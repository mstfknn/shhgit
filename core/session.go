package core

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

type Session struct {
	sync.Mutex

	Version          string
	Log              *Logger
	Options          *Options
	Config           *Config
	Signatures       []Signature
	Repositories     chan GitResource
	Gists            chan string
	Comments         chan string
	Context          context.Context
	Clients          chan *GitHubClientWrapper
	ExhaustedClients chan *GitHubClientWrapper
	CsvWriter        *csv.Writer
	SearchRegex      *regexp.Regexp
}

var (
	session     *Session
	sessionSync sync.Once
	err         error
)

func (s *Session) Start() {
	s.InitLogger()
	s.InitThreads()
	s.InitSignatures()
	s.InitGitHubClients()
	s.InitCsvWriter()
	s.InitSearchRegex()
}

func (s *Session) InitSearchRegex() {
	if *s.Options.SearchQuery != "" {
		var err error
		s.SearchRegex, err = regexp.Compile(*s.Options.SearchQuery)
		if err != nil {
			s.Log.Fatal("Invalid search query regex '%s': %s", *s.Options.SearchQuery, err)
		}
	}
}

func (s *Session) InitLogger() {
	s.Log = &Logger{}
	s.Log.SetDebug(*s.Options.Debug)
	s.Log.SetSilent(*s.Options.Silent)
}

func (s *Session) InitSignatures() {
	s.Signatures = GetSignatures(s)
}

func (s *Session) InitGitHubClients() {
	if len(*s.Options.Local) <= 0 {
		chanSize := *s.Options.Threads * (len(s.Config.GitHubAccessTokens) + 1)
		s.Clients = make(chan *GitHubClientWrapper, chanSize)
		s.ExhaustedClients = make(chan *GitHubClientWrapper, chanSize)
		for _, token := range s.Config.GitHubAccessTokens {
			ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
			tc := oauth2.NewClient(s.Context, ts)

			client := github.NewClient(tc)
			client.UserAgent = fmt.Sprintf("%s v%s", Name, Version)
			_, _, err := client.Users.Get(s.Context, "")

			if err != nil {
				if _, ok := err.(*github.ErrorResponse); ok {
					s.Log.Warn("Failed to validate token %s[..]: %s", token[:10], err)
					continue
				}
			}

			for i := 0; i <= *s.Options.Threads; i++ {
				s.Clients <- &GitHubClientWrapper{client, token, time.Now().Add(-1 * time.Second)}
			}
		}

		if len(s.Clients) < 1 {
			s.Log.Fatal("No valid GitHub tokens provided. Quitting!")
		}
	}
}

func (s *Session) GetClient() *GitHubClientWrapper {
	for {
		select {
		case client := <-s.Clients:
			s.Log.Debug("Using client with token: %s", client.Token[:10])
			return client

		default:
			// No available clients, check exhausted clients
			exhaustedCount := len(s.ExhaustedClients)
			
			if exhaustedCount > 0 {
				// Collect all exhausted clients to find the one with earliest reset time
				exhaustedClients := make([]*GitHubClientWrapper, 0, exhaustedCount)
				earliestReset := time.Now().Add(24 * time.Hour) // Far future
				var earliestClient *GitHubClientWrapper

				// Drain exhausted clients channel temporarily
				for i := 0; i < exhaustedCount; i++ {
					select {
					case client := <-s.ExhaustedClients:
						exhaustedClients = append(exhaustedClients, client)
						if client.RateLimitedUntil.Before(earliestReset) {
							earliestReset = client.RateLimitedUntil
							earliestClient = client
						}
					default:
						break
					}
				}

				// Put all clients back except the one we'll wait for
				for _, client := range exhaustedClients {
					if client != earliestClient {
						s.ExhaustedClients <- client
					}
				}

				if earliestClient != nil {
					// Check if any client is already available
					if earliestReset.Before(time.Now()) {
						// This client is ready, return it to available pool
						s.Log.Debug("Client %s is ready, returning to pool", earliestClient.Token[:10])
						s.Clients <- earliestClient
						continue
					}

					// Wait for the earliest reset time
					sleepTime := time.Until(earliestReset)
					s.Log.Warn("All GitHub tokens exhausted/rate limited. Earliest reset in %s (token: %s[..])", sleepTime.String(), earliestClient.Token[:10])
					time.Sleep(sleepTime)
					
					// Return this client to available pool
					s.Log.Debug("Returning client %s to pool", earliestClient.Token[:10])
					s.Clients <- earliestClient
					continue
				}
			}

			// No clients available, wait a bit and retry
			s.Log.Debug("Available Clients: %d, Exhausted Clients: %d", len(s.Clients), len(s.ExhaustedClients))
			time.Sleep(time.Millisecond * 1000)
		}
	}
}

// FreeClient returns the GitHub Client to the pool of available,
// non-rate-limited channel of clients in the session
func (s *Session) FreeClient(client *GitHubClientWrapper) {
	if client.RateLimitedUntil.After(time.Now()) {
		s.ExhaustedClients <- client
	} else {
		s.Clients <- client
	}
}

func (s *Session) InitThreads() {
	if *s.Options.Threads == 0 {
		numCPUs := runtime.NumCPU()
		s.Options.Threads = &numCPUs
	}

	runtime.GOMAXPROCS(*s.Options.Threads + 1)
}

func (s *Session) InitCsvWriter() {
	if *s.Options.CsvPath == "" {
		return
	}

	writeHeader := false
	if !PathExists(*s.Options.CsvPath) {
		writeHeader = true
	}

	file, err := os.OpenFile(*s.Options.CsvPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	LogIfError("Could not create/open CSV file", err)

	s.CsvWriter = csv.NewWriter(file)

	if writeHeader {
		s.WriteToCsv([]string{"Repository name", "Signature name", "Matching file", "Matches"})
	}
}

func (s *Session) WriteToCsv(line []string) {
	if *s.Options.CsvPath == "" {
		return
	}

	s.CsvWriter.Write(line)
	s.CsvWriter.Flush()
}

func GetSession() *Session {
	sessionSync.Do(func() {
		session = &Session{
			Context:      context.Background(),
			Repositories: make(chan GitResource, 1000),
			Gists:        make(chan string, 100),
			Comments:     make(chan string, 1000),
		}

		if session.Options, err = ParseOptions(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if session.Config, err = ParseConfig(session.Options); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		session.Start()
	})

	return session
}
