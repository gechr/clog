package clog

import (
	"time"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/gechr/clog/internal/core"
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
	// DurationMinimum hides duration fields below this duration.
	// The default is 0, which shows all values.
	DurationMinimum time.Duration
	// DurationPrecision is the decimal precision for duration display
	// (0 = "3s", 1 = "3.2s"). Ignored when a duration scale applies.
	DurationPrecision int
	// DurationRound is the rounding granularity for duration values.
	// The default is [time.Second]; 0 disables rounding. Ignored when a
	// duration scale applies.
	DurationRound time.Duration
	// DurationScale overrides TimeScale for duration fields. nil inherits
	// TimeScale; a non-nil empty scale uses DurationRound and
	// DurationPrecision instead.
	DurationScale TimeScale
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
	// (0 = "3s", 1 = "3.2s"). Ignored when an elapsed scale applies.
	ElapsedPrecision int
	// ElapsedRound is the rounding granularity for elapsed values.
	// The default is [time.Second]; 0 disables rounding. Ignored when an
	// elapsed scale applies.
	ElapsedRound time.Duration
	// ElapsedScale overrides TimeScale for elapsed and deadline fields. nil
	// inherits TimeScale; a non-nil empty scale uses ElapsedRound and
	// ElapsedPrecision instead. The default is an empty scale so live fields
	// keep a stable whole-second width.
	ElapsedScale TimeScale
	// TimeScale is the shared magnitude-keyed rounding and precision scale for
	// time fields. DurationScale and ElapsedScale can override its inheritance.
	TimeScale TimeScale

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

	// NumberFormat selects how integer fields and both halves of fraction
	// fields are rendered. The zero value is [NumberPlain].
	NumberFormat NumberFormat
	// FractionFormat overrides NumberFormat for fraction fields only.
	// nil means fractions inherit NumberFormat.
	FractionFormat *NumberFormat
	// NumberGroupSeparator is the separator inserted between digit groups
	// for [NumberGrouped]. The default via [DefaultFieldFormats] is ",".
	NumberGroupSeparator string
	// NumberCompactMinimum is the smallest magnitude that [NumberCompact]
	// abbreviates; values below it render using NumberCompactFallback. The
	// default via [DefaultFieldFormats] is 1000.
	NumberCompactMinimum int64
	// NumberCompactFallback selects how [NumberCompact] renders values below
	// NumberCompactMinimum. Only [NumberPlain] and [NumberGrouped] are
	// meaningful; [NumberCompact] is treated as [NumberPlain]. The default
	// via [DefaultFieldFormats] is [NumberGrouped] (e.g. "9,999" before
	// "10K").
	NumberCompactFallback NumberFormat
}

const (
	defaultTimeScaleDecimalMaximum = 10 * time.Second
	defaultTimeScaleDecimalRound   = 100 * time.Millisecond
)

// DefaultFieldFormats returns the default field-format configuration:
// durations shown at millisecond resolution below one second, at up to one
// decimal place below ten seconds, and at whole-second resolution thereafter;
// live elapsed and deadline fields kept at stable whole-second resolution;
// hyperlinks enabled; case-insensitive quantity units; plain numbers with a
// "," group separator and a 1000 compact minimum; and built-in formatters.
func DefaultFieldFormats() FieldFormats {
	return FieldFormats{
		DurationMinimum:         0,
		DurationRound:           time.Second,
		ElapsedMinimum:          time.Second,
		ElapsedRound:            time.Second,
		ElapsedScale:            TimeScale{},
		HyperlinkEnabled:        true,
		NumberCompactFallback:   NumberGrouped,
		NumberCompactMinimum:    1000, //nolint:mnd // default compact threshold
		NumberGroupSeparator:    ",",
		QuantityUnitsIgnoreCase: true,
		TimeScale: TimeScale{
			{Below: time.Second, Round: time.Millisecond},
			{
				Below:     defaultTimeScaleDecimalMaximum,
				Precision: 1,
				Round:     defaultTimeScaleDecimalRound,
				Trim:      true,
			},
			{Round: time.Second},
		},
	}
}

// TimeScale maps time-field magnitudes to rounding granularity and decimal
// precision. Steps must be ordered by ascending Below. The first step whose
// exclusive Below bound exceeds the absolute value applies; a zero Below is a
// catch-all and should be last.
//
//	clog.TimeScale{
//		{Below: time.Second, Round: time.Millisecond},
//		{Below: 10 * time.Second, Precision: 1, Round: 100 * time.Millisecond, Trim: true},
//		{Round: time.Second},
//	}
type TimeScale = core.TimeScale

// TimeScaleStep is one magnitude bracket of a [TimeScale].
type TimeScaleStep = core.TimeScaleStep

