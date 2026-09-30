package ui

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNewPrinterDefaults(t *testing.T) {
	p := NewPrinter()
	if p.Out != os.Stdout || p.In != os.Stdin {
		t.Error("NewPrinter harus default ke stdout/stdin")
	}
}

func TestReaderFallsBackToStdinWhenNil(t *testing.T) {
	p := &Printer{Out: &bytes.Buffer{}, In: nil}
	// In nil → fallback ke os.Stdin; cukup pastikan tidak panik.
	if p.reader() == nil {
		t.Error("reader() tidak boleh nil")
	}
	// Reader harus persisten (objek yang sama dipakai ulang).
	if p.reader() != p.reader() {
		t.Error("reader() harus mengembalikan instance yang sama")
	}
}

func TestWaitCommandEcho(t *testing.T) {
	// waitCommand diuji dengan command trivial yang ada di semua OS.
	var name string
	var args []string
	if os.PathSeparator == '\\' {
		name, args = "cmd", []string{"/c", "echo", "hai"}
	} else {
		name, args = "echo", []string{"hai"}
	}
	if err := waitCommand(name, args...); err != nil {
		t.Fatalf("waitCommand error = %v", err)
	}
}

func TestStepOutputEndsWithNewline(t *testing.T) {
	out := &bytes.Buffer{}
	p := &Printer{Out: out, In: strings.NewReader("")}
	p.Step("selesai")
	if !strings.HasSuffix(out.String(), "\n") {
		t.Error("output harus diakhiri newline")
	}
}
