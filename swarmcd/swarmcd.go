package swarmcd

import (
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/docker/cli/cli/command/stack"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/m-adawi/swarm-cd/util"
)


func Run() {
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
	defer waitGroup.Done()

	logger.Info(fmt.Sprintf("updating %s stack", stackName))
	revision, err := updateStack(stackName)
	if err != nil{
		stackStatus[stackName].Error = err.Error()
		logger.Error(err.Error())
		return
	}

	stackStatus[stackName].Error = ""
	stackStatus[stackName].Revision = revision
	logger.Info(fmt.Sprintf("done updating %s stack", stackName))
}

func updateStack(stackName string) (revision string, err error) {
	revision, err = pullChanges(stackName)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return
	}

	err = decryptSopsFiles(stackName)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt one or more sops files for %s stack: %w", stackName, err)
	}

	err = deployStack(stackName) 
	if err != nil {
		return
	}
	return 
}

func pullChanges(stackName string) (revision string, err error) {
	stackConfig := config.StackConfigs[stackName]
	branch := stackConfig.Branch
	repo := repos[stackConfig.Repo]
	
	workTree, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("could not get %s repo worktree: %w", stackConfig.Repo, err)
	}
	
	err = workTree.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/remotes/origin/" + branch),
		Force: true,
	})
	if err != nil {
		return "", fmt.Errorf("could not checkout branch %s in %s: %w", branch, stackConfig.Repo, err)
	}
	
	pullOptions := &git.PullOptions{
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		RemoteName: "origin",
		Auth: repoAuth[stackConfig.Repo],
	}

	err = workTree.Pull(pullOptions)
	if err != nil {
		// we get this error when provided creds are invalid
		// which can mislead users into thinking they 
		// haven't provided creds correctly
		if err.Error() == "authentication required" {
			err = fmt.Errorf("authentication failed")
		}
		return "", fmt.Errorf("could not pull %s branch in %s repo: %w", branch, stackConfig.Repo,  err)
	}
	
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("could not get HEAD commit hash of %s branch in %s repo: %w", branch, stackConfig.Repo, err)
	}
	// return HEAD commit short hash
	return ref.Hash().String()[:8], nil
}

func decryptSopsFiles(stackName string) (err error) {
	stackConfig := config.StackConfigs[stackName]
	for _, sopsFile := range stackConfig.SopsFiles {
		err = util.DecryptFile(path.Join(config.ReposPath, stackConfig.Repo, sopsFile))
		if err != nil {
			return
		}
	}
	return
}


func deployStack(stackName string) error {
	stackConfig := config.StackConfigs[stackName]
	cmd := stack.NewStackCommand(dockerCli)
	cmd.SetArgs([]string{
		"deploy", "--detach", "--with-registry-auth", "-c",
		path.Join(config.ReposPath, stackConfig.Repo, stackConfig.ComposeFile),
		stackName,
	})
	// To stop printing errors and 
	// usage message to stdout
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err != nil {
		return fmt.Errorf("could not deploy stack %s: %s", stackName, err.Error())
	}
	return nil
}

func GetStackStatus() map[string]*StackStatus {
	return stackStatus
}