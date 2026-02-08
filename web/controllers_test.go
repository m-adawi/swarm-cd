package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter() *gin.Engine {
	r := gin.New()
	r.POST("/webhook", webhookAuthMiddleware(), postWebhook)
	return r
}

func TestWebhookAuthMiddleware_NoKeyConfigured(t *testing.T) {
	// Ensure no webhook key is set
	os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	req, _ := http.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Authorization", "Bearer some-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "webhook not configured" {
		t.Errorf("expected error 'webhook not configured', got '%s'", response["error"])
	}
}

func TestWebhookAuthMiddleware_MissingAuthHeader(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	req, _ := http.NewRequest("POST", "/webhook", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "missing Authorization header" {
		t.Errorf("expected error 'missing Authorization header', got '%s'", response["error"])
	}
}

func TestWebhookAuthMiddleware_InvalidKey(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	req, _ := http.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Authorization", "Bearer wrong-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "invalid webhook key" {
		t.Errorf("expected error 'invalid webhook key', got '%s'", response["error"])
	}
}

func TestWebhookAuthMiddleware_ValidBearerKey(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	req, _ := http.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should pass auth and return OK (updating all stacks with empty body)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestWebhookAuthMiddleware_ValidRawKey(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	req, _ := http.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Authorization", "test-secret-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should pass auth and return OK
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestPostWebhook_UpdateAllStacks(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	req, _ := http.NewRequest("POST", "/webhook", nil)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["message"] != "all stacks update triggered" {
		t.Errorf("expected message 'all stacks update triggered', got '%s'", response["message"])
	}
}

func TestPostWebhook_UpdateAllStacksWithEmptyBody(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	body := bytes.NewBufferString("{}")
	req, _ := http.NewRequest("POST", "/webhook", body)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["message"] != "all stacks update triggered" {
		t.Errorf("expected message 'all stacks update triggered', got '%s'", response["message"])
	}
}

func TestPostWebhook_UpdateSpecificStack_NotFound(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	reqBody := webhookRequest{Stack: "nonexistent-stack"}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer test-secret-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "stack nonexistent-stack not found" {
		t.Errorf("expected error 'stack nonexistent-stack not found', got '%s'", response["error"])
	}
}

func TestPostWebhook_InvalidJSON(t *testing.T) {
	os.Setenv("WEBHOOK_KEY", "test-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	router := setupTestRouter()

	body := bytes.NewBufferString("invalid json")
	req, _ := http.NewRequest("POST", "/webhook", body)
	req.Header.Set("Authorization", "Bearer test-secret-key")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Invalid JSON should trigger update all stacks (graceful fallback)
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["message"] != "all stacks update triggered" {
		t.Errorf("expected message 'all stacks update triggered', got '%s'", response["message"])
	}
}
