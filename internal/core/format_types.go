package core

import "time"

// RawJSON wraps pre-serialized JSON bytes so formatValue can emit them
// verbatim without quoting or escaping.
type RawJSON []byte

// QuantityField wraps a string value with numeric and unit segments (e.g.
// "5m", "5.1km", "100MB") so formatValue can identify it for quantity styling.
type QuantityField string

// ElapsedField wraps a time.Duration so formatValue can identify it
// for elapsed-time styling.
type ElapsedField time.Duration

// Percent holds a percentage value (0–100) with an optional reverse
// gradient flag. When Reverse is true, the gradient is flipped relative
// to the logger's default direction.
type Percent struct {
	Value   float64
	Reverse bool
}
