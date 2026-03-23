package web

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
)

type mockServiceAPI struct {
	services  []swarm.Service
	listErr   error
	updateErr error
	updates   []string // tracks which services were updated
}

func (m *mockServiceAPI) ServiceList(_ context.Context, _ types.ServiceListOptions) ([]swarm.Service, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.services, nil
}

func (m *mockServiceAPI) ServiceUpdate(_ context.Context, _ string, _ swarm.Version, spec swarm.ServiceSpec, _ types.ServiceUpdateOptions) (swarm.ServiceUpdateResponse, error) {
	if m.updateErr != nil {
		return swarm.ServiceUpdateResponse{}, m.updateErr
	}
	m.updates = append(m.updates, spec.Name)
	return swarm.ServiceUpdateResponse{}, nil
}

func setupRestartRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/stacks/:name/restart", restartStack)
	r.POST("/stacks/:name/services/:service/restart", restartService)
	r.POST("/restart", restartAll)
	return r
}

func TestRestartStack_Success(t *testing.T) {
	cleanup := swarmcd.SetStackStatusForTest(map[string]*swarmcd.StackStatus{
		"my-stack": {RepoURL: "https://example.com"},
	})
	defer cleanup()

	mock := &mockServiceAPI{
		services: []swarm.Service{
			{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "my-stack_web"}}},
			{ID: "svc2", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "my-stack_worker"}}},
		},
	}
	cleanupAPI := swarmcd.SetServiceAPIForTest(mock)
	defer cleanupAPI()

	r := setupRestartRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/stacks/my-stack/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(mock.updates) != 2 {
		t.Errorf("expected 2 service updates, got %d", len(mock.updates))
	}
}

func TestRestartStack_NotFound(t *testing.T) {
	cleanup := swarmcd.SetStackStatusForTest(map[string]*swarmcd.StackStatus{})
	defer cleanup()

	r := setupRestartRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/stacks/missing/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRestartService_Success(t *testing.T) {
	cleanup := swarmcd.SetStackStatusForTest(map[string]*swarmcd.StackStatus{
		"my-stack": {RepoURL: "https://example.com"},
	})
	defer cleanup()

	mock := &mockServiceAPI{
		services: []swarm.Service{
			{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "my-stack_web"}}},
			{ID: "svc2", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "my-stack_worker"}}},
		},
	}
	cleanupAPI := swarmcd.SetServiceAPIForTest(mock)
	defer cleanupAPI()

	r := setupRestartRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/stacks/my-stack/services/web/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	if len(mock.updates) != 1 {
		t.Errorf("expected 1 service update, got %d", len(mock.updates))
	}
	if len(mock.updates) > 0 && mock.updates[0] != "my-stack_web" {
		t.Errorf("expected service 'my-stack_web' to be updated, got %q", mock.updates[0])
	}
}

func TestRestartService_NotFoundService(t *testing.T) {
	cleanup := swarmcd.SetStackStatusForTest(map[string]*swarmcd.StackStatus{
		"my-stack": {RepoURL: "https://example.com"},
	})
	defer cleanup()

	mock := &mockServiceAPI{
		services: []swarm.Service{
			{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "my-stack_web"}}},
		},
	}
	cleanupAPI := swarmcd.SetServiceAPIForTest(mock)
	defer cleanupAPI()

	r := setupRestartRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/stacks/my-stack/services/nonexistent/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRestartAll_Success(t *testing.T) {
	cleanup := swarmcd.SetStackStatusForTest(map[string]*swarmcd.StackStatus{
		"stack-a": {RepoURL: "https://example.com"},
		"stack-b": {RepoURL: "https://example.com"},
	})
	defer cleanup()

	mock := &mockServiceAPI{
		services: []swarm.Service{
			{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "stack-a_web"}}},
		},
	}
	cleanupAPI := swarmcd.SetServiceAPIForTest(mock)
	defer cleanupAPI()

	r := setupRestartRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRestartStack_AuthRequired(t *testing.T) {
	cleanup := swarmcd.SetStackStatusForTest(map[string]*swarmcd.StackStatus{
		"my-stack": {RepoURL: "https://example.com"},
	})
	defer cleanup()

	oldToken := apiToken
	apiToken = "test-secret"
	defer func() { apiToken = oldToken }()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	write := r.Group("/")
	write.Use(authMiddleware())
	write.POST("/stacks/:name/restart", restartStack)

	req := httptest.NewRequest(http.MethodPost, "/stacks/my-stack/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRestartStack_ServiceUpdateError(t *testing.T) {
	cleanup := swarmcd.SetStackStatusForTest(map[string]*swarmcd.StackStatus{
		"my-stack": {RepoURL: "https://example.com"},
	})
	defer cleanup()

	mock := &mockServiceAPI{
		services: []swarm.Service{
			{ID: "svc1", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Name: "my-stack_web"}}},
		},
		updateErr: fmt.Errorf("connection refused"),
	}
	cleanupAPI := swarmcd.SetServiceAPIForTest(mock)
	defer cleanupAPI()

	r := setupRestartRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/stacks/my-stack/restart", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
