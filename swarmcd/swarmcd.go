package swarmcd

import (
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/docker/cli/cli/command/stack"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/goccy/go-yaml"
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

	err = rotateConfigsAndSecrets(stackName) 
	if err != nil {
		return
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



func rotateConfigsAndSecrets(stackName string) error {
	stackConfig := config.StackConfigs[stackName]
	composeFile := path.Join(config.ReposPath, stackConfig.Repo, stackConfig.ComposeFile)
	composeFileBytes, err := os.ReadFile(composeFile)
	if err != nil {
		return fmt.Errorf("could not read compose file %s: %w", composeFile, err)
	}
	var composeMap map[string]any
	err = yaml.Unmarshal(composeFileBytes, &composeMap)
	if err != nil {
		return fmt.Errorf("could not parse yaml file %s: %w", composeFile, err)
	}
	
	composeDir := path.Dir(composeFile)
	if configs, ok := composeMap["configs"].(map[string]any); ok {
		err = rotateObjects(configs, composeDir, stackName)
		if err != nil{
			return fmt.Errorf("could not rotate one or more config files of stack %s: %w", stackName, err)
		}
	}
	if secrets, ok := composeMap["secrets"].(map[string]any); ok {
		err = rotateObjects(secrets, composeDir, stackName)
		if err != nil{
			return fmt.Errorf("could not rotate one or more secret files of stack %s: %w", stackName, err)
		}
	}
	
	composeFileBytes, err = yaml.Marshal(composeMap)
	if err != nil {
		return fmt.Errorf("could not store comopse file as yaml after calculating hashes for stack %s", stackName)
	}
	fileInfo, _ := os.Stat(composeFile)
	os.WriteFile(composeFile, composeFileBytes, fileInfo.Mode())
	return nil
}

func rotateObjects(objects map[string]any, objectsDir string, stackName string) error {
	for objectName, object := range objects {
		objectMap, ok := object.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid compose file: %s object must be a map", objectName)
		}
		objectFile, ok := objectMap["file"].(string)
		if !ok {
			return fmt.Errorf("invalid compose file: %s file field must be a string", objectName)
		}
		objectFilePath := path.Join(objectsDir, objectFile)
		configFileBytes, err := os.ReadFile(objectFilePath)
		if err != nil {
			return fmt.Errorf("could not read file %s for rotation: %w", objectFilePath, err)
		}
		hash := fmt.Sprintf("%x", md5.Sum(configFileBytes))[:8]
		objectMap["name"] = stackName + "-" + objectName + "-" + hash
	}
	return nil
}

func GetStackStatus() map[string]*StackStatus {
	return stackStatus
}