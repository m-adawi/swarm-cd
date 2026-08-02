package swarmcd

import (
	"os"
	"path"
	"slices"
	"sync"
	"testing"

	"github.com/m-adawi/swarm-cd/util"
)

// test all the possible combinations of config and stack AlwaysPullContainers settings.
func TestResolveImageMode(t *testing.T) {
	stackFalse := false
	stackTrue := true
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
		{"config=false stack=false", false, &stackFalse, "changed"},
		{"config=false stack=true", false, &stackTrue, "always"},
		{"config=true  stack=false", true, &stackFalse, "changed"},
		{"config=true  stack=true", true, &stackTrue, "always"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Point the package-level config at a fresh Config value so
			// parallel / sequential runs don't interfere with each other.
			originalConfig := config
			config = &util.Config{AlwaysPullContainers: tt.configPull}
			t.Cleanup(func() { config = originalConfig })

			repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
			s := newSwarmStack("test", repo, "main", []string{"docker-compose.yaml"}, nil, "", false, tt.stackPull)

			got := s.resolveImageMode()
			if got != tt.expectedResult {
				t.Errorf("resolveImageMode() = %q, want %q", got, tt.expectedResult)
			}
		})
	}
}

// Non-file objects are ignored by the rotation.
func TestRotateObjectsWithoutFile(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", []string{"docker-compose.yaml"}, nil, "", false, nil)
	objects := map[string]any{
		"my-secret": map[string]any{"external": true},
		"my-plugin-external-secret": map[string]any{
			"driver": "my-driver", "labels": map[string]string{"my_option": "value"},
		},
	}
	if err := stack.rotateObjects(objects, "secrets", t.TempDir()); err != nil {
		t.Fatalf("rotateObjects() error: %v", err)
	}
}

// Secrets are discovered, external secrets are ignored.
func TestSecretDiscovery(t *testing.T) {
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
	composeMap, err := parseStackString(stackString)
	if err != nil {
		t.Fatalf("parseStackString() error: %v", err)
	}

	sopsFiles, err := discoverSecrets(composeMap, "stacks")
	if err != nil {
		t.Fatalf("discoverSecrets() error: %v", err)
	}
	if !slices.Equal(sopsFiles, []string{"stacks/secrets/secret.yaml"}) {
		t.Errorf("discoverSecrets() = %v, want %v", sopsFiles, []string{"stacks/secrets/secret.yaml"})
	}
}

// Secret file paths that are already absolute stay absolute.
func TestSecretDiscoveryKeepsAbsolutePaths(t *testing.T) {
	stackString := []byte(`secrets:
  absolute-secret:
    file: /run/secrets/absolute.yaml
  relative-secret:
    file: secrets/relative.yaml
`)
	composeMap, err := parseStackString(stackString)
	if err != nil {
		t.Fatalf("parseStackString() error: %v", err)
	}

	sopsFiles, err := discoverSecrets(composeMap, "stacks")
	if err != nil {
		t.Fatalf("discoverSecrets() error: %v", err)
	}
	if len(sopsFiles) != 2 {
		t.Fatalf("discoverSecrets() len = %d, want 2", len(sopsFiles))
	}
	if !slices.Contains(sopsFiles, "/run/secrets/absolute.yaml") {
		t.Errorf("discoverSecrets() = %v, missing %q", sopsFiles, "/run/secrets/absolute.yaml")
	}
	if !slices.Contains(sopsFiles, "stacks/secrets/relative.yaml") {
		t.Errorf("discoverSecrets() = %v, missing %q", sopsFiles, "stacks/secrets/relative.yaml")
	}
}

