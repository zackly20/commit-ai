package cmd

import (
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/zackly20/commit-ai/internal/git"
	"github.com/zackly20/commit-ai/internal/ui"
)

// fakeGitRunner merekam argumen commit terakhir untuk assert.
type fakeGitRunner struct {
	lastArgs []string
	out      string
	err      error
}

func (f *fakeGitRunner) Run(dir string, args ...string) (string, error) {
	f.lastArgs = args
	return f.out, f.err
}

func discardPrinter() *ui.Printer {
	return &ui.Printer{Out: io.Discard, In: strings.NewReader("")}
}

func TestTruncateDiff(t *testing.T) {
	// Di bawah batas: utuh, tidak di-truncate.
	got, truncated := truncateDiff("abc", 10)
	if truncated || got != "abc" {
		t.Errorf("truncateDiff(abc,10) = %q, %v", got, truncated)
	}

	// Tepat di batas: utuh.
	got, truncated = truncateDiff("abcdef", 6)
	if truncated || got != "abcdef" {
		t.Errorf("truncateDiff tepat batas = %q, %v", got, truncated)
	}

	// Melebihi batas: dipotong tepat max karakter.
	long := strings.Repeat("x", 100)
	got, truncated = truncateDiff(long, 30)
	if !truncated {
		t.Error("diff 100 karakter dengan batas 30 harus truncated")
	}
	if len(got) != 30 {
		t.Errorf("len = %d, want 30", len(got))
	}
}

func TestFinalizeMessage(t *testing.T) {
	// Message valid langsung lolos.
	msg, err := finalizeMessage("feat(auth): add login", 72)
	if err != nil || msg != "feat(auth): add login" {
		t.Fatalf("finalizeMessage = %q, %v", msg, err)
	}

	// Prose + fence dari model dinormalisasi sampai valid.
	msg, err = finalizeMessage("Berikut hasilnya:\n```text\nfix(api): handle nil pointer\n```", 72)
	if err != nil {
		t.Fatalf("normalize+validate gagal: %v", err)
	}
	if msg != "fix(api): handle nil pointer" {
		t.Errorf("msg = %q", msg)
	}

	// Type tidak dikenal → error.
	if _, err := finalizeMessage("feature: unknown type", 72); err == nil {
		t.Error("type tidak dikenal harus error")
	}

	// Output model terlalu panjang → dinormalisasi (truncate) sampai <= 72
	// lalu valid — sesuai perilaku Normalize (bagian 9 PRD).
	long := "feat: " + strings.Repeat("a", 100)
	msg, err = finalizeMessage(long, 72)
	if err != nil {
		t.Fatalf("output panjang harus di-truncate, bukan error: %v", err)
	}
	if len(msg) > 72 {
		t.Errorf("len = %d, harus <= 72 setelah normalisasi", len(msg))
	}
}

func TestApplyActionCommit(t *testing.T) {
	r := &fakeGitRunner{out: "[main abc] feat: x"}
	svc := git.NewServiceWithRunner("/repo", r)
	p := discardPrinter()

	if err := applyAction(actionCommit, "feat: x", 72, svc, p); err != nil {
		t.Fatalf("applyAction(commit) = %v", err)
	}
	if len(r.lastArgs) < 3 || r.lastArgs[0] != "commit" || r.lastArgs[2] != "feat: x" {
		t.Errorf("args = %v, want commit -m 'feat: x'", r.lastArgs)
	}
}

func TestApplyActionCommitFails(t *testing.T) {
	r := &fakeGitRunner{err: errors.New("nothing staged")}
	svc := git.NewServiceWithRunner("/repo", r)
	p := discardPrinter()

	if err := applyAction(actionCommit, "feat: x", 72, svc, p); err == nil {
		t.Fatal("commit gagal harus mengembalikan error")
	}
}

func TestApplyActionEditUsesEditedMessage(t *testing.T) {
	withEditorEnvCmd(t)
	old := ui.RunEditor
	ui.RunEditor = func(editor, path string) error {
		return os.WriteFile(path, []byte("feat(edited): from editor\n"), 0o644)
	}
	t.Cleanup(func() { ui.RunEditor = old })

	r := &fakeGitRunner{out: "ok"}
	svc := git.NewServiceWithRunner("/repo", r)
	p := discardPrinter()

	if err := applyAction(actionEdit, "feat: original", 72, svc, p); err != nil {
		t.Fatalf("applyAction(edit) = %v", err)
	}
	if len(r.lastArgs) < 3 || r.lastArgs[2] != "feat(edited): from editor" {
		t.Errorf("commit harus memakai message hasil edit, args = %v", r.lastArgs)
	}
}

func TestApplyActionEditEditorError(t *testing.T) {
	withEditorEnvCmd(t)
	old := ui.RunEditor
	ui.RunEditor = func(editor, path string) error { return errors.New("editor mati") }
	t.Cleanup(func() { ui.RunEditor = old })

	r := &fakeGitRunner{}
	svc := git.NewServiceWithRunner("/repo", r)

	if err := applyAction(actionEdit, "feat: x", 72, svc, discardPrinter()); err == nil {
		t.Fatal("editor gagal harus error")
	}
	if r.lastArgs != nil {
		t.Error("tidak boleh commit ketika editor gagal")
	}
}

func TestApplyActionCancel(t *testing.T) {
	r := &fakeGitRunner{}
	svc := git.NewServiceWithRunner("/repo", r)

	if err := applyAction(actionCancel, "feat: x", 72, svc, discardPrinter()); err != nil {
		t.Fatalf("applyAction(cancel) = %v", err)
	}
	if r.lastArgs != nil {
		t.Error("cancel tidak boleh menjalankan git commit")
	}
}

// withEditorEnvCmd set COMMITAI_EDITOR untuk test di package cmd.
func withEditorEnvCmd(t *testing.T) {
	t.Helper()
	t.Setenv("COMMITAI_EDITOR", "mock-editor")
}
