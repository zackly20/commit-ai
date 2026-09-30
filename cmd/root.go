// Package cmd mendefinisikan command CLI CommitAI berbasis Cobra.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zackly20/commit-ai/internal/ai"
	"github.com/zackly20/commit-ai/internal/config"
	"github.com/zackly20/commit-ai/internal/git"
	"github.com/zackly20/commit-ai/internal/prompt"
	"github.com/zackly20/commit-ai/internal/ui"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "commit-ai",
	Short:   "Generate Git commit messages dari staged diff menggunakan Local AI (Ollama)",
	Long: `CommitAI menganalisis staged Git diff dengan model AI lokal (Ollama)
untuk menghasilkan commit message Conventional Commits.

Workflow utama:
  git add .
  commit-ai

Source code tidak dikirim ke layanan cloud — inference berjalan di komputer kamu.`,
	Args:    cobra.NoArgs,
	// Workflow utama: analyze → generate → review → commit.
	RunE:    runWorkflow,
	Version: version,
}

// Opsi flag CLI (precedence tertinggi).
var (
	flagEndpoint string
	flagModel    string
	flagLanguage string
	flagYes      bool
)

func init() {
	rootCmd.PersistentFlags().StringVar(&flagEndpoint, "endpoint", "", "Endpoint Ollama (default http://localhost:11434)")
	rootCmd.PersistentFlags().StringVar(&flagModel, "model", "", "Nama model Ollama (default qwen2.5-coder)")
	rootCmd.PersistentFlags().StringVar(&flagLanguage, "lang", "", "Bahasa commit message (kode bahasa, contoh: en, id)")
	rootCmd.PersistentFlags().BoolVar(&flagYes, "yes", false, "Langsung commit tanpa menampilkan pilihan (konfirmasi tetap muncul)")

	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(configCmd)
}

// Execute menjalankan CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolvedConfig memuat config lalu menerapkan flag CLI.
func resolvedConfig() (config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return cfg, err
	}
	if flagEndpoint != "" {
		cfg.Endpoint = flagEndpoint
	}
	if flagModel != "" {
		cfg.Model = flagModel
	}
	if flagLanguage != "" {
		cfg.Language = flagLanguage
	}
	if flagYes {
		cfg.ConfirmBeforeCommit = false
	}
	if err := config.Validate(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// buildProvider membuat provider AI sesuai config.
func buildProvider(cfg config.Config) (ai.Provider, error) {
	switch cfg.Provider {
	case "ollama":
		return ai.NewOllama(cfg.Endpoint), nil
	default:
		return nil, fmt.Errorf("provider %q tidak didukung", cfg.Provider)
	}
}

// truncateDiff memotong diff ke max karakter. Mengembalikan diff (mungkin
// tetap utuh) dan apakah pemotongan terjadi. Fungsi murni.
func truncateDiff(diff string, max int) (string, bool) {
	if len(diff) <= max {
		return diff, false
	}
	return diff[:max], true
}

// finalizeMessage menormalisasi output model lalu memvalidasinya sebagai
// commit message. Fungsi murni (FR-04 & bagian 9 PRD).
func finalizeMessage(raw string, subjectMax int) (string, error) {
	opts := prompt.Options{SubjectMaxLength: subjectMax}
	message := prompt.Normalize(raw, opts)
	if err := prompt.Validate(message, opts); err != nil {
		return "", err
	}
	return message, nil
}

// runWorkflow adalah alur utama: repo check → diff → generate → review → commit.
func runWorkflow(cmd *cobra.Command, args []string) error {
	return runPipeline(cmd.Context(), true)
}

// generateOnly menghasilkan message tanpa mengeksekusi commit.
func generateOnly(cmd *cobra.Command, args []string) error {
	return runPipeline(cmd.Context(), false)
}

// runPipeline menjalankan seluruh alur; doCommit mengontrol eksekusi commit.
func runPipeline(ctx context.Context, doCommit bool) error {
	p := ui.NewPrinter()

	// Dependency check: git harus ada (bagian 14 PRD).
	if !git.HasGit() {
		p.Error("Git tidak ditemukan di PATH. Install Git terlebih dahulu (https://git-scm.com).")
		return errors.New("missing dependency: git")
	}

	cfg, err := resolvedConfig()
	if err != nil {
		p.Error(err.Error())
		return err
	}

	// FR-01: deteksi repository.
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

	// FR-02: staged diff.
	p.Step("Checking repository...")
	diff, err := svc.StagedDiff()
	if err != nil {
		p.Error(err.Error())
		return err
	}
	if strings.TrimSpace(diff) == "" {
		p.Warn("Tidak ada staged changes.")
		p.Info("Stage perubahan dulu, contoh: git add .")
		return errors.New("no staged changes")
	}

	files, err := svc.StagedFileCount()
	if err != nil {
		p.Error(err.Error())
		return err
	}
	p.Step(fmt.Sprintf("Reading staged changes... (%d file)", files))

	// FR-02: batas ukuran diff.
	if newDiff, truncated := truncateDiff(diff, cfg.MaxDiffChars); truncated {
		agree, cerr := p.Confirm(
			fmt.Sprintf("Diff besar (%d karakter > batas %d). Lanjutkan dengan truncate?", len(diff), cfg.MaxDiffChars),
			true,
		)
		if cerr != nil {
			// Lingkungan non-interaktif: default aman adalah truncate + warning.
			agree = true
		}
		if !agree {
			p.Info("Perbesar max_diff_chars pada konfigurasi atau stage lebih sedikit file.")
			return errors.New("diff terlalu besar")
		}
		diff = newDiff
		p.Warn(fmt.Sprintf("Diff dipotong ke %d karakter.", len(diff)))
	}

	// FR-03: provider & koneksi.
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

	// FR-04/05/06: generate → tampilkan → aksi user.
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		p.Step("Generating commit message...")
		resp, err := provider.Generate(ctx, ai.Request{
			System: prompt.SystemPrompt(),
			User:   prompt.BuildUserPrompt(diff, prompt.Options{Language: cfg.Language, SubjectMaxLength: cfg.SubjectMaxLength}),
		})
		if err != nil {
			p.Error(err.Error())
			choice, cerr := p.Select("Inference gagal. Apa yang mau dilakukan?", []string{"Retry", "Cancel"})
			if cerr != nil || choice != 0 {
				return errors.New("inference dibatalkan")
			}
			lastErr = err
			continue
		}

		message, verr := finalizeMessage(resp, cfg.SubjectMaxLength)
		if verr != nil {
			p.Warn(fmt.Sprintf("Output model tidak valid: %v", verr))
			if attempt == 0 {
				lastErr = verr
				continue // satu kali retry terkontrol (bagian 9 PRD)
			}
			p.Error("Gagal menghasilkan message valid. Coba regenerate manual atau edit sendiri.")
			return verr
		}

		p.Print("")
		p.Print(fmt.Sprintf("Generated commit:\n  %s", message))

		// Mode generate: tampilkan saja, tidak commit.
		if !doCommit {
			p.Info("Mode generate: message tidak di-commit.")
			return nil
		}

		action, err := selectAction(p, cfg)
		if err != nil {
			return err
		}
		if action == actionRegenerate {
			lastErr = nil
			continue // loop ulang generate dengan diff yang sama (FR-08)
		}
		return applyAction(action, message, cfg.SubjectMaxLength, svc, p)
	}
	return lastErr
}

