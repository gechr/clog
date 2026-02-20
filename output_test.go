package clog

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
