// Package ui menyediakan helper output terminal dan prompt interaktif.
// Output tidak bergantung hanya pada warna — simbol teks selalu disertakan
// (NFR Accessibility). Out dan In dapat di-inject untuk keperluan test.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Printer menulis output berlangkah ke sebuah writer dan membaca
// input dari In (default: stdin).
type Printer struct {
	Out io.Writer
	In  io.Reader

	br *bufio.Reader // reader persisten agar antar-prompt tidak kehilangan buffer
}

// NewPrinter membuat Printer ke stdout/stdin.
func NewPrinter() *Printer { return &Printer{Out: os.Stdout, In: os.Stdin} }

// reader mengembalikan bufio.Reader persisten atas input yang di-inject.
func (p *Printer) reader() *bufio.Reader {
	if p.br == nil {
		in := p.In
		if in == nil {
			in = os.Stdin
		}
		p.br = bufio.NewReader(in)
	}
	return p.br
}

func (p *Printer) printf(format string, a ...any) {
	fmt.Fprintf(p.Out, format, a...)
}

// Step menampilkan progres langkah dengan simbol ✓.
func (p *Printer) Step(msg string) { p.printf("✓ %s\n", msg) }

// Info menampilkan informasi dengan simbol •.
func (p *Printer) Info(msg string) { p.printf("• %s\n", msg) }

// Warn menampilkan peringatan dengan simbol !.
func (p *Printer) Warn(msg string) { p.printf("! %s\n", msg) }

// Error menampilkan error dengan simbol x.
func (p *Printer) Error(msg string) { p.printf("x %s\n", msg) }

// Print menampilkan teks apa adanya.
func (p *Printer) Print(msg string) { p.printf("%s\n", msg) }

// Select menampilkan menu pilihan dan mengembalikan indeks pilihan user.
// Navigasi angka 1..n lalu Enter; tahanan panah tidak dibutuhkan agar
// tetap berjalan di semua terminal (Windows included).
func (p *Printer) Select(question string, options []string) (int, error) {
	p.printf("\n? %s\n", question)
	for i, opt := range options {
		p.printf("  %d) %s\n", i+1, opt)
	}
	p.printf("Pilih 1-%d: ", len(options))

	in := p.reader()
	for {
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			return -1, fmt.Errorf("input dibatalkan")
		}
		line = strings.TrimSpace(line)
		var n int
		if _, err := fmt.Sscanf(line, "%d", &n); err == nil && n >= 1 && n <= len(options) {
			return n - 1, nil
		}
		p.printf("Pilihan tidak valid. Masukkan angka 1-%d: ", len(options))
	}
}

// Confirm menanyakan ya/tidak; default saat Enter adalah defaultValue.
func (p *Printer) Confirm(question string, defaultValue bool) (bool, error) {
	suffix := "[y/N]"
	if defaultValue {
		suffix = "[Y/n]"
	}
	p.printf("%s %s: ", question, suffix)

	in := p.reader()
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return false, err
	}
	line = strings.TrimSpace(strings.ToLower(line))
	if line == "" {
		return defaultValue, nil
	}
	return line == "y" || line == "yes", nil
}

// RunEditor menjalankan editor; variabel package agar dapat diganti
// pada unit test (mock editor).
var RunEditor = func(editor, path string) error {
	// Untuk notepad, buka lalu tunggu sampai ditutup.
	if isWindows() && strings.EqualFold(editor, "notepad") {
		return waitCommand("cmd", "/c", "notepad", path)
	}
	parts := strings.Fields(editor)
	return waitCommand(parts[0], append(parts[1:], path)...)
}

// pickEditor menentukan editor dari env COMMITAI_EDITOR, GIT_EDITOR,
// VISUAL, EDITOR, lalu fallback notepad (Windows) / vi (lainnya) — FR-07.
func pickEditor() string {
	editor := firstNonEmpty(
		os.Getenv("COMMITAI_EDITOR"),
		os.Getenv("GIT_EDITOR"),
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
	)
	if editor == "" {
		if isWindows() {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}
	return editor
}

// EditInEditor membuka editor untuk mengubah message dan mengembalikan hasil.
func EditInEditor(message string) (string, error) {
	editor := pickEditor()

	tmp, err := os.CreateTemp("", "commit-ai-*.txt")
	if err != nil {
		return "", fmt.Errorf("buat file sementara: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(message + "\n"); err != nil {
		tmp.Close()
		return "", fmt.Errorf("tulis file sementara: %w", err)
	}
	tmp.Close()

	if err := RunEditor(editor, tmpPath); err != nil {
		return "", fmt.Errorf("buka editor %q: %w", editor, err)
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("baca hasil editor: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	var kept []string
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue // baris komentar diabaikan seperti commit-msg git
		}
		kept = append(kept, l)
	}
	return strings.TrimSpace(strings.Join(kept, "\n")), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}
