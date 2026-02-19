package util

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
)

var (
	vaultClient     *vaultapi.Client
	vaultClientMu   sync.RWMutex
	vaultRenewStop  chan struct{}
	vaultRenewWg    sync.WaitGroup
)

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

	vaultClientMu.Lock()
	vaultClient = client
	vaultClientMu.Unlock()

	Logger.Info("Vault client initialized", slog.String("address", Configs.VaultAddress))

	// Start token renewal goroutine
	startVaultTokenRenewal()

	// Register Vault as an external value resolver.
	RegisterExternalResolver(&vaultResolver{})
	return nil
}

// renewVaultToken attempts to renew the current Vault token.
// Returns error if renewal failed.
func renewVaultToken() error {
	vaultClientMu.RLock()
	client := vaultClient
	vaultClientMu.RUnlock()

	if client == nil {
		return fmt.Errorf("Vault client is not initialized")
	}

	// Renew token for the configured interval (in seconds)
	renewInterval := Configs.VaultTokenRenewInterval * 24 * 60 * 60 // convert days to seconds
	if renewInterval <= 0 {
		renewInterval = 24 * 60 * 60 // default to 1 day
	}

	secret, err := client.Auth().Token().RenewSelf(int(renewInterval))
	if err != nil {
		return fmt.Errorf("failed to renew Vault token: %w", err)
	}

	if secret != nil && secret.Auth != nil {
		Logger.Debug("Vault token renewed successfully",
			slog.Int("ttl", secret.Auth.LeaseDuration),
			slog.Bool("renewable", secret.Auth.Renewable))
	} else {
		Logger.Debug("Vault token renewed successfully")
	}

	return nil
}

// startVaultTokenRenewal starts a background goroutine that periodically
// renews the Vault token based on the configured interval.
func startVaultTokenRenewal() {
	if Configs.VaultTokenRenewInterval <= 0 {
		Configs.VaultTokenRenewInterval = 1 // default to 1 day
	}

	// Calculate renewal interval (renew slightly before expiration)
	// Renew every (renewInterval - 1 hour) to ensure token doesn't expire
	renewInterval := time.Duration(Configs.VaultTokenRenewInterval) * 24 * time.Hour
	if renewInterval > time.Hour {
		renewInterval = renewInterval - time.Hour // renew 1 hour before expiration
	}

	vaultRenewStop = make(chan struct{})
	vaultRenewWg.Add(1)

	go func() {
		defer vaultRenewWg.Done()
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		// Initial renewal attempt after a short delay
		time.Sleep(5 * time.Second)

		for {
			select {
			case <-ticker.C:
				if err := renewVaultToken(); err != nil {
					Logger.Warn("Failed to renew Vault token in background", slog.String("error", err.Error()))
				}
			case <-vaultRenewStop:
				Logger.Info("Vault token renewal stopped")
				return
			}
		}
	}()

	Logger.Info("Vault token renewal started",
		slog.Int("renew_interval_days", Configs.VaultTokenRenewInterval))
}

// vaultResolver implements ExternalResolver for HashiCorp Vault KV v2.
// It resolves references of the form:
//   vault:<path>#<key>
//
// Example:
//   vault:kv/data/dev#CLIENT_SECRET
//
// It is KV v2 aware: if response contains "data" object, it reads keys from it.
type vaultResolver struct{}

func (r *vaultResolver) Resolve(ref string) (string, bool, error) {
	if !strings.HasPrefix(ref, "vault:") {
		return "", false, nil
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

	vaultClientMu.RLock()
	client := vaultClient
	vaultClientMu.RUnlock()

	if client == nil {
		return "", true, fmt.Errorf("Vault is not initialized")
	}

	secret, err := client.Logical().Read(secretPath)
	if err != nil {
		// Check if error is due to expired token and try to renew
		errStr := err.Error()
		if strings.Contains(errStr, "permission denied") ||
			strings.Contains(errStr, "token") ||
			strings.Contains(errStr, "unauthorized") ||
			strings.Contains(errStr, "403") {
			// Attempt to renew token and retry once
			Logger.Debug("Vault access error detected, attempting token renewal", slog.String("error", errStr))
			if renewErr := renewVaultToken(); renewErr == nil {
				// Retry the read after renewal
				secret, err = client.Logical().Read(secretPath)
				if err != nil {
					return "", true, fmt.Errorf("could not read Vault path %s after token renewal: %w", secretPath, err)
				}
			} else {
				return "", true, fmt.Errorf("could not read Vault path %s: %w (token renewal also failed: %v)", secretPath, err, renewErr)
			}
		} else {
			return "", true, fmt.Errorf("could not read Vault path %s: %w", secretPath, err)
		}
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

	// Convert any scalar type to string; this makes it work even if
	// Vault stored the value as non-string (number, bool, etc).
	switch v := val.(type) {
	case string:
		return v, true, nil
	case []byte:
		return string(v), true, nil
	default:
		return fmt.Sprint(v), true, nil
	}
}

