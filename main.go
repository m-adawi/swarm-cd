package main

import (
	"fmt"
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
		for stackName := range config.StackConfigs {
			err := updateStack(stackName)
			logIfError(err)
		}
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)
	}

}

func updateStack(stackName string) (err error) {
	err = pullChanges(stackName)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return
	}
	stackConfig := config.StackConfigs[stackName]
	cmd := stack.NewStackCommand(dockerCli)
	cmd.SetArgs([]string{
		"deploy", "-c",
		path.Join(config.ReposPath, stackConfig.Repo, stackConfig.ComposeFile),
		stackName,
	})
	// To stop printing errors and 
	// usage message to stdout
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err = cmd.Execute()
	if err != nil {
		return fmt.Errorf("could not deploy stack %s: %s", stackName, err.Error())
	}
	return 
}

func pullChanges(stackName string) (err error) {
	stackConfig := config.StackConfigs[stackName]
	repoConfig := config.RepoConfigs[stackConfig.Repo]
	branch := stackConfig.Branch
	workTree, err := repos[stackConfig.Repo].Worktree()
	if err != nil {
		return fmt.Errorf("could not get %s repo worktree: %w", stackConfig.Repo, err)
	}
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(branch),
	})
	if err != nil {
		return fmt.Errorf("could not checkout branch %s: %w", branch, err)
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
	if err != nil {
		return fmt.Errorf("could not pull %s branch in %s repo: %w", branch, stackConfig.Repo,  err)
	}
	return 
}


func logIfError(err error) {
	if err != nil {
		log.Println(err)
	}
}