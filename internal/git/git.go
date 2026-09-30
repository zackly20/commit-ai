// Package git membungkus Git CLI: deteksi repository, staged diff,
// dan eksekusi commit. Semua eksekusi lewat interface Runner agar
// mudah di-mock pada unit test.
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Runner mengeksekusi command git dan mengembalikan stdout.
type Runner interface {
	Run(dir string, args ...string) (string, error)
}

// ExecRunner menjalankan git binary asli.
type ExecRunner struct{}

// Run menjalankan `git args...` pada dir dan mengembalikan stdout.
func (ExecRunner) Run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.Output()
	if err != nil {
		return "", wrapGitError(err, args)
	}
	return string(out), nil
}

// wrapGitError menyertakan stderr Git agar error tetap informatif (FR-09).
func wrapGitError(err error, args []string) error {
	var exitErr *exec.ExitError
	if ok := asExitError(err, &exitErr); ok && len(exitErr.Stderr) > 0 {
		return fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(exitErr.Stderr)))
	}
	return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
}

func asExitError(err error, target **exec.ExitError) bool {
	e, ok := err.(*exec.ExitError)
	if ok {
		*target = e
	}
	return ok
}

// Service menyediakan operasi Git pada sebuah direktori.
type Service struct {
	runner Runner
	dir    string
}

// NewService membuat Service dengan runner asli (git binary).
func NewService(dir string) *Service {
	return &Service{runner: ExecRunner{}, dir: dir}
}

// NewServiceWithRunner membuat Service dengan runner kustom (untuk test).
func NewServiceWithRunner(dir string, r Runner) *Service {
	return &Service{runner: r, dir: dir}
}

// IsRepo memeriksa apakah dir berada di dalam Git repository,
// termasuk subdirectory dengan root di parent (FR-01).
func (s *Service) IsRepo() bool {
	_, err := s.runner.Run(s.dir, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

// StagedDiff mengembalikan output `git diff --cached` (FR-02).
func (s *Service) StagedDiff() (string, error) {
	out, err := s.runner.Run(s.dir, "diff", "--cached")
	if err != nil {
		return "", fmt.Errorf("read staged diff: %w", err)
	}
	return out, nil
}

// Diff mengembalikan working tree diff (perubahan yang belum di-stage).
// Dipakai `commit-ai explain --unstaged`.
func (s *Service) Diff() (string, error) {
	out, err := s.runner.Run(s.dir, "diff")
	if err != nil {
		return "", fmt.Errorf("read working diff: %w", err)
	}
	return out, nil
}

// StagedFileCount mengembalikan jumlah file yang di-stage.
func (s *Service) StagedFileCount() (int, error) {
	out, err := s.runner.Run(s.dir, "diff", "--cached", "--name-only")
	if err != nil {
		return 0, fmt.Errorf("list staged files: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return 0, nil
	}
	return len(strings.Split(out, "\n")), nil
}

// Commit membuat commit dengan message yang diberikan (FR-09).
// commit dipanggil dengan --allow-empty-message=false default; message
// sudah melewati validasi di layer prompt.
func (s *Service) Commit(message string) (string, error) {
	if strings.TrimSpace(message) == "" {
		return "", fmt.Errorf("commit message kosong")
	}
	out, err := s.runner.Run(s.dir, "commit", "-m", message)
	if err != nil {
		return "", fmt.Errorf("commit failed: %w", err)
	}
	return out, nil
}

// HasGit memeriksa apakah binary git tersedia di PATH.
func HasGit() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
