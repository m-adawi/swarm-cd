package swarmcd

import (
	"bytes"
	"os"
	"path"
	"sync"
	"testing"
)

// External objects are ignored by the rotation
func TestRotateExternalObjects(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	var valuesMap map[string]any
	stack := NewSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, valuesMap, "template")
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
	var valuesMap map[string]any
	stack := NewSwarmStack("test", repo, "main", "stacks/docker-compose.yaml", nil, "", false, valuesMap, "template")
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

func TestStackGeneration(t *testing.T) {
	type args struct {
		compose        string
		valueFile      string
		globalFile     string
		templateFolder string
	}
	tests := []struct {
		name      string
		args      args
		expected  string
		templated bool
	}{
		{name: "novar", args: args{compose: "basic_compose.yaml", valueFile: "", globalFile: "", templateFolder: ""}, expected: "basic_compose.yaml", templated: false},
		{name: "novarused", args: args{compose: "basic_compose.yaml", valueFile: "values.yaml", globalFile: "globals.yaml", templateFolder: ""}, expected: "basic_compose.yaml", templated: false},
		{name: "unusedtemplates", args: args{compose: "basic_compose.yaml", valueFile: "", globalFile: "", templateFolder: "templates"}, expected: "basic_compose.yaml", templated: false},
		{name: "unusedinvalidtemplates", args: args{compose: "basic_compose.yaml", valueFile: "", globalFile: "", templateFolder: "templates_invalid"}, expected: "basic_compose.yaml", templated: false},

		{name: "varreplacement", args: args{compose: "replacement_compose.yaml", valueFile: "values.yaml", globalFile: "", templateFolder: ""}, expected: "varreplacement_expected.yaml", templated: true},
		{name: "globalreplacement", args: args{compose: "replacement_compose.yaml", valueFile: "", globalFile: "globals.yaml", templateFolder: ""}, expected: "globalreplacement_expected.yaml", templated: true},
		{name: "override", args: args{compose: "replacement_compose.yaml", valueFile: "values.yaml", globalFile: "globals.yaml", templateFolder: ""}, expected: "override_expected.yaml", templated: true},
		{name: "basictemplate", args: args{compose: "basictemplate_compose.yaml", valueFile: "", globalFile: "", templateFolder: "templates"}, expected: "basictemplate_expected.yaml", templated: true},
		{name: "varintemplate", args: args{compose: "varintemplate_compose.yaml", valueFile: "values.yaml", globalFile: "globals.yaml", templateFolder: "templates"}, expected: "varintemplate_expected.yaml", templated: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &stackRepo{name: tt.name, path: "../test_data/stack_test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
			var globalValuesMap map[string]any
			var err error
			if tt.args.globalFile != "" {
				globalPath := path.Join(repo.path, tt.args.globalFile)
				globalValuesMap, err = ParseValuesFile(globalPath, "global")
				if err != nil {
					t.Errorf("%s: global file %s could not be parsed: %s", tt.name, globalPath, err)
				}

			}
			stack := NewSwarmStack("test", repo, "main", tt.args.compose, nil, tt.args.valueFile, false, globalValuesMap, tt.args.templateFolder)
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
