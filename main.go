package main

import (
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/docker/cli/cli/command/stack"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

var repoLocks map[string]*sync.Mutex = make(map[string]*sync.Mutex)

func main() {
	logger.Info("starting SwarmCD")
	for {
		var waitGroup sync.WaitGroup
		logger.Info("updating stacks...")	
		for stackName := range config.StackConfigs {
			waitGroup.Add(1)
			go updateStackThread(&waitGroup, stackName)
		}
		waitGroup.Wait()
		logger.Info("waiting for the update interval")
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)
	}

}

func updateStackThread(waitGroup *sync.WaitGroup, stackName string) {
	repoLock := repoLocks[config.StackConfigs[stackName].Repo]
	repoLock.Lock()
	defer repoLock.Unlock()
	logger.Info(fmt.Sprintf("updating %s stack", stackName))
	err := updateStack(stackName)
	if err != nil{
		logger.Error(err.Error())
	} else {
		logger.Info(fmt.Sprintf("done updating %s stack", stackName))
	}
	waitGroup.Done()
}

func updateStack(stackName string) (err error) {
	err = pullChanges(stackName)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return
	}
	stackConfig := config.StackConfigs[stackName]
	cmd := stack.NewStackCommand(dockerCli)
	cmd.SetArgs([]string{
		"deploy", "--detach", "-c",
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
	// repos[stackConfig.Repo].//Branch(branch)//.Fetch(&git.FetchOptions{})
	workTree, err := repos[stackConfig.Repo].Worktree()
	if err != nil {
		return fmt.Errorf("could not get %s repo worktree: %w", stackConfig.Repo, err)
	}
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/remotes/origin/" + branch),
	})
	if err != nil {
		return fmt.Errorf("could not checkout branch %s: %w", branch, err)
	}
	pullOptions := &git.PullOptions{
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		RemoteName: "origin",
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
