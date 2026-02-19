package util

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	consulapi "github.com/hashicorp/consul/api"
)

var consulClient *consulapi.Client

// InitConsul initializes a shared Consul client and registers a resolver
// for values coming from Consul KV.
//
// If consul_address is empty, Consul integration is disabled (no-op).
//
// Token resolution order:
// - config.yaml: consul_token
// - env: CONSUL_TOKEN
func InitConsul() error {
	if Configs.ConsulAddress == "" {
		Logger.Info("Consul integration disabled (consul_address is empty)")
		return nil
	}

	cfg := consulapi.DefaultConfig()
	cfg.Address = Configs.ConsulAddress

	token := strings.TrimSpace(Configs.ConsulToken)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("CONSUL_TOKEN"))
	}
	if token != "" {
		cfg.Token = token
	}

	client, err := consulapi.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("could not create Consul client: %w", err)
	}

	consulClient = client

	// Register Consul as an external value resolver.
	RegisterExternalResolver(&consulResolver{})
	return nil
}

// consulResolver implements ExternalResolver for Consul KV.
// It resolves references of the form:
//   consul:<key>
//   consul:<key>#<field>
//
// Examples:
//   consul:variables/dev/SECRET           -> whole value
//   consul:variables/dev/SECRET#password -> JSON field "password" from the value
//
// If no "#field" is specified, the raw value is returned as string.
// If "#field" is specified, the value is treated as JSON and the field is extracted.
type consulResolver struct{}

func (r *consulResolver) Resolve(ref string) (string, bool, error) {
	if !strings.HasPrefix(ref, "consul:") {
		return "", false, nil
	}
	if consulClient == nil {
		return "", true, fmt.Errorf("Consul is not initialized")
	}

	withoutPrefix := strings.TrimPrefix(ref, "consul:")
	parts := strings.SplitN(withoutPrefix, "#", 2)
	key := strings.TrimSpace(parts[0])
	var field string
	if len(parts) == 2 {
		field = strings.TrimSpace(parts[1])
	}
	if key == "" {
		return "", true, fmt.Errorf("invalid consul reference %q, key must be non-empty", ref)
	}
	if len(parts) == 2 && field == "" {
		return "", true, fmt.Errorf("invalid consul reference %q, field after # must be non-empty", ref)
	}

	kv := consulClient.KV()
	pair, _, err := kv.Get(key, nil)
	if err != nil {
		return "", true, fmt.Errorf("could not read Consul key %s: %w", key, err)
	}
	if pair == nil || pair.Value == nil {
		return "", true, fmt.Errorf("Consul key %s not found or empty", key)
	}

	// No field requested: return raw value as string.
	if field == "" {
		return string(pair.Value), true, nil
	}

	// Field requested: interpret the value as JSON and extract the field.
	var data map[string]any
	if err := json.Unmarshal(pair.Value, &data); err != nil {
		return "", true, fmt.Errorf("Consul key %s does not contain valid JSON: %w", key, err)
	}

	val, ok := data[field]
	if !ok {
		return "", true, fmt.Errorf("field %s not found in JSON at Consul key %s", field, key)
	}

	// Convert any scalar type to string; this makes it work even if
	// the JSON stored the value as non-string (number, bool, etc).
	switch v := val.(type) {
	case string:
		return v, true, nil
	case []byte:
		return string(v), true, nil
	default:
		return fmt.Sprint(v), true, nil
	}
}

