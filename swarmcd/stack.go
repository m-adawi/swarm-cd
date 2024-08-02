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

type composeObject struct {
	configs map[string]*struct {
		file string
		name string
	}
	secrets map[string]*struct {
		file string
		name string	
	}
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
		err = swarmStack.rednerAllTemplates() 
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
		swarmStack.composePath,
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
	composeObject, composeMap, err := swarmStack.readComposeFile()
	if err != nil {
		return err
	}

	if composeObject.configs != nil {
		err = swarmStack.rotateObjects(composeObject.configs)
		if err != nil{
			return fmt.Errorf("could not rotate one or more configs of stack %s: %w", swarmStack.name, err)
		}
		composeMap["configs"] = composeObject.configs
	}

	if composeObject.secrets != nil {
		err = swarmStack.rotateObjects(composeObject.secrets)
		if err != nil{
			return fmt.Errorf("could not rotate one or more secrets of stack %s: %w", swarmStack.name, err)
		}
		composeMap["secrets"] = composeObject.secrets
	}

	composeFileBytes, err := yaml.Marshal(composeMap)
	if err != nil {
		return fmt.Errorf("could not store comopse file as yaml after rotating objects for stack %s", swarmStack.name)
	}
	fileInfo, _ := os.Stat(swarmStack.composePath)
	os.WriteFile(swarmStack.composePath, composeFileBytes, fileInfo.Mode())
	return nil
}

func (swarmStack *swarmStack) readComposeFile() (composeObject *composeObject, composeMap map[string]any, err error) {
	composeFileBytes, err := os.ReadFile(swarmStack.composePath)
	if err != nil {
		err = fmt.Errorf("could not read compose file %s: %w", swarmStack.composePath, err)
		return
	}
	err = yaml.Unmarshal(composeFileBytes, &composeMap)
	if err != nil {
		err = fmt.Errorf("could not parse yaml file %s: %w", swarmStack.composePath, err)
		return
	}
	err = yaml.Unmarshal(composeFileBytes, &composeObject)
	if err != nil {
		err = fmt.Errorf("could unmarshal yaml file %s into compose object: %w", swarmStack.composePath, err)
		return
	}
	return 
}

func (swarmStack *swarmStack) rotateObjects(objects map[string]*struct{file string; name string}) error {
	objectsDir := path.Dir(swarmStack.composePath)
	for objectName, object := range objects {
		objectFilePath := path.Join(objectsDir, object.file)
		configFileBytes, err := os.ReadFile(objectFilePath)
		if err != nil {
			return fmt.Errorf("could not read file %s for rotation: %w", objectFilePath, err)
		}
		hash := fmt.Sprintf("%x", md5.Sum(configFileBytes))[:8]
		object.name = swarmStack.name + "-" + objectName + "-" + hash
	}
	return nil
}


func (swarmStack *swarmStack) rednerAllTemplates() error {
	err := swarmStack.renderTemplate(swarmStack.composePath)
	if err != nil {
		return err
	}
	composeObject, _, err := swarmStack.readComposeFile()
	if err != nil {
		return err
	}
	
	if composeObject.configs != nil {
		for _, config := range composeObject.configs {
			configPath := path.Join(path.Dir(swarmStack.composePath), config.file)
			err = swarmStack.renderTemplate(configPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (swarmStack *swarmStack) renderTemplate(filepath string) error {
	valuesBytes, err := os.ReadFile(swarmStack.valuesFile)
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
