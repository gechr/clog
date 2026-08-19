package clog

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
	xmath "github.com/gechr/x/math"
	"github.com/lucasb-eyer/go-colorful"
)

// styleDuration renders a duration string (from [time.Duration.String]) with
// gradient coloring when active ([style.Config.DurationGradient] non-empty and
// gradientMax > 0), otherwise with separate styles for numeric and
// unit segments using the [style.Config.FieldDuration] segment styles. Returns "" when no styles apply.
func styleDuration(
	s string,
	originalValue any,
	styles *style.Config,
	gradientMax time.Duration,
) string {
	if styled := styleDurationGradient(s, originalValue, styles, gradientMax); styled != "" {
		return styled
	}

	return styleNumberUnit(
		s,
		styles.FieldDuration.Number,
		styles.FieldDuration.Unit,
		styles.DurationUnits,
		styles.DurationThresholds,
		true,
	)
}

// asDurationField normalizes a duration-typed field value to a
// [core.DurationField], accepting both the wrapped type produced by
// [Event.Duration] and a raw [time.Duration] (e.g. passed via [Event.Any])
// with no per-field overrides. ok is false for any other type.
func asDurationField(v any) (core.DurationField, bool) {
	switch val := v.(type) {
	case core.DurationField:
		return val, true
	case time.Duration:
		return core.DurationField{Value: val}, true
	default:
		return core.DurationField{}, false
	}
}

// resolveGradient returns the active stops and mode after applying the
// field-level overrides (if any) over the logger defaults.
func resolveGradient(
	defStops []style.ColorStop,
	defMode style.GradientMode,
	overrideStops []style.ColorStop,
	overrideMode *style.GradientMode,
) ([]style.ColorStop, style.GradientMode) {
	stops := defStops
	if len(overrideStops) > 0 {
		stops = overrideStops
	}

	mode := defMode
	if overrideMode != nil {
		mode = *overrideMode
	}
	return stops, mode
}

// gradientColor picks the color at position t along stops, honoring mode.
// A single stop is used as-is.
func gradientColor(t float64, stops []style.ColorStop, mode style.GradientMode) colorful.Color {
	switch {
	case len(stops) == 1:
		return stops[0].Color
	case mode == style.GradientStep:
		return style.StepGradient(t, stops)
	default:
		return style.InterpolateGradient(t, stops)
	}
}

// renderGradient renders s with the gradient color at position t as its
// foreground.
func renderGradient(s string, t float64, stops []style.ColorStop, mode style.GradientMode) string {
	c := gradientColor(t, stops, mode)
	ls := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
	return ls.Render(s)
}

// styleDurationGradient colors the entire duration string based on value/max.
// Returns "" when the gradient is inactive (no stops, zero max, or wrong type).
// A field-level override set via [duration.WithGradientMax], [duration.WithGradient],
// or [duration.WithGradientMode] takes precedence over the logger defaults
// passed in via gradientMax and styles.
func styleDurationGradient(
	s string,
	originalValue any,
	styles *style.Config,
	gradientMax time.Duration,
) string {
	df, ok := asDurationField(originalValue)
	if !ok {
		return ""
	}

	stops, mode := resolveGradient(
		styles.DurationGradient,
		styles.DurationGradientMode,
		df.Gradient,
		df.GradientMode,
	)
	return styleValueGradient(s, stops, mode, df.Value, gradientMax, df.GradientMax)
}

// styleValueGradient colors s by interpolating stops at value/max, shared by
// the duration and elapsed gradients. A field-level max override takes
// precedence over the logger default. Returns "" when the gradient is
// inactive (no stops or non-positive max).
func styleValueGradient(
	s string,
	stops []style.ColorStop,
	mode style.GradientMode,
	value, gradientMax time.Duration,
	gmOverride *time.Duration,
) string {
	if len(stops) == 0 {
		return ""
	}

	gm := gradientMax
	if gmOverride != nil {
		gm = *gmOverride
	}
	if gm <= 0 {
		return ""
	}

	t := xmath.Clamp01(float64(value) / float64(gm))
	return renderGradient(s, t, stops, mode)
}

// styleElapsed renders an elapsed-time string. When the elapsed gradient is
// active ([style.Config.ElapsedGradient] non-empty and gradientMax > 0),
// the entire string is colored by interpolating the gradient based on
// elapsed/max. Otherwise it falls back to the number/unit split path using
// the [style.Config.FieldElapsed] segment styles.
// Returns "" when no styles apply.
func styleElapsed(
	s string,
	originalValue any,
	styles *style.Config,
	gradientMax time.Duration,
) string {
	if styled := styleElapsedGradient(s, originalValue, styles, gradientMax); styled != "" {
		return styled
	}
	return styleElapsedNumberUnit(s, styles)
}

