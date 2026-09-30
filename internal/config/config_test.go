package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Provider != "ollama" {
		t.Errorf("Provider = %q", cfg.Provider)
	}
	if cfg.Endpoint != "http://localhost:11434" {
		t.Errorf("Endpoint = %q", cfg.Endpoint)
	}
	if cfg.Model != "qwen2.5-coder" {
		t.Errorf("Model = %q", cfg.Model)
	}
	if cfg.MaxDiffChars != 20000 || cfg.SubjectMaxLength != 72 {
		t.Errorf("MaxDiffChars = %d, SubjectMaxLength = %d", cfg.MaxDiffChars, cfg.SubjectMaxLength)
	}
}

func TestLoadPathPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("model: llama3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "llama3" {
		t.Errorf("Model = %q, want llama3 (override)", cfg.Model)
	}
	if cfg.Endpoint != "http://localhost:11434" {
		t.Errorf("Endpoint = %q, want default (partial override)", cfg.Endpoint)
	}
}

func TestLoadPathMissingFile(t *testing.T) {
	cfg, err := LoadPath(filepath.Join(t.TempDir(), "tidak-ada.yaml"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if cfg != Defaults() {
		t.Errorf("cfg = %+v, want defaults", cfg)
	}
}

func TestLoadPathInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("model: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPath(path); err == nil {
		t.Fatal("want error for invalid YAML")
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "cfg.yaml")

	want := Defaults()
	want.Model = "mistral"
	want.ConfirmBeforeCommit = false

	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("round trip mismatch: got %+v want %+v", got, want)
	}
}

func TestValidate(t *testing.T) {
	if err := Validate(Defaults()); err != nil {
		t.Fatalf("defaults should validate: %v", err)
	}
	bad := Defaults()
	bad.Provider = "openai"
	if err := Validate(bad); err == nil {
		t.Error("non-ollama provider should fail validation in MVP")
	}
	bad2 := Defaults()
	bad2.MaxDiffChars = 0
	if err := Validate(bad2); err == nil {
		t.Error("max_diff_chars=0 should fail validation")
	}
}
