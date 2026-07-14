package clog

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputSetSuppressEchoDuringAnimationsNonTTYNoop(t *testing.T) {
	var buf bytes.Buffer
	o := TestOutput(&buf)

	// A non-TTY writer has no terminal to control: neither enabling nor
	// disabling may create a live region as a side effect.
	o.SetSuppressEchoDuringAnimations(true)
	assert.Nil(t, o.region.Load())
	o.SetSuppressEchoDuringAnimations(false)
	assert.Nil(t, o.region.Load())
}

// fakeTTYOutput fabricates a TTY-flagged Output over a plain buffer so the
// echo-controller wiring can be exercised without a real terminal (the
// controller's ioctls no-op on the non-terminal fd).
func fakeTTYOutput(t *testing.T, buf *bytes.Buffer) *Output {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "fake-tty")
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })

	o := TestOutput(buf)
	o.isTTY = true
	o.fd = int(f.Fd())
	return o
}

func TestLoggerSuppressEchoSurvivesOutputReplacement(t *testing.T) {
	var buf bytes.Buffer
	l := New(fakeTTYOutput(t, &buf))

	// Enabling installs a controller (and thus a region) on the current output.
	l.SetSuppressEchoDuringAnimations(true)
	require.NotNil(t, l.output.region.Load())

	// The setting sticks to the logger: a replacement output gets it too.
	replacement := fakeTTYOutput(t, &buf)
	l.SetOutput(replacement)
	assert.NotNil(t, replacement.region.Load())

	// With the setting off, a replacement output is left untouched.
	l.SetSuppressEchoDuringAnimations(false)
	untouched := fakeTTYOutput(t, &buf)
	l.SetOutput(untouched)
	assert.Nil(t, untouched.region.Load())
}

func TestStderr(t *testing.T) {
	out := Stderr(ColorNever)

	assert.NotNil(t, out)
	assert.Equal(t, os.Stderr, out.Writer())
}

func TestIsTTY(t *testing.T) {
	t.Run("test_output", func(t *testing.T) {
		var buf bytes.Buffer

		out := TestOutput(&buf)

		assert.False(t, out.IsTTY(), "TestOutput should not be a TTY")
	})

	t.Run("new_output_non_fd", func(t *testing.T) {
		var buf bytes.Buffer

		out := NewOutput(&buf, ColorNever)

		assert.False(t, out.IsTTY(), "non-fd writer should not be a TTY")
	})
}

func TestWidth(t *testing.T) {
	t.Run("non_tty_returns_zero", func(t *testing.T) {
		var buf bytes.Buffer

		out := TestOutput(&buf)

		assert.Equal(t, 0, out.Width())
	})

	t.Run("cached_returns_same", func(t *testing.T) {
		var buf bytes.Buffer

		out := TestOutput(&buf)

		first := out.Width()
		second := out.Width()

		assert.Equal(t, first, second)
	})
}

func TestRefreshWidth(t *testing.T) {
	var buf bytes.Buffer

	out := TestOutput(&buf)

	w1 := out.Width()
	out.RefreshWidth()
	w2 := out.Width()

	// For non-TTY, both should be 0.
	assert.Equal(t, 0, w1)
	assert.Equal(t, 0, w2)
}

func TestCursorPositionUsesInjectedQuery(t *testing.T) {
	var buf bytes.Buffer

	out := TestOutput(&buf)
	out.isTTY = true
	want := cursorPosition{row: 7, column: 11}
	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return want, true
	}

	got, ok := out.cursorPosition()

	require.True(t, ok)
	assert.Equal(t, want, got)
}

func TestParseCursorPositionReport(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, ok := parseCursorPositionReport("\x1b[12;34R")

		require.True(t, ok)
		assert.Equal(t, cursorPosition{row: 12, column: 34}, got)
	})

	t.Run("valid_with_prefix", func(t *testing.T) {
		got, ok := parseCursorPositionReport("noise\x1b[3;9R")

		require.True(t, ok)
		assert.Equal(t, cursorPosition{row: 3, column: 9}, got)
	})

	t.Run("invalid", func(t *testing.T) {
		_, ok := parseCursorPositionReport("\x1b[3;R")

		assert.False(t, ok)
	})
}
