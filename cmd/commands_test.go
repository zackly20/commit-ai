package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/zackly20/commit-ai/internal/config"
)

// TestInitCmdCreatesTemplate: `commit-ai init` menulis template, idempoten.
func TestInitCmdCreatesTemplate(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("init pertama error = %v", err)
	}
	data, err := os.ReadFile(config.ProjectPath())
	if err != nil {
		t.Fatalf("file template harus dibuat: %v", err)
	}
	if !strings.Contains(string(data), "provider: ollama") {
		t.Errorf("template harus berisi konfigurasi default:\n%s", data)
	}

	// init kedua tidak boleh menimpa.
	if err := initCmd.RunE(initCmd, nil); err != nil {
		t.Fatalf("init kedua error = %v", err)
	}
	data2, _ := os.ReadFile(config.ProjectPath())
	if string(data2) != string(data) {
		t.Error("init kedua tidak boleh mengubah file yang sudah ada")
	}
}

// TestConfigSetGetRoundTrip: config set lalu get menghasilkan nilai sama.
func TestConfigSetGetRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())

	// Project config belum ada → set harus menulis ke project config.
	if err := configSetCmd.RunE(configSetCmd, []string{"model", "llama3-test"}); err != nil {
		t.Fatalf("config set error = %v", err)
	}
	// Load via resolvedConfig harus membaca nilai baru.
	flagYes = false
	cfg, err := resolvedConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "llama3-test" {
		t.Errorf("model = %q, want llama3-test", cfg.Model)
	}

	// config get membaca kembali.
	old := flagModel
	flagModel = ""
	if err := configGetCmd.RunE(configGetCmd, []string{"model"}); err != nil {
		t.Fatalf("config get error = %v", err)
	}
	flagModel = old

	if err := configGetCmd.RunE(configGetCmd, []string{"key-tidak-ada"}); err == nil {
		t.Error("config get key tidak dikenal harus error")
	}
}

// TestConfigSetInvalidValue: validasi dijalankan sebelum simpan.
func TestConfigSetInvalidValue(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := configSetCmd.RunE(configSetCmd, []string{"max_diff_chars", "abc"}); err == nil {
		t.Error("nilai non-angka harus error")
	}
	if err := configSetCmd.RunE(configSetCmd, []string{"provider", "openai"}); err == nil {
		t.Error("provider tidak didukung harus error saat validasi")
	}
	// File tidak boleh terlanjur dibuat saat validasi gagal.
	if _, err := os.Stat(config.ProjectPath()); !os.IsNotExist(err) {
		t.Error("config gagal validasi tidak boleh menulis file")
	}
}

// TestConfigShowOutput: config show mencetak seluruh key.
func TestConfigShowOutput(t *testing.T) {
	t.Chdir(t.TempDir())

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = configShowCmd.RunE(configShowCmd, nil)
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	out := string(buf[:n])
	for _, want := range []string{"provider:", "endpoint:", "model:", "confirm_before_commit:"} {
		if !strings.Contains(out, want) {
			t.Errorf("config show output tidak mengandung %q:\n%s", want, out)
		}
	}
}

// TestConfigResetCmd: reset mengembalikan default di project yang punya config.
func TestConfigResetCmd(t *testing.T) {
	t.Chdir(t.TempDir())

	// Reset tanpa project config harus error.
	if err := configResetCmd.RunE(configResetCmd, nil); err == nil {
		t.Error("reset tanpa project config harus error")
	}

	if err := os.WriteFile(config.ProjectPath(), []byte("model: custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := configResetCmd.RunE(configResetCmd, nil); err != nil {
		t.Fatalf("reset error = %v", err)
	}
	cfg, err := config.LoadPath(config.ProjectPath())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != config.Defaults().Model {
		t.Errorf("model = %q, want default setelah reset", cfg.Model)
	}
}
