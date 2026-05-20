package swarmcd

import (
	"sync"
	"testing"

	"github.com/m-adawi/swarm-cd/util"
)

func boolPtr(v bool) *bool { return &v }

// test all the possible combinations of config and stack AlwaysPullContainers settings.
func TestResolveImageMode(t *testing.T) {
	tests := []struct {
		name           string
		configPull     bool
		stackPull      *bool
		expectedResult string
	}{
		// Stack setting is unset: falls back to global config
		{"config=false stack=unset", false, nil, "changed"},
		{"config=true  stack=unset", true, nil, "always"},
		// Stack setting is explicit: overrides global config
		{"config=false stack=false", false, boolPtr(false), "changed"},
		{"config=false stack=true", false, boolPtr(true), "always"},
		{"config=true  stack=false", true, boolPtr(false), "changed"},
		{"config=true  stack=true", true, boolPtr(true), "always"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Point the package-level config at a fresh Config value so
			// parallel / sequential runs don't interfere with each other.
			originalConfig := config
			config = &util.Config{AlwaysPullContainers: tt.configPull}
			t.Cleanup(func() { config = originalConfig })

			repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
			s := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, tt.stackPull)

			got := s.resolveImageMode()
			if got != tt.expectedResult {
				t.Errorf("resolveImageMode() = %q, want %q", got, tt.expectedResult)
			}
		})
	}
}

// Non-file objects are ignored by the rotation
func TestRotateObjectsWithoutFile(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, nil)
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
	stack := newSwarmStack("test", repo, "main", "stacks/docker-compose.yaml", nil, "", false, nil)
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
