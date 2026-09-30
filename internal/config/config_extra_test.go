package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectPath(t *testing.T) {
	if got := ProjectPath(); got != ".commit-ai.yaml" {
		t.Errorf("ProjectPath() = %q", got)
	}
}

// withFakeConfigHome mengarahkan os.UserConfigDir ke tmp/cfg pada semua OS:
// Windows membaca %AppData%, Unix membaca XDG_CONFIG_HOME atau HOME/.config.
func withFakeConfigHome(t *testing.T, tmp string) {
	t.Helper()
	t.Setenv("AppData", filepath.Join(tmp, "cfg"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "cfg"))
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
}

func TestGlobalPathUnderUserConfigDir(t *testing.T) {
	tmp := t.TempDir()
	withFakeConfigHome(t, tmp)

	gp, err := GlobalPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(gp), "commit-ai/config.yaml") {
		t.Errorf("GlobalPath() = %q, harus berakhir di commit-ai/config.yaml", gp)
	}
}

func TestLoadReadsGlobalLayer(t *testing.T) {
	tmp := t.TempDir()
	withFakeConfigHome(t, tmp)

	dir := filepath.Join(tmp, "cfg", "commit-ai")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("model: global-model\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Pindah ke direktori tanpa .commit-ai.yaml.
	t.Chdir(t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "global-model" {
		t.Errorf("Model = %q, want global-model dari layer global", cfg.Model)
	}
}

func TestLoadNullValueKeepsDefault(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(path, []byte("model:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != Defaults().Model {
		t.Errorf("Model = %q; null di YAML tidak boleh menimpa default", cfg.Model)
	}
}

func TestTemplateYAMLValid(t *testing.T) {
	tmpl := TemplateYAML()
	// Template harus bisa di-parse dan menghasilkan default penuh.
	dir := t.TempDir()
	path := filepath.Join(dir, "from-template.yaml")
	if err := os.WriteFile(path, []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatalf("template harus valid YAML: %v", err)
	}
	if cfg != Defaults() {
		t.Errorf("template menghasilkan %+v, want defaults", cfg)
	}
}

func TestValidateRemainingBranches(t *testing.T) {
	base := Defaults()

	cases := []struct {
		name  string
		mutat func(*Config)
	}{
		{"endpoint kosong", func(c *Config) { c.Endpoint = "" }},
		{"model kosong", func(c *Config) { c.Model = "" }},
		{"format tidak dikenal", func(c *Config) { c.Format = "json" }},
		{"subject_max_length nol", func(c *Config) { c.SubjectMaxLength = 0 }},
	}
	for _, c := range cases {
		cfg := base
		c.mutat(&cfg)
		if err := Validate(cfg); err == nil {
			t.Errorf("%s harus gagal validasi", c.name)
		}
	}
}
