package swarmcd

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/Masterminds/sprig/v3"
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
	globalValuesMap map[string]any
	templateFolder  string
	templated       bool
}

func NewSwarmStack(name string, repo *stackRepo, branch string, composePath string, sopsFiles []string, valuesFile string, discoverSecrets bool, globalValuesMap map[string]any, templateFolder string) *swarmStack {
	log := logger.With(
		slog.String("stack", name),
		slog.String("branch", branch),
	)
	if repo != nil {
		templateFolder = path.Join(repo.path, templateFolder)
	}

	_, err := os.Stat(templateFolder)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			log.Error("Cannot access template folder due to permission", "err", err, "folder", templateFolder)
		}
		templateFolder = ""
	}

	return &swarmStack{
		name:            name,
		repo:            repo,
		branch:          branch,
		composePath:     composePath,
		sopsFiles:       sopsFiles,
		valuesFile:      valuesFile,
		discoverSecrets: discoverSecrets,
		globalValuesMap: globalValuesMap,
		templateFolder:  templateFolder,
		templated:       false,
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

	stackBytes, err := swarmStack.GenerateStack()
	if err != nil {
		return
	}

	status, exists := stackStatus[swarmStack.name]
	if !exists {
		return "", fmt.Errorf("Stack %s exists, but it has no status.", swarmStack.name)
	}
	status.Templated = swarmStack.templated

	log.Debug("parsing stack content...")
	stackContents, err := swarmStack.parseStackString([]byte(stackBytes))
	if err != nil {
		return
	}

	log.Debug("decrypting secrets...")
	err = swarmStack.decryptSopsFiles(stackContents)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt one or more sops files for %s stack: %w", swarmStack.name, err)
	}

	if config.AutoRotate {
		log.Debug("rotating configs and secrets...")
		err = swarmStack.rotateConfigsAndSecrets(stackContents)
		if err != nil {
			return
		}
	}

	log.Debug("writing stack to file...")
	err = swarmStack.writeStack(stackContents)
	if err != nil {
		return
	}

	log.Debug("deploying stack...")
	err = swarmStack.deployStack()
	return
}

func (swarmStack *swarmStack) GenerateStack() (stackBytes []byte, err error) {
	log := logger.With(
		slog.String("stack", swarmStack.name),
		slog.String("branch", swarmStack.branch),
	)
	log.Debug("reading stack file...")
	swarmStack.templated = false
	stackBytes, err = swarmStack.ReadStack()
	if err != nil {
		return
	}

	mergedValuesMap := make(map[string]any)
	maps.Copy(mergedValuesMap, swarmStack.globalValuesMap)

	if swarmStack.valuesFile != "" {
		valuesFile := swarmStack.valuesFile
		if swarmStack.repo != nil {
			valuesFile = path.Join(swarmStack.repo.path, swarmStack.valuesFile)
		}
		var valuesMap map[string]any
		valuesMap, err = ParseValuesFile(valuesFile, swarmStack.name+" stack")
		if err != nil {
			return
		}
		maps.Copy(mergedValuesMap, valuesMap)
	}

	if len(mergedValuesMap) == 0 && swarmStack.templateFolder == "" {
		// No need to continue, this file isn't templated
		return
	}

	log.Debug("rendering template...")
	templ, err := template.New(swarmStack.name).Funcs(sprig.FuncMap()).Parse(string(stackBytes[:]))
	if err != nil {
		return nil, fmt.Errorf("could not parse %s stack compose file as a Go template: %w", swarmStack.name, err)
	}

	if swarmStack.templateFolder != "" {
		log.Debug("Loading template folder...")

		pattern := path.Join(swarmStack.templateFolder, "*.tmpl")
		filenames, err := filepath.Glob(pattern)
		if err == nil {
			if filenames != nil {
				_, err = templ.ParseFiles(filenames...)
			} else {
				log.Debug("Skipping, folder empty", "folder", swarmStack.templateFolder)
			}
		}
		if err != nil {
			log.Warn("Could not parse templates, trying to generate stack without them.", "error", err)
		}
	}

	var stackContents bytes.Buffer
	err = templ.Execute(&stackContents, map[string]map[string]any{"Values": mergedValuesMap})
	if err != nil {
		return nil, fmt.Errorf("error rending %s stack compose template: %w", swarmStack.name, err)
	}
	// If there hasn't been any variable replacement, then it's not templated.
	swarmStack.templated = !bytes.Equal(stackContents.Bytes(), stackBytes)
	return stackContents.Bytes(), nil
}

func (swarmStack *swarmStack) ReadStack() ([]byte, error) {
	composeFile := swarmStack.composePath
	if swarmStack.repo != nil {
		composeFile = path.Join(swarmStack.repo.path, swarmStack.composePath)
	}
	composeFileBytes, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("could not read compose file %s: %w", composeFile, err)
	}
	return composeFileBytes, nil
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

func ParseValuesFile(valuesFile string, source string) (map[string]any, error) {
	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return nil, fmt.Errorf("could not read %s values file: %w", source, err)
	}
	var valuesMap map[string]any
	err = yaml.Unmarshal(valuesBytes, &valuesMap)
	if err != nil {
		return nil, fmt.Errorf("could not parse yaml from values file: %w", err)
	}
	return valuesMap, nil
}
