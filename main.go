package main

import (
	"log"
	"path"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/command/stack"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	// "github.com/docker/docker/cli/cli/command/stack"
	// "github.com/docker/docker/cli/cli/flags"
)


type StackConfig struct {
	Name string
	Repo string
	Branch string
	ComposeFile string `json:"compose_file"`
}

type RepoConfig struct {
	Url	string
	Username string
	Password string
}

var stackConfigs []StackConfig

var repoConfigs map[string]RepoConfig

var repos map[string] *git.Repository = make(map[string]*git.Repository)

const reposPath string = "repos/"

var dockerCli *command.DockerCli

const updateInterval = 120

func main() {
	for {
		for _, stackConfig := range stackConfigs {
			updateStack(stackConfig)
		}
		time.Sleep(updateInterval * time.Second)
	}

}


func updateStack(stackConfig StackConfig) (err error) {
	err = pullChanges(stackConfig)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Print(err)
		return 
	}
	cmd := stack.NewStackCommand(dockerCli)
	cmd.SetArgs([]string{"deploy", "-c", path.Join(reposPath, stackConfig.Repo, stackConfig.ComposeFile), stackConfig.Name})
	err = cmd.Execute()
	if err != nil {
		log.Print(err)
		return 
	}
	return nil
}


func pullChanges(stackConfig StackConfig) error {
	repoConfig := repoConfigs[stackConfig.Repo]
	repo := repos[stackConfig.Repo]
	branch := stackConfig.Branch
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil { 
		return err
	}
	pullOptions := &git.PullOptions{
		ReferenceName: plumbing.NewBranchReferenceName(branch),
	}
	if repoConfig.Password != "" && repoConfig.Username != "" {
		pullOptions.Auth = &http.BasicAuth{
			Username: repoConfig.Username,
			Password: repoConfig.Password,
		}
	}
	err = w.Pull(pullOptions)
	return err
}


