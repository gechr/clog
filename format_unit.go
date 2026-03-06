package clog

import (
	"strconv"
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/field/duration"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
)

// styleDuration renders a duration string (from [time.Duration.String]) with
// gradient coloring when active ([style.Config.DurationGradient] non-empty and
// [duration.GradientMax] > 0), otherwise with separate styles for numeric and
// unit segments using [style.Config.FieldDurationNumber] and
// [style.Config.FieldDurationUnit]. Returns "" when no styles apply.
func styleDuration(s string, originalValue any, styles *style.Config) string {
	if styled := styleDurationGradient(s, originalValue, styles); styled != "" {
		return styled
	}

	return styleNumberUnit(
		s,
		styles.FieldDurationNumber,
		styles.FieldDurationUnit,
		styles.DurationUnits,
		styles.DurationThresholds,
		true,
	)
}

// styleDurationGradient colors the entire duration string based on value/max.
// Returns "" when the gradient is inactive (no stops, zero max, or wrong type).
func styleDurationGradient(s string, originalValue any, styles *style.Config) string {
	if len(styles.DurationGradient) == 0 {
		return ""
	}

	gm := duration.GradientMax()
	if gm <= 0 {
		return ""
	}

	d, ok := originalValue.(time.Duration)
	if !ok {
		return ""
	}

	t := core.Clamp01(float64(d) / float64(gm))

	var c colorful.Color
	switch {
	case len(styles.DurationGradient) == 1:
		c = styles.DurationGradient[0].Color
	case styles.DurationGradientMode == style.GradientStep:
		c = style.StepGradient(t, styles.DurationGradient)
	default:
		c = style.InterpolateGradient(t, styles.DurationGradient)
	}

	ls := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
	return ls.Render(s)
}

// styleElapsed renders an elapsed-time string. When the elapsed gradient is
// active ([style.Config.ElapsedGradient] non-empty and [elapsed.GradientMax] > 0),
// the entire string is colored by interpolating the gradient based on
// elapsed/max. Otherwise it falls back to the number/unit split path using
// [style.Config.FieldElapsedNumber] and [style.Config.FieldElapsedUnit].
// Returns "" when no styles apply.
func styleElapsed(s string, originalValue any, styles *style.Config) string {
	if styled := styleElapsedGradient(s, originalValue, styles); styled != "" {
		return styled
	}

	// Number/unit split path.
	numStyle := styles.FieldElapsedNumber
	if numStyle == nil {
		numStyle = styles.FieldDurationNumber
	}

	unitStyle := styles.FieldElapsedUnit
	if unitStyle == nil {
		unitStyle = styles.FieldDurationUnit
	}

	return styleNumberUnit(
		s,
		numStyle,
		unitStyle,
		styles.DurationUnits,
		styles.DurationThresholds,
		true,
	)
}

// styleElapsedGradient colors the entire elapsed string based on elapsed/max.
// Returns "" when the gradient is inactive (no stops, zero max, or wrong type).
func styleElapsedGradient(s string, originalValue any, styles *style.Config) string {
	if len(styles.ElapsedGradient) == 0 {
		return ""
	}

	gm := elapsed.GradientMax()
	if gm <= 0 {
		return ""
	}

	ef, ok := originalValue.(core.ElapsedField)
	if !ok {
		return ""
	}

	t := core.Clamp01(float64(time.Duration(ef)) / float64(gm))

	var c colorful.Color
	switch {
	case len(styles.ElapsedGradient) == 1:
		c = styles.ElapsedGradient[0].Color
	case styles.ElapsedGradientMode == style.GradientStep:
		c = style.StepGradient(t, styles.ElapsedGradient)
	default:
		c = style.InterpolateGradient(t, styles.ElapsedGradient)
	}

	ls := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
	return ls.Render(s)
}

// stylePercent renders a percentage string with a gradient color based on the
// value. The color is interpolated from the [style.Config.PercentGradient] stops and
// applied as the foreground on top of [style.Config.FieldPercent] (if set).
// originalValue must be a [core.Percent] value.
// When reverse is true the gradient position is flipped (1-t), making 0% green
// and 100% red - suitable for metrics where a low value is good.
// Returns "" when both FieldPercent and PercentGradient are nil/empty.
func stylePercent(valStr string, originalValue any, styles *style.Config, reverse bool) string {
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
		t := p.Value / percent.Scale()
		if reverse {
			t = 1 - t
		}

		var c colorful.Color
		if len(styles.PercentGradient) == 1 {
			c = styles.PercentGradient[0].Color
		} else {
			c = style.InterpolateGradient(t, styles.PercentGradient)
		}

		ls = ls.Foreground(lipgloss.Color(c.Clamped().Hex()))
	}
	return ls.Render(valStr)
}

// styleQuantity renders a quantity string with separate styles for the numeric
// and unit segments (e.g. "5" in FieldQuantityNumber, "km" in FieldQuantityUnit).
// Per-unit overrides in [style.Config.QuantityUnits] take priority over [style.Config.FieldQuantityUnit].
// Returns "" when both default styles are nil and no unit overrides match,
// or the string is not a valid quantity pattern.
func styleQuantity(s string, styles *style.Config, ignoreCase bool) string {
	return styleNumberUnit(
		s,
		styles.FieldQuantityNumber,
		styles.FieldQuantityUnit,
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
