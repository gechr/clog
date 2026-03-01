// Package quantity provides quantity field configuration for clog.
package quantity

import "sync/atomic"

// unitsIgnoreCase controls whether unit matching is case-insensitive.
var unitsIgnoreCase atomic.Bool

func init() {
	unitsIgnoreCase.Store(true) // default: case-insensitive
}

// SetUnitsIgnoreCase enables or disables case-insensitive unit matching.
// Defaults to true.
func SetUnitsIgnoreCase(enabled bool) {
	unitsIgnoreCase.Store(enabled)
}

// UnitsIgnoreCase reports whether unit matching is case-insensitive.
func UnitsIgnoreCase() bool {
	return unitsIgnoreCase.Load()
}
