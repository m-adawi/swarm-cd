package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
)

// chdirTemp changes to a temp directory for the duration of the test.
// This isolates PersistConfigs file writes.
func chdirTemp(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

// createTestGitRepo creates a bare git repo with a single commit containing
// a docker-compose.yaml file. Returns the bare repo path (usable as URL).
func createTestGitRepo(t *testing.T, dir, name string) string {
	t.Helper()

	repoDir := filepath.Join(dir, name+".git")
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Init bare repo with main as default branch
	gitRun(t, repoDir, "git", "init", "--bare")
	gitRun(t, repoDir, "git", "symbolic-ref", "HEAD", "refs/heads/main")

	// Create work dir, init, add content, push
	workDir := filepath.Join(dir, name+"-work")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		t.Fatalf("mkdir work: %v", err)
	}

	gitRun(t, workDir, "git", "init")
	gitRun(t, workDir, "git", "config", "user.email", "test@test.com")
	gitRun(t, workDir, "git", "config", "user.name", "test")
	gitRun(t, workDir, "git", "checkout", "-b", "main")

	composeContent := "version: '3'\nservices:\n  web:\n    image: nginx\n"
	if err := os.WriteFile(filepath.Join(workDir, "docker-compose.yaml"), []byte(composeContent), 0644); err != nil {
		t.Fatalf("write compose: %v", err)
	}

	gitRun(t, workDir, "git", "add", ".")
	gitRun(t, workDir, "git", "commit", "-m", "initial")
	gitRun(t, workDir, "git", "remote", "add", "origin", repoDir)
	gitRun(t, workDir, "git", "push", "-u", "origin", "main")

	return repoDir
}

func gitRun(t *testing.T, dir, command string, args ...string) {
	t.Helper()
	cmd := exec.Command(command, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("command %q in %s failed: %v\n%s", command+" "+strings.Join(args, " "), dir, err, out)
	}
}

func setupPatchRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PATCH("/stacks/:name", patchStack)
	return r
}

func TestPatchStack_UpdateRef(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"branch": "develop"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status := swarmcd.GetStackStatus()
	if status["my-stack"].RefType != "branch" {
		t.Errorf("expected ref_type 'branch', got %q", status["my-stack"].RefType)
	}
	if status["my-stack"].RefValue != "develop" {
		t.Errorf("expected ref_value 'develop', got %q", status["my-stack"].RefValue)
	}
}

func TestPatchStack_BranchToTag(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"tag": "v1.0.0"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status := swarmcd.GetStackStatus()
	if status["my-stack"].RefType != "tag" {
		t.Errorf("expected ref_type 'tag', got %q", status["my-stack"].RefType)
	}
	if status["my-stack"].RefValue != "v1.0.0" {
		t.Errorf("expected ref_value 'v1.0.0', got %q", status["my-stack"].RefValue)
	}
}

func TestPatchStack_TagToBranch(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Tag:         "v1.0.0",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"branch": "main"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status := swarmcd.GetStackStatus()
	if status["my-stack"].RefType != "branch" {
		t.Errorf("expected ref_type 'branch', got %q", status["my-stack"].RefType)
	}
	if status["my-stack"].RefValue != "main" {
		t.Errorf("expected ref_value 'main', got %q", status["my-stack"].RefValue)
	}
}

