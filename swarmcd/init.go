package swarmcd

import (
	"fmt"
	"log/slog"
	"os"
	"path"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/m-adawi/swarm-cd/util"
)

type StackStatus struct {
	Error    string
	Revision string
	RepoURL  string
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
	for repoName, repoConfig := range config.RepoConfigs {
		repoPath := path.Join(config.ReposPath, repoName)
		auth, err := createRepoAuth(repoName)
		if err != nil {
			return fmt.Errorf("create auth for repo %q: %w", repoName, err)
		}
		repos[repoName], err = newStackRepo(repoName, repoPath, repoConfig.Url, repoConfig.DefaultReference, auth)
		if err != nil {
			return fmt.Errorf("create stack repo %q: %w", repoName, err)
		}
	}
	return nil
}

func initStacks() error {
	for stack, stackConfig := range config.StackConfigs {
		stackRepo, ok := repos[stackConfig.Repo]
		if !ok {
			return fmt.Errorf("error initializing %s stack, no such repo: %s", stack, stackConfig.Repo)
		}
		swarmStack := newSwarmStackFromConfig(stack, stackRepo, stackConfig, config.SopsSecretsDiscovery)
		stacks = append(stacks, swarmStack)
		stackStatus[stack] = &StackStatus{}
		stackStatus[stack].RepoURL = stackRepo.url
	}
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
