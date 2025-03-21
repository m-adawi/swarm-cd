package swarmcd

import (
	"crypto/md5"
	"fmt"
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

func TestRotateObjects(t *testing.T) {
	fileName, fileContent, swarm := setupTestStack(t)

	objects := map[string]any{
		"service1": map[string]any{"file": fileName},
	}

	err := swarm.rotateObjects(objects)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedHash := fmt.Sprintf("%x", md5.Sum(fileContent))[:8]
	expectedName := "test-stack-service1-" + expectedHash
	if objects["service1"].(map[string]any)["name"] != expectedName {
		t.Errorf("Expected name %s, got %s", expectedName, objects["service1"].(map[string]any)["name"])
	}
}

func TestRotateObjectsHandlesExternalTrue(t *testing.T) {
	configFile, _, swarm := setupTestStack(t)

	objects := map[string]any{
		"config1": map[string]any{"external": true},
		"config2": map[string]any{"file": configFile},
	}

	err := swarm.rotateObjects(objects)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if _, exists := objects["config1"].(map[string]any)["name"]; exists {
		t.Errorf("Expected config1 to be unmodified, but 'name' was set")
	}
}

func TestRotateObjectsInvalidMap(t *testing.T) {
	_, _, swarm := setupTestStack(t)

	objects := map[string]any{"service1": "invalid"}

	err := swarm.rotateObjects(objects)
	if err == nil {
		t.Fatalf("Expected an error but got none")
	}
	expectedErr := "invalid compose file: service1 object must be a map"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestRotateObjectsMissingFileField(t *testing.T) {
	_, _, swarm := setupTestStack(t)

	objects := map[string]any{"service1": map[string]any{}}

	err := swarm.rotateObjects(objects)
	if err == nil {
		t.Fatalf("Expected an error but got none")
	}
	expectedErr := "invalid compose file: service1 file field must be a string"
	if err.Error() != expectedErr {
		t.Errorf("Expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestRotateObjectsFileNotFound(t *testing.T) {
	swarm := &swarmStack{name: "test-stack", repo: &stackRepo{path: "nonexistent"}, composePath: "docker-compose.yml"}
	objects := map[string]any{"service1": map[string]any{"file": "missing.txt"}}

	err := swarm.rotateObjects(objects)
	if err == nil {
		t.Fatalf("Expected an error but got none")
	}
}

func setupTestStack(t *testing.T) (string, []byte, *swarmStack) {
	tempDir := t.TempDir()
	fileName := "testfile.txt"
	filePath := path.Join(tempDir, fileName)
	fileContent := []byte("test content")
	os.WriteFile(filePath, fileContent, 0644)

	repo := &stackRepo{path: tempDir}
	swarm := &swarmStack{name: "test-stack", repo: repo, composePath: "docker-compose.yml"}
	return fileName, fileContent, swarm
}
