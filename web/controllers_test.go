package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/m-adawi/swarm-cd/swarmcd"
)

func servicesRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/stacks/:name/services", getStackServices)
	return router
}

// withStubServices swaps the getStackServicesFn seam for the duration of a test.
func withStubServices(t *testing.T, stub func(name string) ([]swarmcd.ServiceStatus, error)) {
	t.Helper()
	original := getStackServicesFn
	getStackServicesFn = stub
	t.Cleanup(func() { getStackServicesFn = original })
}

func TestGetStackServices_NotFound(t *testing.T) {
	withStubServices(t, func(string) ([]swarmcd.ServiceStatus, error) {
		return nil, swarmcd.ErrStackNotFound
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stacks/unknown/services", nil)
	servicesRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGetStackServices_OK(t *testing.T) {
	withStubServices(t, func(name string) ([]swarmcd.ServiceStatus, error) {
		return []swarmcd.ServiceStatus{
			{ID: "1", Name: "web_nginx", Image: "nginx:1.27", Mode: "replicated", RunningTasks: 3, DesiredTasks: 3, Health: swarmcd.HealthHealthy},
		}, nil
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stacks/web/services", nil)
	servicesRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var got []swarmcd.ServiceStatus
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(services) = %d, want 1", len(got))
	}
	if got[0].Name != "web_nginx" || got[0].Health != swarmcd.HealthHealthy {
		t.Errorf("unexpected service payload: %+v", got[0])
	}
}

func TestGetStackServices_InternalError(t *testing.T) {
	withStubServices(t, func(string) ([]swarmcd.ServiceStatus, error) {
		return nil, errors.New("docker daemon unreachable")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/stacks/web/services", nil)
	servicesRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
