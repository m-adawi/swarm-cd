package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"sigs.k8s.io/yaml"
	// "github.com/docker/docker/cli/cli/command/stack"
	// "github.com/docker/docker/cli/cli/flags"
)

func init() {
	err := initStacks()
	handleError(err)

	err = initRepos()
	handleError(err)

	err = initDockerCli()
	handleError(err)
}

func initStacks() error {
    stackFile, err := os.Open("stacks.yaml") 
	if err != nil {
		return err
	}
	defer stackFile.Close()
	
	stackFileBytes, err := io.ReadAll(stackFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(stackFileBytes, &stackConfigs)
	if err != nil {
		return err
	}
	return nil
}

func initRepos() (err error) {
	reposFile, err := os.Open("repos.yaml") 
	if err != nil {
		return err
	}
	defer reposFile.Close()
	
	reposFileBytes, err := io.ReadAll(reposFile)
	if err != nil {
		return err
	}
	
	err = yaml.Unmarshal(reposFileBytes, &repoConfigs)
	if err != nil {
		return err
	}

	for repoName, repoConfig := range repoConfigs {
		repoPath := path.Join(reposPath, repoName)
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
		fmt.Print(err)
		os.Exit(1)
	}
}