func TestDiscoverSecretsFromComposeFilesUsesEachComposeDirectory(t *testing.T) {
	composeArtifacts := []composeArtifact{
		{
			sourcePath: "/repo/tenant-a/compose.yaml",
			composeMap: map[string]any{
				"secrets": map[string]any{
					"tenant-a-secret": map[string]any{"file": "secrets/tenant-a.enc.yaml"},
				},
			},
		},
		{
			sourcePath: "/repo/tenant-b/compose.yaml",
			composeMap: map[string]any{
				"secrets": map[string]any{
					"tenant-b-secret": map[string]any{"file": "secrets/tenant-b.enc.yaml"},
				},
			},
		},
	}

	sopsFiles, err := discoverSecretsFromComposeFiles(composeArtifacts)
	if err != nil {
		t.Fatalf("discoverSecretsFromComposeFiles() error: %v", err)
	}
	want := []string{
		"/repo/tenant-a/secrets/tenant-a.enc.yaml",
		"/repo/tenant-b/secrets/tenant-b.enc.yaml",
	}
	if !slices.Equal(sopsFiles, want) {
		t.Errorf("discoverSecretsFromComposeFiles() = %v, want %v", sopsFiles, want)
	}
}

func TestResolveSopsFilesForDecryptionDeduplicatesDiscoveredPaths(t *testing.T) {
	repoPath := t.TempDir()
	repo := &stackRepo{name: "test", path: repoPath, url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", []string{"compose-a.yaml", "compose-b.yaml"}, nil, "", true, nil)

	composeArtifacts := []composeArtifact{
		{
			sourcePath: path.Join(repoPath, "tenant", "compose-a.yaml"),
			composeMap: map[string]any{
				"secrets": map[string]any{
					"shared-secret-a": map[string]any{"file": "secrets/shared.enc.yaml"},
				},
			},
		},
		{
			sourcePath: path.Join(repoPath, "tenant", "compose-b.yaml"),
			composeMap: map[string]any{
				"secrets": map[string]any{
					"shared-secret-b": map[string]any{"file": "secrets/shared.enc.yaml"},
				},
			},
		},
	}

	resolvedPaths, err := stack.resolveSopsFilesForDecryption(composeArtifacts)
	if err != nil {
		t.Fatalf("resolveSopsFilesForDecryption() error: %v", err)
	}
	want := []string{path.Join(repoPath, "tenant", "secrets", "shared.enc.yaml")}
	if !slices.Equal(resolvedPaths, want) {
		t.Errorf("resolveSopsFilesForDecryption() = %v, want %v", resolvedPaths, want)
	}
}

func TestWriteDeploymentArtifactUsesSourceComposeDirectory(t *testing.T) {
	repoPath := t.TempDir()
	composeDir := path.Join(repoPath, "tenant")
	if err := os.MkdirAll(composeDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error: %v", err)
	}

	repo := &stackRepo{name: "test", path: repoPath, url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", []string{"tenant/compose.yaml"}, nil, "", false, nil)

	artifactPath, err := stack.writeDeploymentArtifact(map[string]any{
		"services": map[string]any{
			"app": map[string]any{"image": "nginx"},
		},
	}, path.Join(composeDir, "compose.yaml"))
	if err != nil {
		t.Fatalf("writeDeploymentArtifact() error: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(artifactPath) })

	if got := path.Dir(artifactPath); got != composeDir {
		t.Errorf("writeDeploymentArtifact() dir = %q, want %q", got, composeDir)
	}
	artifactInfo, err := os.Stat(artifactPath)
	if err != nil {
		t.Fatalf("os.Stat() error: %v", err)
	}
	if artifactInfo.IsDir() {
		t.Errorf("writeDeploymentArtifact() artifact %q is a directory", artifactPath)
	}
}

func TestDeployStackArgsPreservesConfiguredOrderAndRepeats(t *testing.T) {
	originalConfig := config
	config = &util.Config{AlwaysPullContainers: true}
	t.Cleanup(func() { config = originalConfig })

	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test-stack", repo, "main", []string{"a.yaml", "b.yaml"}, nil, "", false, nil)

	args := stack.deployStackArgs([]string{"a.yaml", "b.yaml", "a.yaml"})
	want := []string{
		"deploy", "--detach", "--with-registry-auth",
		"--resolve-image", "always",
		"-c", "a.yaml",
		"-c", "b.yaml",
		"-c", "a.yaml",
		"test-stack",
	}
	if !slices.Equal(args, want) {
		t.Errorf("deployStackArgs() = %v, want %v", args, want)
	}
}
