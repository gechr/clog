// Package fx provides terminal animation types (spinners, progress bars,
// shimmers, and pulses) with composable builders and group orchestration.
package fx

import (
	"context"
	"io"
	"sync/atomic"
	"time"

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

	// RunAnimation runs the animation loop for a single builder.
	RunAnimation(ctx context.Context, cfg AnimationConfig) error

	// RunGroup runs the group render loop.
	RunGroup(ctx context.Context, g *Group) error
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

// AnimationConfig holds the parameters for running a single animation.
type AnimationConfig struct {
	Builder   *Builder
	FieldsPtr *atomic.Pointer[[]core.Field]
	MsgPtr    *atomic.Pointer[string]
	StartTime time.Time
	SymbolPtr *atomic.Pointer[string]
	Task      TaskFunc
}

// DoneEvent holds the parameters for logging a done event.
type DoneEvent struct {
	Err    error
	Fields []core.Field
	Level  core.Level
	Msg    string
	Parts  *[]core.Part
	Symbol *string
}
