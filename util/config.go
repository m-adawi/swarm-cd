package util

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/viper"
)

type StackConfig struct {
	Repo                 string
	Branch               string
	ComposeFile          string   `mapstructure:"compose_file"`
	ValuesFile           string   `mapstructure:"values_file"`
	SopsFiles            []string `mapstructure:"sops_files"`
	SopsSecretsDiscovery bool     `mapstructure:"sops_secrets_discovery"`
	AlwaysPullContainers *bool    `mapstructure:"always_pull_containers"`
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
	Concurrency          int                     `mapstructure:"concurrency"`
	AutoRotate           bool                    `mapstructure:"auto_rotate"`
	StackConfigs         map[string]*StackConfig `mapstructure:"stacks"`
	RepoConfigs          map[string]*RepoConfig  `mapstructure:"repos"`
	SopsSecretsDiscovery bool                    `mapstructure:"sops_secrets_discovery"`
	Address              string                  `mapstructure:"address"`
	AlwaysPullContainers bool                    `mapstructure:"always_pull_containers"`
	GlobalValues         map[string]any          `mapstructure:"global_values"`
}

var Configs Config

func LoadConfigs() (err error) {
	configsPath := getConfigsPath()
	Logger.Info(fmt.Sprintf("using configuration path: %s", configsPath))
	err = ReadConfig(configsPath)
	if err != nil {
		return fmt.Errorf("could not read configuration file: %w", err)
	}
	if Configs.RepoConfigs == nil {
		err = readRepoConfigs(configsPath)
		if err != nil {
			return fmt.Errorf("could not read repos file: %w", err)
		}
	}
	if Configs.StackConfigs == nil {
		err = readStackConfigs(configsPath)
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
	validateConfig()
	return nil
}

func getConfigsPath() string {
	if path := os.Getenv("CONFIGS_PATH"); path != "" {
		return path
	}
	return "."
}

const defaultWorkers = 3

// ReadConfig loads the main config file. If configPath points to an existing
// file it is used directly; otherwise it is treated as a directory to search.
// An empty string falls back to the current directory.
func ReadConfig(configPath string) (err error) {
	configViper := viper.New()
	configViper.SetConfigName("config")
	if configPath == "" {
		configViper.AddConfigPath(".")
	} else if info, statErr := os.Stat(configPath); statErr == nil && !info.IsDir() {
		configViper.SetConfigFile(configPath)
	} else {
		configViper.AddConfigPath(configPath)
	}
	configViper.SetDefault("update_interval", 120)
	configViper.SetDefault("concurrency", defaultWorkers)
	configViper.SetDefault("repos_path", "repos")
	configViper.SetDefault("auto_rotate", true)
	configViper.SetDefault("sops_secrets_discovery", false)
	configViper.SetDefault("always_pull_containers", true)
	configViper.SetDefault("address", "0.0.0.0:8080")
	err = configViper.ReadInConfig()
	if err != nil && !errors.As(err, &viper.ConfigFileNotFoundError{}) {
		return
	}
	return configViper.Unmarshal(&Configs)
}

func readRepoConfigs(path string) (err error) {
	reposViper := viper.New()
	reposViper.SetConfigName("repos")
	reposViper.AddConfigPath(path)
	err = reposViper.ReadInConfig()
	if err != nil {
		return
	}
	return reposViper.Unmarshal(&Configs.RepoConfigs)
}

func readStackConfigs(path string) (err error) {
	stacksViper := viper.New()
	stacksViper.SetConfigName("stacks")
	stacksViper.AddConfigPath(path)
	err = stacksViper.ReadInConfig()
	if err != nil {
		return
	}
	return stacksViper.Unmarshal(&Configs.StackConfigs)
}

func ReadGlobalValues(globalPath string) (err error) {
	globalViper := viper.New()
	globalViper.SetConfigName("global_values")
	globalViper.AddConfigPath(getConfigsPath())
	if globalPath != "" {
		globalViper.SetConfigFile(globalPath)
	}
	err = globalViper.ReadInConfig()
	if err != nil {
		if errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return nil
		}
		return
	}
	return globalViper.Unmarshal(&Configs.GlobalValues)
}

func validateConfig() {
	if Configs.Concurrency <= 0 {
		Logger.Warn(fmt.Sprintf("Invalid `config.concurrency value`, using default: %v", defaultWorkers))
		Configs.Concurrency = defaultWorkers
	}
}
