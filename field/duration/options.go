// Package duration provides options for time.Duration field rendering in clog.
package duration

import (
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
)

// Duration is the value type for duration fields.
type Duration = core.DurationField

// TimeScale is a magnitude-keyed rounding and precision scale.
type TimeScale = core.TimeScale

// Option configures how a duration field is rendered.
type Option func(*Duration)

// WithGradientMax returns an [Option] that overrides the logger's duration
// gradient maximum for this field. The gradient runs from the first stop at
// 0 to the last stop at max.
//
//	clog.Info().Duration("latency", d, duration.WithGradientMax(3*time.Second))
func WithGradientMax(maximum time.Duration) Option {
	return func(d *Duration) {
		d.GradientMax = &maximum
	}
}

// WithGradient returns an [Option] that overrides the logger's duration
// gradient color stops for this field.
func WithGradient(stops ...style.ColorStop) Option {
	return func(d *Duration) {
		d.Gradient = stops
	}
}

// WithGradientMode returns an [Option] that overrides the logger's duration
// gradient transition mode ([style.GradientFade] or [style.GradientStep])
// for this field.
func WithGradientMode(mode style.GradientMode) Option {
	return func(d *Duration) {
		d.GradientMode = &mode
	}
}

// WithMinimum returns an [Option] that overrides the logger's duration
// minimum threshold for this field - the field is hidden entirely when its
// value falls below minimum.
func WithMinimum(minimum time.Duration) Option {
	return func(d *Duration) {
		d.Minimum = &minimum
	}
}

// WithRound returns an [Option] that overrides the logger's duration rounding
// granularity for this field. 0 disables rounding for this field.
func WithRound(round time.Duration) Option {
	return func(d *Duration) {
		d.Round = &round
	}
}

// WithScale returns an [Option] that overrides the logger's duration scale for
// this field. An empty scale uses the logger's scalar duration settings.
func WithScale(scale TimeScale) Option {
	return func(d *Duration) {
		d.Scale = scale
	}
}

// WithOmitOnDone returns an [Option] that controls whether an animated
// duration field is omitted from the fx.Builder done row.
func WithOmitOnDone(omit bool) Option {
	return func(d *Duration) {
		d.OmitOnDone = omit
	}
}

// Apply applies the given options to d.
func Apply(d *Duration, opts ...Option) {
	for _, o := range opts {
		o(d)
	}
}
