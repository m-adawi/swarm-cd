package swarmcd

import (
	"os"
	"path"
	"sync"
	"testing"
)

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

// External objects are ignored by the rotation
func TestRotateObjectsHandlesExternalTrue(t *testing.T) {
	configFile, _, swarm := setupTestStack(t)

	objects := map[string]any{
		"config1": map[string]any{"external": true},
		"config2": map[string]any{"file": configFile},
	}

	err := swarm.rotateObjects(objects, "secrets")
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

	err := swarm.rotateObjects(objects, "secrets")
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

	err := swarm.rotateObjects(objects, "secrets")
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

	err := swarm.rotateObjects(objects, "secrets")
	if err == nil {
		t.Fatalf("Expected an error but got none")
	}
}

// Secrets are discovered, external secrets are ignored
func TestSecretDiscovery(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "stacks/docker-compose.yaml", nil, "", false)
	stackString := []byte(`services:
  my-service:
    image: my-image
    secrets:
      - my-secret
      - my-external-secret
secrets:
  my-secret:
    file: secrets/secret.yaml
  my-external-secret:
    external: true`)
	composeMap, err := stack.parseStackString(stackString)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	sopsFiles, err := discoverSecrets(composeMap, stack.composePath)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if len(sopsFiles) != 1 {
		t.Errorf("unexpected number of sops files: %d", len(sopsFiles))
	}
	if sopsFiles[0] != "stacks/secrets/secret.yaml" {
		t.Errorf("unexpected sops file: %s", sopsFiles[0])
	}
}

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
