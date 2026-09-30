// Package ai mendefinisikan interface provider AI dan implementasi
// klien Ollama HTTP API. MVP hanya menyediakan Ollama, namun interface
// Provider menjaga layer tetap terpisah (NFR Maintainability).
package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Request berisi input inference ke provider.
type Request struct {
	System string
	User   string
}

// Response berisi hasil inference.
type Response struct {
	Message string
}

// Provider adalah interface AI provider yang dapat diganti-ganti.
type Provider interface {
	// CheckConnection memverifikasi provider aktif dan model tersedia.
	CheckConnection(ctx context.Context, model string) error
	// Generate menjalankan inference dan mengembalikan teks hasil.
	Generate(ctx context.Context, req Request) (string, error)
}

// OllamaProvider adalah klien Ollama HTTP API (/api/chat).
type OllamaProvider struct {
	Endpoint string
	HTTP     *http.Client
}

// NewOllama membuat OllamaProvider dengan timeout default.
func NewOllama(endpoint string) *OllamaProvider {
	return &OllamaProvider{
		Endpoint: strings.TrimRight(endpoint, "/"),
		HTTP:     &http.Client{Timeout: 10 * time.Minute},
	}
}

// ollamaChatRequest adalah payload /api/chat.
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResponse struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error"`
}

// CheckConnection memeriksa Ollama aktif dan model tersedia (FR-03).
func (o *OllamaProvider) CheckConnection(ctx context.Context, model string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.Endpoint+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	resp, err := o.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("ollama tidak dapat dijangkau di %s; pastikan Ollama berjalan (contoh: `ollama serve`)", o.Endpoint)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("ollama merespons %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return fmt.Errorf("parse response /api/tags: %w", err)
	}
	for _, m := range tags.Models {
		name := m.Name
		// "model:latest" dianggap sama dengan "model".
		if name == model || strings.SplitN(name, ":", 2)[0] == model {
			return nil
		}
	}
	return fmt.Errorf("model %q belum tersedia di Ollama; jalankan `ollama pull %s` terlebih dahulu", model, model)
}

// Generate memanggil /api/chat tanpa streaming dan mengembalikan pesan hasil.
func (o *OllamaProvider) Generate(ctx context.Context, req Request) (string, error) {
	payload := ollamaChatRequest{
		Messages: []ollamaMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.User},
		},
		Stream:  false,
		Options: map[string]any{"temperature": 0.2},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, o.Endpoint+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gagal menghubungi Ollama di %s; pastikan Ollama berjalan", o.Endpoint)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama merespons %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var chat ollamaChatResponse
	if err := json.Unmarshal(respBody, &chat); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}
	if chat.Error != "" {
		return "", fmt.Errorf("ollama error: %s", chat.Error)
	}
	return chat.Message.Content, nil
}
