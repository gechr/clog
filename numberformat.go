package clog

import "github.com/gechr/clog/internal/core"

// NumberFormat controls how numeric field values (plain integers and both
// halves of a fraction) are rendered. Configure it per-logger with
// [Logger.SetNumberFormat] (all numbers) or [Logger.SetFractionFormat]
// (fractions only, falling back to the number format when unset).
type NumberFormat = core.NumberFormat

const (
	// NumberPlain renders numbers verbatim with no grouping (the default),
	// e.g. "1234567".
	NumberPlain = core.NumberPlain
	// NumberGrouped inserts a separator every three digits, e.g.
	// "1,234,567". Set the separator with [Logger.SetNumberGroupSeparator].
	NumberGrouped = core.NumberGrouped
	// NumberCompact renders an abbreviated form using K/M/B/T suffixes,
	// e.g. "1.2M". Values whose magnitude is below the minimum set with
	// [Logger.SetNumberCompactMinimum] render verbatim.
	NumberCompact = core.NumberCompact
)
