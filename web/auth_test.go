package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestRouter creates a minimal router with auth middleware for testing.
func newTestRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	write := r.Group("/")
	write.Use(authMiddleware())
	write.POST("/test", handler)
	return r
}

func TestAuth_NoTokenConfigured(t *testing.T) {
	oldToken := apiToken
	apiToken = ""
	defer func() { apiToken = oldToken }()

	r := newTestRouter(func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer some-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}

	body := decodeBody(t, w)
	if body["error"] == nil {
		t.Error("expected error in response")
	}
}

func TestAuth_MissingHeader(t *testing.T) {
	oldToken := apiToken
	apiToken = "test-secret"
	defer func() { apiToken = oldToken }()

	r := newTestRouter(func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_InvalidFormat(t *testing.T) {
	oldToken := apiToken
	apiToken = "test-secret"
	defer func() { apiToken = oldToken }()

	r := newTestRouter(func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_WrongToken(t *testing.T) {
	oldToken := apiToken
	apiToken = "correct-token"
	defer func() { apiToken = oldToken }()

	r := newTestRouter(func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuth_ValidToken(t *testing.T) {
	oldToken := apiToken
	apiToken = "correct-token"
	defer func() { apiToken = oldToken }()

	r := newTestRouter(func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	req.Header.Set("Authorization", "Bearer correct-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuth_MutationAPIEnabled(t *testing.T) {
	oldToken := apiToken
	defer func() { apiToken = oldToken }()

	apiToken = ""
	if MutationAPIEnabled() {
		t.Error("expected MutationAPIEnabled=false when token is empty")
	}

	apiToken = "some-token"
	if !MutationAPIEnabled() {
		t.Error("expected MutationAPIEnabled=true when token is set")
	}
}
