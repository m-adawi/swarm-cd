package swarmcd

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/m-adawi/swarm-cd/util"
)

type StackStatus struct {
	Error    string
	Revision string
	RepoURL  string
}

var repoLocks map[string]*sync.Mutex = make(map[string]*sync.Mutex)

var stackStatus map[string]*StackStatus = map[string]*StackStatus{}

var config *util.Config = &util.Configs

var logger *slog.Logger = util.Logger

var repos map[string]*git.Repository = make(map[string]*git.Repository)

var repoAuth map[string]*http.BasicAuth = make(map[string]*http.BasicAuth)

var dockerCli *command.DockerCli

func Init() (err error) {
	err = initRepos()
	if err != nil {
		return err
	}
	err = initStacks()
	if err != nil {
		return err
	}
	err = initDockerCli()
	if err != nil {
		return err
	}
	return
}

func initRepos() (err error) {
	for repoName, repoConfig := range config.RepoConfigs {
		repoPath := path.Join(config.ReposPath, repoName)
		var repo *git.Repository
		cloneOptions := &git.CloneOptions{
			URL:   repoConfig.Url,
			Depth: 1,
		}
		repoAuth[repoName], err = createHTTPBasicAuth(repoName)
		if err != nil {
			return
		}
		cloneOptions.Auth = repoAuth[repoName]
		repo, err = git.PlainClone(repoPath, false, cloneOptions)

		if err != nil {
			if errors.Is(err, git.ErrRepositoryAlreadyExists) {
				repo, err = git.PlainOpen(repoPath)
				if err != nil {
					return fmt.Errorf("could not open existing repo %s: %w", repoName, err)
				}
			} else {
				// we get this error when provided creds are invalid
				// which can mislead users into thinking they
				// haven't provided creds correctly
				if err.Error() == "authentication required" {
					err = fmt.Errorf("authentication failed")
				}
				return fmt.Errorf("could not clone repo %s: %w", repoName, err)
			}
		}
		repos[repoName] = repo
		repoLocks[repoName] = &sync.Mutex{}
	}

	return nil
}

func createHTTPBasicAuth(repoName string) (*http.BasicAuth, error) {
	repoConfig := config.RepoConfigs[repoName]
	// assume repo is public and no auth is required
	if repoConfig.Username == "" && repoConfig.Password == "" && repoConfig.PasswordFile == "" {
		return nil, nil
	}

	if repoConfig.Username == "" {
		return nil, fmt.Errorf("you must set username for the repo %s", repoName)
	}

	if repoConfig.Password == "" && repoConfig.PasswordFile == "" {
		return nil, fmt.Errorf("you must set one of password or password_file properties for the repo %s", repoName)
	}

	var password string
	if repoConfig.Password != "" {
		password = repoConfig.Password
	} else {
		passwordBytes, err := os.ReadFile(repoConfig.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("could not read password file %s for repo %s", repoConfig.PasswordFile, repoName)
		}
		// trim newline and whitespaces
		password = strings.TrimSpace(string(passwordBytes))
	}

	return &http.BasicAuth{
		Username: repoConfig.Username,
		Password: password,
	}, nil
}

func initStacks() error {
	for stack, stackConfig := range config.StackConfigs {
		stackStatus[stack] = &StackStatus{}
		repoConfig, ok := config.RepoConfigs[stackConfig.Repo]
		if !ok {
			return fmt.Errorf("error initializing %s stack, no such repo: %s", stack, stackConfig.Repo)
		}
		stackStatus[stack].RepoURL = repoConfig.Url
	}
	return nil
}

func initDockerCli() (err error) {
	// suppress command outputs (errors are returned as objects)
	nullFile, _ := os.Open("/dev/null")
	defer nullFile.Close()
	dockerCli, err = command.NewDockerCli(command.WithOutputStream(nullFile), command.WithErrorStream(nullFile))
	if err != nil {
		return fmt.Errorf("could not create a docker cli object: %w", err)
	}
	err = dockerCli.Initialize(flags.NewClientOptions())

	if err != nil {
		return fmt.Errorf("could not initialize docker cli object: %w", err)
	}
	return nil
}
