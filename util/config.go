package util

import (
	"errors"
	"path/filepath"
	"fmt"
	"log/slog"

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
	StackConfigsPath     string                  `mapstructure:"stacks_path"`
	RepoConfigs          map[string]*RepoConfig  `mapstructure:"repos"`
	RepoConfigsPath      string                  `mapstructure:"repos_path"`
	SopsSecretsDiscovery bool                    `mapstructure:"sops_secrets_discovery"`
	Address              string                  `mapstructure:"address"`
	GlobalValues         map[string]any          `mapstructure:"global_values"`
	GlobalValuesPath     string                  `mapstructure:"global_values_path"`
}

var Configs Config
var ConfigDir string

func LoadConfigs() (err error) {
	err = ReadConfig("")
	if err != nil {
		return fmt.Errorf("could not read configuration file: %w", err)
	}
	if Configs.RepoConfigs != nil && Configs.RepoConfigsPath != "" {
		slog.Warn("Both repos and repos_path provided, ignoring repos_path.")
		Configs.RepoConfigsPath = ""
	} else if Configs.RepoConfigs == nil {
		err = readRepoConfigs(Configs.RepoConfigsPath)
		if err != nil {
			return fmt.Errorf("could not read repos file: %w", err)
		}
	}
	if Configs.StackConfigs != nil && Configs.StackConfigsPath != "" {
		slog.Warn("Both stacks and stacks_path provided, ignoring stacks_path.")
		Configs.StackConfigsPath = ""
	} else if Configs.StackConfigs == nil {
		err = readStackConfigs(Configs.StackConfigsPath)
		if err != nil {
			return fmt.Errorf("could not load stacks file: %w", err)
		}
	}
	if Configs.GlobalValues != nil && Configs.GlobalValuesPath != "" {
		slog.Warn("Both global_values and global_values_path provided, ignoring global_values_path.")
		Configs.GlobalValuesPath = ""
	} else if Configs.GlobalValues == nil {
		err = ReadGlobalValues(Configs.GlobalValuesPath)
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
	ConfigDir = filepath.Dir(configViper.ConfigFileUsed())
	return configViper.Unmarshal(&Configs)
}

func defaultConfigViper(configName string, filePath string) (*viper.Viper){
	v := viper.New()
	v.SetConfigName(configName)
	v.AddConfigPath(ConfigDir)
	if filePath != "" {
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(ConfigDir, filePath)
		}
		v.SetConfigFile(filePath)
	}
	return v
}

func readRepoConfigs(reposPath string) (err error) {
	reposViper := defaultConfigViper("repos", reposPath)
	err = reposViper.ReadInConfig()
	if err != nil {
		return
	}
	return reposViper.Unmarshal(&Configs.RepoConfigs)
}

func readStackConfigs(stacksPath string) (err error) {
	stacksViper := defaultConfigViper("stacks", stacksPath)
	err = stacksViper.ReadInConfig()
	if err != nil {
		return
	}
	return stacksViper.Unmarshal(&Configs.StackConfigs)
}

func ReadGlobalValues(globalPath string) (err error) {
	globalViper := defaultConfigViper("global_values", globalPath)
	err = globalViper.ReadInConfig()
	if err != nil {
		return
	}
	return globalViper.Unmarshal(&Configs.GlobalValues)
}
