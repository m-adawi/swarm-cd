package swarmcd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/m-adawi/swarm-cd/util"
)

// PatchRequest represents a partial update to a stack's configuration.
// Pointer fields distinguish "not provided" (nil) from "set to empty" ("").
type PatchRequest struct {
	Branch      *string `json:"branch"`
	Tag         *string `json:"tag"`
	ComposeFile *string `json:"compose_file"`
	RepoURL     *string `json:"repo_url"`
}

// PatchResponse contains the result of a successful patch operation.
type PatchResponse struct {
	Message  string   `json:"message"`
	Warnings []string `json:"warnings,omitempty"`
}

// ValidationError is returned when a patch request is invalid.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// NotFoundError is returned when a requested stack does not exist.
type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string { return e.Message }

// PatchStack applies a partial update to the named stack's configuration.
// It uses a three-phase approach to respect lock ordering:
//   - Phase 1 (stateMu.RLock): validate the request and gather references
//   - Phase 2 (no locks): perform I/O (git clone for repo URL changes)
//   - Phase 3 (stateMu.Lock): apply state changes and persist
//
// This ensures stateMu is never held while repo.lock is acquired, which
// would violate the lock ordering rule (see swarmcd.go).
func PatchStack(name string, req PatchRequest) (*PatchResponse, error) {
	// --- Phase 1: Validate under RLock ---
	stateMu.RLock()

	if _, ok := stackStatus[name]; !ok {
		stateMu.RUnlock()
		return nil, &NotFoundError{Message: fmt.Sprintf("stack '%s' not found", name)}
	}

	stackCfg, ok := config.StackConfigs[name]
	if !ok {
		stateMu.RUnlock()
		return nil, &NotFoundError{Message: fmt.Sprintf("stack config '%s' not found", name)}
	}

	if req.Branch == nil && req.Tag == nil && req.ComposeFile == nil && req.RepoURL == nil {
		stateMu.RUnlock()
		return nil, &ValidationError{Message: "patch request must include at least one field"}
	}

	if req.Branch != nil && req.Tag != nil {
		stateMu.RUnlock()
		return nil, &ValidationError{Message: "cannot set both branch and tag in the same request"}
	}

	var warnings []string
	var repo *stackRepo
	var repoName string

	if req.RepoURL != nil {
		repoName = stackCfg.Repo
		repo = repos[repoName]

		shared := findSharedRepoStacks(repoName, name)
		if len(shared) > 0 {
			warnings = append(warnings, fmt.Sprintf("repo '%s' is also used by: %v — they will be affected", repoName, shared))
		}
	}

	stateMu.RUnlock()

	// --- Phase 2: I/O without any locks held ---
	if req.RepoURL != nil {
		if err := cloneToTempThenSwap(repo, *req.RepoURL); err != nil {
			return nil, &ValidationError{Message: fmt.Sprintf("failed to update repo URL: %v", err)}
		}
	}

	// --- Phase 3: Apply state changes under Lock ---
	stateMu.Lock()
	defer stateMu.Unlock()

	// Re-validate after re-acquiring lock (state may have changed)
	status, ok := stackStatus[name]
	if !ok {
		return nil, &NotFoundError{Message: fmt.Sprintf("stack '%s' not found", name)}
	}

	var stack *swarmStack
	for _, s := range stacks {
		if s.name == name {
			stack = s
			break
		}
	}
	if stack == nil {
		return nil, &NotFoundError{Message: fmt.Sprintf("stack object '%s' not found", name)}
	}

	// Re-read stackCfg under write lock
	stackCfg = config.StackConfigs[name]

	if req.RepoURL != nil {
		repoCfg := config.RepoConfigs[repoName]
		repoCfg.Url = *req.RepoURL
		status.RepoURL = *req.RepoURL
	}

	if req.Branch != nil || req.Tag != nil {
		updateSwarmStackRef(stack, stackCfg, status, req.Branch, req.Tag)
	}

	if req.ComposeFile != nil {
		updateSwarmStackComposeFile(stack, stackCfg, status, *req.ComposeFile)
	}

	if err := util.PersistConfigs(); err != nil {
		return nil, fmt.Errorf("failed to persist configs: %w", err)
	}

	return &PatchResponse{
		Message:  fmt.Sprintf("stack '%s' updated successfully", name),
		Warnings: warnings,
	}, nil
}

// findSharedRepoStacks returns the names of other stacks that use the same repo.
func findSharedRepoStacks(repoName, excludeStack string) []string {
	var shared []string
	for name, cfg := range config.StackConfigs {
		if name != excludeStack && cfg.Repo == repoName {
			shared = append(shared, name)
		}
	}
	return shared
}

// cloneToTempThenSwap clones the new URL to a temp directory in the same
// parent as the repo, then swaps the repo's path and git object.
// Must be called WITHOUT holding stateMu (acquires repo.lock internally).
func cloneToTempThenSwap(repo *stackRepo, newURL string) error {
	repo.lock.Lock()
	defer repo.lock.Unlock()

	// Create temp dir in the same parent to ensure same filesystem for rename
	tmpDir, err := os.MkdirTemp(filepath.Dir(repo.path), "swarmcd-repo-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	gitRepo, err := git.PlainClone(tmpDir, false, &git.CloneOptions{
		URL:  newURL,
		Auth: repo.auth,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("failed to clone %s: %w", newURL, err)
	}

	oldPath := repo.path
	if err := os.RemoveAll(oldPath); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("failed to remove old repo dir: %w", err)
	}
	if err := os.Rename(tmpDir, oldPath); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("failed to move new repo into place: %w", err)
	}

	repo.gitRepoObject = gitRepo
	repo.url = newURL
	return nil
}

// updateSwarmStackRef updates the branch/tag ref on the stack.
func updateSwarmStackRef(stack *swarmStack, cfg *util.StackConfig, status *StackStatus, branch, tag *string) {
	if branch != nil {
		stack.branch = *branch
		stack.tag = ""
		cfg.Branch = *branch
		cfg.Tag = ""
		status.RefType = "branch"
		status.RefValue = *branch
	} else if tag != nil {
		stack.tag = *tag
		stack.branch = ""
		cfg.Tag = *tag
		cfg.Branch = ""
		status.RefType = "tag"
		status.RefValue = *tag
	}
}

// updateSwarmStackComposeFile updates the compose file path on the stack.
func updateSwarmStackComposeFile(stack *swarmStack, cfg *util.StackConfig, status *StackStatus, composeFile string) {
	stack.composePath = composeFile
	cfg.ComposeFile = composeFile
	status.ComposeFile = composeFile
}
