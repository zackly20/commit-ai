package cmd

import "io"

// ioDiscard mengembalikan writer pembuang output (untuk test Printer).
func ioDiscard() io.Writer { return io.Discard }
