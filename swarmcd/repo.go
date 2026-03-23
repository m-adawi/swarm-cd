package swarmcd

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

type stackRepo struct {
	name          string
	lock          *sync.Mutex
	url           string
	gitRepoObject *git.Repository
	auth          *http.BasicAuth
	path          string
}

func newStackRepo(name string, path string, url string, auth *http.BasicAuth) (*stackRepo, error) {
	var repo *git.Repository
	cloneOptions := &git.CloneOptions{
		URL:  url,
		Auth: auth,
	}
	repo, err := git.PlainClone(path, false, cloneOptions)

	if err != nil {
		if errors.Is(err, git.ErrRepositoryAlreadyExists) {
			repo, err = git.PlainOpen(path)
			if err != nil {
				return nil, fmt.Errorf("could not open existing repo %s: %w", name, err)
			}
		} else {
			// we get this error when provided creds are invalid
			// which can mislead users into thinking they
			// haven't provided creds correctly
			if err.Error() == "authentication required" {
				err = fmt.Errorf("authentication failed")
			}
			return nil, fmt.Errorf("could not clone repo %s: %w", name, err)
		}
	}
	return &stackRepo{
		name:          name,
		path:          path,
		url:           url,
		auth:          auth,
		lock:          &sync.Mutex{},
		gitRepoObject: repo,
	}, nil
}

func (repo *stackRepo) pullChanges(branch string) (revision string, err error) {
	log := logger.With(slog.String("repo", repo.name), slog.String("branch", branch))

	log.Debug("getting repo worktree...")
	workTree, err := repo.gitRepoObject.Worktree()
	if err != nil {
		return "", fmt.Errorf("could not get %s repo worktree: %w", repo.name, err)
	}

	log.Debug("checking out branch...")
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/remotes/origin/" + branch),
		Force:  true,
	})
	if err != nil {
		return "", fmt.Errorf("could not checkout branch %s in %s: %w", branch, repo.name, err)
	}

	pullOptions := &git.PullOptions{
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		RemoteName:    "origin",
		Auth:          repo.auth,
	}

	log.Debug("pulling changes...")
	err = workTree.Pull(pullOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		// we get this error when provided creds are invalid
		// which can mislead users into thinking they
		// haven't provided creds correctly
		if err.Error() == "authentication required" {
			err = fmt.Errorf("authentication failed")
		}
		return "", fmt.Errorf("could not pull %s branch in %s repo: %w", branch, repo.name, err)
	}

	log.Debug("getting revision...")
	ref, err := repo.gitRepoObject.Head()
	if err != nil {
		return "", fmt.Errorf("could not get HEAD commit hash of %s branch in %s repo: %w", branch, repo.name, err)
	}
	// return HEAD commit short hash
	return ref.Hash().String()[:8], nil
}

func (repo *stackRepo) fetchTag(tag string) (revision string, err error) {
	log := logger.With(slog.String("repo", repo.name), slog.String("tag", tag))

	log.Debug("getting repo worktree...")
	workTree, err := repo.gitRepoObject.Worktree()
	if err != nil {
		return "", fmt.Errorf("could not get %s repo worktree: %w", repo.name, err)
	}

	// Fetch the specific tag from remote
	log.Debug("fetching tag...")
	fetchOptions := &git.FetchOptions{
		RemoteName: "origin",
		Auth:       repo.auth,
		RefSpecs:   []gitconfig.RefSpec{gitconfig.RefSpec("+refs/tags/" + tag + ":refs/tags/" + tag)},
		Force:      true,
	}
	err = repo.gitRepoObject.Fetch(fetchOptions)
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		if err.Error() == "authentication required" {
			err = fmt.Errorf("authentication failed")
		}
		return "", fmt.Errorf("could not fetch tag %s in %s repo: %w", tag, repo.name, err)
	}

	log.Debug("checking out tag...")
	tagRef := plumbing.ReferenceName("refs/tags/" + tag)
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: tagRef,
		Force:  true,
	})
	if err != nil {
		return "", fmt.Errorf("could not checkout tag %s in %s: %w", tag, repo.name, err)
	}

	log.Debug("getting revision...")
	ref, err := repo.gitRepoObject.Head()
	if err != nil {
		return "", fmt.Errorf("could not get HEAD commit hash of tag %s in %s repo: %w", tag, repo.name, err)
	}
	// return HEAD commit short hash
	return ref.Hash().String()[:8], nil
}
