package swarmcd

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/docker/cli/cli/command/stack"
	"github.com/goccy/go-yaml"
	"github.com/m-adawi/swarm-cd/util"
)

type swarmStack struct {
	name             string
	repo             *stackRepo
	branch           string
	composePath      string
	sopsFiles        []string
	valuesFile       string
	discoverSecrets  bool
	environmentsFile string
	envVars          map[string]string
}

type environmentsConfig struct {
	Environments map[string]map[string]string `yaml:"environments"`
}

func newSwarmStack(name string, repo *stackRepo, branch string, composePath string, sopsFiles []string, valuesFile string, discoverSecrets bool, environmentsFile string) *swarmStack {
	return &swarmStack{
		name:             name,
		repo:             repo,
		branch:           branch,
		composePath:      composePath,
		sopsFiles:        sopsFiles,
		valuesFile:       valuesFile,
		discoverSecrets:  discoverSecrets,
		environmentsFile: environmentsFile,
		envVars:          nil, // Will be loaded from environments file
	}
}

func (swarmStack *swarmStack) updateStack() (revision string, err error) {
	log := logger.With(
		slog.String("stack", swarmStack.name),
		slog.String("branch", swarmStack.branch),
	)

	log.Debug("pulling changes...")
	revision, err = swarmStack.repo.pullChanges(swarmStack.branch)
	if err != nil {
		return
	}
	log.Debug("changes pulled", "revision", revision)

	shouldDeploy, err := swarmStack.loadEnvironmentVars()
	if err != nil {
		return
	}
	if !shouldDeploy {
		log.Info("stack skipped due to environment filtering")
		return revision, nil
	}

	stackBytes, err := swarmStack.readStack()
	if err != nil {
		return
	}

	if swarmStack.valuesFile != "" {
		stackBytes, err = swarmStack.renderComposeTemplate(stackBytes)
		if err != nil {
			return
		}
	}

	stackContents, err := swarmStack.parseStackString([]byte(stackBytes))
	if err != nil {
		return
	}

	err = swarmStack.replaceEnvVars(stackContents)
	if err != nil {
		return
	}
	err = swarmStack.decryptSopsFiles(stackContents)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt one or more sops files for %s stack: %w", swarmStack.name, err)
	}

	err = swarmStack.rotateConfigsAndSecrets(stackContents)
	if err != nil {
		return
	}

	err = swarmStack.writeStack(stackContents)
	if err != nil {
		return
	}

	err = swarmStack.deployStack()
	return
}

func (swarmStack *swarmStack) readStack() ([]byte, error) {
	composeFile := path.Join(swarmStack.repo.path, swarmStack.composePath)
	composeFileBytes, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("could not read compose file %s: %w", composeFile, err)
	}
	return composeFileBytes, nil
}

func (swarmStack *swarmStack) loadEnvironmentVars() (shouldDeploy bool, err error) {
	// If no environments file is specified, always deploy (no filtering)
	if swarmStack.environmentsFile == "" {
		return true, nil
	}

	// If environments file is specified but no environment label is set on the node,
	// this stack is environment-filtered and should NOT be deployed
	if currentEnvironment == "" {
		logger.Info("skipping stack with environment filtering when no environment is set on node",
			"stack", swarmStack.name,
			"environments_file", swarmStack.environmentsFile)
		return false, nil
	}

	// Read the environments file from the repository
	envFilePath := path.Join(swarmStack.repo.path, swarmStack.environmentsFile)
	envFileBytes, err := os.ReadFile(envFilePath)
	if err != nil {
		// If file doesn't exist, it's not an error - just skip loading
		if os.IsNotExist(err) {
			logger.Warn("environments file not found, deploying without environment variables",
				"stack", swarmStack.name,
				"file", swarmStack.environmentsFile)
			return true, nil
		}
		return false, fmt.Errorf("could not read environments file %s: %w", envFilePath, err)
	}

	// Parse the environments file
	var envConfig environmentsConfig
	err = yaml.Unmarshal(envFileBytes, &envConfig)
	if err != nil {
		return false, fmt.Errorf("could not parse environments file %s: %w", envFilePath, err)
	}

	// Get the variables for the current environment
	if envVars, ok := envConfig.Environments[currentEnvironment]; ok {
		swarmStack.envVars = envVars
		if len(envVars) > 0 {
			logger.Info("loaded environment variables",
				"stack", swarmStack.name,
				"environment", currentEnvironment,
				"count", len(envVars))
		}
		return true, nil
	} else {
		// Current environment is not defined in the environments file
		// This means this stack should not be deployed in this environment
		logger.Info("skipping stack not configured for current environment",
			"stack", swarmStack.name,
			"environment", currentEnvironment,
			"available_environments", getKeys(envConfig.Environments))
		return false, nil
	}
}

