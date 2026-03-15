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

// Group creates a new animation group using the [Default] logger.
func Group(ctx context.Context, opts ...GroupOption) *fx.Group {
	return Default.Group(ctx, opts...)
}

// Group creates a new animation group.
func (l *Logger) Group(ctx context.Context, opts ...GroupOption) *fx.Group {
	g := &fx.Group{Ctx: ctx, Log: fxLogger{l}}
	for _, opt := range opts {
		opt(g)
	}
	return g
}
