package clog

import (
	"sync/atomic"

	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/bar"
)

// Bar creates a new [fx.Builder] using the [Default] logger with a
// determinate progress bar animation.
// total is the maximum progress value. Use [Update.SetProgress] to update progress.
func Bar(msg string, total int, opts ...bar.Option) *fx.Builder {
	return Default.Bar(msg, total, opts...)
}

// Bar creates a new [fx.Builder] with a determinate progress bar animation.
// total is the maximum progress value. Use [Update.SetProgress] to update progress.
func (l *Logger) Bar(msg string, total int, opts ...bar.Option) *fx.Builder {
	if total <= 0 {
		total = 1
	}

	progressPtr := new(atomic.Int64)
	totalPtr := new(atomic.Int64)
	totalPtr.Store(int64(total))

	return fx.NewBuilder(fx.BuilderConfig{
		Logger:        fxLogger{l},
		Mode:          fx.AnimationBar,
		Level:         LevelInfo,
		PercentMax:    l.loadFieldFormats().PercentMaximum,
		Message:       msg,
		BarConfig:     bar.Apply(opts...),
		BarProgress:   progressPtr,
		BarTotal:      totalPtr,
		SpinnerConfig: l.resolveSpinnerConfig(),
	})
}
