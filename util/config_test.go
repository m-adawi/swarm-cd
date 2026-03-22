package util

import (
	"os"
	"strings"
	"testing"
)

// ---------- Config conflict detection tests ----------

// TestConfigConflict_InlineStacksNoFile verifies that when stacks are
// defined inline in config.yaml and no stacks.yaml exists on disk,
// the conflict is detected and the "won't persist" warning is emitted.
func TestConfigConflict_InlineStacksNoFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	origConfigs := Configs
	origConflict := Conflict
	defer func() {
		Configs = origConfigs
		Conflict = origConflict
	}()

	cfgYAML := `
stacks:
  my-stack:
    repo: my-repo
    branch: main
`
	if err := os.WriteFile("config.yaml", []byte(cfgYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile("repos.yaml", []byte("my-repo:\n  url: https://example.com\n"), 0644); err != nil {
		t.Fatalf("write repos.yaml: %v", err)
	}

	Configs = Config{}
	if err := LoadConfigs(); err != nil {
		t.Fatalf("LoadConfigs: %v", err)
	}

	if !Conflict.StacksInline {
		t.Error("expected StacksInline=true")
	}
	if Conflict.StacksFileExists {
		t.Error("expected StacksFileExists=false")
	}

	warnings := ConfigWarnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "API changes to stacks won't persist") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'won't persist' warning for stacks, got %v", warnings)
	}
}

// TestConfigConflict_InlineStacksWithFile verifies that when stacks are
// defined inline in config.yaml AND stacks.yaml also exists on disk,
// the "ignored" warning is emitted.
func TestConfigConflict_InlineStacksWithFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	origConfigs := Configs
	origConflict := Conflict
	defer func() {
		Configs = origConfigs
		Conflict = origConflict
	}()

	cfgYAML := `
stacks:
  my-stack:
    repo: my-repo
    branch: main
`
	if err := os.WriteFile("config.yaml", []byte(cfgYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile("stacks.yaml", []byte("other-stack:\n  repo: other\n"), 0644); err != nil {
		t.Fatalf("write stacks.yaml: %v", err)
	}
	if err := os.WriteFile("repos.yaml", []byte("my-repo:\n  url: https://example.com\n"), 0644); err != nil {
		t.Fatalf("write repos.yaml: %v", err)
	}

	Configs = Config{}
	if err := LoadConfigs(); err != nil {
		t.Fatalf("LoadConfigs: %v", err)
	}

	if !Conflict.StacksInline {
		t.Error("expected StacksInline=true")
	}
	if !Conflict.StacksFileExists {
		t.Error("expected StacksFileExists=true")
	}

	warnings := ConfigWarnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "stacks.yaml is ignored") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'stacks.yaml is ignored' warning, got %v", warnings)
	}
}

// TestConfigConflict_SplitOnly verifies that when stacks are only in
// stacks.yaml (not inline), no conflict is detected.
func TestConfigConflict_SplitOnly(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	origConfigs := Configs
	origConflict := Conflict
	defer func() {
		Configs = origConfigs
		Conflict = origConflict
	}()

	cfgYAML := `
update_interval: 60
`
	if err := os.WriteFile("config.yaml", []byte(cfgYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile("stacks.yaml", []byte("my-stack:\n  repo: my-repo\n  branch: main\n"), 0644); err != nil {
		t.Fatalf("write stacks.yaml: %v", err)
	}
	if err := os.WriteFile("repos.yaml", []byte("my-repo:\n  url: https://example.com/repo\n"), 0644); err != nil {
		t.Fatalf("write repos.yaml: %v", err)
	}

	Configs = Config{}
	if err := LoadConfigs(); err != nil {
		t.Fatalf("LoadConfigs: %v", err)
	}

	if Conflict.StacksInline {
		t.Error("expected StacksInline=false")
	}
	if Conflict.ReposInline {
		t.Error("expected ReposInline=false")
	}

	warnings := ConfigWarnings()
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for split-only config, got %v", warnings)
	}
}

// TestConfigConflict_InlineReposNoFile verifies that when repos are
// defined inline in config.yaml and no repos.yaml exists on disk,
// the "won't persist" warning is emitted.
func TestConfigConflict_InlineReposNoFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	origConfigs := Configs
	origConflict := Conflict
	defer func() {
		Configs = origConfigs
		Conflict = origConflict
	}()

	cfgYAML := `
repos:
  my-repo:
    url: https://example.com/repo
`
	if err := os.WriteFile("config.yaml", []byte(cfgYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile("stacks.yaml", []byte("my-stack:\n  repo: my-repo\n  branch: main\n"), 0644); err != nil {
		t.Fatalf("write stacks.yaml: %v", err)
	}

	Configs = Config{}
	if err := LoadConfigs(); err != nil {
		t.Fatalf("LoadConfigs: %v", err)
	}

	if !Conflict.ReposInline {
		t.Error("expected ReposInline=true")
	}
	if Conflict.ReposFileExists {
		t.Error("expected ReposFileExists=false")
	}

	warnings := ConfigWarnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "API changes to repos won't persist") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'won't persist' warning for repos, got %v", warnings)
	}
}

// TestConfigConflict_InlineReposWithFile verifies that when repos are
// defined inline AND repos.yaml exists, the "ignored" warning is emitted.
func TestConfigConflict_InlineReposWithFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(origDir)

	origConfigs := Configs
	origConflict := Conflict
	defer func() {
		Configs = origConfigs
		Conflict = origConflict
	}()

	cfgYAML := `
repos:
  my-repo:
    url: https://example.com/repo
`
	if err := os.WriteFile("config.yaml", []byte(cfgYAML), 0644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	if err := os.WriteFile("repos.yaml", []byte("other-repo:\n  url: https://other.com\n"), 0644); err != nil {
		t.Fatalf("write repos.yaml: %v", err)
	}
	if err := os.WriteFile("stacks.yaml", []byte("my-stack:\n  repo: my-repo\n  branch: main\n"), 0644); err != nil {
		t.Fatalf("write stacks.yaml: %v", err)
	}

	Configs = Config{}
	if err := LoadConfigs(); err != nil {
		t.Fatalf("LoadConfigs: %v", err)
	}

	if !Conflict.ReposInline {
		t.Error("expected ReposInline=true")
	}
	if !Conflict.ReposFileExists {
		t.Error("expected ReposFileExists=true")
	}

	warnings := ConfigWarnings()
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "repos.yaml is ignored") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'repos.yaml is ignored' warning, got %v", warnings)
	}
}
