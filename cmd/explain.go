package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zackly20/commit-ai/internal/ai"
	"github.com/zackly20/commit-ai/internal/git"
	"github.com/zackly20/commit-ai/internal/prompt"
	"github.com/zackly20/commit-ai/internal/ui"
)

// flagUnstaged: jelaskan working tree diff, bukan staged diff.
var flagUnstaged bool

var explainCmd = &cobra.Command{
	Use:   "explain",
	Short: "Jelaskan isi perubahan (diff) dengan Local AI",
	Long: `Explain menganalisis diff dan menjelaskan perubahan dalam bahasa alami.

Default: menjelaskan perubahan yang sudah di-stage (git diff --cached).
Gunakan --unstaged untuk menjelaskan perubahan yang belum di-stage.

Tidak ada file yang diubah atau di-commit oleh command ini.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExplain(cmd.Context())
	},
}

func init() {
	explainCmd.Flags().BoolVar(&flagUnstaged, "unstaged", false, "Jelaskan working tree diff (belum di-stage) alih-alih staged diff")
	rootCmd.AddCommand(explainCmd)
}

// runExplain menjalankan alur explain: repo → diff → AI → tampilkan.
func runExplain(ctx context.Context) error {
	p := ui.NewPrinter()

	if !git.HasGit() {
		p.Error("Git tidak ditemukan di PATH. Install Git terlebih dahulu (https://git-scm.com).")
		return errors.New("missing dependency: git")
	}

	cfg, err := resolvedConfig()
	if err != nil {
		p.Error(err.Error())
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		p.Error(err.Error())
		return err
	}
	svc := git.NewService(cwd)
	if !svc.IsRepo() {
		p.Error("Bukan Git repository. Masuk ke folder repository kamu dulu, contoh: cd my-project")
		return errors.New("not a git repository")
	}

	// Sumber diff sesuai flag (FR Phase 2: explain).
	var diff string
	if flagUnstaged {
		p.Step("Reading working tree changes...")
		diff, err = svc.Diff()
	} else {
		p.Step("Reading staged changes...")
		diff, err = svc.StagedDiff()
	}
	if err != nil {
		p.Error(err.Error())
		return err
	}
	if strings.TrimSpace(diff) == "" {
		p.Warn("Tidak ada perubahan untuk dijelaskan.")
		if flagUnstaged {
			p.Info("Ubah beberapa file dulu, atau jalankan tanpa --unstaged untuk melihat staged diff.")
		} else {
			p.Info("Stage perubahan dulu (git add .), atau pakai --unstaged untuk perubahan yang belum di-stage.")
		}
		return errors.New("no changes to explain")
	}

	if newDiff, truncated := truncateDiff(diff, cfg.MaxDiffChars); truncated {
		diff = newDiff
		p.Warn(fmt.Sprintf("Diff dipotong ke %d karakter (max_diff_chars).", len(diff)))
	}

	provider, err := buildProvider(cfg)
	if err != nil {
		p.Error(err.Error())
		return err
	}
	p.Step("Checking Ollama...")
	if err := provider.CheckConnection(ctx, cfg.Model); err != nil {
		p.Error(err.Error())
		return err
	}
	p.Step(fmt.Sprintf("Model: %s", cfg.Model))

	p.Step("Generating explanation...")
	resp, err := provider.Generate(ctx, ai.Request{
		System: prompt.ExplainSystemPrompt(),
		User:   prompt.BuildExplainUserPrompt(diff, prompt.Options{Language: cfg.Language, SubjectMaxLength: cfg.SubjectMaxLength}),
	})
	if err != nil {
		p.Error(err.Error())
		return err
	}

	explanation := prompt.NormalizeExplanation(resp)
	p.Print("")
	p.Print(explanation)
	return nil
}
