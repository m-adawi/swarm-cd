// test_helpers.go lives in the swarmcd package (not a _test.go file) so that
// test files in other packages (e.g., web/) can call these helpers to safely
// swap internal state. This is intentional — SwarmCD is a standalone binary,
// not a library, so the exported surface area is not a concern.
package swarmcd

import (
	"sync"

	"github.com/m-adawi/swarm-cd/util"
)

// SetStackStatusForTest replaces the internal stackStatus map with the given
// test data and returns a cleanup function that restores the original state.
// This is intended for use in tests only.
func SetStackStatusForTest(testStatus map[string]*StackStatus) func() {
	stateMu.Lock()
	oldStatus := stackStatus
	oldStacks := stacks
	stackStatus = testStatus
	stacks = nil
	stateMu.Unlock()

	return func() {
		stateMu.Lock()
		stackStatus = oldStatus
		stacks = oldStacks
		stateMu.Unlock()
	}
}

// SetRuntimeInfoForTest replaces the internal runtimeInfo with the given
// test data and returns a cleanup function that restores the original state.
// This is intended for use in tests only.
func SetRuntimeInfoForTest(info RuntimeInfo) func() {
	stateMu.Lock()
	old := runtimeInfo
	runtimeInfo = info
	stateMu.Unlock()
	return func() {
		stateMu.Lock()
		runtimeInfo = old
		stateMu.Unlock()
	}
}

// TestStackDef describes a stack for test setup purposes.
type TestStackDef struct {
	Name        string
	RepoName    string
	RepoPath    string
	RepoURL     string
	Branch      string
	Tag         string
	ComposeFile string
}

// SetFullStateForTest builds and replaces all internal state (stackStatus,
// stacks, repos, and util.Configs) from the given definitions. Returns a
// cleanup function that restores the original state.
func SetFullStateForTest(defs []TestStackDef) func() {
	oldRepoConfigs := config.RepoConfigs
	oldStackConfigs := config.StackConfigs

	newStatus := make(map[string]*StackStatus)
	newRepos := make(map[string]*stackRepo)
	newRepoCfgs := make(map[string]*util.RepoConfig)
	newStackCfgs := make(map[string]*util.StackConfig)
	var newStacks []*swarmStack

	for _, d := range defs {
		// Create repo if not already created
		if _, ok := newRepos[d.RepoName]; !ok {
			newRepos[d.RepoName] = &stackRepo{
				name:          d.RepoName,
				path:          d.RepoPath,
				url:           d.RepoURL,
				auth:          nil,
				lock:          &sync.Mutex{},
				gitRepoObject: nil,
			}
			newRepoCfgs[d.RepoName] = &util.RepoConfig{Url: d.RepoURL}
		}

		repo := newRepos[d.RepoName]

		var refType, refValue string
		if d.Tag != "" {
			refType = "tag"
			refValue = d.Tag
		} else if d.Branch != "" {
			refType = "branch"
			refValue = d.Branch
		} else {
			refType = "branch"
			refValue = "main"
		}

		newStatus[d.Name] = &StackStatus{
			RepoURL:     d.RepoURL,
			RefType:     refType,
			RefValue:    refValue,
			ComposeFile: d.ComposeFile,
		}

		newStackCfgs[d.Name] = &util.StackConfig{
			Repo:        d.RepoName,
			Branch:      d.Branch,
			Tag:         d.Tag,
			ComposeFile: d.ComposeFile,
		}

		s := newSwarmStack(d.Name, repo, d.Branch, d.Tag, d.ComposeFile, nil, "", false)
		newStacks = append(newStacks, s)
	}

	stateMu.Lock()
	oldStatus := stackStatus
	oldStacks := stacks
	oldRepos := repos
	stackStatus = newStatus
	stacks = newStacks
	repos = newRepos
	stateMu.Unlock()

	config.RepoConfigs = newRepoCfgs
	config.StackConfigs = newStackCfgs

	return func() {
		stateMu.Lock()
		stackStatus = oldStatus
		stacks = oldStacks
		repos = oldRepos
		stateMu.Unlock()
		config.RepoConfigs = oldRepoConfigs
		config.StackConfigs = oldStackConfigs
	}
}
