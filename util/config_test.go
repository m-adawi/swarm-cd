package util

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestGetConfigsPath(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		envSet   bool
		want     string
	}{
		{
			name:     "env var set with path",
			envValue: "/custom/config/path",
			envSet:   true,
			want:     "/custom/config/path",
		},
		{
			name:     "env var not set",
			envValue: "",
			envSet:   false,
			want:     ".",
		},
		{
			name:     "env var set to empty string",
			envValue: "",
			envSet:   true,
			want:     ".",
		},
	}
	const envVar = "CONFIGS_PATH"
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envSet {
				t.Setenv(envVar, tt.envValue)
			} else {
				t.Setenv(envVar, "")
				os.Unsetenv(envVar)
			}

			if got := getConfigsPath(); got != tt.want {
				t.Errorf("getConfigsPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStackConfigComposeFileChain(t *testing.T) {
	tests := []struct {
		name   string
		config StackConfig
		want   []string
	}{
		{
			name: "canonical compose files",
			config: StackConfig{
				ComposeFiles: []string{"base.yaml", "prod.yaml"},
			},
			want: []string{"base.yaml", "prod.yaml"},
		},
		{
			name: "legacy compose file fallback",
			config: StackConfig{
				ComposeFile: "compose.yaml",
			},
			want: []string{"compose.yaml"},
		},
		{
			name:   "no compose file configured",
			config: StackConfig{},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ComposeFileChain()
			if !slices.Equal(got, tt.want) {
				t.Fatalf("ComposeFileChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateConfigComposeFilesRules(t *testing.T) {
	originalConfigs := Configs
	t.Cleanup(func() {
		Configs = originalConfigs
	})

	tests := []struct {
		name      string
		stack     StackConfig
		wantError string
	}{
		{
			name: "legacy and canonical fields are both set",
			stack: StackConfig{
				ComposeFile:  "compose.yaml",
				ComposeFiles: []string{"base.yaml", "prod.yaml"},
			},
			wantError: "use either compose_file or compose_files, not both",
		},
		{
			name: "compose files list is empty",
			stack: StackConfig{
				ComposeFiles: []string{},
			},
			wantError: "compose_files must not be empty",
		},
		{
			name: "compose files list contains empty value",
			stack: StackConfig{
				ComposeFiles: []string{"base.yaml", ""},
			},
			wantError: "compose_files[1] must not be empty",
		},
		{
			name: "valid canonical compose files",
			stack: StackConfig{
				ComposeFiles: []string{"base.yaml", "prod.yaml"},
			},
		},
		{
			name: "valid legacy compose file",
			stack: StackConfig{
				ComposeFile: "compose.yaml",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stackConfig := tt.stack
			Configs = Config{
				Concurrency: defaultWorkers,
				StackConfigs: map[string]*StackConfig{
					"test-stack": &stackConfig,
				},
			}

			err := validateConfig()
			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("validateConfig() returned unexpected error: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("validateConfig() expected error containing %q, got nil", tt.wantError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("validateConfig() error = %q, want substring %q", err.Error(), tt.wantError)
			}
		})
	}
}

func TestValidateConfigSetsDefaultConcurrency(t *testing.T) {
	originalConfigs := Configs
	t.Cleanup(func() {
		Configs = originalConfigs
	})

	Configs = Config{
		Concurrency: 0,
		StackConfigs: map[string]*StackConfig{
			"test-stack": {
				ComposeFile: "compose.yaml",
			},
		},
	}

	err := validateConfig()
	if err != nil {
		t.Fatalf("validateConfig() returned unexpected error: %v", err)
	}
	if Configs.Concurrency != defaultWorkers {
		t.Fatalf("validateConfig() concurrency = %d, want %d", Configs.Concurrency, defaultWorkers)
	}
}
