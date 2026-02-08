package util

import (
	"os"
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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up env var after test
			originalValue, originalSet := os.LookupEnv("CONFIGS_PATH")
			defer func() {
				if originalSet {
					os.Setenv("CONFIGS_PATH", originalValue)
				} else {
					os.Unsetenv("CONFIGS_PATH")
				}
			}()

			if tt.envSet {
				os.Setenv("CONFIGS_PATH", tt.envValue)
			} else {
				os.Unsetenv("CONFIGS_PATH")
			}

			if got := getConfigsPath(); got != tt.want {
				t.Errorf("getConfigsPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
