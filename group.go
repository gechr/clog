package clog

import (
	"context"

	"github.com/gechr/clog/fx"
)

// Group creates a new animation group using the [Default] logger.
func Group(ctx context.Context) *fx.Group {
	return Default.Group(ctx)
}

// Group creates a new animation group.
func (l *Logger) Group(ctx context.Context) *fx.Group {
	return &fx.Group{Ctx: ctx, Log: fxLogger{l}}
}
