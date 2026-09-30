// Package config menangani konfigurasi CommitAI: default bawaan, global,
// project (.commit-ai.yaml), dan flag CLI — dengan precedence:
// flags > project > global > default.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config menyimpan seluruh pengaturan CommitAI.
type Config struct {
	Provider            string `yaml:"provider"`
	Endpoint            string `yaml:"endpoint"`
	Model               string `yaml:"model"`
	Format              string `yaml:"format"`
	Language            string `yaml:"language"`
	MaxDiffChars        int    `yaml:"max_diff_chars"`
	SubjectMaxLength    int    `yaml:"subject_max_length"`
	ConfirmBeforeCommit bool   `yaml:"confirm_before_commit"`
}

// Defaults mengembalikan konfigurasi bawaan sesuai PRD.
func Defaults() Config {
	return Config{
		Provider:            "ollama",
		Endpoint:            "http://localhost:11434",
		Model:               "qwen2.5-coder",
		Format:              "conventional",
		Language:            "en",
		MaxDiffChars:        20000,
		SubjectMaxLength:    72,
		ConfirmBeforeCommit: true,
	}
}

// GlobalDir mengembalikan direktori konfigurasi global (~/.config/commit-ai).
func GlobalDir() (string, error) {
	if d, err := os.UserConfigDir(); err == nil {
		return filepath.Join(d, "commit-ai"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "commit-ai"), nil
}

// GlobalPath mengembalikan path file konfigurasi global.
func GlobalPath() (string, error) {
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// ProjectPath mengembalikan path file .commit-ai.yaml pada cwd.
func ProjectPath() string {
	return ".commit-ai.yaml"
}

// Load memuat konfigurasi dengan precedence default < global < project.
// File yang tidak ada di-skip; error YAML pada file yang ada bersifat fatal.
func Load() (Config, error) {
	cfg := Defaults()

	globalPath, err := GlobalPath()
	if err != nil {
		return cfg, fmt.Errorf("resolve global config dir: %w", err)
	}
	if err := mergeFile(&cfg, globalPath); err != nil {
		return cfg, err
	}
	if err := mergeFile(&cfg, ProjectPath()); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// LoadPath memuat konfigurasi dari satu file di atas default
// (dipakai untuk melihat/mengubah file project secara terisolasi).
func LoadPath(path string) (Config, error) {
	cfg := Defaults()
	if err := mergeFile(&cfg, path); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// mergeFile membaca satu file YAML (jika ada) dan menimpa field yang diset.
func mergeFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}
	// yaml.v3 hanya menimpa field yang eksplisit diset di dokumen;
	// field yang absen atau null dibiarkan mempertahankan nilai sebelumnya.
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}

// Save menulis konfigurasi ke file YAML.
func Save(path string, cfg Config) error {
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// Validate memeriksa nilai konfigurasi dan mengembalikan error yang jelas.
func Validate(cfg Config) error {
	if cfg.Provider != "ollama" {
		return fmt.Errorf("provider %q tidak didukung; MVP hanya mendukung \"ollama\"", cfg.Provider)
	}
	if cfg.Endpoint == "" {
		return fmt.Errorf("endpoint tidak boleh kosong")
	}
	if cfg.Model == "" {
		return fmt.Errorf("model tidak boleh kosong")
	}
	if cfg.Format != "conventional" {
		return fmt.Errorf("format %q tidak didukung; gunakan \"conventional\"", cfg.Format)
	}
	if cfg.MaxDiffChars <= 0 {
		return fmt.Errorf("max_diff_chars harus > 0")
	}
	if cfg.SubjectMaxLength <= 0 {
		return fmt.Errorf("subject_max_length harus > 0")
	}
	return nil
}

// TemplateYAML mengembalikan isi template .commit-ai.yaml untuk `commit-ai init`.
func TemplateYAML() string {
	return `# CommitAI configuration
# Priority: CLI flags > this file > global config > defaults
provider: ollama
endpoint: http://localhost:11434
model: qwen2.5-coder
format: conventional
language: en
max_diff_chars: 20000
subject_max_length: 72
confirm_before_commit: true
`
}
