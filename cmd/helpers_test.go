package cmd

import (
	"io"
	"os"
	"testing"
)

// ioDiscard mengembalikan writer pembuang output (untuk test Printer).
func ioDiscard() io.Writer { return io.Discard }

// writeProjectConfig menulis .commit-ai.yaml dengan konten yang diberikan.
func writeProjectConfig(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(".commit-ai.yaml", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
