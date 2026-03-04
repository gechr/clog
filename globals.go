package clog

import (
	"time"

	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/field/quantity"
)

// SetElapsedGradientMax sets the max duration for elapsed gradient coloring (process-global).
// The gradient maps 0 → max onto the configured color stops; 0 disables the gradient.
// Delegates to [elapsed.SetGradientMax].
func SetElapsedGradientMax(d time.Duration) { elapsed.SetGradientMax(d) }

// SetElapsedFormatFunc sets the elapsed format function for all loggers (process-global).
// Delegates to [elapsed.SetFormatFunc].
func SetElapsedFormatFunc(fn func(time.Duration) string) { elapsed.SetFormatFunc(fn) }

// SetElapsedMinimum sets the elapsed minimum threshold for all loggers (process-global).
// Delegates to [elapsed.SetMinimum].
func SetElapsedMinimum(d time.Duration) { elapsed.SetMinimum(d) }

// SetElapsedPrecision sets the elapsed precision for all loggers (process-global).
// Delegates to [elapsed.SetPrecision].
func SetElapsedPrecision(precision int) { elapsed.SetPrecision(precision) }

// SetElapsedRound sets the elapsed rounding granularity for all loggers (process-global).
// Delegates to [elapsed.SetRound].
func SetElapsedRound(d time.Duration) { elapsed.SetRound(d) }

// SetPercentFormatFunc sets the percent format function for all loggers (process-global).
// Delegates to [percent.SetFormatFunc].
func SetPercentFormatFunc(fn func(float64) string) { percent.SetFormatFunc(fn) }

// SetPercentReverseGradient sets the percent gradient inversion for all loggers (process-global).
// Delegates to [percent.SetReverseGradient].
func SetPercentReverseGradient(reverse bool) { percent.SetReverseGradient(reverse) }

// SetPercentPrecision sets the percent precision for all loggers (process-global).
// Delegates to [percent.SetPrecision].
func SetPercentPrecision(precision int) { percent.SetPrecision(precision) }

// SetPercentScale sets the percent scale for all loggers (process-global).
// The default is 1.0, meaning percent values are passed as fractions
// (e.g. 0.75 → "75%"). Set to 100 for legacy 0–100 input.
// Delegates to [percent.SetScale].
func SetPercentScale(s float64) { percent.SetScale(s) }

// SetQuantityUnitsIgnoreCase sets case-insensitive quantity unit matching for all loggers (process-global).
// Delegates to [quantity.SetUnitsIgnoreCase].
func SetQuantityUnitsIgnoreCase(ignoreCase bool) { quantity.SetUnitsIgnoreCase(ignoreCase) }
