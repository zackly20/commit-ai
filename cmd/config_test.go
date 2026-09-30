package cmd

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/zackly20/commit-ai/internal/config"
)

func TestConfigValueAllKeys(t *testing.T) {
	cfg := config.Defaults()
	keys := map[string]string{
		"provider":              cfg.Provider,
		"endpoint":              cfg.Endpoint,
		"model":                 cfg.Model,
		"format":                cfg.Format,
		"language":              cfg.Language,
		"max_diff_chars":        "20000",
		"subject_max_length":    "72",
		"confirm_before_commit": "true",
	}
	for k, want := range keys {
		got, ok := configValue(cfg, k)
		if !ok {
			t.Errorf("configValue(%q) tidak dikenal, seharusnya valid", k)
			continue
		}
		if got != want {
			t.Errorf("configValue(%q) = %q, want %q", k, got, want)
		}
	}

	// Key tidak case-sensitive.
	if _, ok := configValue(cfg, "MODEL"); !ok {
		t.Error("configValue harus case-insensitive")
	}
	if _, ok := configValue(cfg, "tidak-ada"); ok {
		t.Error("configValue(key tidak dikenal) harus ok=false")
	}
}

func TestSetConfigValueStrings(t *testing.T) {
	cfg := config.Defaults()

	pairs := map[string]string{
		"provider": "ollama",
		"endpoint": "http://localhost:9999",
		"model":    "llama3",
		"format":   "conventional",
		"language": "id",
	}
	for k, v := range pairs {
		if err := setConfigValue(&cfg, k, v); err != nil {
			t.Fatalf("setConfigValue(%q) error = %v", k, err)
		}
	}
	if cfg.Endpoint != "http://localhost:9999" || cfg.Model != "llama3" || cfg.Language != "id" {
		t.Errorf("nilai tidak tersimpan: %+v", cfg)
	}
}

func TestSetConfigValueBooleans(t *testing.T) {
	for _, v := range []string{"false", "0", "no", "FALSE"} {
		cfg := config.Defaults()
		if err := setConfigValue(&cfg, "confirm_before_commit", v); err != nil {
			t.Fatalf("setConfigValue(%q) error = %v", v, err)
		}
		if cfg.ConfirmBeforeCommit {
			t.Errorf("confirm_before_commit=%q harus false", v)
		}
	}
	for _, v := range []string{"true", "1", "yes", "YES"} {
		cfg := config.Config{}
		if err := setConfigValue(&cfg, "confirm_before_commit", v); err != nil {
			t.Fatalf("setConfigValue(%q) error = %v", v, err)
		}
		if !cfg.ConfirmBeforeCommit {
			t.Errorf("confirm_before_commit=%q harus true", v)
		}
	}
	if err := setConfigValue(&config.Config{}, "confirm_before_commit", "mungkin"); err == nil {
		t.Error("nilai boolean tidak valid harus error")
	}
}

func TestSetConfigValueInts(t *testing.T) {
	cfg := config.Defaults()
	if err := setConfigValue(&cfg, "max_diff_chars", "50000"); err != nil {
		t.Fatal(err)
	}
	if cfg.MaxDiffChars != 50000 {
		t.Errorf("MaxDiffChars = %d, want 50000", cfg.MaxDiffChars)
	}
	if err := setConfigValue(&cfg, "subject_max_length", "abc"); err == nil {
		t.Error("nilai non-angka harus error")
	}
	if err := setConfigValue(&cfg, "key_tidak_ada", "1"); err == nil {
		t.Error("key tidak dikenal harus error")
	}
}

func TestSetInt(t *testing.T) {
	var n int
	if err := setInt(&n, "x", "42"); err != nil || n != 42 {
		t.Errorf("setInt(42) = %d, %v", n, err)
	}
	if err := setInt(&n, "x", "1.5"); err == nil {
		t.Error("setInt(1.5) harus error")
	}
}

func TestPrintConfig(t *testing.T) {
	// Alihkan stdout sementara untuk menangkap output printConfig.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printConfig(config.Defaults())

	w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)

	s := string(out)
	for _, want := range []string{
		"provider: ollama",
		"endpoint: http://localhost:11434",
		"model: qwen2.5-coder",
		"max_diff_chars: 20000",
		"subject_max_length: 72",
		"confirm_before_commit: true",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("printConfig output tidak mengandung %q:\n%s", want, s)
		}
	}
}
