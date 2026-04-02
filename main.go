package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/eth0izzle/shhgit/core"
	"github.com/fatih/color"
)

type MatchEvent struct {
	Url       string
	Matches   []string
	Signature string
	File      string
	Stars     int
	Source    core.GitResourceType
}

var session = core.GetSession()

func ProcessRepositories() {
	threadNum := *session.Options.Threads

	for i := 0; i < threadNum; i++ {
		go func(tid int) {
			for {
				repository := <-session.Repositories

				repo, err := core.GetRepository(session, repository.Id)

				if err != nil {
					session.Log.Warn("Failed to retrieve repository %d: %s", repository.Id, err)
					continue
				}

				if repo.GetPermissions()["pull"] &&
					uint(repo.GetStargazersCount()) >= *session.Options.MinimumStars &&
					uint(repo.GetSize()) < *session.Options.MaximumRepositorySize {

					processRepositoryOrGist(repo.GetCloneURL(), repository.Ref, repo.GetStargazersCount(), core.GITHUB_SOURCE)
				}
			}
		}(i)
	}
}

func ProcessGists() {
	threadNum := *session.Options.Threads

	for i := 0; i < threadNum; i++ {
		go func(tid int) {
			for {
				gistUrl := <-session.Gists
				processRepositoryOrGist(gistUrl, "", -1, core.GIST_SOURCE)
			}
		}(i)
	}
}

func ProcessComments() {
	threadNum := *session.Options.Threads

	for i := 0; i < threadNum; i++ {
			go func(tid int) {
				for {
					commentBody := <-session.Comments
					func() {
						dir := core.GetTempDir(core.GetHash(commentBody))
						defer os.RemoveAll(dir)
						os.WriteFile(filepath.Join(dir, "comment.ignore"), []byte(commentBody), 0600)
						checkSignatures(dir, "ISSUE", 0, core.GITHUB_COMMENT)
					}()
				}
		}(i)
	}
}

func processRepositoryOrGist(url string, ref string, stars int, source core.GitResourceType) {
	dir := core.GetTempDir(core.GetHash(url))
	defer os.RemoveAll(dir) // Ensure directory is removed after processing

	_, err := core.CloneRepository(session, url, ref, dir)

	if err != nil {
		session.Log.Debug("[%s] Cloning failed: %s", url, err.Error())
		return
	}

	session.Log.Debug("[%s] Cloning %s in to %s", url, ref, strings.Replace(dir, *session.Options.TempDirectory, "", -1))
	checkSignatures(dir, url, stars, source)
	
	// Repository will be cleaned up by defer, regardless of whether matches were found
}

func checkSignatures(dir string, url string, stars int, source core.GitResourceType) (matchedAny bool) {
	for _, file := range core.GetMatchingFiles(dir) {
		var (
			matches          []string
			relativeFileName string
		)
		if strings.Contains(dir, *session.Options.TempDirectory) {
			relativeFileName = strings.Replace(file.Path, *session.Options.TempDirectory, "", -1)
		} else {
			relativeFileName = strings.Replace(file.Path, dir, "", -1)
		}

		if session.SearchRegex != nil {
			for _, match := range session.SearchRegex.FindAllSubmatch(file.Contents, -1) {
				matches = append(matches, string(match[0]))
			}

			if matches != nil {
				count := len(matches)
				m := strings.Join(matches, ", ")
				session.Log.Important("[%s] %d %s for %s in file %s: %s", url, count, core.Pluralize(count, "match", "matches"), color.GreenString("Search Query"), relativeFileName, color.YellowString(m))
				session.WriteToCsv([]string{url, "Search Query", relativeFileName, m})
			}
		} else {
			for _, signature := range session.Signatures {
				if matched, part := signature.Match(file); matched {
					if part == core.PartContents {
						if matches = signature.GetContentsMatches(file.Contents); len(matches) > 0 {
							// Filter out matches containing blacklisted strings and remove duplicates
							filteredMatches := make([]string, 0)
							seenMatches := make(map[string]bool)
							for _, match := range matches {
								shouldFilter := false
								for _, blacklistedString := range session.Config.BlacklistedStrings {
									if strings.Contains(strings.ToLower(match), strings.ToLower(blacklistedString)) {
										shouldFilter = true
										break
									}
								}
								if !shouldFilter && !seenMatches[match] {
									seenMatches[match] = true
									filteredMatches = append(filteredMatches, match)
								}
							}
							
							if len(filteredMatches) > 0 {
								count := len(filteredMatches)
								m := strings.Join(filteredMatches, ", ")
								publish(&MatchEvent{Source: source, Url: url, Matches: filteredMatches, Signature: signature.Name(), File: relativeFileName, Stars: stars})
								matchedAny = true

								session.Log.Important("[%s] %d %s for %s in file %s: %s", url, count, core.Pluralize(count, "match", "matches"), color.GreenString(signature.Name()), relativeFileName, color.YellowString(m))
								session.WriteToCsv([]string{url, signature.Name(), relativeFileName, m})
							}
						}
					} else {
						if *session.Options.PathChecks {
							publish(&MatchEvent{Source: source, Url: url, Matches: matches, Signature: signature.Name(), File: relativeFileName, Stars: stars})
							matchedAny = true

							session.Log.Important("[%s] Matching file %s for %s", url, color.YellowString(relativeFileName), color.GreenString(signature.Name()))
							session.WriteToCsv([]string{url, signature.Name(), relativeFileName, ""})
						}

						if *session.Options.EntropyThreshold > 0 && file.CanCheckEntropy() {
							scanner := bufio.NewScanner(bytes.NewReader(file.Contents))

							for scanner.Scan() {
								line := scanner.Text()

								if len(line) > 6 && len(line) < 100 {
									entropy := core.GetEntropy(line)

									if entropy >= *session.Options.EntropyThreshold {
										blacklistedMatch := false

										for _, blacklistedString := range session.Config.BlacklistedStrings {
											if strings.Contains(strings.ToLower(line), strings.ToLower(blacklistedString)) {
												blacklistedMatch = true
											}
										}

										if !blacklistedMatch {
											// Check if this looks like an API key (contains "api" or "key" in context)
											lineLower := strings.ToLower(line)
											signatureName := "High entropy string"
											
											// If line contains "api" or "key" keywords, classify as "API Key"
											if strings.Contains(lineLower, "api") || strings.Contains(lineLower, "key") || 
											   strings.Contains(lineLower, "token") || strings.Contains(lineLower, "secret") {
												signatureName = "API Key"
											}
											
											publish(&MatchEvent{Source: source, Url: url, Matches: []string{line}, Signature: signatureName, File: relativeFileName, Stars: stars})
											matchedAny = true

											session.Log.Important("[%s] Potential secret in %s = %s", url, color.YellowString(relativeFileName), color.GreenString(line))
											session.WriteToCsv([]string{url, signatureName, relativeFileName, line})
										}
									}
								}
							}
						}
					}
				}
			}
		}

		if !matchedAny && len(*session.Options.Local) <= 0 {
			os.Remove(file.Path)
		}
	}
	return
}

