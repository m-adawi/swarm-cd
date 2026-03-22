package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
	"github.com/m-adawi/swarm-cd/util"
)

// setupTestStacks populates the swarmcd package-level state with test data
// and returns a cleanup function to restore the original state.
func setupTestStacks(t *testing.T, data map[string]*swarmcd.StackStatus) func() {
	t.Helper()
	return swarmcd.SetStackStatusForTest(data)
}

// decodeBody is a test helper that decodes a JSON response body into a map.
func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	return body
}

func TestGetStacks_ReturnsAllFields(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().Truncate(time.Second)
	earlier := now.Add(-1 * time.Hour)

	cleanup := setupTestStacks(t, map[string]*swarmcd.StackStatus{
		"my-stack": {
			Error:          "some error",
			Revision:       "abc12345",
			RepoURL:        "https://github.com/example/repo",
			RefType:        "branch",
			RefValue:       "main",
			ComposeFile:    "docker-compose.yaml",
			LastChangeAt:   &earlier,
			LastDeployedAt: &now,
		},
	})
	defer cleanup()

	r := gin.New()
	r.GET("/stacks", getStacks)

	req := httptest.NewRequest(http.MethodGet, "/stacks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var stacks []stackResponse
	if err := json.Unmarshal(w.Body.Bytes(), &stacks); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(stacks) != 1 {
		t.Fatalf("expected 1 stack, got %d", len(stacks))
	}

	s := stacks[0]

	if s.Name != "my-stack" {
		t.Errorf("expected name %q, got %q", "my-stack", s.Name)
	}
	if s.RepoURL != "https://github.com/example/repo" {
		t.Errorf("expected repo_url %q, got %q", "https://github.com/example/repo", s.RepoURL)
	}
	if s.RefType != "branch" {
		t.Errorf("expected ref_type %q, got %q", "branch", s.RefType)
	}
	if s.RefValue != "main" {
		t.Errorf("expected ref_value %q, got %q", "main", s.RefValue)
	}
	if s.Revision != "abc12345" {
		t.Errorf("expected revision %q, got %q", "abc12345", s.Revision)
	}
	if s.ComposeFile != "docker-compose.yaml" {
		t.Errorf("expected compose_file %q, got %q", "docker-compose.yaml", s.ComposeFile)
	}
	if s.Error != "some error" {
		t.Errorf("expected error %q, got %q", "some error", s.Error)
	}
	if s.LastChangeAt == nil {
		t.Fatal("expected last_change_at to be non-nil")
	}
	if !s.LastChangeAt.Equal(earlier) {
		t.Errorf("expected last_change_at %v, got %v", earlier, *s.LastChangeAt)
	}
	if s.LastDeployedAt == nil {
		t.Fatal("expected last_deployed_at to be non-nil")
	}
	if !s.LastDeployedAt.Equal(now) {
		t.Errorf("expected last_deployed_at %v, got %v", now, *s.LastDeployedAt)
	}

	// Verify JSON field names are snake_case by checking raw JSON
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil {
		t.Fatalf("failed to decode raw response: %v", err)
	}
	expectedKeys := []string{"name", "repo_url", "ref_type", "ref_value", "revision", "compose_file", "error", "last_change_at", "last_deployed_at"}
	for _, key := range expectedKeys {
		if _, ok := raw[0][key]; !ok {
			t.Errorf("expected JSON key %q not found in response", key)
		}
	}
}

func TestGetStacks_SortOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cleanup := setupTestStacks(t, map[string]*swarmcd.StackStatus{
		"charlie": {
			RepoURL:  "https://github.com/example/repo",
			RefType:  "branch",
			RefValue: "main",
		},
		"alpha": {
			RepoURL:  "https://github.com/example/repo",
			RefType:  "tag",
			RefValue: "v1.0.0",
		},
		"bravo": {
			RepoURL:  "https://github.com/example/repo",
			RefType:  "branch",
			RefValue: "develop",
		},
	})
	defer cleanup()

	r := gin.New()
	r.GET("/stacks", getStacks)

	req := httptest.NewRequest(http.MethodGet, "/stacks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var stacks []stackResponse
	if err := json.Unmarshal(w.Body.Bytes(), &stacks); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(stacks) != 3 {
		t.Fatalf("expected 3 stacks, got %d", len(stacks))
	}

	expectedOrder := []string{"alpha", "bravo", "charlie"}
	for i, expected := range expectedOrder {
		if stacks[i].Name != expected {
			t.Errorf("position %d: expected %q, got %q", i, expected, stacks[i].Name)
		}
	}
}

// ---------- GET /stacks/:name tests ----------

func TestGetStackByName_Found(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now().Truncate(time.Second)
	earlier := now.Add(-1 * time.Hour)

	cleanup := setupTestStacks(t, map[string]*swarmcd.StackStatus{
		"my-stack": {
			Error:          "",
			Revision:       "abc12345",
			RepoURL:        "https://github.com/example/repo",
			RefType:        "branch",
			RefValue:       "main",
			ComposeFile:    "docker-compose.yaml",
			LastChangeAt:   &earlier,
			LastDeployedAt: &now,
		},
		"other-stack": {
			Revision: "def67890",
			RepoURL:  "https://github.com/example/other",
			RefType:  "tag",
			RefValue: "v1.0.0",
		},
	})
	defer cleanup()

	r := gin.New()
	r.GET("/stacks/:name", getStack)

	req := httptest.NewRequest(http.MethodGet, "/stacks/my-stack", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var s stackResponse
	if err := json.Unmarshal(w.Body.Bytes(), &s); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if s.Name != "my-stack" {
		t.Errorf("expected name %q, got %q", "my-stack", s.Name)
	}
	if s.RepoURL != "https://github.com/example/repo" {
		t.Errorf("expected repo_url %q, got %q", "https://github.com/example/repo", s.RepoURL)
	}
	if s.RefType != "branch" {
		t.Errorf("expected ref_type %q, got %q", "branch", s.RefType)
	}
	if s.RefValue != "main" {
		t.Errorf("expected ref_value %q, got %q", "main", s.RefValue)
	}
}

func TestGetStackByName_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cleanup := setupTestStacks(t, map[string]*swarmcd.StackStatus{
		"existing-stack": {
			Revision: "abc12345",
			RepoURL:  "https://github.com/example/repo",
			RefType:  "branch",
			RefValue: "main",
		},
	})
	defer cleanup()

	r := gin.New()
	r.GET("/stacks/:name", getStack)

	req := httptest.NewRequest(http.MethodGet, "/stacks/foo", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	body := decodeBody(t, w)
	expected := "stack 'foo' not found"
	if body["error"] != expected {
		t.Errorf("expected error %q, got %q", expected, body["error"])
	}
}

// ---------- GET /health tests ----------

func TestGetHealth_ReturnsBootTime(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bootedAt := time.Now().Add(-5 * time.Minute).Truncate(time.Second)
	cleanupRuntime := swarmcd.SetRuntimeInfoForTest(swarmcd.RuntimeInfo{
		BootedAt: bootedAt,
		Version:  "dev",
	})
	defer cleanupRuntime()

	cleanupStacks := setupTestStacks(t, map[string]*swarmcd.StackStatus{})
	defer cleanupStacks()

	oldInterval := util.Configs.UpdateInterval
	util.Configs.UpdateInterval = 120
	defer func() { util.Configs.UpdateInterval = oldInterval }()

	r := gin.New()
	r.GET("/health", getHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := decodeBody(t, w)

	if body["status"] != "healthy" {
		t.Errorf("expected status %q, got %q", "healthy", body["status"])
	}

	// Verify booted_at is present and parseable
	bootedAtStr, ok := body["booted_at"].(string)
	if !ok {
		t.Fatalf("expected booted_at to be a string, got %T", body["booted_at"])
	}
	parsedBoot, err := time.Parse(time.RFC3339Nano, bootedAtStr)
	if err != nil {
		t.Fatalf("failed to parse booted_at %q: %v", bootedAtStr, err)
	}
	if !parsedBoot.Equal(bootedAt) {
		t.Errorf("expected booted_at %v, got %v", bootedAt, parsedBoot)
	}

	// Verify uptime is positive
	uptimeRaw, ok := body["uptime_seconds"].(float64)
	if !ok {
		t.Fatalf("expected uptime_seconds to be a number, got %T", body["uptime_seconds"])
	}
	if uptimeRaw <= 0 {
		t.Errorf("expected positive uptime_seconds, got %v", uptimeRaw)
	}

	if body["version"] != "dev" {
		t.Errorf("expected version %q, got %q", "dev", body["version"])
	}

	if body["update_interval_seconds"] != float64(120) {
		t.Errorf("expected update_interval_seconds=120, got %v", body["update_interval_seconds"])
	}
}

func TestGetHealth_ReturnsStackCount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bootedAt := time.Now().Add(-1 * time.Minute)
	cleanupRuntime := swarmcd.SetRuntimeInfoForTest(swarmcd.RuntimeInfo{
		BootedAt: bootedAt,
		Version:  "1.2.3",
	})
	defer cleanupRuntime()

	cleanupStacks := setupTestStacks(t, map[string]*swarmcd.StackStatus{
		"stack-a": {Revision: "aaa", RepoURL: "https://github.com/a"},
		"stack-b": {Revision: "bbb", RepoURL: "https://github.com/b"},
		"stack-c": {Revision: "ccc", RepoURL: "https://github.com/c"},
	})
	defer cleanupStacks()

	oldInterval := util.Configs.UpdateInterval
	util.Configs.UpdateInterval = 60
	defer func() { util.Configs.UpdateInterval = oldInterval }()

	r := gin.New()
	r.GET("/health", getHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := decodeBody(t, w)

	if body["stacks_managed"] != float64(3) {
		t.Errorf("expected stacks_managed=3, got %v", body["stacks_managed"])
	}

	if body["version"] != "1.2.3" {
		t.Errorf("expected version %q, got %q", "1.2.3", body["version"])
	}
}

// ---------- GET /health config_warnings tests ----------

func TestGetHealth_ReturnsConfigWarnings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bootedAt := time.Now().Add(-1 * time.Minute)
	cleanupRuntime := swarmcd.SetRuntimeInfoForTest(swarmcd.RuntimeInfo{
		BootedAt: bootedAt,
		Version:  "dev",
	})
	defer cleanupRuntime()

	cleanupStacks := setupTestStacks(t, map[string]*swarmcd.StackStatus{})
	defer cleanupStacks()

	oldInterval := util.Configs.UpdateInterval
	util.Configs.UpdateInterval = 120
	defer func() { util.Configs.UpdateInterval = oldInterval }()

	// Simulate inline config conflict (stacks inline, no split file)
	oldConflict := util.Conflict
	util.Conflict = util.ConfigConflict{
		StacksInline:     true,
		ReposInline:      false,
		StacksFileExists: false,
		ReposFileExists:  false,
	}
	defer func() { util.Conflict = oldConflict }()

	r := gin.New()
	r.GET("/health", getHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := decodeBody(t, w)

	rawWarnings, ok := body["config_warnings"]
	if !ok {
		t.Fatal("expected config_warnings key in health response")
	}
	warnings, ok := rawWarnings.([]interface{})
	if !ok {
		t.Fatalf("expected config_warnings to be an array, got %T", rawWarnings)
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if warnings[0] != "API changes to stacks won't persist across restarts" {
		t.Errorf("unexpected warning: %v", warnings[0])
	}
}

func TestGetHealth_NoConfigWarnings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	bootedAt := time.Now().Add(-1 * time.Minute)
	cleanupRuntime := swarmcd.SetRuntimeInfoForTest(swarmcd.RuntimeInfo{
		BootedAt: bootedAt,
		Version:  "dev",
	})
	defer cleanupRuntime()

	cleanupStacks := setupTestStacks(t, map[string]*swarmcd.StackStatus{})
	defer cleanupStacks()

	oldInterval := util.Configs.UpdateInterval
	util.Configs.UpdateInterval = 120
	defer func() { util.Configs.UpdateInterval = oldInterval }()

	// No conflicts — split config only
	oldConflict := util.Conflict
	util.Conflict = util.ConfigConflict{}
	defer func() { util.Conflict = oldConflict }()

	r := gin.New()
	r.GET("/health", getHealth)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := decodeBody(t, w)

	if _, ok := body["config_warnings"]; ok {
		t.Error("expected no config_warnings key when there are no conflicts")
	}
}
