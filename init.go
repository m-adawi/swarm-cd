package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var repos map[string]*git.Repository = make(map[string]*git.Repository)

var dockerCli *command.DockerCli

func init() {
	err := initConfigs()
	handleInitError(err)

	err = initRepos()
	handleInitError(err)

	err = initDockerCli()
	handleInitError(err)
}


func initRepos() (err error) {
	for repoName, repoConfig := range config.RepoConfigs {
		repoPath := path.Join(config.ReposPath, repoName)
		var repo *git.Repository
		cloneOptions := &git.CloneOptions{
			URL:      repoConfig.Url,
			Depth: 1,
		}
		if repoConfig.Password != "" && repoConfig.Username != "" {
			cloneOptions.Auth = &http.BasicAuth{
				Username: repoConfig.Username,
				Password: repoConfig.Password,
			}
		}
		repo, err = git.PlainClone(repoPath, false, cloneOptions)		

		if err != nil {
			if errors.Is(err, git.ErrRepositoryAlreadyExists) {
				repo, err = git.PlainOpen(repoPath)
				if err != nil {
					return fmt.Errorf("could not open existing repo %s: %w", repoName, err)
				}
			} else  {
				return fmt.Errorf("could not clone repo %s: %w", repoName, err)
			}
		}
		repos[repoName] = repo
		repoLocks[repoName] = &sync.Mutex{}
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

func handleInitError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}