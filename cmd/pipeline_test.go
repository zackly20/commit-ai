package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Integration test ini menjalankan pipeline penuh tanpa LLM sungguhan:
// Ollama digantikan mock HTTP server — cocok untuk mesin ber-speks rendah.

func gitAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git tidak tersedia, lewati integration test")
	}
}

// setupRepo membuat repository Git sementara dengan satu file di-stage
// dan project config yang menunjuk ke model mock.
func setupRepo(t *testing.T, endpoint string) {
	t.Helper()
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

	if err := os.WriteFile("a.txt", []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "a.txt")

	cfgYaml := "model: mock-model\nendpoint: " + endpoint + "\n"
	if err := os.WriteFile(".commit-ai.yaml", []byte(cfgYaml), 0o644); err != nil {
		t.Fatal(err)
	}
}

// mockOllama memulai server tiruan Ollama yang selalu mengembalikan
// commit message yang diberikan.
func mockOllama(t *testing.T, message string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]string{{"name": "mock-model:latest"}},
			})
		case "/api/chat":
			var req struct {
				Messages []struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			if len(req.Messages) != 2 {
				t.Errorf("want 2 messages (system+user), got %d", len(req.Messages))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"message": map[string]string{"role": "assistant", "content": message},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// headSubject mengembalikan subject commit terakhir; string kosong berarti
// belum ada commit.
func headSubject(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "log", "-1", "--pretty=%s").CombinedOutput()
	if err != nil {
		return "" // repo tanpa commit
	}
	return strings.TrimSpace(string(out))
}

// TestPipelineGenerateOnly: alur penuh mode generate, tidak boleh commit.
func TestPipelineGenerateOnly(t *testing.T) {
	gitAvailable(t)
	srv := mockOllama(t, "feat(mock): add greeting module")
	setupRepo(t, srv.URL)

	if err := runPipeline(context.Background(), false); err != nil {
		t.Fatalf("runPipeline(generate) = %v", err)
	}
	if headSubject(t) != "" {
		t.Errorf("mode generate tidak boleh membuat commit, dapat: %q", headSubject(t))
	}
}

// TestPipelineCommitWithYes: happy path AC-01 — generate lalu commit.
func TestPipelineCommitWithYes(t *testing.T) {
	gitAvailable(t)
	srv := mockOllama(t, "feat(mock): add greeting module")
	setupRepo(t, srv.URL)

	oldYes := flagYes
	flagYes = true // lewati menu interaktif; commit tetap eksplisit
	t.Cleanup(func() { flagYes = oldYes })

	if err := runPipeline(context.Background(), true); err != nil {
		t.Fatalf("runPipeline(commit) = %v", err)
	}
	if got := headSubject(t); got != "feat(mock): add greeting module" {
		t.Errorf("head subject = %q, want %q", got, "feat(mock): add greeting module")
	}
}

// TestPipelineNoStagedChanges: AC-07 — harus error dengan pesan jelas.
func TestPipelineNoStagedChanges(t *testing.T) {
	gitAvailable(t)
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
	if err := os.WriteFile("a.txt", []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	} // tidak di-stage

	err := runPipeline(context.Background(), true)
	if err == nil {
		t.Fatal("want error for no staged changes")
	}
	if !strings.Contains(err.Error(), "staged") {
		t.Errorf("error = %v, should mention staged", err)
	}
}

// TestPipelineOllamaDown: AC-08 — error dapat ditindaklanjuti.
func TestPipelineOllamaDown(t *testing.T) {
	gitAvailable(t)
	setupRepo(t, "http://127.0.0.1:1") // port mati

	err := runPipeline(context.Background(), true)
	if err == nil {
		t.Fatal("want error when ollama unreachable")
	}
}

// TestProjectConfigPickedUp memastikan .commit-ai.yaml dibaca oleh pipeline.
func TestProjectConfigPickedUp(t *testing.T) {
	gitAvailable(t)
	srv := mockOllama(t, "feat(mock): loaded project config")
	setupRepo(t, srv.URL)

	cfg, err := resolvedConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "mock-model" {
		t.Errorf("model = %q, want mock-model dari project config", cfg.Model)
	}
	if cfg.Endpoint != srv.URL {
		t.Errorf("endpoint = %q, want %q dari project config", cfg.Endpoint, srv.URL)
	}
}
