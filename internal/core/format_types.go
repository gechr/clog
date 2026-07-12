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
// Minimum overrides the logger's [FieldFormats.ElapsedMinimum] threshold and
// Round the logger's [FieldFormats.ElapsedRound] granularity for this field
// when non-nil. OmitOnDone removes the field from fx.Builder done
// rows while leaving the live animation field visible. Trailing
// pins the field to the end of the row when fx.Builder.ResolveDynamicFields
// reorders fields for animated rows.
type ElapsedField struct {
	Value        time.Duration
	GradientMax  *time.Duration
	Gradient     []style.ColorStop
	GradientMode *style.GradientMode
	Minimum      *time.Duration
	Round        *time.Duration
	OmitOnDone   bool
	Trailing     bool
}

// DeadlineField wraps a countdown so formatValue can identify it for
// deadline styling. Remaining is the displayed value (counting down from From
// to 0); coloring is based on the consumed time (From - Remaining) against an
// implicit gradient maximum of From, so a fresh deadline uses the gradient's
// first stop and an expired one uses the last. Gradient and GradientMode
// override the logger's elapsed gradient settings for this field when
// non-nil/non-empty. Round overrides the logger's [FieldFormats.ElapsedRound]
// granularity for this field when non-nil.
// OmitOnDone removes the field from fx.Builder done rows while leaving
// the live animation countdown visible. Trailing pins the field to the end of
// the row when fx.Builder.ResolveDynamicFields reorders fields for animated
// rows.
type DeadlineField struct {
	Remaining    time.Duration
	From         time.Duration
	Gradient     []style.ColorStop
	GradientMode *style.GradientMode
	Round        *time.Duration
	OmitOnDone   bool
	Trailing     bool
}

// DurationField wraps a time.Duration so formatValue can identify it for
// duration styling. GradientMax, Gradient, and GradientMode override the
// logger's duration gradient settings for this field when non-nil/non-empty.
// Minimum overrides the logger's [FieldFormats.DurationMinimum] threshold and
// Round the logger's [FieldFormats.DurationRound] granularity for this field
// when non-nil. OmitOnDone removes the field from fx.Builder done
// rows while leaving the live animation field visible.
type DurationField struct {
	Value        time.Duration
	GradientMax  *time.Duration
	Gradient     []style.ColorStop
	GradientMode *style.GradientMode
	Minimum      *time.Duration
	Round        *time.Duration
	OmitOnDone   bool
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
