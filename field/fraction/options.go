// Package fraction provides options for fraction field rendering in clog.
package fraction

import "github.com/gechr/clog/internal/core"

// Fraction is the value type for fraction fields.
type Fraction = core.Fraction

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
