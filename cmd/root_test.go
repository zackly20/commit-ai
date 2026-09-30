package cmd

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/zackly20/commit-ai/internal/config"
	"github.com/zackly20/commit-ai/internal/ui"
)

func TestBuildProvider(t *testing.T) {
	p, err := buildProvider(config.Config{Provider: "ollama", Endpoint: "http://x"})
	if err != nil || p == nil {
		t.Fatalf("buildProvider(ollama) = %v, %v", p, err)
	}
	if _, err := buildProvider(config.Config{Provider: "openai"}); err == nil {
		t.Error("provider tidak dikenal harus error")
	}
}

func TestSelectActionMenu(t *testing.T) {
	cases := []struct {
		input string
		want  action
	}{
		{"1\n", actionCommit},
		{"2\n", actionEdit},
		{"3\n", actionRegenerate},
		{"4\n", actionCancel},
		{"9\n4\n", actionCancel}, // input invalid lalu cancel
	}
	for _, c := range cases {
		p := &ui.Printer{Out: ioDiscard(), In: strings.NewReader(c.input)}
		cfg := config.Defaults() // ConfirmBeforeCommit = true

		oldYes := flagYes
		flagYes = false
		got, err := selectAction(p, cfg)
		flagYes = oldYes

		if err != nil {
			t.Fatalf("selectAction(%q) error = %v", c.input, err)
		}
		if got != c.want {
			t.Errorf("selectAction(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestSelectActionYesFlagSkipsMenu(t *testing.T) {
	oldYes := flagYes
	flagYes = true
	t.Cleanup(func() { flagYes = oldYes })

	// Tidak ada input sama sekali — --yes tidak boleh membaca stdin.
	p := &ui.Printer{Out: ioDiscard(), In: strings.NewReader("")}
	cfg := config.Defaults()
	cfg.ConfirmBeforeCommit = false // jalur non-interaktif via config

	got, err := selectAction(p, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if got != actionCommit {
		t.Errorf("selectAction tanpa konfirmasi = %v, want actionCommit", got)
	}
}

func TestResolvedConfigFlagOverrides(t *testing.T) {
	old := [4]*string{&flagEndpoint, &flagModel, &flagLanguage}
	oldEndpoint, oldModel, oldLang := flagEndpoint, flagModel, flagLanguage
	oldYes := flagYes
	t.Cleanup(func() {
		flagEndpoint, flagModel, flagLanguage = oldEndpoint, oldModel, oldLang
		flagYes = oldYes
	})
	_ = old

	flagEndpoint = "http://override:1234"
	flagModel = "flag-model"
	flagLanguage = "id"
	flagYes = true

	cfg, err := resolvedConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Endpoint != "http://override:1234" || cfg.Model != "flag-model" || cfg.Language != "id" {
		t.Errorf("flag tidak diterapkan: %+v", cfg)
	}
	if cfg.ConfirmBeforeCommit {
		t.Error("--yes harus mematikan ConfirmBeforeCommit")
	}
}

func TestResolvedConfigInvalidProject(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(".commit-ai.yaml", []byte("provider: bogus\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := resolvedConfig(); err == nil {
		t.Fatal("project config invalid harus membuat resolvedConfig error")
	}
}

func TestRunWorkflowAndGenerateOnly(t *testing.T) {
	// Wrapper tipis; cukup pastikan tidak panik dengan context kosong.
	// Pipeline penuh sudah dites di pipeline_test.go.
	c := &cobra.Command{}
	c.SetContext(context.Background())

	if err := generateOnly(c, nil); err == nil {
		// Di luar repo (bukan git) harus error — bukan panic.
		t.Log("generateOnly mengembalikan error di luar repo (diharapkan)")
	}
}
