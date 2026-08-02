package swarmcd

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"log/slog"
	"os"
	"path"
	"text/template"

	"github.com/docker/cli/cli/command/stack"
	"github.com/goccy/go-yaml"
	"github.com/m-adawi/swarm-cd/util"
)

type swarmStack struct {
	name                 string
	repo                 *stackRepo
	branch               string
	composePaths         []string
	sopsFiles            []string
	valuesFile           string
	discoverSecrets      bool
	alwaysPullContainers *bool
}

func newSwarmStack(name string, repo *stackRepo, branch string, composePaths []string, sopsFiles []string, valuesFile string, discoverSecrets bool, alwaysPullContainers *bool) *swarmStack {
	return &swarmStack{
		name:                 name,
		repo:                 repo,
		branch:               branch,
		composePaths:         composePaths,
		sopsFiles:            sopsFiles,
		valuesFile:           valuesFile,
		discoverSecrets:      discoverSecrets,
		alwaysPullContainers: alwaysPullContainers,
	}
}

func newSwarmStackFromConfig(name string, repo *stackRepo, stackConfig *util.StackConfig, globalSecretsDiscovery bool) *swarmStack {
	return newSwarmStack(
		name,
		repo,
		stackConfig.Branch,
		stackConfig.ComposeFileChain(),
		stackConfig.SopsFiles,
		stackConfig.ValuesFile,
		globalSecretsDiscovery || stackConfig.SopsSecretsDiscovery,
		stackConfig.AlwaysPullContainers,
	)
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

	log.Debug("building templating function...")
	templatingFunction, err := swarmStack.buildTemplatingFunction()
	if err != nil {
		return
	}

	log.Debug("reading compose chain...")
	composeArtifacts, err := readComposeArtifacts(swarmStack.composeFilePaths(), templatingFunction)
	if err != nil {
		return
	}

	log.Debug("decrypting secrets...")
	err = swarmStack.decryptSopsFiles(composeArtifacts)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt one or more sops files for %s stack: %w", swarmStack.name, err)
	}

	if config.AutoRotate {
		log.Debug("rotating configs and secrets...")
		for _, composeArtifact := range composeArtifacts {
			err = swarmStack.rotateConfigsAndSecrets(composeArtifact.composeMap, path.Dir(composeArtifact.sourcePath))
			if err != nil {
				return
			}
		}
	}

	deploymentArtifacts := make([]string, 0, len(composeArtifacts))
	defer func() {
		for _, deploymentArtifact := range deploymentArtifacts {
			_ = os.Remove(deploymentArtifact)
		}
	}()

	log.Debug("writing deployment artifacts...")
	for _, composeArtifact := range composeArtifacts {
		deploymentArtifact, writeErr := swarmStack.writeDeploymentArtifact(composeArtifact.composeMap, composeArtifact.sourcePath)
		if writeErr != nil {
			err = writeErr
			return
		}
		deploymentArtifacts = append(deploymentArtifacts, deploymentArtifact)
	}

	log.Debug("deploying stack...")
	err = swarmStack.deployStack(deploymentArtifacts)
	return
}

func (swarmStack *swarmStack) composeFilePaths() []string {
	composeFiles := make([]string, 0, len(swarmStack.composePaths))
	for _, composePath := range swarmStack.composePaths {
		composeFiles = append(composeFiles, path.Join(swarmStack.repo.path, composePath))
	}
	return composeFiles
}

type templatingFunction func(composeFile string, composeFileBytes []byte) ([]byte, error)

func (swarmStack *swarmStack) buildTemplatingFunction() (templatingFunction, error) {
	if swarmStack.valuesFile != "" {
		valuesMap, err := swarmStack.readValuesFile()
		if err != nil {
			return nil, err
		}
		return func(composeFile string, composeFileBytes []byte) ([]byte, error) {
			return swarmStack.renderComposeTemplate(composeFile, composeFileBytes, valuesMap)
		}, nil
	} else {
		return func(composeFile string, composeFileBytes []byte) ([]byte, error) {
			return composeFileBytes, nil
		}, nil
	}
}

func (swarmStack *swarmStack) readValuesFile() (map[string]any, error) {
	valuesFile := path.Join(swarmStack.repo.path, swarmStack.valuesFile)
	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return nil, fmt.Errorf("could not read %s stack values file: %w", swarmStack.name, err)
	}
	var valuesMap map[string]any
	err = yaml.Unmarshal(valuesBytes, &valuesMap)
	if err != nil {
		return nil, fmt.Errorf("could not parse %s stack values file as yaml: %w", swarmStack.name, err)
	}
	return valuesMap, nil
}

