package clog

import (
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/shimmer"
)

// Shimmer creates a new [fx.Builder] using the [Default] logger with an
// animated gradient shimmer on the message text.
// Each character is colored independently based on its position in the wave.
// With no options, the default shimmer gradient, direction, and speed are used.
// Use [shimmer.WithGradient], [shimmer.WithDirection], and [shimmer.WithSpeed]
// to customise the animation.
func Shimmer(msg string, opts ...shimmer.Option) *fx.Builder {
	return Default().Shimmer(msg, opts...)
}

// Shimmer creates a new [fx.Builder] with an animated gradient shimmer on the message text.
// Each character is colored independently based on its position in the wave.
// With no options, the default shimmer gradient, direction, and speed are used.
// Use [shimmer.WithGradient], [shimmer.WithDirection], and [shimmer.WithSpeed]
// to customise the animation.
func (l *Logger) Shimmer(msg string, opts ...shimmer.Option) *fx.Builder {
	cfg := shimmer.Apply(opts...)
	return fx.NewBuilder(fx.BuilderConfig{
		Logger:        fxLogger{l},
		Mode:          fx.AnimationShimmer,
		Level:         LevelInfo,
		Message:       msg,
		ShimmerDir:    cfg.Direction,
		ShimmerStops:  cfg.Gradient,
		Speed:         cfg.Speed,
		SpinnerConfig: l.resolveSpinnerConfig(),
	})
}
