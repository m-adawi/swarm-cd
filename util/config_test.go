package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetWebhookKey_EnvVar(t *testing.T) {
	// Setup
	os.Setenv("WEBHOOK_KEY", "env-secret-key")
	defer os.Unsetenv("WEBHOOK_KEY")

	// Also set config values to verify env var takes priority
	Configs.WebhookKey = "config-key"
	Configs.WebhookKeyFile = ""
	defer func() {
		Configs.WebhookKey = ""
		Configs.WebhookKeyFile = ""
	}()

	key := GetWebhookKey()
	if key != "env-secret-key" {
		t.Errorf("expected 'env-secret-key', got '%s'", key)
	}
}

func TestGetWebhookKey_File(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("WEBHOOK_KEY")

	// Create a temp file with the key
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "webhook_key")
	err := os.WriteFile(keyFile, []byte("file-secret-key\n"), 0600)
	if err != nil {
		t.Fatalf("failed to create temp key file: %v", err)
	}

	// Set config to use the file
	Configs.WebhookKeyFile = keyFile
	Configs.WebhookKey = "config-key"
	defer func() {
		Configs.WebhookKey = ""
		Configs.WebhookKeyFile = ""
	}()

	key := GetWebhookKey()
	if key != "file-secret-key" {
		t.Errorf("expected 'file-secret-key', got '%s'", key)
	}
}

func TestGetWebhookKey_FileTrimsWhitespace(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("WEBHOOK_KEY")

	// Create a temp file with the key and whitespace
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "webhook_key")
	err := os.WriteFile(keyFile, []byte("  secret-with-spaces  \n\n"), 0600)
	if err != nil {
		t.Fatalf("failed to create temp key file: %v", err)
	}

	Configs.WebhookKeyFile = keyFile
	Configs.WebhookKey = ""
	defer func() {
		Configs.WebhookKeyFile = ""
	}()

	key := GetWebhookKey()
	if key != "secret-with-spaces" {
		t.Errorf("expected 'secret-with-spaces', got '%s'", key)
	}
}

func TestGetWebhookKey_FileNotFound(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("WEBHOOK_KEY")

	// Set config to use a non-existent file
	Configs.WebhookKeyFile = "/nonexistent/path/webhook_key"
	Configs.WebhookKey = "config-key"
	defer func() {
		Configs.WebhookKey = ""
		Configs.WebhookKeyFile = ""
	}()

	key := GetWebhookKey()
	// Should return empty string when file read fails
	if key != "" {
		t.Errorf("expected empty string when file not found, got '%s'", key)
	}
}

func TestGetWebhookKey_ConfigValue(t *testing.T) {
	// Ensure env var is not set
	os.Unsetenv("WEBHOOK_KEY")

	// Set only config value
	Configs.WebhookKey = "config-secret-key"
	Configs.WebhookKeyFile = ""
	defer func() {
		Configs.WebhookKey = ""
	}()

	key := GetWebhookKey()
	if key != "config-secret-key" {
		t.Errorf("expected 'config-secret-key', got '%s'", key)
	}
}

func TestGetWebhookKey_NoKeyConfigured(t *testing.T) {
	// Ensure nothing is set
	os.Unsetenv("WEBHOOK_KEY")
	Configs.WebhookKey = ""
	Configs.WebhookKeyFile = ""

	key := GetWebhookKey()
	if key != "" {
		t.Errorf("expected empty string, got '%s'", key)
	}
}

func TestGetWebhookKey_Priority(t *testing.T) {
	// Test that priority is: env var > file > config
	tmpDir := t.TempDir()
	keyFile := filepath.Join(tmpDir, "webhook_key")
	err := os.WriteFile(keyFile, []byte("file-key"), 0600)
	if err != nil {
		t.Fatalf("failed to create temp key file: %v", err)
	}

	// Set all three
	os.Setenv("WEBHOOK_KEY", "env-key")
	Configs.WebhookKeyFile = keyFile
	Configs.WebhookKey = "config-key"
	defer func() {
		os.Unsetenv("WEBHOOK_KEY")
		Configs.WebhookKey = ""
		Configs.WebhookKeyFile = ""
	}()

	// Env var should win
	key := GetWebhookKey()
	if key != "env-key" {
		t.Errorf("expected env var to take priority, got '%s'", key)
	}

	// Unset env var, file should win
	os.Unsetenv("WEBHOOK_KEY")
	key = GetWebhookKey()
	if key != "file-key" {
		t.Errorf("expected file to take priority over config, got '%s'", key)
	}

	// Unset file, config should be used
	Configs.WebhookKeyFile = ""
	key = GetWebhookKey()
	if key != "config-key" {
		t.Errorf("expected config value, got '%s'", key)
	}
}
