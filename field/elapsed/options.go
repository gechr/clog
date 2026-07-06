// Package elapsed provides options for elapsed-time field rendering in clog.
package elapsed

import (
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
)

// Elapsed is the value type for elapsed-time fields.
type Elapsed = core.ElapsedField

// Option configures how an elapsed-time field is rendered.
type Option func(*Elapsed)

// WithGradientMax returns an [Option] that overrides the logger's elapsed
// gradient maximum for this field. The gradient runs from the first stop at
// 0 to the last stop at max.
//
//	clog.Info().Elapsed("elapsed", elapsed.WithGradientMax(3*time.Second))
func WithGradientMax(maximum time.Duration) Option {
	return func(e *Elapsed) {
		e.GradientMax = &maximum
	}
}

// WithGradient returns an [Option] that overrides the logger's elapsed
// gradient color stops for this field.
//
//	clog.Info().Elapsed("elapsed", elapsed.WithGradient(
//	    style.ColorStop{Position: 0, Color: green},
//	    style.ColorStop{Position: 1, Color: red},
//	))
func WithGradient(stops ...style.ColorStop) Option {
	return func(e *Elapsed) {
		e.Gradient = stops
	}
}

// WithGradientMode returns an [Option] that overrides the logger's elapsed
// gradient transition mode ([style.GradientFade] or [style.GradientStep])
// for this field.
func WithGradientMode(mode style.GradientMode) Option {
	return func(e *Elapsed) {
		e.GradientMode = &mode
	}
}

// WithMinimum returns an [Option] that overrides the logger's elapsed minimum
// threshold for this field - the field is hidden entirely when its duration
// falls below minimum.
func WithMinimum(minimum time.Duration) Option {
	return func(e *Elapsed) {
		e.Minimum = &minimum
	}
}

// Apply applies the given options to e.
func Apply(e *Elapsed, opts ...Option) {
	for _, o := range opts {
		o(e)
	}
}
