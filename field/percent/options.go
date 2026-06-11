// Package percent provides options for percent field rendering in clog.
package percent

import "github.com/gechr/clog/internal/core"

// Percent is the value type for percent fields.
type Percent = core.Percent

// Option configures how a percent field is rendered.
type Option func(*Percent)

// WithReverseGradient returns an [Option] that flips the gradient
// direction for this field relative to the logger default. If the logger is
// using the normal gradient (red=0%, green=100%), the field renders inverted
// (green=0%, red=100%), and vice versa.
func WithReverseGradient() Option {
	return func(p *Percent) {
		p.Reverse = true
	}
}

// WithMaximum returns an [Option] that overrides the logger's percent
// maximum for this field. For example, use WithMaximum(100) when passing
// values in the 0–100 range instead of the default 0–1 range.
func WithMaximum(m float64) Option {
	return func(p *Percent) {
		p.Maximum = &m
	}
}

// Apply applies the given options to p.
func Apply(p *Percent, opts []Option) {
	for _, o := range opts {
		o(p)
	}
}

// EffectiveMaximum returns the per-field maximum if set, otherwise def.
// A def of 0 (or any non-positive value) means the documented default of 1.0.
func EffectiveMaximum(p Percent, def float64) float64 {
	if p.Maximum != nil {
		return *p.Maximum
	}
	if def > 0 {
		return def
	}
	return 1
}
