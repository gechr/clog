package clog

import (
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/spinner"
)

// Spinner creates a new [fx.Builder] using the [Default] logger with a
// rotating spinner animation.
func Spinner(msg string, opts ...spinner.Option) *fx.Builder {
	return Default().Spinner(msg, opts...)
}

// Spinner creates a new [fx.Builder] with a rotating spinner animation.
func (l *Logger) Spinner(msg string, opts ...spinner.Option) *fx.Builder {
	base := spinner.ApplyTo(l.resolveSpinnerConfig(), opts...)

	return fx.NewBuilder(fx.BuilderConfig{
		AnimatedSymbol: true,
		Logger:         fxLogger{l},
		Mode:           fx.AnimationNone,
		Level:          LevelInfo,
		Message:        msg,
		SpinnerConfig:  base,
	})
}
