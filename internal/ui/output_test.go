package ui

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// newTestPrinter membuat Printer dengan Out/In berupa buffer in-memory.
func newTestPrinter(input string) (*Printer, *bytes.Buffer) {
	out := &bytes.Buffer{}
	p := &Printer{
		Out: out,
		In:  strings.NewReader(input),
	}
	return p, out
}

func TestSelectPicksValidOption(t *testing.T) {
	p, out := newTestPrinter("2\n")
	idx, err := p.Select("What do you want to do?", []string{"Commit", "Edit", "Regenerate", "Cancel"})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if idx != 1 {
		t.Fatalf("Select() = %d, want 1 (Edit)", idx)
	}
	if !strings.Contains(out.String(), "? What do you want to do?") {
		t.Errorf("output harus menampilkan pertanyaan, dapat: %q", out.String())
	}
	if !strings.Contains(out.String(), "4) Cancel") {
		t.Errorf("output harus menampilkan semua opsi bernomor")
	}
}

func TestSelectRetriesOnInvalidThenAccepts(t *testing.T) {
	p, _ := newTestPrinter("abc\n99\n0\n3\n")
	idx, err := p.Select("pilih", []string{"a", "b", "c"})
	if err != nil {
		t.Fatalf("Select() error = %v", err)
	}
	if idx != 2 {
		t.Fatalf("Select() = %d, want 2", idx)
	}
}

func TestSelectStopsAtEOF(t *testing.T) {
	p, _ := newTestPrinter("")
	if _, err := p.Select("pilih", []string{"a", "b"}); err == nil {
		t.Fatal("want error ketika input habis (EOF)")
	}
}

func TestConfirmDefaults(t *testing.T) {
	// Enter kosong → default
	p, _ := newTestPrinter("\n")
	got, err := p.Confirm("lanjut?", true)
	if err != nil || got != true {
		t.Fatalf("Confirm() = %v, %v; want true, nil", got, err)
	}

	p2, _ := newTestPrinter("\n")
	got2, err2 := p2.Confirm("lanjut?", false)
	if err2 != nil || got2 != false {
		t.Fatalf("Confirm() = %v, %v; want false, nil", got2, err2)
	}
}

func TestConfirmYesNoVariants(t *testing.T) {
	cases := []struct {
		input string
		def   bool
		want  bool
	}{
		{"y\n", false, true},
		{"Y\n", false, true},
		{"yes\n", false, true},
		{"YES\n", false, true},
		{"n\n", true, false},
		{"no\n", true, false},
		{"No\n", true, false},
		{"apapun\n", true, false},
	}
	for _, c := range cases {
		p, _ := newTestPrinter(c.input)
		got, err := p.Confirm("q?", c.def)
		if err != nil {
			t.Fatalf("Confirm(%q) error = %v", c.input, err)
		}
		if got != c.want {
			t.Errorf("Confirm(%q, def=%v) = %v, want %v", c.input, c.def, got, c.want)
		}
	}
}

func TestPrinterSymbols(t *testing.T) {
	p, out := newTestPrinter("")
	p.Step("satu")
	p.Info("dua")
	p.Warn("tiga")
	p.Error("empat")
	p.Print("lima")

	got := out.String()
	for _, want := range []string{"✓ satu", "• dua", "! tiga", "x empat", "lima"} {
		if !strings.Contains(got, want) {
			t.Errorf("output tidak mengandung %q:\n%s", want, got)
		}
	}
}

// --- EditInEditor ---

// withEditorEnv set env editor dan kembalikan fungsi restore.
func withEditorEnv(t *testing.T, value string) {
	t.Helper()
	name := "COMMITAI_EDITOR"
	old, had := os.LookupEnv(name)
	os.Setenv(name, value)
	t.Cleanup(func() {
		if had {
			os.Setenv(name, old)
		} else {
			os.Unsetenv(name)
		}
	})
}

// mockEditorFail memaksa RunEditor gagal untuk menguji error path.
func mockEditorFail(t *testing.T) {
	t.Helper()
	old := RunEditor
	RunEditor = func(editor, path string) error { return errors.New("editor mati") }
	t.Cleanup(func() { RunEditor = old })
}

// mockEditorAppend menyimulasikan editor yang menambah baris komentar.
func mockEditorAppend(t *testing.T) {
	t.Helper()
	old := RunEditor
	RunEditor = func(editor, path string) error {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString("\n# baris komentar harus diabaikan\n")
		return err
	}
	t.Cleanup(func() { RunEditor = old })
}

func TestEditInEditorUsesMockEditor(t *testing.T) {
	withEditorEnv(t, "mock-editor")

	var gotEditor string
	old := RunEditor
	RunEditor = func(editor, path string) error {
		gotEditor = editor
		// Simulasikan user menambahkan teks saat mengedit.
		return os.WriteFile(path, []byte("feat(edited): new message\n"), 0o644)
	}
	t.Cleanup(func() { RunEditor = old })

	got, err := EditInEditor("feat(original): old message")
	if err != nil {
		t.Fatalf("EditInEditor() error = %v", err)
	}
	if gotEditor != "mock-editor" {
		t.Errorf("editor = %q, want dari env COMMITAI_EDITOR", gotEditor)
	}
	if got != "feat(edited): new message" {
		t.Errorf("hasil edit = %q, want message yang ditulis mock", got)
	}
}

func TestEditInEditorStripsCommentLines(t *testing.T) {
	withEditorEnv(t, "mock-editor")
	mockEditorAppend(t)

	got, err := EditInEditor("fix: keep me")
	if err != nil {
		t.Fatalf("EditInEditor() error = %v", err)
	}
	if got != "fix: keep me" {
		t.Errorf("hasil = %q, baris komentar harus dibuang", got)
	}
}

func TestEditInEditorEditorError(t *testing.T) {
	withEditorEnv(t, "mock-editor")
	mockEditorFail(t)

	if _, err := EditInEditor("feat: x"); err == nil {
		t.Fatal("want error ketika editor gagal dijalankan")
	}
}

func TestEditInEditorEnvPrecedence(t *testing.T) {
	// COMMITAI_EDITOR harus menang sebelum fallback notepad/vi.
	withEditorEnv(t, "")
	os.Setenv("GIT_EDITOR", "git-editor")
	os.Setenv("VISUAL", "visual-editor")
	defer os.Unsetenv("GIT_EDITOR")
	defer os.Unsetenv("VISUAL")

	if got := pickEditor(); got != "git-editor" {
		t.Errorf("pickEditor() = %q, want git-editor (urutan env)", got)
	}

	os.Unsetenv("GIT_EDITOR")
	if got := pickEditor(); got != "visual-editor" {
		t.Errorf("pickEditor() = %q, want visual-editor", got)
	}
}

func TestEditInEditorFallsBackToDefault(t *testing.T) {
	// Pastikan tidak ada env editor yang tersisa dari test lain.
	for _, k := range []string{"COMMITAI_EDITOR", "GIT_EDITOR", "VISUAL", "EDITOR"} {
		t.Setenv(k, "")
	}
	want := "vi"
	if isWindows() {
		want = "notepad"
	}
	if got := pickEditor(); got != want {
		t.Errorf("pickEditor() = %q, want %q", got, want)
	}
}

