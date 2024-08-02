package swarmcd

import (
	"crypto/md5"
	"fmt"
	"os"
	"path"
	"text/template"

	"github.com/docker/cli/cli/command/stack"
	"github.com/go-git/go-git/v5"
	"github.com/goccy/go-yaml"
	"github.com/m-adawi/swarm-cd/util"
)

type swarmStack struct {
	name        string
	repo        *stackRepo
	branch      string
	composePath string
	sopsFiles   []string
	valuesFile  string
}

func newSwarmStack(name string, repo *stackRepo, branch string, composePath string, sopsFiles []string, valuesFile string) *swarmStack {
	return &swarmStack{
		name:        name,
		repo:        repo,
		branch:      branch,
		composePath: composePath,
		sopsFiles:   sopsFiles,
		valuesFile: valuesFile,
	}
}


func (swarmStack *swarmStack) updateStack() (revision string, err error) {
	revision, err = swarmStack.repo.pullChanges(swarmStack.branch)
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return
	}

	err = swarmStack.decryptSopsFiles()
	if err != nil {
		return "", fmt.Errorf("failed to decrypt one or more sops files for %s stack: %w", swarmStack.name, err)
	}

	if swarmStack.valuesFile != "" {
		err = swarmStack.renderComposeTemplate() 
		if err != nil {
			return
		}
	}

	err = swarmStack.rotateConfigsAndSecrets() 
	if err != nil {
		return
	}

	err = swarmStack.deployStack() 
	if err != nil {
		return
	}
	return 
}

func (swarmStack *swarmStack) decryptSopsFiles() (err error) {
	for _, sopsFile := range swarmStack.sopsFiles {
		err = util.DecryptFile(path.Join(swarmStack.repo.path, sopsFile))
		if err != nil {
			return
		}
	}
	return
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

func (swarmStack *swarmStack) rotateConfigsAndSecrets() error {
	composeMap, err := swarmStack.readComposeFile()
	if err != nil {
		return err
	}

	_, ok := composeMap["configs"]
	// if configs are defined rotate them
	if ok {
		configsMap, ok := composeMap["configs"].(map[string]any)
		if !ok {
			return fmt.Errorf("could not read %s stack configs: should be a map", swarmStack.name, err)
		}
		err = swarmStack.rotateObjects(configsMap)
		if err != nil{
			return fmt.Errorf("could not rotate one or more configs of stack %s: %w", swarmStack.name, err)
		}
	}

	_, ok = composeMap["secrets"]
	// if secrets are defined rotate them
	if ok {
		secretsMap, ok := composeMap["secrets"].(map[string]any)
		if !ok {
			return fmt.Errorf("could not read %s stack secrets: should be a map", swarmStack.name, err)
		}
		err = swarmStack.rotateObjects(secretsMap)
		if err != nil{
			return fmt.Errorf("could not rotate one or more secrets of stack %s: %w", swarmStack.name, err)
		}
	}

	composeFileBytes, err := yaml.Marshal(composeMap)
	if err != nil {
		return fmt.Errorf("could not store comopse file as yaml after calculating hashes for stack %s", swarmStack.name)
	}
	composeFile := path.Join(swarmStack.repo.path, swarmStack.composePath)
	fileInfo, _ := os.Stat(composeFile)
	os.WriteFile(composeFile, composeFileBytes, fileInfo.Mode())
	return nil
}

func (swarmStack *swarmStack) readComposeFile() (map[string]any, error) {
	composeFile := path.Join(swarmStack.repo.path, swarmStack.composePath)
	composeFileBytes, err := os.ReadFile(composeFile)
	if err != nil {
		return nil, fmt.Errorf("could not read compose file %s: %w", composeFile, err)
	}
	var composeMap map[string]any
	err = yaml.Unmarshal(composeFileBytes, &composeMap)
	if err != nil {
		return nil, fmt.Errorf("could not parse yaml file %s: %w", composeFile, err)
	}
	return composeMap, nil
	// value, ok := composeMap[key]
	// if !ok {
	// 	return nil, fmt.Errorf("key %s does not exist in %s stack compose file", key, swarmStack.name)
	// }
	// return value, nil
}

func (swarmStack *swarmStack) rotateObjects(objects map[string]any) error {
	objectsDir := path.Dir(path.Join(swarmStack.repo.path, swarmStack.composePath))
	for objectName, object := range objects {
		objectMap, ok := object.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid compose file: %s object must be a map", objectName)
		}
		objectFile, ok := objectMap["file"].(string)
		if !ok {
			return fmt.Errorf("invalid compose file: %s file field must be a string", objectName)
		}
		objectFilePath := path.Join(objectsDir, objectFile)
		configFileBytes, err := os.ReadFile(objectFilePath)
		if err != nil {
			return fmt.Errorf("could not read file %s for rotation: %w", objectFilePath, err)
		}
		hash := fmt.Sprintf("%x", md5.Sum(configFileBytes))[:8]
		objectMap["name"] = swarmStack.name + "-" + objectName + "-" + hash
	}
	return nil
}


func (swarmStack *swarmStack) rednerAllTemplates() error {
	composeFile := path.Join(config.ReposPath, swarmStack.repo.path, swarmStack.composePath)
	err := swarmStack.renderTemplate(composeFile)
	if err != nil {
		return err
	}

}

func (swarmStack *swarmStack) renderTemplate(filepath string) error {
	valuesFile := path.Join(config.ReposPath, swarmStack.repo.path, swarmStack.valuesFile)
	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return fmt.Errorf("could not read %s stack values file: %w", swarmStack.name, err)
	}
	var valuesMap map[string]any 
	yaml.Unmarshal(valuesBytes, &valuesMap) 
	templ, err := template.New(path.Base(filepath)).ParseFiles(filepath)
	if err != nil {
		return fmt.Errorf("could not parse %s stack file %s as a Go template: %w", swarmStack.name, filepath, err)
	}
	composeFileWriter, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("could not open %s stack file %s: %w", swarmStack.name, filepath, err)
	}
	err = templ.Execute(composeFileWriter, map[string]map[string]any{"Values": valuesMap})
	if err != nil {
		return fmt.Errorf("error rending %s stack %s template: %w", swarmStack.name, filepath, err)
	}
	return nil
}

func (swarmStack *swarmStack) renderComposeTemplate() error {
	filepath := path.Join(config.ReposPath, swarmStack.repo.path, swarmStack.composePath)
	valuesFile := path.Join(config.ReposPath, swarmStack.repo.path, swarmStack.valuesFile)
	valuesBytes, err := os.ReadFile(valuesFile)
	if err != nil {
		return fmt.Errorf("could not read %s stack values file: %w", swarmStack.name, err)
	}
	var valuesMap map[string]any 
	yaml.Unmarshal(valuesBytes, &valuesMap) 
	templ, err := template.New(path.Base(filepath)).ParseFiles(filepath)
	if err != nil {
		return fmt.Errorf("could not parse %s stack compose file as a Go template: %w", swarmStack.name, err)
	}
	composeFileWriter, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("could not open %s stack compose file: %w", swarmStack.name, err)
	}
	err = templ.Execute(composeFileWriter, map[string]map[string]any{"Values": valuesMap})
	if err != nil {
		return fmt.Errorf("error rending %s stack compose template: %w", swarmStack.name, err)
	}
	return nil
}
