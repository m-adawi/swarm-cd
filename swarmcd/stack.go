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
	name            string
	repo            *stackRepo
	branch          string
	composePath     string
	sopsFiles       []string
	valuesFile      string
	discoverSecrets bool
}

func newSwarmStack(name string, repo *stackRepo, branch string, composePath string, sopsFiles []string, valuesFile string, discoverSecrets bool) *swarmStack {
	return &swarmStack{
		name:            name,
		repo:            repo,
		branch:          branch,
		composePath:     composePath,
		sopsFiles:       sopsFiles,
		valuesFile:      valuesFile,
		discoverSecrets: discoverSecrets,
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

	db, err := initDB(getDBFilePath())
	if err != nil {
		logger.Error(fmt.Sprintf("failed to open database: %s", err))
		return
	}
	defer db.Close()

	lastRevision, deployedStackHash, err := loadLastDeployedRevision(db, swarmStack.name)
	if err != nil {
		return "", fmt.Errorf("failed to read revision from db for %s stack: %w", swarmStack.name, err)
	}

	if lastRevision == "" {
		logger.Info(fmt.Sprintf("%s no last revision found", swarmStack.name))
	}

	if lastRevision == revision {
		logger.Info(fmt.Sprintf("%s revision unchanged: stack up-to-date on rev: %s", swarmStack.name, revision))
		return revision, nil
	}

	logger.Info(fmt.Sprintf("%s new revision revision found %s! will update the stack", swarmStack.name, revision))

	log.Debug("reading stack file...")
	stackBytes, err := swarmStack.readStack()
	if err != nil {
		return "", fmt.Errorf("failed to read stack for %s stack: %w", swarmStack.name, err)
	}

	if swarmStack.valuesFile != "" {
		log.Debug("rendering template...")
		stackBytes, err = swarmStack.renderComposeTemplate(stackBytes)
	}
	if err != nil {
		return "", fmt.Errorf("failed to render compose template for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("parsing stack content...")
	stackContents, err := swarmStack.parseStackString([]byte(stackBytes))
	if err != nil {
		return "", fmt.Errorf("failed to parse stack content for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("decrypting secrets...")
	err = swarmStack.decryptSopsFiles(stackContents)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt one or more sops files for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("rotating configs and secrets...")
	// This contains all the new data of the config and secrets.
	// We use this to determine if the stack has changed, and we
	// need to redeploy.
	dataBytes, err := swarmStack.rotateConfigsAndSecrets(stackContents)
	if err != nil {
		return "", fmt.Errorf("failed to rotate configs and secrets for %s stack: %w", swarmStack.name, err)
	}

	dataToHash := append(stackBytes, dataBytes...)
	newStackHash := computeHash(dataToHash)
	logger.Debug(fmt.Sprintf("%s Old Stack hash: %s", swarmStack.name, fmtHash(deployedStackHash)))
	logger.Debug(fmt.Sprintf("%s New Stack hash: %s", swarmStack.name, fmtHash(newStackHash)))
	if newStackHash == deployedStackHash {
		logger.Info(fmt.Sprintf("%s stack file hash unchanged, hash=%s. Will skip deployment of revision: %s", swarmStack.name, fmtHash(deployedStackHash), revision))
		logger.Info(fmt.Sprintf("%s stack remains at revision: %s", swarmStack.name, lastRevision))
		return revision, nil
	} else {
		logger.Info(fmt.Sprintf("%s new stack file with hash=%s found. Will continue with deployment of revision: %s", swarmStack.name, fmtHash(newStackHash), revision))
	}

	log.Debug("writing stack to file...")
	err = swarmStack.writeStack(stackContents)
	if err != nil {
		return "", fmt.Errorf("failed to write stack to file for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("deploying stack...")
	err = swarmStack.deployStack()
	if err != nil {
		return revision, fmt.Errorf("failed to deploy stack for  %s stack: %w", swarmStack.name, err)
	}

	log.Debug("saving current revision to db...")
	err = saveLastDeployedRevision(db, swarmStack.name, revision, dataToHash)
	if err != nil {
		return revision, fmt.Errorf("failed to save revision to db for  %s stack: %w", swarmStack.name, err)
	}

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

func (swarmStack *swarmStack) renderComposeTemplate(templateContents []byte) ([]byte, error) {
	valuesFile := path.Join(config.ReposPath, swarmStack.repo.path, swarmStack.valuesFile)
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
	for _, sopsFile := range sopsFiles {
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
			secretFile, ok := secretMap["file"].(string)
			if !ok {
				return nil, fmt.Errorf("invalid compose file: %s file field must be a string", secretName)
			}
			objectDir := path.Join(path.Dir(composePath), secretFile)
			sopsFiles = append(sopsFiles, objectDir)
		}
	}
	return sopsFiles, nil
}

func (swarmStack *swarmStack) rotateConfigsAndSecrets(composeMap map[string]any) ([]byte, error) {
	var dataBytes []byte
	if configs, ok := composeMap["configs"].(map[string]any); ok {
		configBytes, err := swarmStack.rotateObjects(configs)
		dataBytes = append(dataBytes, configBytes...)

		if err != nil {
			return nil, fmt.Errorf("could not rotate one or more config files of stack %s: %w", swarmStack.name, err)
		}
	}
	if secrets, ok := composeMap["secrets"].(map[string]any); ok {
		secretsByte, err := swarmStack.rotateObjects(secrets)
		dataBytes = append(dataBytes, secretsByte...)
		if err != nil {
			return nil, fmt.Errorf("could not rotate one or more secret files of stack %s: %w", swarmStack.name, err)
		}
	}
	return dataBytes, nil
}

func (swarmStack *swarmStack) rotateObjects(objects map[string]any) ([]byte, error) {
	objectsDir := path.Dir(path.Join(swarmStack.repo.path, swarmStack.composePath))
	var configBytes []byte
	for objectName, object := range objects {
		objectMap, ok := object.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid compose file: %s object must be a map", objectName)
		}
		// If "external" field exists and is true, skip processing
		if external, exists := objectMap["external"].(bool); exists && external {
			continue
		}
		objectFile, ok := objectMap["file"].(string)
		if !ok {
			return nil, fmt.Errorf("invalid compose file: %s file field must be a string", objectName)
		}
		objectFilePath := path.Join(objectsDir, objectFile)
		configFileBytes, err := os.ReadFile(objectFilePath)
		configBytes = append(configBytes, configFileBytes...)
		if err != nil {
			return nil, fmt.Errorf("could not read file %s for rotation: %w", objectFilePath, err)
		}
		hash := fmt.Sprintf("%x", md5.Sum(configFileBytes))[:8]
		objectMap["name"] = swarmStack.name + "-" + objectName + "-" + hash
	}
	return configBytes, nil
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

func fmtHash(hash string) string {
	var shortHash string
	if len(hash) >= 8 {
		return hash[:8]
	}
	return "<empty-hash>"
}
