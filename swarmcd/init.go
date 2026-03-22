package swarmcd

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"

	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/m-adawi/swarm-cd/util"
)

type StackStatus struct {
	Error          string
	Revision       string
	RepoURL        string
	RefType        string
	RefValue       string
	ComposeFile    string
	LastChangeAt   *time.Time
	LastDeployedAt *time.Time
}

// RuntimeInfo holds instance-level metadata set at startup.
type RuntimeInfo struct {
	BootedAt time.Time
	Version  string
}

// Version is the application version, overridden at build time via
// -ldflags "-X github.com/m-adawi/swarm-cd/swarmcd.Version=..."
var Version = "dev"

var runtimeInfo RuntimeInfo

var config *util.Config = &util.Configs

var logger *slog.Logger = util.Logger

var repos map[string]*stackRepo = map[string]*stackRepo{}

var dockerCli *command.DockerCli

func Init() (err error) {
	runtimeInfo = RuntimeInfo{
		BootedAt: time.Now(),
		Version:  Version,
	}

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
		auth, err := createHTTPBasicAuth(repoName)
		if err != nil {
			return err
		}
		repos[repoName], err = newStackRepo(repoName, repoPath, repoConfig.Url, auth)
		if err != nil {
			return err
		}
	}
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
	for stack, stackConfig := range config.StackConfigs {
		stackRepo, ok := repos[stackConfig.Repo]
		if !ok {
			return fmt.Errorf("error initializing %s stack, no such repo: %s", stack, stackConfig.Repo)
		}

		// Validate branch/tag mutual exclusivity and determine ref
		branch := stackConfig.Branch
		tag := stackConfig.Tag
		var refType, refValue string

		if branch != "" && tag != "" {
			return fmt.Errorf("error initializing %s stack: cannot specify both branch and tag", stack)
		} else if tag != "" {
			refType = "tag"
			refValue = tag
		} else if branch != "" {
			refType = "branch"
			refValue = branch
		} else {
			// Default to main branch for backward compatibility
			branch = "main"
			refType = "branch"
			refValue = "main"
		}

		discoverSecrets := config.SopsSecretsDiscovery || stackConfig.SopsSecretsDiscovery
		swarmStack := newSwarmStack(stack, stackRepo, branch, tag, stackConfig.ComposeFile, stackConfig.SopsFiles, stackConfig.ValuesFile, discoverSecrets)

		stateMu.Lock()
		stacks = append(stacks, swarmStack)
		stackStatus[stack] = &StackStatus{
			RepoURL:     stackRepo.url,
			RefType:     refType,
			RefValue:    refValue,
			ComposeFile: stackConfig.ComposeFile,
		}
		stateMu.Unlock()
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
