package clog

import (
	"time"

	"github.com/gechr/clog/field/hyperlink"
)

// This file provides per-field convenience setters over [FieldFormats], so
// callers can tweak a single formatting option without the
// read-modify-write dance of [Logger.FieldFormats] + [Logger.SetFieldFormats].
// Each setter applies to one field and preserves the rest of the snapshot.

// mutateHyperlinks applies fn to a copy of the current formats snapshot,
// stores it, and pushes the hyperlink subset down to the logger's output
// (hyperlink rendering lives on the Output, which has no logger access).
func (l *Logger) mutateHyperlinks(fn func(*FieldFormats)) {
	f := *l.loadFieldFormats()
	fn(&f)
	l.fieldFormats.Store(&f)

	l.mu.Lock()
	out := l.output
	l.mu.Unlock()
	out.setHyperlinks(f.hyperlinkConfig())
}

// SetDurationFormat sets a custom formatter for [time.Duration] fields (also
// used for elapsed fields when no elapsed formatter is set). nil restores the
// built-in format.
func (l *Logger) SetDurationFormat(format func(time.Duration) string) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.DurationFormat = format })
}

// SetDurationGradientMax sets the duration mapped to the end of the duration
// gradient. 0 disables the gradient.
func (l *Logger) SetDurationGradientMax(maximum time.Duration) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.DurationGradientMax = maximum })
}

// SetDurationMinimum hides duration fields below this duration. 0 shows all
// values and is the default.
func (l *Logger) SetDurationMinimum(minimum time.Duration) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.DurationMinimum = minimum })
}

// SetDurationScale sets the magnitude-keyed rounding and precision scale for
// duration fields. nil inherits [FieldFormats.TimeScale]; a non-nil empty
// scale disables rounding and decimal display. See [TimeScale].
func (l *Logger) SetDurationScale(scale TimeScale) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.DurationScale = scale })
}

// SetElapsedFormat sets a custom formatter for elapsed-time fields (takes
// priority over the duration formatter). nil restores the built-in format.
func (l *Logger) SetElapsedFormat(format func(time.Duration) string) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.ElapsedFormat = format })
}

// SetElapsedGradientMax sets the duration mapped to the end of the elapsed
// gradient. 0 disables the gradient.
func (l *Logger) SetElapsedGradientMax(maximum time.Duration) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.ElapsedGradientMax = maximum })
}

// SetElapsedMinimum hides elapsed fields below this duration. 0 shows all
// values. Defaults to [time.Second].
func (l *Logger) SetElapsedMinimum(minimum time.Duration) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.ElapsedMinimum = minimum })
}

// SetElapsedScale sets the magnitude-keyed rounding and precision scale for
// elapsed (and deadline) fields. nil inherits [FieldFormats.TimeScale]; a
// non-nil empty scale disables rounding and decimal display. See [TimeScale].
func (l *Logger) SetElapsedScale(scale TimeScale) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.ElapsedScale = scale })
}

// SetTimeScale sets the shared time scale and clears the duration and elapsed
// overrides so all time fields inherit it. nil disables scale-based rounding
// and decimal display for all time fields.
func (l *Logger) SetTimeScale(scale TimeScale) {
	l.mutateFieldFormats(func(f *FieldFormats) {
		f.TimeScale = scale
		f.DurationScale = nil
		f.ElapsedScale = nil
	})
}

// SetTimeGradientMax sets both [FieldFormats.DurationGradientMax] and
// [FieldFormats.ElapsedGradientMax] to max in one call, since duration and
// elapsed fields are usually given the same gradient ceiling. 0 disables both
// gradients.
func (l *Logger) SetTimeGradientMax(maximum time.Duration) {
	l.mutateFieldFormats(func(f *FieldFormats) {
		f.DurationGradientMax = maximum
		f.ElapsedGradientMax = maximum
	})
}

// SetHyperlinkEnabled enables or disables all hyperlink rendering.
func (l *Logger) SetHyperlinkEnabled(enabled bool) {
	l.mutateHyperlinks(func(f *FieldFormats) { f.HyperlinkEnabled = enabled })
}

// SetHyperlinkColumnFormat sets the URL format for file+line+column links.
// Accepts a full format string or a preset name (e.g. "vscode").
func (l *Logger) SetHyperlinkColumnFormat(format string) {
	l.mutateHyperlinks(func(f *FieldFormats) {
		f.HyperlinkColumnFormat = hyperlink.Expand(format, "column")
	})
}

// SetHyperlinkDirFormat sets the URL format for directory links.
// Accepts a full format string or a preset name (e.g. "vscode").
func (l *Logger) SetHyperlinkDirFormat(format string) {
	l.mutateHyperlinks(func(f *FieldFormats) {
		f.HyperlinkDirFormat = hyperlink.Expand(format, "path")
	})
}

// SetHyperlinkFileFormat sets the URL format for file-only links.
// Accepts a full format string or a preset name (e.g. "vscode").
func (l *Logger) SetHyperlinkFileFormat(format string) {
	l.mutateHyperlinks(func(f *FieldFormats) {
		f.HyperlinkFileFormat = hyperlink.Expand(format, "path")
	})
}

// SetHyperlinkLineFormat sets the URL format for file+line links.
// Accepts a full format string or a preset name (e.g. "vscode").
func (l *Logger) SetHyperlinkLineFormat(format string) {
	l.mutateHyperlinks(func(f *FieldFormats) {
		f.HyperlinkLineFormat = hyperlink.Expand(format, "line")
	})
}

// SetHyperlinkPathFormat sets the generic fallback URL format for any path.
// Accepts a full format string or a preset name (e.g. "vscode").
func (l *Logger) SetHyperlinkPathFormat(format string) {
	l.mutateHyperlinks(func(f *FieldFormats) {
		f.HyperlinkPathFormat = hyperlink.Expand(format, "path")
	})
}

// SetPercentFormat sets a custom formatter for percent fields; it receives the
// display value already scaled to 0–100. nil restores the built-in format.
func (l *Logger) SetPercentFormat(format func(float64) string) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.PercentFormat = format })
}

// SetPercentMaximum sets the input range maximum for percent values. 0 means
// the default of 1.0 (fractions); set 100 for 0–100 input.
func (l *Logger) SetPercentMaximum(maximum float64) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.PercentMaximum = maximum })
}

// SetPercentPrecision sets the decimal precision for percent display
// (0 = "75%", 1 = "75.0%").
func (l *Logger) SetPercentPrecision(precision int) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.PercentPrecision = precision })
}

// SetPercentReverseGradient flips the percent gradient direction (green at 0%,
// red at 100%) for usage-style metrics.
func (l *Logger) SetPercentReverseGradient(reverse bool) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.PercentReverseGradient = reverse })
}

// SetQuantityUnitsIgnoreCase enables or disables case-insensitive quantity
// unit matching.
func (l *Logger) SetQuantityUnitsIgnoreCase(ignore bool) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.QuantityUnitsIgnoreCase = ignore })
}
