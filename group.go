package clog

import (
	"context"

	"github.com/gechr/clog/fx"
)

// GroupOption configures a group before it starts rendering.
type GroupOption func(*fx.Group)

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
