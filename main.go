package main

import (
	"log"
	"path"
	"time"
	"github.com/docker/cli/cli/command/stack"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)



func main() {
	for {
		for stackName, _ := range config.StackConfigs {
			updateStack(stackName)
		}
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)
	}

}

func updateStack(stackName string) (err error) {
	err = pullChanges(stackName)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Print(err)
		return
	}
	stackConfig := config.StackConfigs[stackName]
	cmd := stack.NewStackCommand(dockerCli)
	cmd.SetArgs([]string{
		"deploy", "-c",
		path.Join(config.ReposPath, stackConfig.Repo, stackConfig.ComposeFile),
		stackName,
	})
	err = cmd.Execute()
	if err != nil {
		log.Print(err)
		return
	}
	return nil
}

func pullChanges(stackName string) error {
	stackConfig := config.StackConfigs[stackName]
	repoConfig := config.RepoConfigs[stackConfig.Repo]
	branch := stackConfig.Branch
	workTree, err := repos[stackConfig.Repo].Worktree()
	if err != nil {
		return err
	}
	err = workTree.Checkout(&git.CheckoutOptions{
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
	err = workTree.Pull(pullOptions)
	return err
}
