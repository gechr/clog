package clog

import (
	"time"

	"github.com/gechr/clog/field/hyperlink"
)

// FieldFormats holds all field-formatting configuration for a [Logger].
// Set it with [Logger.SetFieldFormats]; read the current snapshot with
// [Logger.FieldFormats]. Zero-value fields mean the documented default
// where one exists.
type FieldFormats struct {
	// DurationFormat overrides the formatter for [time.Duration] fields.
	// It also applies to elapsed fields when ElapsedFormat is nil.
	// nil means the built-in format.
	DurationFormat func(time.Duration) string
	// DurationGradientMax is the duration mapped to the end of the
	// Duration gradient stops. 0 disables the gradient.
	DurationGradientMax time.Duration
	// ElapsedFormat overrides the formatter for elapsed-time fields.
	// nil means DurationFormat, then the built-in format.
	ElapsedFormat func(time.Duration) string
	// ElapsedGradientMax is the duration mapped to the end of the
	// Elapsed gradient stops. 0 disables the gradient.
	ElapsedGradientMax time.Duration
	// ElapsedMinimum hides elapsed fields below this duration.
	// The default is [time.Second]; 0 shows all values.
	ElapsedMinimum time.Duration
	// ElapsedPrecision is the decimal precision for elapsed display
	// (0 = "3s", 1 = "3.2s").
	ElapsedPrecision int
	// ElapsedRound is the rounding granularity for elapsed values.
	// The default is [time.Second]; 0 disables rounding.
	ElapsedRound time.Duration

	// HyperlinkEnabled controls whether hyperlinks are rendered at all.
	HyperlinkEnabled bool
	// HyperlinkColumnFormat is the URL format for file+line+column links.
	// Accepts a full format string or a preset name (e.g. "vscode").
	HyperlinkColumnFormat string
	// HyperlinkDirFormat is the URL format for directory links.
	HyperlinkDirFormat string
	// HyperlinkFileFormat is the URL format for file-only links.
	HyperlinkFileFormat string
	// HyperlinkLineFormat is the URL format for file+line links.
	HyperlinkLineFormat string
	// HyperlinkPathFormat is the generic fallback URL format for any path.
	HyperlinkPathFormat string

	// PercentFormat overrides the formatter for percent fields.
	// It receives the display value (already scaled to 0–100).
	// nil means the built-in format.
	PercentFormat func(float64) string
	// PercentMaximum is the input range maximum for percent values.
	// 0 means the default of 1.0 (fractions); set 100 for 0–100 input.
	PercentMaximum float64
	// PercentPrecision is the decimal precision for percent display
	// (0 = "75%", 1 = "75.0%").
	PercentPrecision int
	// PercentReverseGradient flips the percent gradient direction
	// (green at 0%, red at 100%) for usage-style metrics.
	PercentReverseGradient bool

	// QuantityUnitsIgnoreCase enables case-insensitive quantity unit
	// matching. Note the default via [DefaultFieldFormats] is true.
	QuantityUnitsIgnoreCase bool
}

// DefaultFieldFormats returns the default field-format configuration:
// hyperlinks enabled, elapsed rounded to whole seconds and hidden below one
// second, case-insensitive quantity units, and built-in formatters.
func DefaultFieldFormats() FieldFormats {
	return FieldFormats{
		ElapsedMinimum:          time.Second,
		ElapsedRound:            time.Second,
		HyperlinkEnabled:        true,
		QuantityUnitsIgnoreCase: true,
	}
}

// defaultFieldFormats is the shared immutable default used when a
// formatFieldsOpts carries no explicit formats (e.g. zero-value opts in
// tests).
var defaultFieldFormats = DefaultFieldFormats()

// SetFieldFormats replaces the logger's field-format configuration
// (replace-all semantics, like [Logger.SetParts]). Hyperlink format fields
// accept preset names (e.g. "vscode"), which are expanded on store.
//
//	f := clog.DefaultFieldFormats()
//	f.PercentPrecision = 1
//	logger.SetFieldFormats(f)
func (l *Logger) SetFieldFormats(f FieldFormats) {
	f.HyperlinkPathFormat = hyperlink.Expand(f.HyperlinkPathFormat, "path")
	f.HyperlinkFileFormat = hyperlink.Expand(f.HyperlinkFileFormat, "path")
	f.HyperlinkDirFormat = hyperlink.Expand(f.HyperlinkDirFormat, "path")
	f.HyperlinkLineFormat = hyperlink.Expand(f.HyperlinkLineFormat, "line")
	f.HyperlinkColumnFormat = hyperlink.Expand(f.HyperlinkColumnFormat, "column")
	l.fieldFormats.Store(&f)

	// Hyperlink rendering happens on the Output (it is part of the
	// fx.Output seam and has no logger access), so push the hyperlink
	// subset down to the logger's output.
	l.mu.Lock()
	out := l.output
	l.mu.Unlock()
	out.setHyperlinks(f.hyperlinkConfig())
}

// FieldFormats returns a copy of the logger's current field-format
// configuration.
func (l *Logger) FieldFormats() FieldFormats {
	return *l.loadFieldFormats()
}

// loadFieldFormats returns the logger's current immutable formats snapshot,
// falling back to the package default when none has been set.
func (l *Logger) loadFieldFormats() *FieldFormats {
	if f := l.fieldFormats.Load(); f != nil {
		return f
	}
	return &defaultFieldFormats
}

// hyperlinkConfig derives the hyperlink rendering subset of f.
func (f *FieldFormats) hyperlinkConfig() hyperlink.Config {
	return hyperlink.Config{
		Enabled:      f.HyperlinkEnabled,
		PathFormat:   f.HyperlinkPathFormat,
		FileFormat:   f.HyperlinkFileFormat,
		DirFormat:    f.HyperlinkDirFormat,
		LineFormat:   f.HyperlinkLineFormat,
		ColumnFormat: f.HyperlinkColumnFormat,
	}
}
