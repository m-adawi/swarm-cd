package util

import (
	"errors"
	"fmt"

	"github.com/spf13/viper"
)

type StackConfig struct {
	Repo                 string
	Branch               string
	ComposeFile          string   `mapstructure:"compose_file"`
	ValuesFile           string   `mapstructure:"values_file"`
	SopsFiles            []string `mapstructure:"sops_files"`
	SopsSecretsDiscovery bool     `mapstructure:"sops_secrets_discovery"`
}

type RepoConfig struct {
	Url          string
	Username     string
	Password     string
	PasswordFile string `mapstructure:"password_file"`
}

type Config struct {
	ReposPath            string                  `mapstructure:"repos_path"`
	UpdateInterval       int                     `mapstructure:"update_interval"`
	AutoRotate           bool                    `mapstructure:"auto_rotate"`
	StackConfigs         map[string]*StackConfig `mapstructure:"stacks"`
	RepoConfigs          map[string]*RepoConfig  `mapstructure:"repos"`
	SopsSecretsDiscovery bool                    `mapstructure:"sops_secrets_discovery"`
}

var Configs Config

func LoadConfigs() (err error) {
	err = readConfig()
	if err != nil {
		return fmt.Errorf("could not read configuration file: %w", err)
	}
	if Configs.RepoConfigs == nil {
		err = readRepoConfigs()
		if err != nil {
			return fmt.Errorf("could not read repos file: %w", err)
		}
	}
	if Configs.StackConfigs == nil {
		err = readStackConfigs()
		if err != nil {
			return fmt.Errorf("could not load stacks file: %w", err)
		}
	}
	return
}

func readConfig() (err error) {
	configViper := viper.New()
	configViper.SetConfigName("config")
	configViper.AddConfigPath(".")
	configViper.SetDefault("update_interval", 120)
	configViper.SetDefault("repos_path", "repos")
	configViper.SetDefault("auto_rotate", true)
	configViper.SetDefault("sops_secrets_discovery", false)
	err = configViper.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		return
	}
	err = configViper.Unmarshal(&Configs)
	if err != nil {
		return
	}
	return
}

func readRepoConfigs() (err error) {
	reposViper := viper.New()
	reposViper.SetConfigName("repos")
	reposViper.AddConfigPath(".")
	err = reposViper.ReadInConfig()
	if err != nil {
		return
	}
	err = reposViper.Unmarshal(&Configs.RepoConfigs)
	if err != nil {
		return
	}
	return
}

func readStackConfigs() (err error) {
	stacksViper := viper.New()
	stacksViper.SetConfigName("stacks")
	stacksViper.AddConfigPath(".")
	err = stacksViper.ReadInConfig()
	if err != nil {
		return
	}
	err = stacksViper.Unmarshal(&Configs.StackConfigs)
	if err != nil {
		return
	}
	return
}
