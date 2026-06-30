// Package fraction provides options for fraction field rendering in clog.
package fraction

import "github.com/gechr/clog/internal/core"

// Fraction is the value type for fraction fields.
type Fraction = core.Fraction

// NumberFormat selects how a fraction's current/total numbers are rendered.
type NumberFormat = core.NumberFormat

// Option configures how a fraction field is rendered.
type Option func(*Fraction)

// Apply applies the given options to f.
func Apply(f *Fraction, opts ...Option) {
	for _, o := range opts {
		o(f)
	}
}

// WithReverseGradient returns an [Option] that flips the gradient
// direction for this field relative to the logger default.
func WithReverseGradient() Option {
	return func(f *Fraction) {
		f.Reverse = true
	}
}

// WithFormat returns an [Option] that overrides the numeric format for this
// fraction field, taking precedence over the logger's number and fraction
// formats.
func WithFormat(format NumberFormat) Option {
	return func(f *Fraction) {
		f.Format = &format
	}
}
