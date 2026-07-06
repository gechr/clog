package core

import (
	"time"

	"github.com/gechr/clog/style"
)

// RawJSON wraps pre-serialized JSON bytes so formatValue can emit them
// verbatim without quoting or escaping.
type RawJSON []byte

// QuantityField wraps a string value with numeric and unit segments (e.g.
// "5m", "5.1km", "100MB") so formatValue can identify it for quantity styling.
type QuantityField string

// ElapsedField wraps a time.Duration so formatValue can identify it for
// elapsed-time styling. GradientMax, Gradient, and GradientMode override the
// logger's elapsed gradient settings for this field when non-nil/non-empty.
type ElapsedField struct {
	Value        time.Duration
	GradientMax  *time.Duration
	Gradient     []style.ColorStop
	GradientMode *style.GradientMode
}

// DurationField wraps a time.Duration so formatValue can identify it for
// duration styling. GradientMax, Gradient, and GradientMode override the
// logger's duration gradient settings for this field when non-nil/non-empty.
type DurationField struct {
	Value        time.Duration
	GradientMax  *time.Duration
	Gradient     []style.ColorStop
	GradientMode *style.GradientMode
}

// Percent holds a percentage value (0–100) with an optional reverse
// gradient flag. When Reverse is true, the gradient is flipped relative
// to the logger's default direction. Maximum overrides the global maximum
// for this field when non-nil.
type Percent struct {
	Value   float64
	Reverse bool
	Maximum *float64
}

// Fraction holds a current/total pair rendered as "current/total" with
// gradient color styling based on current/total progress. When Reverse
// is true, the gradient is flipped (useful for countdowns). Format overrides
// the logger's numeric formatting for this field when non-nil.
type Fraction struct {
	Format  *NumberFormat
	Current int
	Total   int
	Reverse bool
}
