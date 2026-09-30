package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupRealRepo membuat repository git sungguhan untuk ExecRunner.
func setupRealRepo(t *testing.T) string {
	t.Helper()
	if !HasGit() {
		t.Skip("git tidak tersedia")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "t@t.co")
	run("config", "user.name", "t")
	return dir
}

func TestExecRunnerRealGit(t *testing.T) {
	dir := setupRealRepo(t)

	r := ExecRunner{}
	out, err := r.Run(dir, "status", "--short")
	if err != nil {
		t.Fatalf("Run(status) error = %v", err)
	}
	if out != "" {
		t.Errorf("repo baru harus bersih, dapat %q", out)
	}

	// Error path: command git yang salah harus membawa stderr.
	if _, err := r.Run(dir, "log", "--oneline"); err == nil {
		t.Error("git log pada repo tanpa commit harus error")
	} else if !strings.Contains(err.Error(), "git log") {
		t.Errorf("error harus menyebut command, dapat: %v", err)
	}
}

func TestExecRunnerInvalidCommand(t *testing.T) {
	r := ExecRunner{}
	if _, err := r.Run(t.TempDir(), "perintah-tidak-ada"); err == nil {
		t.Error("git subcommand tidak dikenal harus error")
	}
}

func TestWrapGitErrorNonExit(t *testing.T) {
	plain := &testError{"boom"}
	got := wrapGitError(plain, []string{"status"})
	if !strings.Contains(got.Error(), "boom") || !strings.Contains(got.Error(), "git status") {
		t.Errorf("wrapGitError = %v, harus membawa pesan asli + command", got)
	}
}

func TestAsExitError(t *testing.T) {
	var target *exec.ExitError
	if ok := asExitError(error(&exec.ExitError{}), &target); !ok {
		t.Error("asExitError harus true untuk *exec.ExitError")
	}
	if ok := asExitError(&testError{"x"}, &target); ok {
		t.Error("asExitError harus false untuk error biasa")
	}
}

func TestNewServiceUsesExecRunner(t *testing.T) {
	dir := setupRealRepo(t)
	s := NewService(dir)
	if !s.IsRepo() {
		t.Error("NewService pada repo asli harus mendeteksi repo")
	}
}

func TestCommitRealRepo(t *testing.T) {
	dir := setupRealRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := NewService(dir)
	if _, err := s.runner.Run(dir, "add", "a.txt"); err != nil {
		t.Fatal(err)
	}
	out, err := s.Commit("feat: test commit")
	if err != nil {
		t.Fatalf("Commit error = %v", err)
	}
	if !strings.Contains(out, "feat: test commit") {
		t.Errorf("output commit = %q", out)
	}
}

func TestHasGit(t *testing.T) {
	if !HasGit() {
		t.Skip("git tidak tersedia di lingkungan ini")
	}
}
