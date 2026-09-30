package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/zackly20/commit-ai/internal/git"
	"github.com/zackly20/commit-ai/internal/prompt"
)

func TestExplainSystemPromptContract(t *testing.T) {
	sp := prompt.ExplainSystemPrompt()
	if !strings.Contains(sp, "Explain what the provided Git diff changes") {
		t.Error("system prompt explain harus meminta penjelasan diff")
	}
	if strings.Contains(sp, "commit message") {
		t.Error("explain tidak boleh diminta membuat commit message")
	}
}

func TestBuildExplainUserPrompt(t *testing.T) {
	got := prompt.BuildExplainUserPrompt("diff --git a/x b/x", prompt.Options{Language: "id"})
	if !strings.Contains(got, `Respond in language code "id"`) {
		t.Errorf("user prompt harus menyebut bahasa, dapat: %s", got)
	}
	if !strings.Contains(got, "Git diff:") {
		t.Error("user prompt harus memuat diff")
	}
}

func TestNormalizeExplanation(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			"buang fence",
			"```text\nBaris satu.\nBaris dua.\n```",
			"Baris satu.\nBaris dua.",
		},
		{
			"rapikan blank lines",
			"\n\nA.\n\n\n\nB.\n\n",
			"A.\n\nB.",
		},
		{
			"trim spasi",
			"A.   \n  B.",
			"A.\n  B.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := prompt.NormalizeExplanation(c.raw); got != c.want {
				t.Errorf("NormalizeExplanation() = %q, want %q", got, c.want)
			}
		})
	}
}

// setupExplainRepo menyiapkan repo dengan commit awal; mengembalikan dir.
func setupExplainRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git tidak tersedia")
	}
	dir := t.TempDir()
	t.Chdir(dir)

	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Tester")
	if err := os.WriteFile("a.txt", []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")
	run("commit", "-q", "-m", "init")
	return dir
}

func TestGitDiffUnstaged(t *testing.T) {
	dir := setupExplainRepo(t)

	if err := os.WriteFile(dir+"/a.txt", []byte("hello world changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := git.NewService(dir)
	diff, err := svc.Diff()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "+hello world changed") {
		t.Errorf("diff = %q, harus memuat perubahan", diff)
	}

	// Staged diff kosong karena tidak ada yang di-stage.
	staged, err := svc.StagedDiff()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(staged) != "" {
		t.Errorf("staged diff harus kosong, dapat %q", staged)
	}
}

func TestExplainPipelineStaged(t *testing.T) {
	dir := setupExplainRepo(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"mock-model:latest"}]}`))
		case "/api/chat":
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"Menambahkan fungsi utilitas baru."}}`))
		}
	}))
	t.Cleanup(srv.Close)
	writeProjectConfig(t, "model: mock-model\nendpoint: "+srv.URL+"\n")

	// Stage perubahan baru agar staged diff tidak kosong.
	if err := os.WriteFile(dir+"/b.txt", []byte("new module\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := execGit("add", "b.txt"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	setFlagYes(t, true)
	if err := runExplain(context.Background()); err != nil {
		t.Fatalf("runExplain = %v", err)
	}
}

func TestExplainPipelineUnstaged(t *testing.T) {
	dir := setupExplainRepo(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"mock-model:latest"}]}`))
		case "/api/chat":
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"Mengubah isi file utama."}}`))
		}
	}))
	t.Cleanup(srv.Close)
	writeProjectConfig(t, "model: mock-model\nendpoint: "+srv.URL+"\n")

	if err := os.WriteFile(dir+"/a.txt", []byte("changed content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := flagUnstaged
	flagUnstaged = true
	t.Cleanup(func() { flagUnstaged = old })

	if err := runExplain(context.Background()); err != nil {
		t.Fatalf("runExplain(--unstaged) = %v", err)
	}
}

func TestExplainPipelineNoChanges(t *testing.T) {
	setupExplainRepo(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	t.Cleanup(srv.Close)
	writeProjectConfig(t, "model: mock-model\nendpoint: "+srv.URL+"\n")

	err := runExplain(context.Background())
	if err == nil {
		t.Fatal("want error ketika tidak ada perubahan")
	}
	if !strings.Contains(err.Error(), "explain") {
		t.Errorf("err = %v, harus menyebut explain", err)
	}
}

// TestExplainPipelineOllamaDown: Ollama mati → error dapat ditindaklanjuti.
func TestExplainPipelineOllamaDown(t *testing.T) {
	setupExplainRepo(t)
	writeProjectConfig(t, "model: mock-model\nendpoint: http://127.0.0.1:1\n")

	// Stage sesuatu agar lolos cek diff.
	if err := os.WriteFile("c.txt", []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := execGit("add", "c.txt"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	if err := runExplain(context.Background()); err == nil {
		t.Fatal("want error ketika ollama tidak dapat dijangkau")
	}
}

// TestExplainPipelineTruncatesLargeDiff: diff besar dipotong tanpa error.
func TestExplainPipelineTruncatesLargeDiff(t *testing.T) {
	dir := setupExplainRepo(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_, _ = w.Write([]byte(`{"models":[{"name":"mock-model:latest"}]}`))
		case "/api/chat":
			_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"Penjelasan singkat."}}`))
		}
	}))
	t.Cleanup(srv.Close)
	writeProjectConfig(t, "model: mock-model\nendpoint: "+srv.URL+"\nmax_diff_chars: 20\n")

	if err := os.WriteFile(dir+"/big.txt", []byte(strings.Repeat("z", 500)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := execGit("add", "big.txt"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	if err := runExplain(context.Background()); err != nil {
		t.Fatalf("runExplain dengan diff besar = %v", err)
	}
}
