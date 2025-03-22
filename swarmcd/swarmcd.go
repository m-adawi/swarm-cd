package swarmcd

import (
	"fmt"
	"github.com/m-adawi/swarm-cd/util"
	"sync"
	"time"
)

var stackStatus = map[string]*StackStatus{}
var stacks = map[string]*swarmStack{}

func Run() {
	logger.Info("starting SwarmCD")
	for {
		var waitGroup sync.WaitGroup
		logger.Info("updating stacks...")
		for _, swarmStack := range stacks {
			logger.Debug(fmt.Sprintf("Starting go routine for %v", swarmStack.name))
			waitGroup.Add(1)
			go updateStackThread(swarmStack, &waitGroup)
		}
		waitGroup.Wait()
		logger.Info("waiting for the update interval")
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)

		logger.Info("check if new repos or new stacks are available")
		updateStackConfigs()
	}
}

func updateStackConfigs() {
	err := util.LoadConfigs()
	if err != nil {
		logger.Info("Error calling loadConfig again: %v", err)
		return
	}

	err = initRepos()
	if err != nil {
		logger.Info("Error calling initRepos again: %v", err)
	}

	err = initStacks()
	if err != nil {
		logger.Info("Error calling initStacks again: %v", err)
	}
}

func updateStackThread(swarmStack *swarmStack, waitGroup *sync.WaitGroup) {
	repoLock := swarmStack.repo.lock
	repoLock.Lock()
	defer repoLock.Unlock()
	defer waitGroup.Done()

	logger.Info(fmt.Sprintf("%s updating stack", swarmStack.name))
	stackMetadata, err := swarmStack.updateStack()
	if err != nil {
		stackStatus[swarmStack.name].Error = err.Error()
		logger.Error(err.Error())
		return
	}

	stackStatus[swarmStack.name].Error = ""
	stackStatus[swarmStack.name].Revision = stackMetadata.repoRevision
	stackStatus[swarmStack.name].DeployedStackRevision = stackMetadata.deployedStackRevision
	stackStatus[swarmStack.name].DeployedAt = stackMetadata.deployedAt.Format(time.RFC3339)
	logger.Info(fmt.Sprintf("%s done updating stack", swarmStack.name))
}

func GetStackStatus() map[string]*StackStatus {
	return stackStatus
}
