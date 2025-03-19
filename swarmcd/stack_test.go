package swarmcd

import (
	"os"
	"path"
	"testing"
)

func setupSwarmStackWithValues(t *testing.T, values string) *swarmStack {
	t.Helper()
	tempDir := t.TempDir()
	valuesFilePath := path.Join(tempDir, "values.yaml")
	if err := os.WriteFile(valuesFilePath, []byte(values), 0644); err != nil {
		t.Fatalf("Failed to write values file: %v", err)
	}

	return &swarmStack{
		name:            "testStack",
		repo:            &stackRepo{path: tempDir},
		valuesFile:      "values.yaml",
		discoverSecrets: false,
	}
}

func TestRenderComposeTemplate(t *testing.T) {
	tests := []struct {
		name     string
		values   string
		template string
		expected string
	}{
		{"Basic", "key: value", "Service: {{ .Values.key }}", "Service: value"},
		{"Nested", "nested:\n  key: nestedValue", "Service: {{ .Values.nested.key }}", "Service: nestedValue"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			swarm := setupSwarmStackWithValues(t, tt.values)
			result, err := swarm.renderComposeTemplate([]byte(tt.template))
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if string(result) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, string(result))
			}
		})
	}
}
