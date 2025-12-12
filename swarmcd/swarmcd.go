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
	for {
		var waitGroup sync.WaitGroup
		logger.Info("updating stacks...")
		for _, swarmStack := range stacks {
			waitGroup.Add(1)
			go updateStackThread(swarmStack, &waitGroup)
		}
		waitGroup.Wait()
		logger.Info("waiting for the update interval")
		time.Sleep(time.Duration(Config.UpdateInterval) * time.Second)
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

func UpdateAllStackInRepo(repoName string) {
	logger.Info("Update webhook stacks for repo " + repoName)
	for _, stack := range stacks {
		if stack.repo.name == repoName {
			logger.Info("Start update " + stack.name)
			stack.repo.lock.Lock()
			_, err := stack.updateStack()
			if err != nil {
				logger.Error(err.Error())
			}
			stack.repo.lock.Unlock()
		}
	}
}

func GetStackStatus() map[string]*StackStatus {
	return stackStatus
}
