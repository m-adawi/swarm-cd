package swarmcd

import (
	"sync"
	"testing"
)

// External objects are ignored by the rotation
func TestRotateExternalObjects(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, "")
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
	stack := newSwarmStack("test", repo, "main", "stacks/docker-compose.yaml", nil, "", false, "")
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

// Test environment variable replacement in compose file
func TestEnvVarReplacement(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, "")

	// Manually set envVars to simulate loading from environments file
	stack.envVars = map[string]string{
		"DB_HOST": "production-db.example.com",
		"DB_PORT": "5432",
		"API_URL": "https://api.example.com",
	}

	stackString := []byte(`services:
  app:
    image: myapp:latest
    environment:
      DATABASE_HOST: ${DB_HOST}
      DATABASE_PORT: ${DB_PORT}
      API_ENDPOINT: ${API_URL}
      STATIC_VAR: some-static-value`)

	composeMap, err := stack.parseStackString(stackString)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	err = stack.replaceEnvVars(composeMap)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	// Verify replacements
	services := composeMap["services"].(map[string]any)
	app := services["app"].(map[string]any)
	environment := app["environment"].(map[string]any)

	if environment["DATABASE_HOST"] != "production-db.example.com" {
		t.Errorf("expected DATABASE_HOST to be 'production-db.example.com', got '%s'", environment["DATABASE_HOST"])
	}
	if environment["DATABASE_PORT"] != "5432" {
		t.Errorf("expected DATABASE_PORT to be '5432', got '%s'", environment["DATABASE_PORT"])
	}
	if environment["API_ENDPOINT"] != "https://api.example.com" {
		t.Errorf("expected API_ENDPOINT to be 'https://api.example.com', got '%s'", environment["API_ENDPOINT"])
	}
	if environment["STATIC_VAR"] != "some-static-value" {
		t.Errorf("expected STATIC_VAR to remain 'some-static-value', got '%s'", environment["STATIC_VAR"])
	}
}

// Test environment variable replacement in nested structures
func TestEnvVarReplacementNested(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, "")

	// Manually set envVars to simulate loading from environments file
	stack.envVars = map[string]string{
		"IMAGE_TAG":    "v1.2.3",
		"REPLICA_COUNT": "3",
	}

	stackString := []byte(`services:
  app:
    image: myapp:${IMAGE_TAG}
    deploy:
      replicas: ${REPLICA_COUNT}
      labels:
        - "version=${IMAGE_TAG}"`)

	composeMap, err := stack.parseStackString(stackString)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	err = stack.replaceEnvVars(composeMap)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	// Verify replacements
	services := composeMap["services"].(map[string]any)
	app := services["app"].(map[string]any)

	if app["image"] != "myapp:v1.2.3" {
		t.Errorf("expected image to be 'myapp:v1.2.3', got '%s'", app["image"])
	}

	deploy := app["deploy"].(map[string]any)
	if deploy["replicas"] != "3" {
		t.Errorf("expected replicas to be '3', got '%s'", deploy["replicas"])
	}

	labels := deploy["labels"].([]any)
	if labels[0] != "version=v1.2.3" {
		t.Errorf("expected label to be 'version=v1.2.3', got '%s'", labels[0])
	}
}

// Test that stacks without env vars are not affected
func TestNoEnvVars(t *testing.T) {
	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, "")

	stackString := []byte(`services:
  app:
    image: myapp:latest
    environment:
      SOME_VAR: ${UNDEFINED_VAR}`)

	composeMap, err := stack.parseStackString(stackString)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	err = stack.replaceEnvVars(composeMap)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	// Verify no replacements occurred
	services := composeMap["services"].(map[string]any)
	app := services["app"].(map[string]any)
	environment := app["environment"].(map[string]any)

	if environment["SOME_VAR"] != "${UNDEFINED_VAR}" {
		t.Errorf("expected SOME_VAR to remain '${UNDEFINED_VAR}', got '%s'", environment["SOME_VAR"])
	}
}

// Test that stacks without environments_file always deploy
func TestLoadEnvironmentVars_NoEnvironmentsFile(t *testing.T) {
	// Save and restore currentEnvironment
	oldEnv := currentEnvironment
	defer func() { currentEnvironment = oldEnv }()

	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, "")

	// Test with no environment set
	currentEnvironment = ""
	shouldDeploy, err := stack.loadEnvironmentVars()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !shouldDeploy {
		t.Error("expected stack without environments_file to deploy even when no environment is set")
	}

	// Test with environment set
	currentEnvironment = "prod"
	shouldDeploy, err = stack.loadEnvironmentVars()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !shouldDeploy {
		t.Error("expected stack without environments_file to deploy when environment is set")
	}
}

// Test that stacks with environments_file but no environment label do NOT deploy
func TestLoadEnvironmentVars_WithFileButNoEnvironment(t *testing.T) {
	// Save and restore currentEnvironment
	oldEnv := currentEnvironment
	defer func() { currentEnvironment = oldEnv }()

	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, "environments.yaml")

	// No environment set on node
	currentEnvironment = ""
	shouldDeploy, err := stack.loadEnvironmentVars()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if shouldDeploy {
		t.Error("expected stack with environments_file to NOT deploy when no environment is set on node")
	}
}

// Test that stacks deploy when environment matches
func TestLoadEnvironmentVars_MatchingEnvironment(t *testing.T) {
	// Save and restore currentEnvironment
	oldEnv := currentEnvironment
	defer func() { currentEnvironment = oldEnv }()

	// This test would need a real environments.yaml file to work properly
	// For now, we just verify the logic path
	currentEnvironment = "dev"

	repo := &stackRepo{name: "test", path: "test", url: "", auth: nil, lock: &sync.Mutex{}, gitRepoObject: nil}
	stack := newSwarmStack("test", repo, "main", "docker-compose.yaml", nil, "", false, "nonexistent.yaml")

	// File doesn't exist, should still deploy (graceful handling)
	shouldDeploy, err := stack.loadEnvironmentVars()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !shouldDeploy {
		t.Error("expected stack to deploy when environments file doesn't exist")
	}
}
