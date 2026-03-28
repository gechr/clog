package clog

import (
	"context"

	"github.com/gechr/clog/fx"
)

// GroupOption configures a group before it starts rendering.
type GroupOption func(*fx.Group)

// GroupStatusFunc is called each render tick with the number of completed
// and total tasks. Use the [Update] to set the message and fields.
type GroupStatusFunc = fx.GroupStatusFunc

// WithFieldAlignment sets the group-level field alignment mode.
func WithFieldAlignment(alignment FieldAlignment) GroupOption {
	return func(g *fx.Group) {
		g.FieldAlignment = alignment
	}
}

// WithParallelism limits how many group tasks may run at once.
// Values less than or equal to zero disable the limit.
func WithParallelism(parallelism int) GroupOption {
	return func(g *fx.Group) {
		g.Parallelism = parallelism
	}
}

// WithMonotonic prevents grouped bar fills and their associated percentage
// text from rendering lower than the highest progress fraction previously
// shown for each task.
func WithMonotonic() GroupOption {
	return func(g *fx.Group) {
		g.Monotonic = true
	}
}

// WithFooter adds a status line below the task block, updated each tick.
// The builder provides initial config (level, symbol, message, fields).
// The callback updates the message and fields each tick based on progress.
func WithFooter(b *fx.Builder, fn GroupStatusFunc) GroupOption {
	return func(g *fx.Group) {
		g.Footer = &fx.GroupStatus{Builder: b, Callback: fn}
	}
}

// WithHeader adds a status line above the task block, updated each tick.
// The builder provides initial config (level, symbol, message, fields).
// The callback updates the message and fields each tick based on progress.
func WithHeader(b *fx.Builder, fn GroupStatusFunc) GroupOption {
	return func(g *fx.Group) {
		g.Header = &fx.GroupStatus{Builder: b, Callback: fn}
	}
}

// WithMaxLines caps the number of visible lines in the group render block.
// When set, this takes precedence over the automatic terminal height cap.
// Values less than or equal to zero are ignored.
func WithMaxLines(n int) GroupOption {
	return func(g *fx.Group) {
		g.MaxLines = n
	}
}

// WithMaxHeightPercent caps the group render block to a percentage of the
// terminal height (e.g. 0.5 for half). Clamped to (0, 1].
// When both WithMaxLines and WithMaxHeightPercent are set, the smaller wins.
func WithMaxHeightPercent(percent float64) GroupOption {
	return func(g *fx.Group) {
		g.MaxHeightPercent = max(0, min(percent, 1))
	}
}

// WithHideDone removes completed tasks from the rendered block so that
// only active and pending tasks remain visible. Completed tasks reappear
// in the caller's own logging (e.g. via [WaitResult.Msg]).
func WithHideDone() GroupOption {
	return func(g *fx.Group) {
		g.HideDone = true
	}
}

// WithSyncAnimations controls whether animations in the group share a
// common epoch so that spinners, pulses, and shimmers stay in lockstep.
// Sync is enabled by default.
func WithSyncAnimations(sync bool) GroupOption {
	return func(g *fx.Group) {
		g.SyncAnimations = sync
	}
}

// Group creates a new animation group using the [Default] logger.
func Group(ctx context.Context, opts ...GroupOption) *fx.Group {
	return Default.Group(ctx, opts...)
}

// Group creates a new animation group.
func (l *Logger) Group(ctx context.Context, opts ...GroupOption) *fx.Group {
	g := &fx.Group{Ctx: ctx, Log: fxLogger{l}, SyncAnimations: true}
	for _, opt := range opts {
		opt(g)
	}
	return g
}
