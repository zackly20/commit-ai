// Command mock-ollama adalah server tiruan Ollama untuk menguji CommitAI
// tanpa model AI sungguhan — berguna untuk mesin ber-speksifikasi rendah.
//
// Jalankan:
//
//	go run ./tools/mock-ollama
//
// lalu arahkan CommitAI ke server ini:
//
//	commit-ai --endpoint http://localhost:11434
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
)

type chatRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream bool `json:"stream"`
}

func main() {
	addr := flag.String("addr", "127.0.0.1:11434", "alamat listen")
	model := flag.String("model", "mock-model", "nama model yang dilaporkan tersedia")
	message := flag.String("message", "feat(mock): commit message from mock server", "commit message yang dikembalikan")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/tags", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("GET /api/tags")
		writeJSON(w, map[string]any{
			"models": []map[string]string{{"name": *model + ":latest"}},
		})
	})
	mux.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		log.Printf("POST /api/chat model=%s stream=%v", req.Model, req.Stream)
		writeJSON(w, map[string]any{
			"message": map[string]string{"role": "assistant", "content": *message},
		})
	})

	fmt.Printf("mock-ollama listening on http://%s (model: %s)\n", *addr, *model)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
