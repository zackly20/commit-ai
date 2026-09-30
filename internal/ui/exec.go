package ui

import (
	"os"
	"os/exec"
)

// waitCommand menjalankan command dan menunggu sampai selesai.
// Dipakai oleh implementasi default RunEditor.
func waitCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
