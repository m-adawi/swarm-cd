package swarmcd

import (
	"os"
	"path"
	"slices"
	"sync"
	"testing"

	"github.com/m-adawi/swarm-cd/util"
	"github.com/stretchr/testify/require"
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
	err := stack.rotateObjects(objects, "secrets", t.TempDir())
	require.NoError(t, err)
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
	require.NoError(t, err)

	sopsFiles, err := discoverSecrets(composeMap, "stacks")
	require.NoError(t, err)
	require.Equal(t, []string{"stacks/secrets/secret.yaml"}, sopsFiles)
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
	require.NoError(t, err)

	sopsFiles, err := discoverSecrets(composeMap, "stacks")
	require.NoError(t, err)
	require.Len(t, sopsFiles, 2)
	require.True(t, slices.Contains(sopsFiles, "/run/secrets/absolute.yaml"))
	require.True(t, slices.Contains(sopsFiles, "stacks/secrets/relative.yaml"))
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
	require.NoError(t, err)
	require.Equal(t, []string{
		"/repo/tenant-a/secrets/tenant-a.enc.yaml",
		"/repo/tenant-b/secrets/tenant-b.enc.yaml",
	}, sopsFiles)
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
	require.NoError(t, err)
	require.Equal(t, []string{path.Join(repoPath, "tenant", "secrets", "shared.enc.yaml")}, resolvedPaths)
}

func TestWriteDeploymentArtifactUsesSourceComposeDirectory(t *testing.T) {
	repoPath := t.TempDir()
	composeDir := path.Join(repoPath, "tenant")
	require.NoError(t, os.MkdirAll(composeDir, 0o755))

	repo := &stackRepo{name: "test", path: repoPath, url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", []string{"tenant/compose.yaml"}, nil, "", false, nil)

	artifactPath, err := stack.writeDeploymentArtifact(map[string]any{
		"services": map[string]any{
			"app": map[string]any{"image": "nginx"},
		},
	}, path.Join(composeDir, "compose.yaml"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(artifactPath) })

	require.Equal(t, composeDir, path.Dir(artifactPath))
	artifactInfo, err := os.Stat(artifactPath)
	require.NoError(t, err)
	require.False(t, artifactInfo.IsDir())
}

func TestDeployStackArgsPreservesConfiguredOrderAndRepeats(t *testing.T) {
	originalConfig := config
	config = &util.Config{AlwaysPullContainers: true}
	t.Cleanup(func() { config = originalConfig })

	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test-stack", repo, "main", []string{"a.yaml", "b.yaml"}, nil, "", false, nil)

	args := stack.deployStackArgs([]string{"a.yaml", "b.yaml", "a.yaml"})
	require.Equal(t, []string{
		"deploy", "--detach", "--with-registry-auth",
		"--resolve-image", "always",
		"-c", "a.yaml",
		"-c", "b.yaml",
		"-c", "a.yaml",
		"test-stack",
	}, args)
}
