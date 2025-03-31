package swarmcd

import (
	"fmt"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/m-adawi/swarm-cd/util"
	"log/slog"
	"os"
	"path"
	"strings"
)

type StackStatus struct {
	Error                 string
	Revision              string
	DeployedStackRevision string
	DeployedAt            string
	RepoURL               string
}

var config *util.Config = &util.Configs

var logger *slog.Logger = util.Logger

var repos map[string]*stackRepo = map[string]*stackRepo{}

var dockerCli *command.DockerCli

func Init() (err error) {
	err = initRepos()
	if err != nil {
		return err
	}
	err = initStacks()
	if err != nil {
		return err
	}
	err = initDockerCli()
	if err != nil {
		return err
	}
	return
}

func initRepos() error {
	var newRepos = map[string]*stackRepo{}

	for repoName, repoConfig := range config.RepoConfigs {
		if repo, exists := repos[repoName]; exists {
			newRepos[repoName] = repo
			delete(repos, repoName)
			continue
		}

		repoPath := path.Join(config.ReposPath, repoName)
		auth, err := createHTTPBasicAuth(repoName)
		if err != nil {
			return err
		}
		newRepos[repoName], err = newStackRepo(repoName, repoPath, repoConfig.Url, auth)
		if err != nil {
			return err
		}
	}

	if len(repos) != 0 {
		logger.Info(fmt.Sprintf("Some repos were removed from the stack: %v", repos))
	}

	repos = newRepos

	return nil
}

func createHTTPBasicAuth(repoName string) (*http.BasicAuth, error) {
	repoConfig := config.RepoConfigs[repoName]
	// assume repo is public and no auth is required
	if repoConfig.Username == "" && repoConfig.Password == "" && repoConfig.PasswordFile == "" {
		return nil, nil
	}

	if repoConfig.Username == "" {
		return nil, fmt.Errorf("you must set username for the repo %s", repoName)
	}

	if repoConfig.Password == "" && repoConfig.PasswordFile == "" {
		return nil, fmt.Errorf("you must set one of password or password_file properties for the repo %s", repoName)
	}

	var password string
	if repoConfig.Password != "" {
		password = repoConfig.Password
	} else {
		passwordBytes, err := os.ReadFile(repoConfig.PasswordFile)
		if err != nil {
			return nil, fmt.Errorf("could not read password file %s for repo %s", repoConfig.PasswordFile, repoName)
		}
		// trim newline and whitespaces
		password = strings.TrimSpace(string(passwordBytes))
	}

	return &http.BasicAuth{
		Username: repoConfig.Username,
		Password: password,
	}, nil
}

func initStacks() error {
	var newStacks = map[string]*swarmStack{}
	var newStackStatus = map[string]*StackStatus{}

	for stack, stackConfig := range config.StackConfigs {
		logger.Info(fmt.Sprintf("reading stackConfig for stack: %v", stack))

		stackRepo, ok := repos[stackConfig.Repo]
		if !ok {
			return fmt.Errorf("error reading %s stack, no such repo: %s", stack, stackConfig.Repo)
		}

		discoverSecrets := config.SopsSecretsDiscovery || stackConfig.SopsSecretsDiscovery
		swarmStack := newSwarmStack(stack, stackRepo, stackConfig.Branch, stackConfig.ComposeFile, stackConfig.SopsFiles, stackConfig.ValuesFile, discoverSecrets)

		newStacks[stack] = swarmStack
		if _, exists := stacks[stack]; exists {
			delete(stacks, stack)
		}

		newStackStatus[stack] = &StackStatus{}
		newStackStatus[stack].RepoURL = stackRepo.url
		if _, exists := stackStatus[stack]; exists {
			delete(stacks, stack)
		}
	}

	if len(stacks) != 0 {
		logger.Info(fmt.Sprintf("Some stacks were removed: %v", stacks))
		// Todo: do we need to do something for this.
	}

	stacks = newStacks
	stackStatus = newStackStatus

	return nil
}

func initDockerCli() (err error) {
	// suppress command outputs (errors are returned as objects)
	nullFile, _ := os.Open("/dev/null")
	defer nullFile.Close()
	dockerCli, err = command.NewDockerCli(command.WithOutputStream(nullFile), command.WithErrorStream(nullFile))
	if err != nil {
		return fmt.Errorf("could not create a docker cli object: %w", err)
	}
	err = dockerCli.Initialize(flags.NewClientOptions())

	if err != nil {
		return fmt.Errorf("could not initialize docker cli object: %w", err)
	}
	return nil
}
