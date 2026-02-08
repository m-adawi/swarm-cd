package swarmcd

import (
	"fmt"
	"sync"
	"time"
)

var stackStatus = map[string]*StackStatus{}
var stacks []*swarmStack

func Run() {
	logger.Info("starting SwarmCD")
	for {
		logger.Debug("starting update loop")
		var waitGroup sync.WaitGroup
		stacksChannel := make(chan *swarmStack, len(stacks))

		// Start worker pool
		logger.Debug(fmt.Sprintf("worker count: %v", config.Concurrency))
		for range config.Concurrency {
			go worker(stacksChannel, &waitGroup)
		}

		// Send stacks to workers
		for _, swarmStack := range stacks {
			logger.Debug(fmt.Sprintf("Queueing stack %v for update", swarmStack.name))
			waitGroup.Add(1)
			stacksChannel <- swarmStack
		}
		close(stacksChannel)

		// Wait for all workers to complete
		waitGroup.Wait()

		logger.Info("waiting for the update interval")
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)
	}
}

func worker(stacks <-chan *swarmStack, waitGroup *sync.WaitGroup) {
	for swarmStack := range stacks {
		updateStackThread(swarmStack)
		waitGroup.Done()
	}
}

func updateStackThread(swarmStack *swarmStack) {
	repoLock := swarmStack.repo.lock
	repoLock.Lock()
	defer repoLock.Unlock()

	logger.Debug(fmt.Sprintf("%s checking if stack needs to be updated", swarmStack.name))
	revision, err := swarmStack.updateStack()
	if err != nil {
		stackStatus[swarmStack.name].Error = err.Error()
		logger.Error(err.Error())
		return
	}

	stackStatus[swarmStack.name].Error = ""
	stackStatus[swarmStack.name].Revision = revision
	logger.Debug(fmt.Sprintf("%s stack updates check done", swarmStack.name))
}

func GetStackStatus() map[string]*StackStatus {
	return stackStatus
}