// styleElapsedNumberUnit renders an elapsed-family string via the number/unit
// split path, falling back to the duration styles when the elapsed-specific
// ones are unset.
func styleElapsedNumberUnit(s string, styles *style.Config) string {
	seg := styles.FieldElapsed.Or(styles.FieldDuration)
	return styleNumberUnit(
		s,
		seg.Number,
		seg.Unit,
		styles.DurationUnits,
		styles.DurationThresholds,
		true,
	)
}

// styleElapsedGradient colors the entire elapsed string based on elapsed/max.
// Returns "" when the gradient is inactive (no stops, zero max, or wrong type).
// A field-level override set via [elapsed.WithGradientMax], [elapsed.WithGradient],
// or [elapsed.WithGradientMode] takes precedence over the logger defaults
// passed in via gradientMax and styles.
func styleElapsedGradient(
	s string,
	originalValue any,
	styles *style.Config,
	gradientMax time.Duration,
) string {
	ef, ok := originalValue.(core.ElapsedField)
	if !ok {
		return ""
	}

	stops, mode := resolveGradient(
		styles.ElapsedGradient,
		styles.ElapsedGradientMode,
		ef.Gradient,
		ef.GradientMode,
	)
	return styleValueGradient(s, stops, mode, ef.Value, gradientMax, ef.GradientMax)
}

// styleDeadline renders a countdown string. When the deadline gradient is
// active ([style.Config.DeadlineGradient] non-empty and From > 0), the entire
// string is colored by interpolating the gradient based on the consumed time
// (From - Remaining) / From, so a fresh deadline uses the first stop and an
// expiring one the last. Otherwise it falls back to the number/unit split
// path using the [style.Config.FieldElapsed] segment styles. Returns "" when no styles apply.
func styleDeadline(
	s string,
	originalValue any,
	styles *style.Config,
) string {
	if styled := styleDeadlineGradient(s, originalValue, styles); styled != "" {
		return styled
	}
	return styleElapsedNumberUnit(s, styles)
}

// styleDeadlineGradient colors the entire countdown string based on
// (From - Remaining) / From - the deadline's From is the implicit gradient
// maximum, so no GradientMax configuration applies. Returns "" when the
// gradient is inactive (no stops, non-positive From, or wrong type).
// A field-level override set via [deadline.WithGradient] or
// [deadline.WithGradientMode] takes precedence over the logger defaults.
func styleDeadlineGradient(
	s string,
	originalValue any,
	styles *style.Config,
) string {
	df, ok := originalValue.(core.DeadlineField)
	if !ok {
		return ""
	}

	stops, mode := resolveGradient(
		styles.DeadlineGradient,
		styles.DeadlineGradientMode,
		df.Gradient,
		df.GradientMode,
	)
	if len(stops) == 0 {
		return ""
	}

	if df.From <= 0 {
		return ""
	}

	consumed := df.From - df.Remaining
	t := xmath.Clamp01(float64(consumed) / float64(df.From))
	return renderGradient(s, t, stops, mode)
}

// stylePercent renders a percentage string with a gradient color based on the
// value. The color is interpolated from the [style.Config.PercentGradient] stops and
// applied as the foreground on top of [style.Config.FieldPercent] (if set).
// originalValue must be a [core.Percent] value.
// When reverse is true the gradient position is flipped (1-t), making 0% green
// and 100% red - suitable for metrics where a low value is good.
// Returns "" when both FieldPercent and PercentGradient are nil/empty.
func stylePercent(
	valStr string,
	originalValue any,
	styles *style.Config,
	reverse bool,
	maximum float64,
) string {
	p, ok := originalValue.(core.Percent)
	if !ok {
		return ""
	}
	if p.Reverse {
		reverse = !reverse // toggle whatever the logger default is
	}

	hasGradient := len(styles.PercentGradient) > 0

	if !hasGradient && styles.FieldPercent == nil {
		return ""
	}

	// Start from the base style (bold, italic, etc.) or a blank one.
	var ls lipgloss.Style
	if styles.FieldPercent != nil {
		ls = *styles.FieldPercent
	}

	// Apply gradient foreground on top of the base style.
	if hasGradient {
		t := percentFraction(p, maximum)
		if reverse {
			t = 1 - t
		}

		c := gradientColor(t, styles.PercentGradient, style.GradientFade)
		ls = ls.Foreground(lipgloss.Color(c.Clamped().Hex()))
	}
	return ls.Render(valStr)
}

