package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCheckConnectionOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Errorf("path = %s, want /api/tags", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"qwen2.5-coder:latest"},{"name":"llama3:8b"}]}`))
	}))
	defer srv.Close()

	p := NewOllama(srv.URL)
	if err := p.CheckConnection(context.Background(), "qwen2.5-coder"); err != nil {
		t.Fatalf("CheckConnection() = %v, want nil", err)
	}
}

func TestCheckConnectionModelMissing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"models":[{"name":"llama3:latest"}]}`))
	}))
	defer srv.Close()

	p := NewOllama(srv.URL)
	err := p.CheckConnection(context.Background(), "qwen2.5-coder")
	if err == nil {
		t.Fatal("want error for missing model")
	}
	if !strings.Contains(err.Error(), "ollama pull") {
		t.Errorf("error should suggest `ollama pull`, got: %v", err)
	}
}

func TestCheckConnectionServerDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewOllama(srv.URL)
	if err := p.CheckConnection(context.Background(), "m"); err == nil {
		t.Fatal("want error for 500 response")
	}
}

func TestGenerate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("path = %s, want /api/chat", r.URL.Path)
		}
		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if len(req.Messages) != 2 {
			t.Errorf("messages = %d, want 2 (system+user)", len(req.Messages))
		}
		if req.Stream {
			t.Error("stream should be false")
		}
		_ = json.NewEncoder(w).Encode(ollamaChatResponse{
			Message: ollamaMessage{Role: "assistant", Content: "feat: add feature"},
		})
	}))
	defer srv.Close()

	p := NewOllama(srv.URL)
	got, err := p.Generate(context.Background(), Request{System: "sys", User: "usr"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "feat: add feature" {
		t.Errorf("Generate() = %q", got)
	}
}

func TestGenerateAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ollamaChatResponse{Error: "model not found"})
	}))
	defer srv.Close()

	p := NewOllama(srv.URL)
	if _, err := p.Generate(context.Background(), Request{}); err == nil {
		t.Fatal("want error for API error response")
	}
}

func TestEndpointTrailingSlash(t *testing.T) {
	p := NewOllama("http://localhost:11434/")
	if p.Endpoint != "http://localhost:11434" {
		t.Errorf("Endpoint = %q, trailing slash should be trimmed", p.Endpoint)
	}
}