// resolveScale resolves a field-specific scale through the shared TimeScale.
// nil inherits; a non-nil empty scale deliberately falls back to scalars.
func (f *FieldFormats) resolveScale(d time.Duration, specific TimeScale) (TimeScaleStep, bool) {
	if specific != nil {
		return specific.Resolve(d)
	}
	return f.TimeScale.Resolve(d)
}

// resolveRound resolves the rounding granularity for a raw duration value: a
// per-field override wins, then the per-call scale bracket, then the
// field-specific scale (falling back to the shared TimeScale), then the
// scalar fallback.
func (f *FieldFormats) resolveRound(
	raw time.Duration,
	override *time.Duration,
	scale, fieldScale TimeScale,
	scalar time.Duration,
) time.Duration {
	if override != nil {
		return *override
	}
	if scale != nil {
		if step, ok := scale.Resolve(raw); ok {
			return step.Round
		}
		return scalar
	}
	if step, ok := f.resolveScale(raw, fieldScale); ok {
		return step.Round
	}
	return scalar
}

// resolveDisplay resolves the precision and trim settings for a duration
// value: the per-call scale bracket wins, then the field-specific scale
// (falling back to the shared TimeScale), then the scalar precision.
func (f *FieldFormats) resolveDisplay(
	d time.Duration,
	scale, fieldScale TimeScale,
	scalarPrecision int,
) (int, bool) {
	if scale != nil {
		if step, ok := scale.Resolve(d); ok {
			return step.Precision, step.Trim
		}
		return scalarPrecision, false
	}
	if step, ok := f.resolveScale(d, fieldScale); ok {
		return step.Precision, step.Trim
	}
	return scalarPrecision, false
}

// durationRound resolves the rounding granularity for a raw duration value: a
// per-field override wins, then the DurationScale bracket, then DurationRound.
func (f *FieldFormats) durationRound(
	raw time.Duration,
	override *time.Duration,
	scale TimeScale,
) time.Duration {
	return f.resolveRound(raw, override, scale, f.DurationScale, f.DurationRound)
}

// durationDisplay resolves display settings for a duration value.
func (f *FieldFormats) durationDisplay(d time.Duration, scale TimeScale) (int, bool) {
	return f.resolveDisplay(d, scale, f.DurationScale, f.DurationPrecision)
}

// elapsedRound mirrors [FieldFormats.durationRound] for elapsed and deadline
// fields, drawing on ElapsedScale then ElapsedRound.
func (f *FieldFormats) elapsedRound(
	raw time.Duration,
	override *time.Duration,
	scale TimeScale,
) time.Duration {
	return f.resolveRound(raw, override, scale, f.ElapsedScale, f.ElapsedRound)
}

// elapsedDisplay mirrors [FieldFormats.durationDisplay] for elapsed and
// deadline fields.
func (f *FieldFormats) elapsedDisplay(d time.Duration, scale TimeScale) (int, bool) {
	return f.resolveDisplay(d, scale, f.ElapsedScale, f.ElapsedPrecision)
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

// mutateFieldFormats applies fn to a copy of the current formats snapshot and
// stores the result. Hyperlink formats in the snapshot are already expanded,
// so no re-expansion is needed.
func (l *Logger) mutateFieldFormats(fn func(*FieldFormats)) {
	f := *l.loadFieldFormats()
	fn(&f)
	l.fieldFormats.Store(&f)
}

// SetNumberFormat sets how integer fields and both halves of fraction fields
// are rendered. It applies to fractions too unless [Logger.SetFractionFormat]
// overrides them. Defaults to [NumberPlain].
func (l *Logger) SetNumberFormat(format NumberFormat) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.NumberFormat = format })
}

// SetFractionFormat overrides the numeric format for fraction fields only.
// When unset, fractions fall back to [Logger.SetNumberFormat].
func (l *Logger) SetFractionFormat(format NumberFormat) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.FractionFormat = &format })
}

// SetNumberGroupSeparator sets the separator inserted between digit groups for
// [NumberGrouped] (e.g. "," for "1,234,567"). Defaults to ",".
func (l *Logger) SetNumberGroupSeparator(sep string) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.NumberGroupSeparator = sep })
}

// SetNumberCompactMinimum sets the smallest magnitude that [NumberCompact]
// abbreviates; values below it render using the compact fallback (see
// [Logger.SetNumberCompactFallback]). Defaults to 1000.
func (l *Logger) SetNumberCompactMinimum(minimum int64) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.NumberCompactMinimum = minimum })
}

// SetNumberCompactFallback sets how [NumberCompact] renders values below the
// compact minimum: [NumberGrouped] (the default, e.g. "9,999") or
// [NumberPlain] (e.g. "9999").
func (l *Logger) SetNumberCompactFallback(format NumberFormat) {
	l.mutateFieldFormats(func(f *FieldFormats) { f.NumberCompactFallback = format })
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