func getKeys(m map[string]map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (swarmStack *swarmStack) renderComposeTemplate(templateContents []byte) ([]byte, error) {
	valuesFile := path.Join(swarmStack.repo.path, swarmStack.valuesFile)
	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return nil, fmt.Errorf("could not read %s stack values file: %w", swarmStack.name, err)
	}
	var valuesMap map[string]any
	yaml.Unmarshal(valuesBytes, &valuesMap)
	templ, err := template.New(swarmStack.name).Parse(string(templateContents[:]))
	if err != nil {
		return nil, fmt.Errorf("could not parse %s stack compose file as a Go template: %w", swarmStack.name, err)
	}
	var stackContents bytes.Buffer
	err = templ.Execute(&stackContents, map[string]map[string]any{"Values": valuesMap})
	if err != nil {
		return nil, fmt.Errorf("error rending %s stack compose template: %w", swarmStack.name, err)
	}
	return stackContents.Bytes(), nil
}

func (swarmStack *swarmStack) parseStackString(stackContent []byte) (map[string]any, error) {
	var composeMap map[string]any
	err := yaml.Unmarshal(stackContent, &composeMap)
	if err != nil {
		return nil, fmt.Errorf("could not parse stack yaml: %w", err)
	}
	return composeMap, nil
}

func (swarmStack *swarmStack) replaceEnvVars(composeMap map[string]any) error {
	// If no environment variables are defined for this stack, skip replacement
	if len(swarmStack.envVars) == 0 {
		return nil
	}

	// Recursively replace variables in the compose map
	replaceInMap(composeMap, swarmStack.envVars)

	return nil
}

// replaceInMap recursively walks through a map and replaces string values
// containing variable references like ${VAR_NAME} with their actual values
func replaceInMap(data any, envVars map[string]string) {
	switch v := data.(type) {
	case map[string]any:
		for key, value := range v {
			switch val := value.(type) {
			case string:
				// Replace environment variables in string values
				v[key] = replaceEnvInString(val, envVars)
			case map[string]any:
				// Recursively process nested maps
				replaceInMap(val, envVars)
			case []any:
				// Recursively process arrays
				replaceInMap(val, envVars)
			}
		}
	case []any:
		for i, item := range v {
			switch val := item.(type) {
			case string:
				v[i] = replaceEnvInString(val, envVars)
			case map[string]any:
				replaceInMap(val, envVars)
			case []any:
				replaceInMap(val, envVars)
			}
		}
	}
}

// replaceEnvInString replaces ${VAR_NAME} patterns in a string with their values
func replaceEnvInString(str string, envVars map[string]string) string {
	result := str
	for key, value := range envVars {
		// Replace ${VAR_NAME} format
		result = strings.ReplaceAll(result, "${"+key+"}", value)
		// Also support $VAR_NAME format (but not if followed by alphanumeric to avoid partial matches)
		// This is a simple implementation - for production you might want regex
		result = strings.ReplaceAll(result, "$"+key, value)
	}
	return result
}

func (swarmStack *swarmStack) decryptSopsFiles(composeMap map[string]any) (err error) {
	var sopsFiles []string
	if !swarmStack.discoverSecrets {
		sopsFiles = swarmStack.sopsFiles
	} else {
		sopsFiles, err = discoverSecrets(composeMap, swarmStack.composePath)
		if err != nil {
			return
		}
	}
	log := logger.With(
		slog.String("stack", swarmStack.name),
		slog.String("branch", swarmStack.branch),
	)
	for _, sopsFile := range sopsFiles {
		log.Debug("decrypting secret...", "secret", sopsFile)
		err = util.DecryptFile(path.Join(swarmStack.repo.path, sopsFile))
		if err != nil {
			return
		}
	}
	return
}

