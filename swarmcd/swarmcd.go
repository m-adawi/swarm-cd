package swarmcd

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
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

// UpdateStack triggers an update for a specific stack by name.
// Returns an error if the stack is not found.
func UpdateStack(stackName string) error {
	for _, swarmStack := range stacks {
		if swarmStack.name == stackName {
			var waitGroup sync.WaitGroup
			waitGroup.Add(1)
			go updateStackThread(swarmStack, &waitGroup)
			waitGroup.Wait()
			if stackStatus[stackName].Error != "" {
				return errors.New(stackStatus[stackName].Error)
			}
			return nil
		}
	}
	return fmt.Errorf("stack %s not found", stackName)
}

// UpdateAllStacks triggers an update for all configured stacks.
func UpdateAllStacks() {
	var waitGroup sync.WaitGroup
	logger.Info("webhook: updating all stacks...")
	for _, swarmStack := range stacks {
		waitGroup.Add(1)
		go updateStackThread(swarmStack, &waitGroup)
	}
	waitGroup.Wait()
	logger.Info("webhook: done updating all stacks")
}