func styleFraction(valStr string, originalValue any, styles *style.Config, reverse bool) string {
	f, ok := originalValue.(core.Fraction)
	if !ok {
		return ""
	}
	if f.Reverse {
		reverse = !reverse
	}

	hasGradient := len(styles.PercentGradient) > 0

	if !hasGradient && styles.FieldPercent == nil && styles.FieldFractionSeparator == nil {
		return ""
	}

	var ls lipgloss.Style
	if styles.FieldPercent != nil {
		ls = *styles.FieldPercent
	}

	if hasGradient {
		t := float64(f.Current) / float64(max(f.Total, 1))
		if reverse {
			t = 1 - t
		}

		c := gradientColor(t, styles.PercentGradient, style.GradientFade)
		ls = ls.Foreground(lipgloss.Color(c.Clamped().Hex()))
	}

	return renderFraction(valStr, ls, fractionSeparatorStyle(styles, ls))
}

// fractionSeparatorStyle resolves the style for the "/" in a fraction. An
// explicit [style.Config.FieldFractionSeparator] wins; otherwise the separator
// keeps the value's current color but adds the faint attribute, so it reads as
// a dimmed version of whatever color (gradient or base) the numbers use.
func fractionSeparatorStyle(styles *style.Config, base lipgloss.Style) lipgloss.Style {
	if styles.FieldFractionSeparator != nil {
		return *styles.FieldFractionSeparator
	}
	return base.Faint(true)
}

// renderFraction renders "current/total" with valueStyle on the numbers and
// sepStyle on the single "/". When no separator is present it falls back to
// styling the whole string.
func renderFraction(valStr string, valueStyle, sepStyle lipgloss.Style) string {
	before, after, found := strings.CutLast(valStr, "/")
	if !found {
		return valueStyle.Render(valStr)
	}
	return valueStyle.Render(before) +
		sepStyle.Render("/") +
		valueStyle.Render(after)
}

// styleQuantity renders a quantity string with separate styles for the numeric
// and unit segments (e.g. "5" in FieldQuantity.Number, "km" in FieldQuantity.Unit).
// Per-unit overrides in [style.Config.QuantityUnits] take priority over the [style.Config.FieldQuantity] unit style.
// Returns "" when both default styles are nil and no unit overrides match,
// or the string is not a valid quantity pattern.
func styleQuantity(s string, styles *style.Config, ignoreCase bool) string {
	return styleNumberUnit(
		s,
		styles.FieldQuantity.Number,
		styles.FieldQuantity.Unit,
		styles.QuantityUnits,
		styles.QuantityThresholds,
		ignoreCase,
	)
}

// styleNumberUnit renders a string with separate styles for numeric and unit
// segments. unitOverrides provides per-unit style lookups; thresholds provides
// magnitude-based style overrides per unit; ignoreCase controls whether unit
// matching is case-insensitive.
// Returns "" when both default styles are nil, no unit overrides or thresholds
// apply, or the string is not a valid quantity pattern.
func styleNumberUnit(
	s string,
	numStyle, unitStyle *lipgloss.Style,
	unitOverrides style.Map,
	thresholds style.ThresholdMap,
	ignoreCase bool,
) string {
	if numStyle == nil && unitStyle == nil && len(unitOverrides) == 0 && len(thresholds) == 0 {
		return ""
	}

	if !isQuantityString(s) {
		return ""
	}

	var buf strings.Builder

	runes := []rune(s)
	i := 0

	// Buffer the most recently parsed number segment so we can apply
	// threshold-based style overrides once we know the following unit.
	var pendingNum string
	var pendingSpaces string

	for i < len(runes) {
		r := runes[i]

		switch {
		case unicode.IsDigit(r) || r == '.' || r == '-':
			// Flush any prior pending number (defensive; valid quantities always pair num+unit).
			renderPendingNum(&buf, pendingNum, pendingSpaces, numStyle)

			start := i
			if r == '-' {
				i++
			}

			for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
				i++
			}

			pendingNum = string(runes[start:i])
			pendingSpaces = ""

		case unicode.IsLetter(r):
			start := i
			for i < len(runes) && unicode.IsLetter(runes[i]) {
				i++
			}

			unit := string(runes[start:i])

			// Resolve effective styles for this number+unit pair.
			effNumStyle, effUnitStyle := resolveSegmentStyles(
				pendingNum, unit,
				numStyle, unitStyle,
				unitOverrides, thresholds,
				ignoreCase,
			)

			// Render the pending number with the resolved style.
			if pendingNum != "" {
				if effNumStyle != nil {
					buf.WriteString(effNumStyle.Render(pendingNum))
				} else {
					buf.WriteString(pendingNum)
				}

				buf.WriteString(pendingSpaces)

				pendingNum = ""
				pendingSpaces = ""
			}

			// Render the unit.
			if effUnitStyle != nil {
				buf.WriteString(effUnitStyle.Render(unit))
			} else {
				buf.WriteString(unit)
			}

		case r == ' ':
			if pendingNum != "" {
				pendingSpaces += string(r)
			} else {
				buf.WriteRune(r)
			}

			i++

		default:
			renderPendingNum(&buf, pendingNum, pendingSpaces, numStyle)

			pendingNum = ""
			pendingSpaces = ""
			buf.WriteRune(r)
			i++
		}
	}

	// Flush any trailing pending number.
	renderPendingNum(&buf, pendingNum, pendingSpaces, numStyle)
	return buf.String()
}

