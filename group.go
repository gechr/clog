package clog

import (
	"context"

	"github.com/gechr/clog/fx"
)

// GroupStatusFunc is called each render tick with the number of completed
// and total tasks. Use the [Update] to set the message and fields.
type GroupStatusFunc = fx.GroupStatusFunc

// Group creates a new animation group using the [Default] logger.
// Configure it with options from the fx package, e.g.
// [fx.WithParallelism] or [fx.WithHideDone].
func Group(ctx context.Context, opts ...fx.GroupOption) *fx.Group {
	return Default.Group(ctx, opts...)
}

// Group creates a new animation group. Configure it with options from the
// fx package, e.g. [fx.WithParallelism] or [fx.WithHideDone].
func (l *Logger) Group(ctx context.Context, opts ...fx.GroupOption) *fx.Group {
	g := &fx.Group{Ctx: ctx, Log: fxLogger{l}, SyncAnimations: true}
	for _, opt := range opts {
		opt(g)
	}
	return g
}
