//go:build unix

package clog

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyResize registers ch to receive terminal-resize notifications. On
// Unix the kernel delivers SIGWINCH whenever the controlling terminal's
// dimensions change.
func notifyResize(ch chan os.Signal) {
	signal.Notify(ch, syscall.SIGWINCH)
}
