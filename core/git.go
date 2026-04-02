package core

import (
	"context"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

type GitResourceType int

const (
	LOCAL_SOURCE GitResourceType = iota
	GITHUB_SOURCE
	GITHUB_COMMENT
	GIST_SOURCE
	BITBUCKET_SOURCE
	GITLAB_SOURCE
)

type GitResource struct {
	Id   int64
	Type GitResourceType
	Url  string
	Ref  string
}

func CloneRepository(session *Session, url string, ref string, dir string) (*git.Repository, error) {
	timeout := time.Duration(*session.Options.CloneRepositoryTimeout) * time.Second
	localCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	session.Log.Debug("[%s] Cloning %s in to %s", url, ref, strings.Replace(dir, *session.Options.TempDirectory, "", -1))
	opts := &git.CloneOptions{
		Depth:             1,
		RecurseSubmodules: git.NoRecurseSubmodules,
		URL:               url,
		SingleBranch:      true,
		Tags:              git.NoTags,
		Progress:          nil,
	}

	if ref != "" {
		opts.ReferenceName = plumbing.ReferenceName(ref)
	}

	repository, err := git.PlainCloneContext(localCtx, dir, false, opts)

	if err != nil {
		session.Log.Debug("[%s] Cloning failed: %s", url, err.Error())
		return nil, err
	}

	return repository, nil
}
