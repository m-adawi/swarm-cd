package util

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

// ConfigConflict records whether stacks/repos are defined inline in
// config.yaml (which prevents API persistence from working) and whether
// the corresponding split files also exist on disk.
type ConfigConflict struct {
	StacksInline     bool
	ReposInline      bool
	StacksFileExists bool
	ReposFileExists  bool
}

// Conflict is populated during LoadConfigs and reflects the inline vs.
// split-file state detected at startup.
var Conflict ConfigConflict

type StackConfig struct {
	Repo                 string
	Branch               string
	Tag                  string
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
	Address              string                  `mapstructure:"address"`
}

var Configs Config

func LoadConfigs() (err error) {
	err = readConfig()
	if err != nil {
		return fmt.Errorf("could not read configuration file: %w", err)
	}

	// Detect inline config conflicts and check for split files on disk.
	Conflict = detectConflicts()

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

	// Emit startup log warnings for detected conflicts.
	logConfigWarnings(Logger)

	return
}

// detectConflicts checks whether stacks/repos were loaded inline from
// config.yaml and whether the corresponding split files exist on disk.
func detectConflicts() ConfigConflict {
	c := ConfigConflict{
		StacksInline: Configs.StackConfigs != nil,
		ReposInline:  Configs.RepoConfigs != nil,
	}
	if _, err := os.Stat("stacks.yaml"); err == nil {
		c.StacksFileExists = true
	}
	if _, err := os.Stat("repos.yaml"); err == nil {
		c.ReposFileExists = true
	}
	return c
}

// ConfigWarnings returns human-readable warning strings based on the
// current ConfigConflict state. The returned slice is empty when no
// conflicts exist.
func ConfigWarnings() []string {
	return configWarnings(Conflict)
}

func configWarnings(c ConfigConflict) []string {
	var warnings []string

	if c.StacksInline {
		if c.StacksFileExists {
			warnings = append(warnings, "stacks.yaml is ignored because stacks are defined in config.yaml")
		} else {
			warnings = append(warnings, "API changes to stacks won't persist across restarts")
		}
	}

	if c.ReposInline {
		if c.ReposFileExists {
			warnings = append(warnings, "repos.yaml is ignored because repos are defined in config.yaml")
		} else {
			warnings = append(warnings, "API changes to repos won't persist across restarts")
		}
	}

	return warnings
}

func logConfigWarnings(log *slog.Logger) {
	if log == nil {
		return
	}
	for _, w := range ConfigWarnings() {
		log.Warn(w)
	}
}

func readConfig() (err error) {
	configViper := viper.New()
	configViper.SetConfigName("config")
	configViper.AddConfigPath(".")
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