type action int

const (
	actionCancel action = iota
	actionCommit
	actionEdit
	actionRegenerate
)

// selectAction menampilkan menu Commit/Edit/Regenerate/Cancel (FR-06).
func selectAction(p *ui.Printer, cfg config.Config) (action, error) {
	if flagYes || !cfg.ConfirmBeforeCommit {
		return actionCommit, nil
	}
	choice, err := p.Select("What do you want to do?", []string{"Commit", "Edit", "Regenerate", "Cancel"})
	if err != nil {
		return actionCancel, err
	}
	switch choice {
	case 0:
		return actionCommit, nil
	case 1:
		return actionEdit, nil
	case 2:
		return actionRegenerate, nil
	default:
		return actionCancel, nil
	}
}

// applyAction mengeksekusi pilihan user setelah review (FR-06/07/09).
// Fungsi commit/edit/cancel; regenerate ditangani loop pemanggil.
func applyAction(a action, message string, subjectMax int, svc *git.Service, p *ui.Printer) error {
	switch a {
	case actionEdit:
		edited, err := ui.EditInEditor(message)
		if err != nil {
			p.Error(err.Error())
			return err
		}
		if err := prompt.Validate(edited, prompt.Options{SubjectMaxLength: subjectMax}); err != nil {
			p.Warn("Message hasil edit tidak valid; tetap dipakai sesuai permintaan user.")
		}
		return commitAndReport(svc, edited, p)
	case actionCommit:
		return commitAndReport(svc, message, p)
	default: // actionCancel
		p.Info("Dibatalkan. Tidak ada commit dibuat.")
		return nil
	}
}

// commitAndReport menjalankan git commit dan melaporkan hasilnya (FR-09).
// Exit code mengikuti error return; staged changes tetap utuh saat gagal.
func commitAndReport(svc *git.Service, message string, p *ui.Printer) error {
	out, err := svc.Commit(message)
	if err != nil {
		p.Error(fmt.Sprintf("Git commit gagal: %v", err))
		p.Info("Staged changes tetap utuh; perbaiki lalu coba lagi.")
		return err
	}
	p.Step("Commit created")
	p.Print(strings.TrimSpace(out))
	return nil
}
