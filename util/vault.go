package util

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	vaultapi "github.com/hashicorp/vault/api"
)

var vaultClient *vaultapi.Client

// InitVault initializes a shared Vault client.
// If vault_address is empty, Vault integration is disabled (no-op).
//
// Token resolution order:
// - config.yaml: vault_token
// - env: VAULT_TOKEN
//
// Namespace is optional (vault_namespace).
func InitVault() error {
	if Configs.VaultAddress == "" {
		Logger.Info("Vault integration disabled (vault_address is empty)")
		return nil
	}

	cfg := vaultapi.DefaultConfig()
	cfg.Address = Configs.VaultAddress

	client, err := vaultapi.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("could not create Vault client: %w", err)
	}

	token := strings.TrimSpace(Configs.VaultToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("VAULT_TOKEN"))
	}
	if token == "" {
		return fmt.Errorf("Vault token is not configured (set vault_token or VAULT_TOKEN)")
	}
	client.SetToken(token)

	if ns := strings.TrimSpace(Configs.VaultNamespace); ns != "" {
		client.SetNamespace(ns)
	}

	vaultClient = client
	Logger.Info("Vault client initialized", slog.String("address", Configs.VaultAddress))
	return nil
}

// ResolveVaultReferenceKVv2 resolves a reference in the form:
//   vault:<path>#<key>
//
// Examples:
//   vault:secret/data/myapp/db#password
//
// It is KV v2 aware: if response contains "data" object, it reads keys from it.
// Returns (resolvedValue, true, nil) if it was a vault reference.
// Returns ("", false, nil) if the input is not a vault reference.
func ResolveVaultReferenceKVv2(ref string) (string, bool, error) {
	if !strings.HasPrefix(ref, "vault:") {
		return "", false, nil
	}
	if vaultClient == nil {
		return "", true, fmt.Errorf("Vault is not initialized")
	}

	withoutPrefix := strings.TrimPrefix(ref, "vault:")
	parts := strings.SplitN(withoutPrefix, "#", 2)
	if len(parts) != 2 {
		return "", true, fmt.Errorf("invalid vault reference %q, expected vault:<path>#<key>", ref)
	}

	secretPath := strings.TrimSpace(parts[0])
	key := strings.TrimSpace(parts[1])
	if secretPath == "" || key == "" {
		return "", true, fmt.Errorf("invalid vault reference %q, path and key must be non-empty", ref)
	}

	secret, err := vaultClient.Logical().Read(secretPath)
	if err != nil {
		return "", true, fmt.Errorf("could not read Vault path %s: %w", secretPath, err)
	}
	if secret == nil || secret.Data == nil {
		return "", true, fmt.Errorf("Vault path %s not found or empty", secretPath)
	}

	data := secret.Data
	// KV v2: data is nested under "data"
	if inner, ok := data["data"].(map[string]any); ok {
		data = inner
	}

	val, ok := data[key]
	if !ok {
		return "", true, fmt.Errorf("key %s not found at Vault path %s", key, secretPath)
	}
	strVal, ok := val.(string)
	if !ok {
		return "", true, fmt.Errorf("key %s at Vault path %s is not a string", key, secretPath)
	}
	return strVal, true, nil
}

