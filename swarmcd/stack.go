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

func (swarmStack *swarmStack) updateStack() (stackMetadata *stackMetadata, err error) {
	log := logger.With(
		slog.String("stack", swarmStack.name),
		slog.String("branch", swarmStack.branch),
	)

	log.Debug("pulling changes...")
	revision, err := swarmStack.repo.pullChanges(swarmStack.branch)
	if err != nil {
		return nil, err
	}
	log.Debug("changes pulled", "revision", revision)

	stackDb, err := initStackDB(getDBFilePath(), swarmStack.name)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %s", err)
	}
	defer stackDb.close()

	deployedMetadata, err := stackDb.loadLastDeployedMetadata()
	if err != nil {
		return nil, fmt.Errorf("failed to read revision from stackDb for %s stack: %w", swarmStack.name, err)
	}

	if deployedMetadata.repoRevision == revision {
		logger.Info(fmt.Sprintf("%s revision unchanged: stack up-to-date on rev: %s", swarmStack.name, revision))
		return deployedMetadata, nil
	}
	if deployedMetadata.repoRevision == "" {
		logger.Info(fmt.Sprintf("%s no last revision found", swarmStack.name))
	}

	logger.Info(fmt.Sprintf("%s new revision revision found %s! will update the stack", swarmStack.name, revision))

	log.Debug("reading stack file...")
	stackBytes, err := swarmStack.readStack()
	if err != nil {
		return nil, fmt.Errorf("failed to read stack for %s stack: %w", swarmStack.name, err)
	}

	if swarmStack.valuesFile != "" {
		log.Debug("rendering template...")
		stackBytes, err = swarmStack.renderComposeTemplate(stackBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to render compose template for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("parsing stack content...")
	stackContents, err := swarmStack.parseStackString([]byte(stackBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to parse stack content for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("decrypting secrets...")
	err = swarmStack.decryptSopsFiles(stackContents)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt one or more sops files for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("rotating configs and secrets...")
	err = swarmStack.rotateConfigsAndSecrets(stackContents)
	if err != nil {
		return nil, fmt.Errorf("failed to rotate configs and secrets for %s stack: %w", swarmStack.name, err)
	}

	log.Debug("writing stack to file...")
	writtenBytes, err := swarmStack.writeStack(stackContents)
	if err != nil {
		return nil, fmt.Errorf("failed to write stack to file for %s stack: %w", swarmStack.name, err)
	}

	updatedMetadata := newStackMetadataFromStackData(revision, revision, writtenBytes)

	if swarmStack.shouldDeploy(updatedMetadata, deployedMetadata) {
		log.Debug("deploying stack...")
		err = swarmStack.deployStack()
		if err != nil {
			return nil, fmt.Errorf("failed to deploy stack for  %s stack: %w", swarmStack.name, err)
		}

		log.Debug("saving current stack's metadata to stackDb...")
		err = stackDb.saveLastDeployedMetadata(updatedMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to save new stackMetadata to stackDb for  %s stack: %w", swarmStack.name, err)
		}

		return updatedMetadata, nil
	} else {
		log.Debug("updating deployedMetadata.repoRevision in stackDb...")
		deployedMetadata.repoRevision = revision
		err = stackDb.saveLastDeployedMetadata(deployedMetadata)
		if err != nil {
			return nil, fmt.Errorf("failed to update stackMetadata in stackDb for  %s stack: %w", swarmStack.name, err)
		}

		return deployedMetadata, nil
	}
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

func (swarmStack *swarmStack) writeStack(composeMap map[string]any) ([]byte, error) {
	composeFileBytes, err := yaml.Marshal(composeMap)
	if err != nil {
		return nil, fmt.Errorf("could not store compose file as yaml after calculating hashes for stack %s", swarmStack.name)
	}
	composeFile := path.Join(swarmStack.repo.path, swarmStack.composePath)
	fileInfo, _ := os.Stat(composeFile)
	os.WriteFile(composeFile, composeFileBytes, fileInfo.Mode())
	return composeFileBytes, nil
}

func (swarmStack *swarmStack) shouldDeploy(newMetadata *stackMetadata, deployedMetadata *stackMetadata) bool {
	logger.Debug(fmt.Sprintf("%s Old Stack hash: %s", swarmStack.name, deployedMetadata.fmtHash()))
	logger.Debug(fmt.Sprintf("%s New Stack hash: %s", swarmStack.name, newMetadata.fmtHash()))
	if newMetadata.hash == deployedMetadata.hash {
		logger.Info(fmt.Sprintf("%s stack file hash unchanged, hash=%s. Will skip deployment of revision: %s", swarmStack.name, deployedMetadata.fmtHash(), newMetadata.repoRevision))
		logger.Info(fmt.Sprintf("%s stack remains at revision: %s", swarmStack.name, deployedMetadata.deployedStackRevision))
		return false
	}
	logger.Info(fmt.Sprintf("%s new stack file with hash=%s found. Will continue with deployment of revision: %s", swarmStack.name, newMetadata.fmtHash(), newMetadata.repoRevision))
	return true
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
