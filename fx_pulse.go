package clog

import (
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/pulse"
)

// Pulse creates a new [fx.Builder] using the [Default] logger with an
// animated color pulse on the message text.
// All characters fade uniformly between colors in the gradient.
// With no options, the default pulse gradient and speed are used.
// Use [pulse.WithGradient] and [pulse.WithSpeed] to customise the animation.
func Pulse(msg string, opts ...pulse.Option) *fx.Builder {
	return Default.Pulse(msg, opts...)
}

// Pulse creates a new [fx.Builder] with an animated color pulse on the message text.
// All characters fade uniformly between colors in the gradient.
// With no options, the default pulse gradient and speed are used.
// Use [pulse.WithGradient] and [pulse.WithSpeed] to customise the animation.
func (l *Logger) Pulse(msg string, opts ...pulse.Option) *fx.Builder {
	cfg := pulse.ApplyOptions(opts)
	return fx.NewBuilder(fx.BuilderConfig{
		Logger:       fxLogger{l},
		Mode:         fx.AnimationPulse,
		Level:        LevelInfo,
		Message:      msg,
		PulseStops:   cfg.Gradient,
		Speed:        cfg.Speed,
		SpinnerStyle: l.resolveSpinnerStyle(),
	})
}