func publish(event *MatchEvent) {
	// Check if any match contains blacklisted strings
	for _, blacklistedString := range session.Config.BlacklistedStrings {
		// Check in Matches array
		for _, match := range event.Matches {
			if strings.Contains(strings.ToLower(match), strings.ToLower(blacklistedString)) {
				return // Skip this event if it contains blacklisted string
			}
		}
		// Check in File path
		if strings.Contains(strings.ToLower(event.File), strings.ToLower(blacklistedString)) {
			return // Skip this event if file path contains blacklisted string
		}
	}
	
	// Sanitize fields to prevent stored XSS when rendered in the frontend
	sanitizedMatches := make([]string, len(event.Matches))
	for i, match := range event.Matches {
		sanitizedMatches[i] = html.EscapeString(match)
	}
	sanitizedEvent := &MatchEvent{
		Url:       event.Url,
		Matches:   sanitizedMatches,
		Signature: html.EscapeString(event.Signature),
		File:      html.EscapeString(event.File),
		Stars:     event.Stars,
		Source:    event.Source,
	}

	// todo: implement a modular plugin system to handle the various outputs (console, live, csv, webhooks, etc)
	if len(*session.Options.Live) > 0 {
		data, _ := json.Marshal(sanitizedEvent)
		resp, err := http.Post(*session.Options.Live, "application/json", bytes.NewBuffer(data))
		if err != nil {
			session.Log.Warn("Failed to publish event: %s", err)
			return
		}
		defer resp.Body.Close()
	}
}

func main() {
	session.Log.Info(color.HiBlueString(core.Banner))
	session.Log.Info("\t%s\n", color.HiCyanString(core.Author))
	session.Log.Info("[*] Loaded %s signatures. Using %s worker threads. Temp work dir: %s\n", color.BlueString("%d", len(session.Signatures)), color.BlueString("%d", *session.Options.Threads), color.BlueString(*session.Options.TempDirectory))

	if len(*session.Options.Local) > 0 {
		session.Log.Info("[*] Scanning local directory: %s - skipping public repository checks...", color.BlueString(*session.Options.Local))
		rc := 0
		if checkSignatures(*session.Options.Local, *session.Options.Local, -1, core.LOCAL_SOURCE) {
			rc = 1
		} else {
			session.Log.Info("[*] No matching secrets found in %s!", color.BlueString(*session.Options.Local))
		}
		os.Exit(rc)
	} else {
		if *session.Options.SearchQuery != "" {
			session.Log.Important("Search Query '%s' given. Only returning matching results.", *session.Options.SearchQuery)
		}

		go core.GetRepositories(session)
		go ProcessRepositories()
		go ProcessComments()

		if *session.Options.ProcessGists {
			go core.GetGists(session)
			go ProcessGists()
		}

		spinny := core.ShowSpinner()
		select {}
		spinny()
	}
}
