package clog

import (
	"io"
	"os"
	"sync"

	"github.com/charmbracelet/colorprofile"
	"golang.org/x/term"
)

// Output bundles an [io.Writer] with its detected terminal capabilities
// (TTY, width, color profile). Each [Logger] holds an *Output so that
// capability detection is per-writer instead of per-process.
type Output struct {
	w       io.Writer
	fd      int // -1 for non-fd writers
	isTTY   bool
	profile colorprofile.Profile

	widthMu   sync.Mutex
	widthDone bool
	width     int
}

// NewOutput creates a new Output that wraps w. TTY detection is automatic
// for writers that expose an Fd() uintptr method (e.g. [*os.File]). The
// [ColorMode] determines how colors are handled:
//   - [ColorAuto] respects TTY detection and NO_COLOR.
//   - [ColorAlways] forces colors even on non-TTY writers.
//   - [ColorNever] disables all colors.
func NewOutput(w io.Writer, mode ColorMode) *Output {
	o := &Output{w: w, fd: -1}

	if f, ok := w.(interface{ Fd() uintptr }); ok {
		//nolint:gosec // Fd() fits in int on all supported platforms
		o.fd = int(f.Fd())
		o.isTTY = term.IsTerminal(o.fd)
	}

	o.profile = detectProfile(w, o.isTTY, mode)

	return o
}

// Stdout returns a new Output for [os.Stdout].
func Stdout(mode ColorMode) *Output {
	return NewOutput(os.Stdout, mode)
}

// Stderr returns a new Output for [os.Stderr].
func Stderr(mode ColorMode) *Output {
	return NewOutput(os.Stderr, mode)
}

// TestOutput returns a non-TTY Output with colors disabled, suitable for tests.
func TestOutput(w io.Writer) *Output {
	return NewOutput(w, ColorNever)
}

// Writer returns the underlying [io.Writer].
func (o *Output) Writer() io.Writer { return o.w }

// IsTTY returns true if the writer is connected to a terminal.
func (o *Output) IsTTY() bool { return o.isTTY }

// ColorsDisabled returns true if this output should suppress colors.
func (o *Output) ColorsDisabled() bool {
	return o.profile == colorprofile.NoTTY
}

// Width returns the terminal width, or 0 for non-TTY writers.
// The value is lazily detected and cached; call [Output.RefreshWidth]
// to re-detect.
func (o *Output) Width() int {
	o.widthMu.Lock()
	defer o.widthMu.Unlock()

	if !o.widthDone {
		o.widthDone = true

		if o.isTTY && o.fd >= 0 {
			if w, _, err := term.GetSize(o.fd); err == nil {
				o.width = w
			}
		}
	}

	return o.width
}

// RefreshWidth clears the cached terminal width so that the next call
// to [Output.Width] re-queries the terminal.
func (o *Output) RefreshWidth() {
	o.widthMu.Lock()
	defer o.widthMu.Unlock()
	o.widthDone = false
	o.width = 0
}

// writeString writes s to w, discarding the return values.
func writeString(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

// detectProfile determines the [colorprofile.Profile] for the given writer,
// TTY state, and color mode.
func detectProfile(w io.Writer, isTTY bool, mode ColorMode) colorprofile.Profile {
	switch mode {
	case ColorAlways:
		return colorprofile.TrueColor
	case ColorNever:
		return colorprofile.NoTTY
	case ColorAuto:
		if !isTTY || noColorEnvSet.Load() {
			return colorprofile.NoTTY
		}
	}
	return colorprofile.Detect(w, os.Environ())
}

// IsTerminal returns true if the [Default] logger's output is connected to a terminal.
func IsTerminal() bool {
	Default.mu.Lock()
	defer Default.mu.Unlock()
	return Default.output.IsTTY()
}
