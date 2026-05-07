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
