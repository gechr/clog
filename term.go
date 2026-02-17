package clog

import (
	"os"

	"golang.org/x/term"
)

// IsTerminal returns true if stdout is connected to a terminal.
func IsTerminal() bool {
	//nolint:gosec // Fd() fits in int on all supported platforms
	return term.IsTerminal(int(os.Stdout.Fd()))
}
