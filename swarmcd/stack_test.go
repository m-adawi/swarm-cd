package swarmcd

import (
	"sync"
	"testing"
)

// External objects are ignored by the rotation
func TestRotateExternalObjects(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false)
	objects := map[string]any{
		"my-secret": map[string]any{"external": true},
		"my-plugin-external-secret": map[string]any{
			"driver": "my-driver", "labels": map[string]string{"my_option": "value"},
		},
	}
	err := stack.rotateObjects(objects, "secrets")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// Secrets are discovered, external secrets are ignored
func TestSecretDiscovery(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "stacks/docker-compose.yaml", nil, "", false)
	stackString := []byte(`
secrets:
  my-secret:
    file: secrets/secret.yaml
  my-external-secret:
    external: true
  my-plugin-external-secret:
    driver: my-driver
    labels:
      my_option: value
`)
	composeMap, err := stack.parseStackString(stackString)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	sopsFiles, err := discoverSecrets(composeMap, stack.composePath)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}
	if len(sopsFiles) != 1 {
		t.Errorf("unexpected number of sops files: %d", len(sopsFiles))
		return
	}
	if sopsFiles[0] != "stacks/secrets/secret.yaml" {
		t.Errorf("unexpected sops file: %s", sopsFiles[0])
	}
}
