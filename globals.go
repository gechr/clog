package clog

import (
	"time"

	"github.com/gechr/clog/field/duration"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/field/hyperlink"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/field/quantity"
)

// SetDurationFormatFunc sets the duration format function for all loggers (process-global).
// It applies to both [Event.Duration] fields and [Event.Elapsed] fields.
// For elapsed fields, [SetElapsedFormatFunc] takes priority when both are set.
// Delegates to [duration.SetFormatFunc].
func SetDurationFormatFunc(fn func(time.Duration) string) { duration.SetFormatFunc(fn) }

// SetElapsedFormatFunc sets the elapsed format function for all loggers (process-global).
// Delegates to [elapsed.SetFormatFunc].
func SetElapsedFormatFunc(fn func(time.Duration) string) { elapsed.SetFormatFunc(fn) }

// SetDurationGradientMax sets the max duration for Duration field gradient coloring (process-global).
// The gradient maps 0 → max onto the configured color stops; 0 disables the gradient.
// Delegates to [duration.SetGradientMax].
func SetDurationGradientMax(d time.Duration) { duration.SetGradientMax(d) }

// SetElapsedGradientMax sets the max duration for elapsed gradient coloring (process-global).
// The gradient maps 0 → max onto the configured color stops; 0 disables the gradient.
// Delegates to [elapsed.SetGradientMax].
func SetElapsedGradientMax(d time.Duration) { elapsed.SetGradientMax(d) }

// SetElapsedMinimum sets the elapsed minimum threshold for all loggers (process-global).
// Delegates to [elapsed.SetMinimum].
func SetElapsedMinimum(d time.Duration) { elapsed.SetMinimum(d) }

// SetElapsedPrecision sets the elapsed precision for all loggers (process-global).
// Delegates to [elapsed.SetPrecision].
func SetElapsedPrecision(precision int) { elapsed.SetPrecision(precision) }

// SetElapsedRound sets the elapsed rounding granularity for all loggers (process-global).
// Delegates to [elapsed.SetRound].
func SetElapsedRound(d time.Duration) { elapsed.SetRound(d) }

// SetHyperlinkColumnFormat configures the URL format for file+line+column hyperlinks (process-global).
// Accepts a full format string or a preset name (e.g. "vscode").
// Delegates to [hyperlink.SetColumnFormat].
func SetHyperlinkColumnFormat(format string) { hyperlink.SetColumnFormat(format) }

// SetHyperlinkDirFormat configures the URL format for directory hyperlinks (process-global).
// Delegates to [hyperlink.SetDirFormat].
func SetHyperlinkDirFormat(format string) { hyperlink.SetDirFormat(format) }

// SetHyperlinkEnabled enables or disables all hyperlink rendering (process-global).
// Delegates to [hyperlink.SetEnabled].
func SetHyperlinkEnabled(enabled bool) { hyperlink.SetEnabled(enabled) }

// SetHyperlinkFileFormat configures the URL format for file-only hyperlinks (process-global).
// Delegates to [hyperlink.SetFileFormat].
func SetHyperlinkFileFormat(format string) { hyperlink.SetFileFormat(format) }

// SetHyperlinkLineFormat configures the URL format for file+line hyperlinks (process-global).
// Accepts a full format string or a preset name (e.g. "vscode").
// Delegates to [hyperlink.SetLineFormat].
func SetHyperlinkLineFormat(format string) { hyperlink.SetLineFormat(format) }

// SetHyperlinkPathFormat configures the generic fallback URL format for any path (process-global).
// Accepts a full format string or a preset name.
// Delegates to [hyperlink.SetPathFormat].
func SetHyperlinkPathFormat(format string) { hyperlink.SetPathFormat(format) }

// SetHyperlinkPreset configures all hyperlink format slots using a named preset (process-global).
// Known presets: cursor, kitty, macvim, subl, textmate, vscode, vscode-insiders, vscodium.
// Delegates to [hyperlink.SetPreset].
func SetHyperlinkPreset(name string) error { return hyperlink.SetPreset(name) }

// SetPercentFormatFunc sets the percent format function for all loggers (process-global).
// Delegates to [percent.SetFormatFunc].
func SetPercentFormatFunc(fn func(float64) string) { percent.SetFormatFunc(fn) }

// SetPercentPrecision sets the percent precision for all loggers (process-global).
// Delegates to [percent.SetPrecision].
func SetPercentPrecision(precision int) { percent.SetPrecision(precision) }

// SetPercentReverseGradient sets the percent gradient inversion for all loggers (process-global).
// Delegates to [percent.SetReverseGradient].
func SetPercentReverseGradient(reverse bool) { percent.SetReverseGradient(reverse) }

// SetPercentMaximum sets the percent maximum for all loggers (process-global).
// The default is 1.0, meaning percent values are passed as fractions
// (e.g. 0.75 → "75%"). Set to 100 for 0–100 input.
// Delegates to [percent.SetMaximum].
func SetPercentMaximum(m float64) { percent.SetMaximum(m) }

// SetQuantityUnitsIgnoreCase sets case-insensitive quantity unit matching for all loggers (process-global).
// Delegates to [quantity.SetUnitsIgnoreCase].
func SetQuantityUnitsIgnoreCase(ignoreCase bool) { quantity.SetUnitsIgnoreCase(ignoreCase) }