type composeArtifact struct {
	sourcePath string
	composeMap map[string]any
}

func readComposeArtifacts(composeFilePaths []string, templatingFunction templatingFunction) ([]composeArtifact, error) {
	composeArtifacts := make([]composeArtifact, 0, len(composeFilePaths))
	for _, composeFile := range composeFilePaths {
		composeMap, err := ReadStackFile(composeFile, templatingFunction)
		if err != nil {
			return nil, err
		}
		composeArtifacts = append(composeArtifacts, composeArtifact{
			sourcePath: composeFile,
			composeMap: composeMap,
		})
	}
	return composeArtifacts, nil
}

func ReadStackFile(composeFile string, templatingFunction templatingFunction) (map[string]any, error) {
	composeFileBytes, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("could not read compose file %s: %w", composeFile, err)
	}
	composeFileBytes, err = templatingFunction(composeFile, composeFileBytes)
	if err != nil {
		return nil, err
	}
	return parseStackString(composeFileBytes)
}

func (swarmStack *swarmStack) renderComposeTemplate(composeFilePath string, templateContents []byte, valuesMap map[string]any) ([]byte, error) {
	templ, err := template.New(swarmStack.name).Parse(string(templateContents))
	if err != nil {
		return nil, fmt.Errorf("could not parse compose file %s as a Go template for %s stack: %w", composeFilePath, swarmStack.name, err)
	}
	var stackContents bytes.Buffer
	err = templ.Execute(&stackContents, map[string]map[string]any{"Values": valuesMap})
	if err != nil {
		return nil, fmt.Errorf("error rendering compose template %s for %s stack: %w", composeFilePath, swarmStack.name, err)
	}
	return stackContents.Bytes(), nil
}

func parseStackString(stackContent []byte) (map[string]any, error) {
	var composeMap map[string]any
	err := yaml.Unmarshal(stackContent, &composeMap)
	if err != nil {
		return nil, fmt.Errorf("could not parse stack yaml: %w", err)
	}
	return composeMap, nil
}

func (swarmStack *swarmStack) decryptSopsFiles(composeArtifacts []composeArtifact) (err error) {
	sopsFiles, err := swarmStack.resolveSopsFilesForDecryption(composeArtifacts)
	if err != nil {
		return
	}
	log := logger.With(
		slog.String("stack", swarmStack.name),
		slog.String("branch", swarmStack.branch),
	)
	for _, sopsFile := range sopsFiles {
		log.Debug("decrypting secret...", "secret", sopsFile)
		err = util.DecryptFile(sopsFile)
		if err != nil {
			return
		}
	}
	return
}

func (swarmStack *swarmStack) resolveSopsFilesForDecryption(composeArtifacts []composeArtifact) ([]string, error) {
	var sopsFiles []string
	if !swarmStack.discoverSecrets {
		sopsFiles = swarmStack.sopsFiles
	} else {
		var err error
		sopsFiles, err = discoverSecretsFromComposeFiles(composeArtifacts)
		if err != nil {
			return nil, err
		}
	}

	resolvedPaths := make([]string, 0, len(sopsFiles))
	seen := make(map[string]struct{}, len(sopsFiles))
	for _, sopsFile := range sopsFiles {
		secretFilePath := sopsFile
		if !path.IsAbs(secretFilePath) {
			secretFilePath = path.Join(swarmStack.repo.path, secretFilePath)
		}
		if _, ok := seen[secretFilePath]; ok {
			continue
		}
		seen[secretFilePath] = struct{}{}
		resolvedPaths = append(resolvedPaths, secretFilePath)
	}
	return resolvedPaths, nil
}

func discoverSecretsFromComposeFiles(composeArtifacts []composeArtifact) ([]string, error) {
	sopsFiles := make([]string, 0)
	composeDir := path.Dir(composeArtifacts[0].sourcePath)
	for _, composeArtifact := range composeArtifacts {
		composeSopsFiles, err := discoverSecrets(composeArtifact.composeMap, composeDir)
		if err != nil {
			return nil, err
		}
		sopsFiles = append(sopsFiles, composeSopsFiles...)
	}
	return sopsFiles, nil
}

// filters objects to only those with a "file" key, validating
// types along the way. The returned maps are references to the originals, so
// callers may mutate them (e.g. to set a "name" field).
func getFileObjects(objects map[string]any) (map[string]map[string]any, error) {
	result := make(map[string]map[string]any)
	for objectName, object := range objects {
		objectMap, ok := object.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid compose file: %s object must be a map", objectName)
		}
		// only process objects with a file: key
		objectFileObj, ok := objectMap["file"]
		if !ok {
			continue
		}
		if _, ok = objectFileObj.(string); !ok {
			return nil, fmt.Errorf("invalid compose file: %s file field must be a string", objectName)
		}
		result[objectName] = objectMap
	}
	return result, nil
}

