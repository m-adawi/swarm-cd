package swarmcd

import (
	"bytes"
	"os"
	"path"
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
			s := NewSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, tt.stackPull, nil)

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
	stack := NewSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, nil, nil)
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
	stack := NewSwarmStack("test", repo, "main", "stacks/docker-compose.yaml", nil, "", false, nil, nil)
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

func TestStackGeneration(t *testing.T) {
	type args struct {
		compose       string
		valueFile     string
		globalFile    string
		templatesPath string
	}
	tests := []struct {
		name      string
		args      args
		expected  string
		templated bool
	}{
		{name: "novar", args: args{compose: "basic_compose.yaml", valueFile: "", globalFile: "", templatesPath: ""}, expected: "basic_compose.yaml", templated: false},
		{name: "novarused", args: args{compose: "basic_compose.yaml", valueFile: "values.yaml", globalFile: "globals.yaml", templatesPath: ""}, expected: "basic_compose.yaml", templated: false},
		{name: "unusedtemplates", args: args{compose: "basic_compose.yaml", valueFile: "", globalFile: "", templatesPath: "templates"}, expected: "basic_compose.yaml", templated: false},
		{name: "unusedinvalidtemplates", args: args{compose: "basic_compose.yaml", valueFile: "", globalFile: "", templatesPath: "templates_invalid"}, expected: "basic_compose.yaml", templated: false},

		{name: "varreplacement", args: args{compose: "replacement_compose.yaml", valueFile: "values.yaml", globalFile: "", templatesPath: ""}, expected: "varreplacement_expected.yaml", templated: true},
		{name: "globalreplacement", args: args{compose: "replacement_compose.yaml", valueFile: "", globalFile: "globals.yaml", templatesPath: ""}, expected: "globalreplacement_expected.yaml", templated: true},
		{name: "override", args: args{compose: "replacement_compose.yaml", valueFile: "values.yaml", globalFile: "globals.yaml", templatesPath: ""}, expected: "override_expected.yaml", templated: true},
		{name: "basictemplate", args: args{compose: "basictemplate_compose.yaml", valueFile: "", globalFile: "", templatesPath: "templates"}, expected: "basictemplate_expected.yaml", templated: true},
		{name: "varintemplate", args: args{compose: "varintemplate_compose.yaml", valueFile: "values.yaml", globalFile: "globals.yaml", templatesPath: "templates"}, expected: "varintemplate_expected.yaml", templated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stackRepo{name: tt.name, path: "../test_data/stack_test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil, templatesPath: tt.args.templatesPath}
			var globalValuesMap map[string]any
			var err error
			if tt.args.globalFile != "" {
				globalPath := path.Join(repo.path, tt.args.globalFile)
				globalValuesMap, err = ParseValuesFile(globalPath, "global")
				if err != nil {
					t.Errorf("%s: global file %s could not be parsed: %s", tt.name, globalPath, err)
				}

			}
			stack := NewSwarmStack("test", repo, "main", tt.args.compose, nil, tt.args.valueFile, false, nil, globalValuesMap)
			stack.UpdateTemplatesPath("")
			stackBytes, err := stack.GenerateStack()
			if err != nil {
				t.Errorf("%s: unexpected error: %s", tt.name, err)
			}
			if stack.templated != tt.templated {
				t.Errorf("%s: Template flag was not set correctly", tt.name)
			}
			expectedPath := path.Join(stack.repo.path, tt.expected)
			expectedStack, err := os.ReadFile(expectedPath)
			if !bytes.Equal(stackBytes, expectedStack) {
				t.Errorf("%s: generated stack is different from what was expected: %s", tt.name, string(stackBytes))
			}
		})
	}
}