func TestPatchStack_RejectsEmpty(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPatchStack_RejectsBothBranchAndTag(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"branch": "develop", "tag": "v1.0.0"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPatchStack_NotFound(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"branch": "develop"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/missing", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPatchStack_ComposeFile(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"compose_file": "docker-compose.prod.yaml"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status := swarmcd.GetStackStatus()
	if status["my-stack"].ComposeFile != "docker-compose.prod.yaml" {
		t.Errorf("expected compose_file 'docker-compose.prod.yaml', got %q", status["my-stack"].ComposeFile)
	}
}

func TestPatchStack_PartialUpdate(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"compose_file": "docker-compose.staging.yaml"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status := swarmcd.GetStackStatus()
	// branch should be unchanged
	if status["my-stack"].RefType != "branch" {
		t.Errorf("expected ref_type 'branch', got %q", status["my-stack"].RefType)
	}
	if status["my-stack"].RefValue != "main" {
		t.Errorf("expected ref_value 'main', got %q", status["my-stack"].RefValue)
	}
	// compose_file should be updated
	if status["my-stack"].ComposeFile != "docker-compose.staging.yaml" {
		t.Errorf("expected compose_file 'docker-compose.staging.yaml', got %q", status["my-stack"].ComposeFile)
	}
}

func TestPatchStack_UpdateRepoURL(t *testing.T) {
	tmpDir := t.TempDir()

	// Change to tmpDir so PersistConfigs writes there
	orig, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	// Create two test git repos
	repoPath1 := createTestGitRepo(t, tmpDir, "repo1")
	repoPath2 := createTestGitRepo(t, tmpDir, "repo2")

	// Clone repo1 to simulate the existing working copy
	workDir := filepath.Join(tmpDir, "repos", "my-repo")
	if err := os.MkdirAll(filepath.Dir(workDir), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	gitRun(t, tmpDir, "git", "clone", repoPath1, workDir)

	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    workDir,
			RepoURL:     repoPath1,
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body, _ := json.Marshal(map[string]string{"repo_url": repoPath2})
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status := swarmcd.GetStackStatus()
	if status["my-stack"].RepoURL != repoPath2 {
		t.Errorf("expected repo_url %q, got %q", repoPath2, status["my-stack"].RepoURL)
	}
}

func TestPatchStack_SharedRepo_BranchIsolation(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "stack-a",
			RepoName:    "shared-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
		{
			Name:        "stack-b",
			RepoName:    "shared-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "develop",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"branch": "feature"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/stack-a", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	status := swarmcd.GetStackStatus()
	if status["stack-a"].RefValue != "feature" {
		t.Errorf("expected ref_value 'feature', got %q", status["stack-a"].RefValue)
	}
	if status["stack-b"].RefValue != "develop" {
		t.Errorf("expected stack-b ref_value 'develop', got %q", status["stack-b"].RefValue)
	}
}

func TestPatchStack_SharedRepo_URLChangeWarning(t *testing.T) {
	tmpDir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	repoPath1 := createTestGitRepo(t, tmpDir, "shared")
	repoPath2 := createTestGitRepo(t, tmpDir, "new-origin")

	workDir := filepath.Join(tmpDir, "repos", "shared-repo")
	if err := os.MkdirAll(filepath.Dir(workDir), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	gitRun(t, tmpDir, "git", "clone", repoPath1, workDir)

	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "stack-a",
			RepoName:    "shared-repo",
			RepoPath:    workDir,
			RepoURL:     repoPath1,
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
		{
			Name:        "stack-b",
			RepoName:    "shared-repo",
			RepoPath:    workDir,
			RepoURL:     repoPath1,
			Branch:      "develop",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body, _ := json.Marshal(map[string]string{"repo_url": repoPath2})
	req := httptest.NewRequest(http.MethodPatch, "/stacks/stack-a", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the response contains a shared-repo warning
	var resp swarmcd.PatchResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Warnings) == 0 {
		t.Fatal("expected warnings about shared repo, got none")
	}

	found := false
	for _, w := range resp.Warnings {
		if strings.Contains(w, "shared-repo") && strings.Contains(w, "stack-b") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning mentioning shared-repo and stack-b, got: %v", resp.Warnings)
	}
}

func TestPatchStack_NoAuth(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	oldToken := apiToken
	apiToken = "test-secret"
	defer func() { apiToken = oldToken }()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	write := r.Group("/")
	write.Use(authMiddleware())
	write.PATCH("/stacks/:name", patchStack)

	body := `{"branch": "develop"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPatchStack_AtomicPersistence(t *testing.T) {
	chdirTemp(t)
	cleanup := swarmcd.SetFullStateForTest([]swarmcd.TestStackDef{
		{
			Name:        "my-stack",
			RepoName:    "my-repo",
			RepoPath:    "/tmp/fake-repo",
			RepoURL:     "https://example.com/repo",
			Branch:      "main",
			ComposeFile: "docker-compose.yaml",
		},
	})
	defer cleanup()

	r := setupPatchRouter(t)
	body := `{"branch": "develop"}`
	req := httptest.NewRequest(http.MethodPatch, "/stacks/my-stack", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify stacks.yaml was written
	data, err := os.ReadFile("stacks.yaml")
	if err != nil {
		t.Fatalf("expected stacks.yaml to exist: %v", err)
	}
	if !strings.Contains(string(data), "develop") {
		t.Errorf("expected stacks.yaml to contain 'develop': %s", data)
	}

	// Verify no .tmp files left behind
	entries, _ := os.ReadDir(".")
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
}
