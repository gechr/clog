//go:build darwin || linux

package clog

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoControllerNonTerminalFdNoop(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-a-tty")
	require.NoError(t, err)
	defer f.Close()

	e, ok := newEchoController(int(f.Fd())).(*echoController)
	require.True(t, ok)

	// The termios ioctl fails on a regular file, so Suppress never latches
	// and Restore stays a no-op - repeatedly.
	e.Suppress()
	assert.False(t, e.suppressed)
	e.Restore()
	e.Suppress()
	assert.False(t, e.suppressed)
}
