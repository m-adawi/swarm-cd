package swarmcd

import (
	"fmt"
	"sync"
	"testing"
)

// TestConcurrentStatusRead spawns N goroutines reading GetStackStatus() while
// reconciliation-style goroutines write to stackStatus. This test is designed
// to be run with -race to detect data races.
// This test must not use t.Parallel() — it mutates package-level globals.
func TestConcurrentStatusRead(t *testing.T) {
	// Set up test state: populate stackStatus and stacks with test data
	stateMu.Lock()
	oldStatus := stackStatus
	oldStacks := stacks
	stackStatus = map[string]*StackStatus{}
	stacks = nil

	const numStacks = 5
	repos := make([]*stackRepo, numStacks)
	for i := 0; i < numStacks; i++ {
		name := fmt.Sprintf("test-stack-%d", i)
		repo := &stackRepo{name: name, path: "/tmp/test", url: "http://example.com", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
		repos[i] = repo
		s := newSwarmStack(name, repo, "main", "docker-compose.yaml", nil, "", false)
		stacks = append(stacks, s)
		stackStatus[name] = &StackStatus{
			RepoURL:  "http://example.com",
			Revision: "abc12345",
		}
	}
	stateMu.Unlock()

	// Restore original state when done
	defer func() {
		stateMu.Lock()
		stackStatus = oldStatus
		stacks = oldStacks
		stateMu.Unlock()
	}()

	var wg sync.WaitGroup
	const numReaders = 10
	const numWriters = 5
	const iterations = 100

	// Start reader goroutines that call GetStackStatus() concurrently
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				status := GetStackStatus()
				// Access the returned snapshot to ensure it's a real copy
				for k, v := range status {
					_ = k
					_ = v.Error
					_ = v.Revision
					_ = v.RepoURL
				}
			}
		}()
	}

	// Start writer goroutines that simulate reconciliation writes
	// These follow the correct lock ordering: repo.lock first, then stateMu
	for i := 0; i < numWriters; i++ {
		stackIdx := i % numStacks
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("test-stack-%d", idx)
			repo := repos[idx]
			for j := 0; j < iterations; j++ {
				// Simulate lock ordering: acquire repo.lock, do work, release, then acquire stateMu
				repo.lock.Lock()
				// Simulate some work under repo lock
				revision := fmt.Sprintf("%08x", j)
				repo.lock.Unlock()

				stateMu.Lock()
				stackStatus[name].Revision = revision
				stackStatus[name].Error = ""
				stateMu.Unlock()
			}
		}(stackIdx)
	}

	wg.Wait()
}

// TestGetStackStatusReturnsSnapshot verifies that GetStackStatus returns a
// copy so mutations to the returned map don't affect the original.
// This test must not use t.Parallel() — it mutates package-level globals.
func TestGetStackStatusReturnsSnapshot(t *testing.T) {
	stateMu.Lock()
	oldStatus := stackStatus
	oldStacks := stacks
	stackStatus = map[string]*StackStatus{
		"snap-test": {
			Error:    "",
			Revision: "aabb1122",
			RepoURL:  "http://example.com",
		},
	}
	stacks = nil
	stateMu.Unlock()

	defer func() {
		stateMu.Lock()
		stackStatus = oldStatus
		stacks = oldStacks
		stateMu.Unlock()
	}()

	snapshot := GetStackStatus()

	// Mutate the snapshot
	snapshot["snap-test"].Revision = "MUTATED"
	snapshot["extra"] = &StackStatus{Revision: "bogus"}

	// Verify original is unchanged
	stateMu.RLock()
	if stackStatus["snap-test"].Revision != "aabb1122" {
		t.Errorf("original stackStatus was mutated: got revision %q, want %q", stackStatus["snap-test"].Revision, "aabb1122")
	}
	if _, exists := stackStatus["extra"]; exists {
		t.Error("original stackStatus should not contain 'extra' key")
	}
	stateMu.RUnlock()
}

// TestLockOrderingNoDeadlock verifies that the lock ordering (release repo.lock
// before acquiring stateMu) does not cause deadlock when multiple goroutines
// follow the pattern simultaneously.
// This test must not use t.Parallel() — it mutates package-level globals.
func TestLockOrderingNoDeadlock(t *testing.T) {
	stateMu.Lock()
	oldStatus := stackStatus
	oldStacks := stacks
	stackStatus = map[string]*StackStatus{}
	stacks = nil

	const numStacks = 3
	repos := make([]*stackRepo, numStacks)
	for i := 0; i < numStacks; i++ {
		name := fmt.Sprintf("deadlock-test-%d", i)
		repo := &stackRepo{name: name, path: "/tmp/test", url: "http://example.com", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
		repos[i] = repo
		stacks = append(stacks, newSwarmStack(name, repo, "main", "docker-compose.yaml", nil, "", false))
		stackStatus[name] = &StackStatus{
			RepoURL: "http://example.com",
		}
	}
	stateMu.Unlock()

	defer func() {
		stateMu.Lock()
		stackStatus = oldStatus
		stacks = oldStacks
		stateMu.Unlock()
	}()

	var wg sync.WaitGroup

	// Simulate multiple reconciliation goroutines with correct lock ordering
	for i := 0; i < numStacks; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("deadlock-test-%d", idx)
			repo := repos[idx]
			for j := 0; j < 50; j++ {
				// Correct ordering: repo.lock then release, then stateMu
				repo.lock.Lock()
				revision := fmt.Sprintf("%08x", j)
				repo.lock.Unlock()

				stateMu.Lock()
				stackStatus[name].Revision = revision
				stateMu.Unlock()
			}
		}(i)
	}

	// Concurrent readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = GetStackStatus()
			}
		}()
	}

	wg.Wait()
}
