package swarmcd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/m-adawi/swarm-cd/util"
)

func createRepoAuth(repoName string) (transport.AuthMethod, error) {
	repoConfig := config.RepoConfigs[repoName]

	if strings.HasPrefix(repoConfig.Url, "ssh://") {
		return createSSHAuth(*repoConfig)
	}

	return createHTTPBasicAuth(*repoConfig)
}

func createSSHAuth(repoConfig util.RepoConfig) (transport.AuthMethod, error) {
	username := repoConfig.Username
	if username == "" {
		username = "git"
	}

	password, err := getPassword(repoConfig)
	if err != nil {
		return nil, fmt.Errorf("get password: %w", err)
	}

	return ssh.NewPublicKeysFromFile(repoConfig.Username, repoConfig.CertificateFile, password)
}

func createHTTPBasicAuth(repoConfig util.RepoConfig) (*http.BasicAuth, error) {
	// assume repo is public and no auth is required
	if repoConfig.Username == "" && repoConfig.Password == "" && repoConfig.PasswordFile == "" {
		return nil, nil
	}

	if repoConfig.Username == "" {
		return nil, errors.New("you must set username for the repo")
	}

	if repoConfig.Password == "" && repoConfig.PasswordFile == "" {
		return nil, errors.New("you must set one of password or password_file properties for the repo")
	}

	password, err := getPassword(repoConfig)
	if err != nil {
		return nil, fmt.Errorf("get password: %w", err)
	}

	return &http.BasicAuth{
		Username: repoConfig.Username,
		Password: password,
	}, nil
}

func getPassword(repoConfig util.RepoConfig) (string, error) {
	if repoConfig.Password == "" && repoConfig.PasswordFile == "" {
		return "", nil
	}

	if repoConfig.Password != "" {
		return repoConfig.Password, nil
	}

	passwordBytes, err := os.ReadFile(repoConfig.PasswordFile)
	if err != nil {
		return "", fmt.Errorf("could not read password file %q: %w", repoConfig.PasswordFile, err)
	}

	// trim newline and whitespaces
	return strings.TrimSpace(string(passwordBytes)), nil
}
