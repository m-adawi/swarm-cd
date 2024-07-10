package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var repos map[string]*git.Repository = make(map[string]*git.Repository)

var dockerCli *command.DockerCli

func init() {
	err := initConfigs()
	handleError(err)

	err = initRepos()
	handleError(err)

	err = initDockerCli()
	handleError(err)
}


func initRepos() (err error) {
	for repoName, repoConfig := range config.RepoConfigs {
		repoPath := path.Join(config.ReposPath, repoName)
		var repo *git.Repository
		cloneOptions := &git.CloneOptions{
			URL:      repoConfig.Url,
			Depth: 1,
			Progress: os.Stdout,
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
					return err
				}
			} else  {
				return err
			}
		}
		repos[repoName] = repo
	}

	return nil
}

func initDockerCli() (err error) {
	dockerCli, err = command.NewDockerCli()
	if err != nil { 
		return err
	}
	dockerCli.Initialize(flags.NewClientOptions())
	return nil
}

func handleError(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}