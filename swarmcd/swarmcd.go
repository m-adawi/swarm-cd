package swarmcd

import (
	"fmt"
	"sync"
	"time"
)

var stackStatus map[string]*StackStatus = map[string]*StackStatus{}
var stacks []*swarmStack

// stateMu protects stackStatus and stacks from concurrent access.
// Lock ordering: stateMu must never be acquired while repo.lock is held.
// Acquiring stateMu independently (without any repo.lock) is always safe.
var stateMu sync.RWMutex

func Run() {
	logger.Info("starting SwarmCD")
	for {
		var waitGroup sync.WaitGroup
		logger.Info("updating stacks...")
		stateMu.RLock()
		currentStacks := make([]*swarmStack, len(stacks))
		copy(currentStacks, stacks)
		stateMu.RUnlock()
		for _, swarmStack := range currentStacks {
			waitGroup.Add(1)
			go updateStackThread(swarmStack, &waitGroup)
		}
		waitGroup.Wait()
		logger.Info("waiting for the update interval")
		time.Sleep(time.Duration(config.UpdateInterval) * time.Second)
	}
}

func updateStackThread(swarmStack *swarmStack, waitGroup *sync.WaitGroup) {
	defer waitGroup.Done()

	repoLock := swarmStack.repo.lock
	repoLock.Lock()
	logger.Info(fmt.Sprintf("updating %s stack", swarmStack.name))
	revision, err := swarmStack.updateStack()
	repoLock.Unlock() // Release repo.lock BEFORE acquiring stateMu

	now := time.Now()

	stateMu.Lock()
	status := stackStatus[swarmStack.name]
	if err != nil {
		status.Error = err.Error()
	} else {
		// Update timestamps only when the revision actually changes
		if revision != status.Revision {
			status.LastChangeAt = &now
			status.LastDeployedAt = &now
		}
		status.Error = ""
		status.Revision = revision
	}
	stateMu.Unlock()

	if err != nil {
		logger.Error(err.Error())
		return
	}
	logger.Info(fmt.Sprintf("done updating %s stack", swarmStack.name))
}

// GetRuntimeInfo returns a copy of the instance's runtime metadata.
// Protected by stateMu for consistency, though in practice runtimeInfo
// is only written once during Init() before Run() starts.
func GetRuntimeInfo() RuntimeInfo {
	stateMu.RLock()
	defer stateMu.RUnlock()
	return runtimeInfo
}

// GetStackStatus returns a snapshot of all stack statuses under a read lock.
// Pointer fields (LastChangeAt, LastDeployedAt) are deep-copied so callers
// cannot mutate the original values.
func GetStackStatus() map[string]*StackStatus {
	stateMu.RLock()
	defer stateMu.RUnlock()
	snapshot := make(map[string]*StackStatus, len(stackStatus))
	for k, v := range stackStatus {
		cp := *v
		if v.LastChangeAt != nil {
			t := *v.LastChangeAt
			cp.LastChangeAt = &t
		}
		if v.LastDeployedAt != nil {
			t := *v.LastDeployedAt
			cp.LastDeployedAt = &t
		}
		snapshot[k] = &cp
	}
	return snapshot
}