func discoverSecrets(composeMap map[string]any, composePath string) ([]string, error) {
	var sopsFiles []string
	if secrets, ok := composeMap["secrets"].(map[string]any); ok {
		for secretName, secret := range secrets {
			secretMap, ok := secret.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("invalid compose file: %s secret must be a map", secretName)
			}
			isExternal, ok := secretMap["external"].(bool)
			if ok && isExternal {
				continue
			}
			secretFile, ok := secretMap["file"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid compose file: %s file field must be a string", secretName)
			}
			secretPath := path.Join(path.Dir(composePath), secretFile)
			sopsFiles = append(sopsFiles, secretPath)
		}
	}
	return sopsFiles, nil
}

func (swarmStack *swarmStack) rotateConfigsAndSecrets(composeMap map[string]any) error {
	if configs, ok := composeMap["configs"].(map[string]any); ok {
		err := swarmStack.rotateObjects(configs, "configs")
		if err != nil {
			return fmt.Errorf("could not rotate one or more config files of stack %s: %w", swarmStack.name, err)
		}
	}
	if secrets, ok := composeMap["secrets"].(map[string]any); ok {
		err := swarmStack.rotateObjects(secrets, "secrets")
		if err != nil {
			return fmt.Errorf("could not rotate one or more secret files of stack %s: %w", swarmStack.name, err)
		}
	}
	return nil
}

func (swarmStack *swarmStack) rotateObjects(objects map[string]any, objectType string) error {
	objectsDir := path.Dir(path.Join(swarmStack.repo.path, swarmStack.composePath))
	for objectName, object := range objects {
		log := logger.With(
			slog.String("stack", swarmStack.name),
			slog.String("branch", swarmStack.branch),
			slog.String(objectType, objectName),
		)
		objectMap, ok := object.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid compose file: %s object must be a map", objectName)
		}
		isExternal, ok := objectMap["external"].(bool)
		if ok && isExternal {
			continue
		}
		objectFile, ok := objectMap["file"].(string)
		if !ok {
			return fmt.Errorf("invalid compose file: %s file field must be a string", objectName)
		}
		log.Debug("reading...", "file", objectFile)
		objectFilePath := path.Join(objectsDir, objectFile)
		configFileBytes, err := os.ReadFile(objectFilePath)
		if err != nil {
			return fmt.Errorf("could not read file %s for rotation: %w", objectFilePath, err)
		}
		log.Debug("computing hash...", "file", objectFile)
		hash := fmt.Sprintf("%x", md5.Sum(configFileBytes))[:8]
		newObjectName := swarmStack.name + "-" + objectName + "-" + hash
		log.Debug("renaming...", "new_name", newObjectName)
		objectMap["name"] = newObjectName
	}
	return nil
}

func (swarmStack *swarmStack) writeStack(composeMap map[string]any) error {
	composeFileBytes, err := yaml.Marshal(composeMap)
	if err != nil {
		return fmt.Errorf("could not store compose file as yaml after calculating hashes for stack %s", swarmStack.name)
	}
	composeFile := path.Join(swarmStack.repo.path, swarmStack.composePath)
	fileInfo, _ := os.Stat(composeFile)
	os.WriteFile(composeFile, composeFileBytes, fileInfo.Mode())
	return nil
}

func (swarmStack *swarmStack) deployStack() error {
	cmd := stack.NewStackCommand(dockerCli)
	cmd.SetArgs([]string{
		"deploy", "--detach", "--with-registry-auth", "-c",
		path.Join(swarmStack.repo.path, swarmStack.composePath),
		swarmStack.name,
	})
	// To stop printing errors and
	// usage message to stdout
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err != nil {
		return fmt.Errorf("could not deploy stack %s: %s", swarmStack.name, err)
	}
	return nil
}