func discoverSecrets(composeMap map[string]any, composePath string) ([]string, error) {
	var sopsFiles []string
	if secrets, ok := composeMap["secrets"].(map[string]any); ok {
		fileObjects, err := getFileObjects(secrets)
		if err != nil {
			return nil, err
		}
		for _, secretMap := range fileObjects {
			secretFile := secretMap["file"].(string)
			secretPath := secretFile
			if !path.IsAbs(secretPath) {
				secretPath = path.Join(composePath, secretPath)
			}
			sopsFiles = append(sopsFiles, secretPath)
		}
	}
	return sopsFiles, nil
}

func (swarmStack *swarmStack) rotateConfigsAndSecrets(composeMap map[string]any, composeDir string) error {
	if configs, ok := composeMap["configs"].(map[string]any); ok {
		err := swarmStack.rotateObjects(configs, "configs", composeDir)
		if err != nil {
			return fmt.Errorf("could not rotate one or more config files of stack %s: %w", swarmStack.name, err)
		}
	}
	if secrets, ok := composeMap["secrets"].(map[string]any); ok {
		err := swarmStack.rotateObjects(secrets, "secrets", composeDir)
		if err != nil {
			return fmt.Errorf("could not rotate one or more secret files of stack %s: %w", swarmStack.name, err)
		}
	}
	return nil
}

func (swarmStack *swarmStack) rotateObjects(objects map[string]any, objectType string, objectsDir string) error {
	fileObjects, err := getFileObjects(objects)
	if err != nil {
		return err
	}
	for objectName, objectMap := range fileObjects {
		log := logger.With(
			slog.String("stack", swarmStack.name),
			slog.String("branch", swarmStack.branch),
			slog.String(objectType, objectName),
		)
		objectFile := objectMap["file"].(string)
		log.Debug("reading...", "file", objectFile)
		objectFilePath := objectFile
		if !path.IsAbs(objectFilePath) {
			objectFilePath = path.Join(objectsDir, objectFilePath)
		}
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

func (swarmStack *swarmStack) writeDeploymentArtifact(composeMap map[string]any, composeSourcePath string) (string, error) {
	composeFileBytes, err := yaml.Marshal(composeMap)
	if err != nil {
		return "", fmt.Errorf("could not store compose file as yaml after calculating hashes for stack %s", swarmStack.name)
	}

	composeDir := path.Dir(composeSourcePath)
	deploymentArtifact, err := os.CreateTemp(composeDir, fmt.Sprintf(".swarmcd-deploy-%s-*.yaml", swarmStack.name))
	if err != nil {
		return "", fmt.Errorf("could not create deployment artifact for stack %s: %w", swarmStack.name, err)
	}

	if _, err = deploymentArtifact.Write(composeFileBytes); err != nil {
		deploymentArtifact.Close()
		os.Remove(deploymentArtifact.Name())
		return "", fmt.Errorf("could not write deployment artifact for stack %s: %w", swarmStack.name, err)
	}

	if err = deploymentArtifact.Close(); err != nil {
		os.Remove(deploymentArtifact.Name())
		return "", fmt.Errorf("could not close deployment artifact for stack %s: %w", swarmStack.name, err)
	}

	return deploymentArtifact.Name(), nil
}

func (swarmStack *swarmStack) deployStack(composeFiles []string) error {
	cmd := stack.NewStackCommand(dockerCli)
	cmd.SetArgs(swarmStack.deployStackArgs(composeFiles))
	// To stop printing errors and
	// usage message to stdout
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	err := cmd.Execute()
	if err != nil {
		return fmt.Errorf("could not deploy stack %s: %w", swarmStack.name, err)
	}
	return nil
}

func (swarmStack *swarmStack) deployStackArgs(composeFiles []string) []string {
	args := []string{
		"deploy", "--detach", "--with-registry-auth",
		"--resolve-image", swarmStack.resolveImageMode(),
	}
	for _, composeFile := range composeFiles {
		args = append(args, "-c", composeFile)
	}
	return append(args, swarmStack.name)
}

// Returns "always" when alwaysPullContainers is true, "changed" otherwise.
// The stack-level setting overrides the global config setting.
func (swarmStack *swarmStack) resolveImageMode() string {
	alwaysPull := config.AlwaysPullContainers
	if swarmStack.alwaysPullContainers != nil {
		alwaysPull = *swarmStack.alwaysPullContainers
	}
	if alwaysPull {
		return "always"
	}
	return "changed"
}
