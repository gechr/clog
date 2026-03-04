// Package percent provides options for percent field rendering in clog.
package percent

import (
	"sync/atomic"

	"github.com/gechr/clog/internal/core"
)

// formatFunc holds the global custom format function.
var formatFunc atomic.Pointer[func(float64) string]

// reverse controls whether the gradient is inverted.
var reverse atomic.Bool

// precision holds the decimal precision for formatting.
var precision atomic.Int32

// scale holds the percent scale. A nil pointer means the default (1.0).
var scale atomic.Pointer[float64]

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

// Apply applies the given options to p.
func Apply(p *Percent, opts []Option) {
	for _, o := range opts {
		o(p)
	}
}

// SetFormatFunc configures a custom format function for percent fields.
// When set to nil (the default), the built-in format is used.
func SetFormatFunc(fn func(float64) string) {
	if fn == nil {
		formatFunc.Store(nil)
	} else {
		formatFunc.Store(&fn)
	}
}

// SetReverseGradient reverses the gradient direction for percent fields.
// By default the gradient runs red (0%) → green (100%) - suitable for
// metrics where higher is better. Set reverse=true to flip it to
// green (0%) → red (100%) - suitable for metrics like CPU or disk usage
// where lower is better.
func SetReverseGradient(rev bool) {
	reverse.Store(rev)
}

// SetPrecision sets the number of decimal places for percent display.
// For example, 0 = "75%", 1 = "75.0%". Defaults to 0.
func SetPrecision(n int) {
	precision.Store(int32(n)) //nolint:gosec // precision is a small positive integer
}

// SetScale sets the percent scale. The default is 1.0, meaning percent
// values are passed as fractions (e.g. 0.75 → "75%"). Set to 100 for
// legacy 0–100 input (e.g. 75 → "75%").
func SetScale(s float64) {
	scale.Store(&s)
}

// Scale returns the current percent scale (default 1.0).
func Scale() float64 {
	if p := scale.Load(); p != nil {
		return *p
	}
	return 1
}

// FormatFunc returns the current custom format function, or nil if using default.
func FormatFunc() func(float64) string {
	p := formatFunc.Load()
	if p == nil {
		return nil
	}
	return *p
}

// ReverseGradient reports whether the gradient is inverted.
func ReverseGradient() bool {
	return reverse.Load()
}

// Precision returns the current decimal precision.
func Precision() int {
	return int(precision.Load())
}

// Snapshot captures the current state of all percent configuration.
// Use [Restore] to reset the state in test cleanup.
type Snapshot struct {
	formatFunc *func(float64) string
	reverse    bool
	precision  int32
	scale      *float64
}

// Save captures the current percent configuration so it can be
// restored later with [Restore]. Typical usage in tests:
//
//	snap := percent.Save()
//	t.Cleanup(func() { percent.Restore(snap) })
func Save() Snapshot {
	return Snapshot{
		formatFunc: formatFunc.Load(),
		reverse:    reverse.Load(),
		precision:  precision.Load(),
		scale:      scale.Load(),
	}
}

// Restore resets the percent configuration to a previously saved [Snapshot].
func Restore(s Snapshot) {
	formatFunc.Store(s.formatFunc)
	reverse.Store(s.reverse)
	precision.Store(s.precision)
	scale.Store(s.scale)
}