// isQuantityString reports whether s looks like a quantity: an optional leading
// '-' followed by one or more digit+letter groups with optional spaces between
// the number and unit (e.g. "5m", "5.1km", "100 MB", "2h30m").
func isQuantityString(s string) bool {
	runes := []rune(s)
	i := 0

	if i < len(runes) && runes[i] == '-' {
		i++
	}

	if i >= len(runes) || !unicode.IsDigit(runes[i]) {
		return false
	}

	groups := 0

	for i < len(runes) {
		if !unicode.IsDigit(runes[i]) && runes[i] != '.' {
			return false
		}

		for i < len(runes) && (unicode.IsDigit(runes[i]) || runes[i] == '.') {
			i++
		}

		// Skip optional space between number and unit.
		for i < len(runes) && runes[i] == ' ' {
			i++
		}

		if i >= len(runes) || !unicode.IsLetter(runes[i]) {
			return false
		}

		for i < len(runes) && unicode.IsLetter(runes[i]) {
			i++
		}

		// Skip optional space before next group.
		for i < len(runes) && runes[i] == ' ' {
			i++
		}

		groups++
	}
	return groups > 0
}

// resolveSegmentStyles determines the effective number and unit styles for a
// single number+unit pair, applying threshold overrides when the numeric value
// meets or exceeds a configured threshold.
func resolveSegmentStyles(
	num, unit string,
	numStyle, unitStyle *lipgloss.Style,
	unitOverrides style.Map,
	thresholds style.ThresholdMap,
	ignoreCase bool,
) (*lipgloss.Style, *lipgloss.Style) {
	effNumStyle := numStyle

	effUnitStyle := unitOverrideStyle(unit, unitOverrides, ignoreCase)
	if effUnitStyle == nil {
		effUnitStyle = unitStyle
	}

	if len(thresholds) == 0 || num == "" {
		return effNumStyle, effUnitStyle
	}

	numVal, err := strconv.ParseFloat(num, 64)
	if err != nil {
		return effNumStyle, effUnitStyle
	}

	for _, t := range thresholdForUnit(unit, thresholds, ignoreCase) {
		if numVal >= t.Value {
			if t.Style.Number != nil {
				effNumStyle = t.Style.Number
			}

			if t.Style.Unit != nil {
				effUnitStyle = t.Style.Unit
			}

			break
		}
	}
	return effNumStyle, effUnitStyle
}

// renderPendingNum renders a buffered number segment with optional trailing
// spaces. This is a no-op when num is empty.
func renderPendingNum(buf *strings.Builder, num, spaces string, s *lipgloss.Style) {
	if num == "" {
		return
	}

	if s != nil {
		buf.WriteString(s.Render(num))
	} else {
		buf.WriteString(num)
	}

	buf.WriteString(spaces)
}

// lookupMapKey returns the value for key in m when valid(value) is true.
// When ignoreCase is true and the direct lookup fails, a case-insensitive
// scan of all keys is tried. Returns the zero value of V when no match is found.
func lookupMapKey[V any](key string, m map[string]V, ignoreCase bool, valid func(V) bool) V {
	if v := m[key]; valid(v) {
		return v
	}

	if ignoreCase {
		lower := strings.ToLower(key)
		for k, v := range m {
			if strings.ToLower(k) == lower {
				return v
			}
		}
	}

	var zero V
	return zero
}

// thresholdForUnit looks up quantity thresholds for a unit string.
// When ignoreCase is true, keys are matched case-insensitively.
func thresholdForUnit(
	unit string,
	thresholds style.ThresholdMap,
	ignoreCase bool,
) []style.Threshold {
	return lookupMapKey(
		unit,
		thresholds,
		ignoreCase,
		func(ts []style.Threshold) bool {
			return len(ts) > 0
		},
	)
}

// unitOverrideStyle looks up a per-unit style from the given overrides map.
// When ignoreCase is true, keys are matched case-insensitively.
func unitOverrideStyle(unit string, overrides style.Map, ignoreCase bool) *lipgloss.Style {
	return lookupMapKey(
		unit,
		overrides,
		ignoreCase,
		func(s *lipgloss.Style) bool {
			return s != nil
		},
	)
}
