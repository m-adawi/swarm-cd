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
	Url           string
	Username      string
	Password      string
	PasswordFile  string `mapstructure:"password_file"`
	TemplatesPath string `mapstructure:"templates_path"`
}

type Config struct {
	ReposPath            string                  `mapstructure:"repos_path"`
	UpdateInterval       int                     `mapstructure:"update_interval"`
	AutoRotate           bool                    `mapstructure:"auto_rotate"`
	StackConfigs         map[string]*StackConfig `mapstructure:"stacks"`
	RepoConfigs          map[string]*RepoConfig  `mapstructure:"repos"`
	SopsSecretsDiscovery bool                    `mapstructure:"sops_secrets_discovery"`
	Address              string                  `mapstructure:"address"`
	GlobalValues         map[string]any          `mapstructure:"global_values"`
}

var Configs Config

func LoadConfigs() (err error) {
	err = ReadConfig("")
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
	if Configs.GlobalValues == nil {
		err = ReadGlobalValues("")
		if err != nil {
			return fmt.Errorf("could not load global values file: %w", err)
		}
	}
	return
}

func ReadConfig(configPath string) (err error) {
	configViper := viper.New()
	configViper.SetConfigName("config")
	configViper.AddConfigPath(".")
	if configPath != "" {
		configViper.SetConfigFile(configPath)
	}
	configViper.SetDefault("update_interval", 120)
	configViper.SetDefault("repos_path", "repos")
	configViper.SetDefault("auto_rotate", true)
	configViper.SetDefault("sops_secrets_discovery", false)
	configViper.SetDefault("address", "0.0.0.0:8080")
	err = configViper.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		return
	}
	return configViper.Unmarshal(&Configs)
}

func readRepoConfigs() (err error) {
	reposViper := viper.New()
	reposViper.SetConfigName("repos")
	reposViper.AddConfigPath(".")
	reposViper.SetDefault("templates_path", "")
	err = reposViper.ReadInConfig()
	if err != nil {
		return
	}
	return reposViper.Unmarshal(&Configs.RepoConfigs)
}

func readStackConfigs() (err error) {
	stacksViper := viper.New()
	stacksViper.SetConfigName("stacks")
	stacksViper.AddConfigPath(".")
	err = stacksViper.ReadInConfig()
	if err != nil {
		return
	}
	return stacksViper.Unmarshal(&Configs.StackConfigs)
}

func ReadGlobalValues(globalPath string) (err error) {
	globalViper := viper.New()
	globalViper.SetConfigName("global_values")
	globalViper.AddConfigPath(".")
	if globalPath != "" {
		globalViper.SetConfigFile(globalPath)
	}
	err = globalViper.ReadInConfig()
	if err != nil {
		return
	}
	return globalViper.Unmarshal(&Configs.GlobalValues)
}
