package clog

import (
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/spinner"
)

// Spinner creates a new [fx.Builder] using the [Default] logger with a
// rotating spinner animation.
func Spinner(msg string, opts ...spinner.Option) *fx.Builder {
	return Default.Spinner(msg, opts...)
}

// Spinner creates a new [fx.Builder] with a rotating spinner animation.
func (l *Logger) Spinner(msg string, opts ...spinner.Option) *fx.Builder {
	base := spinner.DefaultStyle()
	l.mu.Lock()
	if l.spinnerStyle != nil {
		base = *l.spinnerStyle
	}
	l.mu.Unlock()

	for _, o := range opts {
		o(&base)
	}

	return fx.NewBuilder(fx.BuilderConfig{
		Logger:       fxLogger{l},
		Mode:         fx.AnimationSpinner,
		Level:        LevelInfo,
		Message:      msg,
		SpinnerStyle: base,
	})
}
