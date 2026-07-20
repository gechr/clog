// Package deadline provides options for countdown deadline field rendering in clog.
package deadline

import (
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
)

// Deadline is the value type for countdown deadline fields.
type Deadline = core.DeadlineField

// TimeScale is a magnitude-keyed rounding and precision scale.
type TimeScale = core.TimeScale

// Option configures how a deadline field is rendered.
type Option func(*Deadline)

// WithGradient returns an [Option] that overrides the logger's deadline
// gradient color stops for this field. The gradient runs from the first stop
// on a fresh deadline to the last stop at expiry.
//
//	clog.Spinner("Waiting for confirmation").Deadline("timeout", 15*time.Second,
//	    deadline.WithGradient(
//	        style.ColorStop{Position: 0, Color: green},
//	        style.ColorStop{Position: 1, Color: red},
//	    ))
func WithGradient(stops ...style.ColorStop) Option {
	return func(d *Deadline) {
		d.Gradient = stops
	}
}

// WithGradientMode returns an [Option] that overrides the logger's deadline
// gradient transition mode ([style.GradientFade] or [style.GradientStep])
// for this field.
func WithGradientMode(mode style.GradientMode) Option {
	return func(d *Deadline) {
		d.GradientMode = &mode
	}
}

// WithRound returns an [Option] that overrides the logger's elapsed rounding
// granularity for this field's countdown. 0 disables rounding for this field.
func WithRound(round time.Duration) Option {
	return func(d *Deadline) {
		d.Round = &round
	}
}

// WithScale returns an [Option] that overrides the logger's elapsed scale for
// this deadline field. An empty scale uses the logger's scalar elapsed
// settings.
func WithScale(scale TimeScale) Option {
	return func(d *Deadline) {
		d.Scale = scale
	}
}

// WithOmitOnDone returns an [Option] that controls whether an animated
// deadline is omitted from the fx.Builder done row. Builder deadlines default
// to true because a final remaining-time value is usually stale or misleading.
func WithOmitOnDone(omit bool) Option {
	return func(d *Deadline) {
		d.OmitOnDone = omit
	}
}

// WithTrailing returns an [Option] that renders the deadline field last on
// the row, after any fields added at runtime via a live Update. Use when the
// deadline is a persistent countdown that should always trail the row's own
// attrs.
func WithTrailing() Option {
	return func(d *Deadline) {
		d.Trailing = true
	}
}

// Apply applies the given options to d.
func Apply(d *Deadline, opts ...Option) {
	for _, o := range opts {
		o(d)
	}
}
