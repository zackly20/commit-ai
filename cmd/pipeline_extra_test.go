package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// execGit menjalankan git di cwd test saat ini.
func execGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// withPipedStdin mengganti os.Stdin dengan pipe berisi input yang diberikan,
// sehingga prompt interaktif di dalam runPipeline bisa diuji.
func withPipedStdin(t *testing.T, input string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString(input); err != nil {
		t.Fatal(err)
	}
	w.Close()

	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
}

func setFlagYes(t *testing.T, v bool) {
	t.Helper()
	old := flagYes
	flagYes = v
	t.Cleanup(func() { flagYes = old })
}

// TestPipelineDiffTruncation: diff melebihi max_diff_chars → konfirmasi "y"
// → truncate → tetap commit.
func TestPipelineDiffTruncation(t *testing.T) {
	gitAvailable(t)
	srv := mockOllama(t, "feat(mock): truncated diff commit")
	setupRepo(t, srv.URL)

	// Perkecil batas via project config agar diff pasti melebihi.
	if err := os.WriteFile(".commit-ai.yaml", []byte("model: mock-model\nendpoint: "+srv.URL+"\nmax_diff_chars: 10\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Timpa a.txt agar diff > 10 karakter.
	if err := os.WriteFile("a.txt", []byte(strings.Repeat("x", 500)+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := execGit("add", "a.txt"); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}

	withPipedStdin(t, "y\n") // konfirmasi truncate
	setFlagYes(t, true)      // lewati menu aksi

	if err := runPipeline(context.Background(), true); err != nil {
		t.Fatalf("runPipeline = %v", err)
	}
	if headSubject(t) != "feat(mock): truncated diff commit" {
		t.Errorf("head subject = %q", headSubject(t))
	}
}

// TestPipelineInferenceFailureCancel: Ollama balas 500 terus → pilih Cancel.
func TestPipelineInferenceFailureCancel(t *testing.T) {
	gitAvailable(t)
	fail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[{"name":"mock-model:latest"}]}`))
			return
		}
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(fail.Close)
	setupRepo(t, fail.URL)

	withPipedStdin(t, "2\n") // Retry=1, Cancel=2
	setFlagYes(t, true)

	err := runPipeline(context.Background(), true)
	if err == nil {
		t.Fatal("want error setelah cancel")
	}
	if !strings.Contains(err.Error(), "dibatalkan") {
		t.Errorf("err = %v, want pesan dibatalkan", err)
	}
	if headSubject(t) != "" {
		t.Error("tidak boleh ada commit setelah inference gagal")
	}
}

// TestPipelineInvalidOutputRetry: model balas teks non-commit → retry sekali
// → tetap invalid → error jelas, tanpa commit.
func TestPipelineInvalidOutputRetry(t *testing.T) {
	gitAvailable(t)
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_, _ = w.Write([]byte(`{"models":[{"name":"mock-model:latest"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"message":{"role":"assistant","content":"ini bukan commit message"}}`))
	}))
	t.Cleanup(bad.Close)
	setupRepo(t, bad.URL)

	setFlagYes(t, true)

	err := runPipeline(context.Background(), true)
	if err == nil {
		t.Fatal("want error untuk output model tidak valid")
	}
	if headSubject(t) != "" {
		t.Error("tidak boleh ada commit")
	}
}

// TestPipelineGenerateModeErrorFree di repo tanpa commit lain memastikan
// mode generate tidak menyentuh git commit.
func TestPipelineGenerateModeNoCommit(t *testing.T) {
	gitAvailable(t)
	srv := mockOllama(t, "feat(mock): only generate")
	setupRepo(t, srv.URL)

	setFlagYes(t, true)
	if err := runPipeline(context.Background(), false); err != nil {
		t.Fatalf("runPipeline(generate) = %v", err)
	}
	if headSubject(t) != "" {
		t.Error("mode generate tidak boleh membuat commit")
	}
}
