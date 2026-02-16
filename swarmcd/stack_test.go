package swarmcd

import (
	"sync"
	"testing"
)

// External objects are ignored by the rotation
func TestRotateExternalObjects(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "", "docker-compose.yaml", nil, "", false)
	objects := map[string]any{
		"my-secret": map[string]any{"external": true},
	}
	err := stack.rotateObjects(objects, "secrets")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

// Secrets are discovered, external secrets are ignored
func TestSecretDiscovery(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "", "stacks/docker-compose.yaml", nil, "", false)
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

// refAttr returns branch attribute when branch is set
func TestRefAttrBranch(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "", "docker-compose.yaml", nil, "", false)
	attr := stack.refAttr()
	if attr.Key != "branch" {
		t.Errorf("expected key 'branch', got '%s'", attr.Key)
	}
	if attr.Value.String() != "main" {
		t.Errorf("expected value 'main', got '%s'", attr.Value.String())
	}
}

// refAttr returns tag attribute when tag is set
func TestRefAttrTag(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "", "v1.2.3", "docker-compose.yaml", nil, "", false)
	attr := stack.refAttr()
	if attr.Key != "tag" {
		t.Errorf("expected key 'tag', got '%s'", attr.Key)
	}
	if attr.Value.String() != "v1.2.3" {
		t.Errorf("expected value 'v1.2.3', got '%s'", attr.Value.String())
	}
}

// newSwarmStack correctly stores tag
func TestNewSwarmStackWithTag(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("mystack", repo, "", "v2.0.0", "compose.yaml", nil, "", false)
	if stack.tag != "v2.0.0" {
		t.Errorf("expected tag 'v2.0.0', got '%s'", stack.tag)
	}
	if stack.branch != "" {
		t.Errorf("expected empty branch, got '%s'", stack.branch)
	}
}

// Verify refAttr prefers tag over branch when both could theoretically be set
func TestRefAttrPrefersTag(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	// Even if branch has a value, tag takes precedence
	stack := newSwarmStack("test", repo, "main", "v1.0.0", "docker-compose.yaml", nil, "", false)
	attr := stack.refAttr()
	if attr.Key != "tag" {
		t.Errorf("expected key 'tag' to take precedence, got '%s'", attr.Key)
	}
}
