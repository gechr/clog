// Package fx provides terminal animation types (spinners, progress bars,
// shimmers, and pulses) with composable builders and group orchestration.
package fx

import (
	"io"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/core"
)

// Logger is the interface that the root clog package implements.
// It decouples the fx animation types from the concrete *clog.Logger so the
// fx package can reference logging operations without importing root clog.
type Logger interface {
	// Done logs a done event after an animation finishes.
	Done(evt DoneEvent)

	// WithIndent returns a Logger with additional indent depth and tree positions.
	WithIndent(depth int, tree []core.TreePos) Logger

	// Output returns the Output associated with this Logger.
	Output() Output

	// TaskConfig snapshots everything rendering needs for one task,
	// captured under the logger's lock. Closures carry the root-only
	// formatting logic into fx.
	TaskConfig(b *Builder) TaskConfig
}

// Output provides terminal capabilities needed by animation types.
type Output interface {
	IsTTY() bool
	Writer() io.Writer
	Width() int
	ColorsDisabled() bool
	PathLink(path string, line, column int) string
	Hyperlink(url, text string) string
}

// RenderOutput extends [Output] with the terminal-geometry queries the
// render loops need to size and reposition the live block.
type RenderOutput interface {
	Output

	Height() int
	// CursorPosition reports the cursor's current 1-based row, if known.
	CursorPosition() (row int, ok bool)
	// ListenResize starts refreshing cached dimensions on terminal resize.
	// Call the returned stop function to release the listener.
	ListenResize() (stop func())
	RefreshWidth()
	RefreshHeight()
}

// TaskConfig is an immutable snapshot of logger settings captured under the
// logger's mutex. It stores exactly what per-tick rendering needs so the
// animation loops never touch the logger after the initial capture.
type TaskConfig struct {
	AnimationInterval time.Duration // minimum tick interval; 0 = no clamp
	Indentation       string        // pre-computed indent string before message
	IsTTY             bool          // output.IsTTY()
	Label             string        // pre-computed padded label for the builder's level
	LevelSymbol       string        // styled padded label for the builder's level
	NonTTYSilent      bool          // suppress all output on non-TTY writers
	Order             []core.Part   // log-line part order
	Out               io.Writer     // output.Writer()
	Output            RenderOutput  // for Width()/Height()/cursor queries
	ReportTimestamp   bool
	TimeFormat        string
	TimeLocation      *time.Location

	// Closures over root-private formatting and styling state.
	FormatFields   func(fields []core.Field) string
	StyleLevel     func(lvl core.Level) string // styled padded label for level overrides
	StyleMessage   func(msg string, lvl core.Level) string
	StyleSymbol    func(symbol string, lvl core.Level) string
	StyleTimestamp func(ts string) string
}

// DoneEvent holds the parameters for logging a done event.
type DoneEvent struct {
	Err      error
	Fields   []core.Field
	Level    core.Level
	Msg      string
	MsgStyle *lipgloss.Style
	Parts    *[]core.Part
	Symbol   *string
}
