package swarmcd

import (
	"fmt"
	"sync"
	"time"
)

var stackStatus map[string]*StackStatus = map[string]*StackStatus{}
var stacks []*swarmStack

func Run() {
	logger.Info("starting SwarmCD")

	err := initDB(getDBFilePath())
	defer closeDB()

	if err != nil {
		logger.Error(fmt.Sprintf("failed to initialize SwarmCD DB: %s", err))
		return
	}

	for {
		var waitGroup sync.WaitGroup
		logger.Info("updating stacks...")
		for _, swarmStack := range stacks {
			waitGroup.Add(1)
			go updateStackThread(swarmStack, &waitGroup)
		}
		waitGroup.Wait()
		logger.Info("waiting for the update interval")
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)
	}
}

func updateStackThread(swarmStack *swarmStack, waitGroup *sync.WaitGroup) {
	repoLock := swarmStack.repo.lock
	repoLock.Lock()
	defer repoLock.Unlock()
	defer waitGroup.Done()

	logger.Info(fmt.Sprintf("updating %s stack", swarmStack.name))
	revision, err := swarmStack.updateStack()

	if err != nil {
		stackStatus[swarmStack.name].Error = err.Error()
		logger.Error(err.Error())
		return
	}

	stackStatus[swarmStack.name].Error = ""
	stackStatus[swarmStack.name].Revision = revision
	logger.Info(fmt.Sprintf("done updating %s stack", swarmStack.name))
}

func GetStackStatus() map[string]*StackStatus {
	return stackStatus
}
