package git

import (
	"strings"
	"testing"
)

// fakeRunner merekam argumen dan mengembalikan output preset.
type fakeRunner struct {
	args   []string
	out    string
	err    error
	calls  int
}

func (f *fakeRunner) Run(dir string, args ...string) (string, error) {
	f.calls++
	f.args = args
	return f.out, f.err
}

func TestIsRepo(t *testing.T) {
	r := &fakeRunner{out: "true"}
	s := NewServiceWithRunner("/repo", r)
	if !s.IsRepo() {
		t.Fatal("IsRepo() = false, want true")
	}
	if len(r.args) < 2 || r.args[0] != "rev-parse" {
		t.Errorf("args = %v, want rev-parse call", r.args)
	}
}

func TestIsRepoFalse(t *testing.T) {
	r := &fakeRunner{err: &testError{"not a repo"}}
	s := NewServiceWithRunner("/not-repo", r)
	if s.IsRepo() {
		t.Fatal("IsRepo() = true, want false")
	}
}

func TestStagedDiff(t *testing.T) {
	r := &fakeRunner{out: "diff --git a/x b/x\n+hello"}
	s := NewServiceWithRunner("/repo", r)
	got, err := s.StagedDiff()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "diff --git") {
		t.Errorf("diff output = %q", got)
	}
	if r.args[0] != "diff" || r.args[1] != "--cached" {
		t.Errorf("args = %v, want diff --cached", r.args)
	}
}

func TestStagedFileCount(t *testing.T) {
	r := &fakeRunner{out: "a.go\nb.go\nc.go"}
	s := NewServiceWithRunner("/repo", r)
	n, err := s.StagedFileCount()
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
}

func TestStagedFileCountEmpty(t *testing.T) {
	r := &fakeRunner{out: "  \n"}
	s := NewServiceWithRunner("/repo", r)
	n, err := s.StagedFileCount()
	if err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("count = %d, want 0", n)
	}
}

func TestCommit(t *testing.T) {
	r := &fakeRunner{out: "[main abc1234] feat: test"}
	s := NewServiceWithRunner("/repo", r)
	if _, err := s.Commit("feat: test"); err != nil {
		t.Fatal(err)
	}
	if r.args[0] != "commit" || r.args[1] != "-m" {
		t.Errorf("args = %v, want commit -m", r.args)
	}
}

func TestCommitEmptyMessage(t *testing.T) {
	s := NewServiceWithRunner("/repo", &fakeRunner{})
	if _, err := s.Commit("   "); err == nil {
		t.Fatal("want error for empty message")
	}
}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
