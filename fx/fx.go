// Package fx provides terminal animation types (spinners, progress bars,
// shimmers, and pulses) with composable builders and group orchestration.
package fx

import (
	"io"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
)

// The types below appear throughout this package's exported signatures. They
// are aliased here so that fx's surface is nameable without importing the
// root clog package - which fx itself cannot import, and which a consumer of
// fx alone has no reason to. Each alias denotes the identical type as its
// clog counterpart, so values pass between the two packages unconverted.
type (
	// Field is one structured key/value pair on a log line.
	Field = core.Field

	// FieldBuilder carries the fluent field methods shared by every builder.
	// [Builder], [Update], and the result builders embed it. T is the
	// concrete builder its methods return for chaining.
	FieldBuilder[T any] = core.FieldBuilder[T]

	// Part identifies a component of a formatted log line, used to override
	// part order (see [Builder.Parts]).
	Part = core.Part

	// Percent is a fractional progress value in the range 0..1, as returned
	// by [Builder.BarPercentValue].
	Percent = core.Percent

	// TreePos identifies a node's position among its siblings in a tree
	// (see [Builder.Tree]).
	TreePos = core.TreePos
)

// The [Part] values, re-exported alongside the type so a part order can be
// expressed without reaching for another package.
const (
	PartTimestamp = core.PartTimestamp
	PartLevel     = core.PartLevel
	PartSymbol    = core.PartSymbol
	PartMessage   = core.PartMessage
	PartFields    = core.PartFields
)

// Logger is the interface that the root clog package implements.
// It decouples the fx animation types from the concrete *clog.Logger so the
// fx package can reference logging operations without importing root clog.
type Logger interface {
	// Done logs a done event after an animation finishes.
	Done(evt DoneEvent)

	// WithIndent returns a Logger with additional indent depth and tree positions.
	WithIndent(depth int, tree []TreePos) Logger

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

// LiveRegion coordinates a single repaintable block of animation lines with
// regular log writes on the same terminal. An [Output] implementation opts
// into a shared region - one that also displaces log lines written to the
// same writer - by additionally providing:
//
//	LiveRegion() *fx.LiveRegion
//
// An Output that does not provide one gets a private region wrapping its
// writer, which coordinates the animations of a single run but not the
// surrounding log output. The root clog Output provides a shared region, and
// the clog package aliases this same type as clog.LiveRegion.
type LiveRegion = core.LiveRegion

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
	// RefreshSize re-queries the terminal size and updates the cached
	// dimensions.
	RefreshSize()
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
	Silent            bool          // suppress all output; the builder's level is below the logger minimum
	Order             []Part        // log-line part order
	Out               io.Writer     // output.Writer()
	Output            RenderOutput  // for Width()/Height()/cursor queries
	ReportTimestamp   bool
	TimeFormat        string
	TimeLocation      *time.Location

	// Closures over root-private formatting and styling state.
	FormatFields   func(fields []Field) string
	StyleLevel     func(lvl level.Level) string // styled padded label for level overrides
	StyleMessage   func(msg string, lvl level.Level) string
	StyleSymbol    func(symbol string, lvl level.Level) string
	StyleTimestamp func(ts string) string
}

// DoneEvent holds the parameters for logging a done event.
type DoneEvent struct {
	Err      error
	Fields   []Field
	Level    level.Level
	Msg      string
	MsgStyle *lipgloss.Style
	Parts    *[]Part
	Symbol   *string
}
