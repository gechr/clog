package clog

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// percent wraps a float64 value (0–100) so [formatValue] can identify it
// for percentage styling with gradient colors.
type percent float64

// quantity wraps a string value with numeric and unit segments (e.g. "5m",
// "5.1km", "100MB") so [formatValue] can identify it for quantity styling.
type quantity string

// rawJSON wraps pre-serialized JSON bytes so [formatValue] can emit them
// verbatim without quoting or escaping.
type rawJSON []byte

// formatFieldsOpts configures field formatting behaviour.
type formatFieldsOpts struct {
	fieldStyleLevel Level
	level           Level
	noColor         bool
	quoteClose      rune // 0 means same as quoteOpen (or default)
	quoteMode       QuoteMode
	quoteOpen       rune // 0 means default ('"' via strconv.Quote)
	styles          *Styles
	timeFormat      string
}

// valueKind classifies a formatted value for type-based styling.
type valueKind int

const (
	kindDefault valueKind = iota
	kindBool
	kindDuration
	kindError
	kindJSON
	kindNumber
	kindPercent
	kindQuantity
	kindSlice
	kindString
	kindTime
)

const (
	percentMax = 100.0

	sliceOpen  = '['
	sliceClose = ']'
	sliceSep   = ", "
)

// formatFields formats fields for display.
// Returns an empty string if fields is empty.
func formatFields(fields []Field, opts formatFieldsOpts) string {
	if len(fields) == 0 {
		return ""
	}

	var buf strings.Builder

	for _, f := range fields {
		buf.WriteString(" ")

		sep := "="
		if opts.styles != nil && opts.styles.SeparatorText != "" {
			sep = opts.styles.SeparatorText
		}

		if !opts.noColor && opts.styles.KeyDefault != nil {
			buf.WriteString(opts.styles.KeyDefault.Render(f.Key))
		} else {
			buf.WriteString(f.Key)
		}

		if !opts.noColor && opts.styles.Separator != nil {
			buf.WriteString(opts.styles.Separator.Render(sep))
		} else {
			buf.WriteString(sep)
		}

		percentPrecision := 0
		if opts.styles != nil {
			percentPrecision = opts.styles.PercentPrecision
		}

		valStr, kind := formatValue(
			f.Value,
			opts.quoteMode,
			opts.quoteOpen,
			opts.quoteClose,
			opts.timeFormat,
			percentPrecision,
		)
		if opts.quoteMode != QuoteNever &&
			(kind == kindDefault || kind == kindString || kind == kindError || kind == kindTime) &&
			(opts.quoteMode == QuoteAlways || needsQuoting(valStr)) {
			valStr = quoteString(valStr, opts.quoteOpen, opts.quoteClose)
		}

		styled := styledFieldValue(f, valStr, kind, opts)
		buf.WriteString(styled)
	}
	return buf.String()
}

// formatValue converts a field value to its string representation.
// The returned valueKind indicates the type category for styling and quoting.
func formatValue(
	v any,
	quoteMode QuoteMode,
	quoteOpen, quoteClose rune,
	timeFormat string,
	percentPrecision int,
) (string, valueKind) {
	switch val := v.(type) {
	case error:
		return val.Error(), kindError
	case rawJSON:
		return string(val), kindJSON
	case string:
		return val, kindString
	case int:
		return strconv.Itoa(val), kindNumber
	case int64:
		return strconv.FormatInt(val, 10), kindNumber
	case uint:
		return strconv.FormatUint(uint64(val), 10), kindNumber
	case uint64:
		return strconv.FormatUint(val, 10), kindNumber
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), kindNumber
	case bool:
		return strconv.FormatBool(val), kindBool
	case percent:
		return strconv.FormatFloat(float64(val), 'f', percentPrecision, 64) + "%", kindPercent
	case quantity:
		return string(val), kindQuantity
	case time.Duration:
		return val.String(), kindDuration
	case time.Time:
		if timeFormat == "" {
			timeFormat = time.DateTime
		}
		return val.Format(timeFormat), kindTime
	case []time.Duration:
		return formatDurationSlice(val, nil), kindSlice
	case []quantity:
		return formatQuantitySlice(val, nil), kindSlice
	case []string:
		return formatStringSlice(val, nil, quoteMode, quoteOpen, quoteClose), kindSlice
	case []int:
		return formatIntSlice(val, nil), kindSlice
	case []int64:
		return formatInt64Slice(val, nil), kindSlice
	case []uint:
		return formatUintSlice(val, nil), kindSlice
	case []uint64:
		return formatUint64Slice(val, nil), kindSlice
	case []float64:
		return formatFloat64Slice(val, nil), kindSlice
	case []bool:
		return formatBoolSlice(val, nil), kindSlice
	case []any:
		return formatAnySlice(val, nil, quoteMode, quoteOpen, quoteClose), kindSlice
	default:
		return fmt.Sprintf("%v", v), kindDefault
	}
}

// formatAnySlice formats a []any slice with comma separation and per-element
// styling. Uses reflection to determine each element's type for highlighting.
func formatAnySlice(
	vals []any,
	styles *Styles,
	quoteMode QuoteMode,
	quoteOpen, quoteClose rune,
) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := fmt.Sprintf("%v", v)
		kind := reflectValueKind(v)

		if quoteMode != QuoteNever &&
			(kind == kindDefault || kind == kindString) &&
			(quoteMode == QuoteAlways || needsQuoting(s)) {
			s = quoteString(s, quoteOpen, quoteClose)
		}

		if styles != nil {
			styled := styleAnyElement(s, v, kind, styles)
			if styled != "" {
				buf.WriteString(styled)

				continue
			}
		}

		buf.WriteString(s)
	}

	buf.WriteByte(sliceClose)
	return buf.String()
}

// formatSlice formats any slice with comma separation and optional per-element styling.
// stringify converts each element to its string representation.
// stylize returns a styled string, or "" to fall back to the plain string.
func formatSlice[T any](
	vals []T,
	styles *Styles,
	stringify func(T) string,
	stylize func(T, string, *Styles) string,
) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := stringify(v)
		if styled := stylize(v, s, styles); styled != "" {
			buf.WriteString(styled)
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteByte(sliceClose)
	return buf.String()
}

// numberSliceStyle is a stylize function for numeric slice elements.
// It applies Styles.FieldNumber when set.
func numberSliceStyle[T any](_ T, s string, styles *Styles) string {
	if styles != nil && styles.FieldNumber != nil {
		return styles.FieldNumber.Render(s)
	}
	return ""
}

// formatBoolSlice formats a bool slice with comma separation.
// When styles is non-nil, individual elements are styled via ValueStyles.
func formatBoolSlice(vals []bool, styles *Styles) string {
	return formatSlice(vals, styles, strconv.FormatBool, func(v bool, s string, st *Styles) string {
		if st != nil {
			if style := st.Values[v]; style != nil {
				return style.Render(s)
			}
		}
		return ""
	})
}

// formatDurationSlice formats a [time.Duration] slice with comma separation.
// When styles is non-nil, individual elements are styled via [styleDuration].
func formatDurationSlice(vals []time.Duration, styles *Styles) string {
	return formatSlice(
		vals,
		styles,
		time.Duration.String,
		func(_ time.Duration, s string, st *Styles) string {
			if st == nil {
				return ""
			}
			return styleDuration(s, st)
		},
	)
}

// formatFloat64Slice formats a float64 slice with comma separation.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatFloat64Slice(vals []float64, styles *Styles) string {
	return formatSlice(vals, styles,
		func(v float64) string {
			return strconv.FormatFloat(v, 'f', -1, 64)
		},
		numberSliceStyle[float64],
	)
}

// formatIntSlice formats an int slice with comma separation.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatIntSlice(vals []int, styles *Styles) string {
	return formatSlice(vals, styles, strconv.Itoa, numberSliceStyle[int])
}

// formatInt64Slice formats an int64 slice with comma separation.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatInt64Slice(vals []int64, styles *Styles) string {
	return formatSlice(vals, styles,
		func(v int64) string {
			return strconv.FormatInt(v, 10)
		},
		numberSliceStyle[int64],
	)
}

// formatUintSlice formats a uint slice with comma separation.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatUintSlice(vals []uint, styles *Styles) string {
	return formatSlice(vals, styles,
		func(v uint) string {
			return strconv.FormatUint(uint64(v), 10)
		},
		numberSliceStyle[uint],
	)
}

// formatQuantitySlice formats a quantity slice with comma separation.
// When styles is non-nil, individual elements are styled via [styleQuantity].
func formatQuantitySlice(vals []quantity, styles *Styles) string {
	return formatSlice(
		vals,
		styles,
		func(v quantity) string {
			return string(v)
		},
		func(_ quantity, s string, st *Styles) string {
			if st == nil {
				return ""
			}
			return styleQuantity(s, st)
		},
	)
}

// formatStringSlice formats a string slice with comma separation and per-element quoting.
// When styles is non-nil, individual elements are styled via ValueStyles.
func formatStringSlice(
	vals []string,
	styles *Styles,
	quoteMode QuoteMode,
	quoteOpen, quoteClose rune,
) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		display := v
		if quoteMode != QuoteNever && (quoteMode == QuoteAlways || needsQuoting(v)) {
			display = quoteString(v, quoteOpen, quoteClose)
		}

		if styles != nil {
			if style := styles.Values[v]; style != nil {
				buf.WriteString(style.Render(display))

				continue
			}

			if styles.FieldString != nil {
				buf.WriteString(styles.FieldString.Render(display))

				continue
			}
		}

		buf.WriteString(display)
	}

	buf.WriteByte(sliceClose)
	return buf.String()
}

// formatUint64Slice formats a uint64 slice with comma separation.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatUint64Slice(vals []uint64, styles *Styles) string {
	return formatSlice(vals, styles,
		func(v uint64) string {
			return strconv.FormatUint(v, 10)
		},
		numberSliceStyle[uint64],
	)
}

// styleAnyElement applies the appropriate style to a single element in a []any slice.
// originalValue is the pre-format typed value for typed Values map lookups.
func styleAnyElement(s string, originalValue any, kind valueKind, styles *Styles) string {
	// Per-value styling (typed key lookup — bool true ≠ string "true").
	if style := lookupValueStyle(originalValue, styles.Values); style != nil {
		return style.Render(s)
	}

	switch kind { //nolint:exhaustive // slices don't appear as individual elements
	case kindString:
		if styles.FieldString != nil {
			return styles.FieldString.Render(s)
		}
	case kindNumber:
		if styles.FieldNumber != nil {
			return styles.FieldNumber.Render(s)
		}
	case kindError:
		if styles.FieldError != nil {
			return styles.FieldError.Render(s)
		}
	case kindDuration:
		if styled := styleDuration(s, styles); styled != "" {
			return styled
		}
	case kindPercent:
		if styled := stylePercent(s, originalValue, styles); styled != "" {
			return styled
		}
	case kindQuantity:
		if styled := styleQuantity(s, styles); styled != "" {
			return styled
		}

		// Fall back to string styling for unrecognized quantity strings.
		if styles.FieldString != nil {
			return styles.FieldString.Render(s)
		}
	case kindTime:
		if styles.FieldTime != nil {
			return styles.FieldTime.Render(s)
		}
	case kindBool, kindDefault, kindJSON:
		// No type-based style for these.
	}
	return ""
}

// styleDuration renders a duration string (from [time.Duration.String]) with
// separate styles for numeric and unit segments using [Styles.FieldDurationNumber]
// and [Styles.FieldDurationUnit]. Returns "" when both styles are nil.
func styleDuration(s string, styles *Styles) string {
	return styleNumberUnit(
		s,
		styles.FieldDurationNumber,
		styles.FieldDurationUnit,
		styles.DurationUnits,
		styles.DurationThresholds,
		true,
	)
}

// styledFieldValue applies styling to a formatted field value.
// Returns the styled string, or the plain valStr if no styling applies.
func styledFieldValue(f Field, valStr string, kind valueKind, opts formatFieldsOpts) string {
	if opts.noColor || opts.level < opts.fieldStyleLevel {
		return valStr
	}

	// KeyStyles takes priority over per-element styling for slices.
	if kind == kindSlice {
		if style := opts.styles.Keys[f.Key]; style != nil {
			return style.Render(valStr)
		}
		return styledSlice(f.Value, opts.styles, opts.quoteMode, opts.quoteOpen, opts.quoteClose)
	}

	if styled := styleValue(valStr, f.Value, f.Key, kind, opts.styles); styled != "" {
		return styled
	}
	return valStr
}

// styledSlice re-formats a slice value with per-element styling.
func styledSlice(v any, styles *Styles, quoteMode QuoteMode, quoteOpen, quoteClose rune) string {
	switch vals := v.(type) {
	case []bool:
		return formatBoolSlice(vals, styles)
	case []time.Duration:
		return formatDurationSlice(vals, styles)
	case []quantity:
		return formatQuantitySlice(vals, styles)
	case []int:
		return formatIntSlice(vals, styles)
	case []int64:
		return formatInt64Slice(vals, styles)
	case []uint:
		return formatUintSlice(vals, styles)
	case []uint64:
		return formatUint64Slice(vals, styles)
	case []float64:
		return formatFloat64Slice(vals, styles)
	case []string:
		return formatStringSlice(vals, styles, quoteMode, quoteOpen, quoteClose)
	case []any:
		return formatAnySlice(vals, styles, quoteMode, quoteOpen, quoteClose)
	default:
		s, _ := formatValue(v, quoteMode, quoteOpen, quoteClose, "", 0)
		return s
	}
}

// styleNumberUnit renders a string with separate styles for numeric and unit
// segments. unitOverrides provides per-unit style lookups; thresholds provides
// magnitude-based style overrides per unit; ignoreCase controls whether unit
// matching is case-insensitive.
// Returns "" when both default styles are nil, no unit overrides or thresholds
// apply, or the string is not a valid quantity pattern.
func styleNumberUnit(
	s string,
	numStyle, unitStyle Style,
	unitOverrides StyleMap,
	thresholds ThresholdMap,
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

// stylePercent renders a percentage string with a gradient color based on the
// value. The color is interpolated from the [Styles.PercentGradient] stops and
// applied as the foreground on top of [Styles.FieldPercent] (if set).
// originalValue must be a [percent] typed value.
// Returns "" when both FieldPercent and PercentGradient are nil/empty.
func stylePercent(valStr string, originalValue any, styles *Styles) string {
	p, ok := originalValue.(percent)
	if !ok {
		return ""
	}

	hasGradient := len(styles.PercentGradient) > 0

	if !hasGradient && styles.FieldPercent == nil {
		return ""
	}

	// Start from the base style (bold, italic, etc.) or a blank one.
	var style lipgloss.Style
	if styles.FieldPercent != nil {
		style = *styles.FieldPercent
	}

	// Apply gradient foreground on top of the base style.
	if hasGradient {
		var c colorful.Color
		if len(styles.PercentGradient) == 1 {
			c = styles.PercentGradient[0].Color
		} else {
			c = interpolateGradient(float64(p)/percentMax, styles.PercentGradient)
		}

		style = style.Foreground(lipgloss.Color(c.Clamped().Hex()))
	}
	return style.Render(valStr)
}

// styleQuantity renders a quantity string with separate styles for the numeric
// and unit segments (e.g. "5" in FieldQuantityNumber, "km" in FieldQuantityUnit).
// Per-unit overrides in [Styles.QuantityUnits] take priority over [Styles.FieldQuantityUnit].
// Returns "" when both default styles are nil and no unit overrides match,
// or the string is not a valid quantity pattern.
func styleQuantity(s string, styles *Styles) string {
	return styleNumberUnit(
		s,
		styles.FieldQuantityNumber,
		styles.FieldQuantityUnit,
		styles.QuantityUnits,
		styles.QuantityThresholds,
		styles.QuantityUnitsIgnoreCase,
	)
}

// styleValue applies the appropriate style to a formatted value.
// Priority: key style -> value style -> type style. Returns "" if no style applies.
// originalValue is the pre-format typed value for typed Values map lookups.
func styleValue(
	valStr string,
	originalValue any,
	key string,
	kind valueKind,
	styles *Styles,
) string {
	// Per-key styling takes priority.
	if style := styles.Keys[key]; style != nil {
		return style.Render(valStr)
	}

	// Per-value styling (typed key lookup — bool true ≠ string "true").
	if style := lookupValueStyle(originalValue, styles.Values); style != nil {
		return style.Render(valStr)
	}

	// Type-based styling.
	switch kind {
	case kindString:
		if styles.FieldString != nil {
			return styles.FieldString.Render(valStr)
		}
	case kindNumber:
		if styles.FieldNumber != nil {
			return styles.FieldNumber.Render(valStr)
		}
	case kindError:
		if styles.FieldError != nil {
			return styles.FieldError.Render(valStr)
		}
	case kindDuration:
		if styled := styleDuration(valStr, styles); styled != "" {
			return styled
		}
	case kindPercent:
		if styled := stylePercent(valStr, originalValue, styles); styled != "" {
			return styled
		}
	case kindQuantity:
		if styled := styleQuantity(valStr, styles); styled != "" {
			return styled
		}

		// Fall back to string styling for unrecognized quantity strings.
		if styles.FieldString != nil {
			return styles.FieldString.Render(valStr)
		}
	case kindTime:
		if styles.FieldTime != nil {
			return styles.FieldTime.Render(valStr)
		}
	case kindJSON:
		return highlightJSON(valStr, styles.FieldJSON)
	case kindBool, kindSlice, kindDefault:
		// No type-based style for these.
	}
	return ""
}

// needsQuoting returns true if the string needs quoting for parseable output.
// Returns false for strings containing ANSI escapes (e.g. hyperlinks) to preserve them.
func needsQuoting(s string) bool {
	if strings.Contains(s, "\x1b") {
		return false // preserve ANSI escape sequences (hyperlinks)
	}

	for _, r := range s {
		if unicode.IsSpace(r) || r == '"' || !strconv.IsPrint(r) {
			return true
		}
	}
	return false
}

// quoteString wraps s in quotes. When open is 0, it uses [strconv.Quote]
// (Go-style double-quoted with escaping). Otherwise it wraps with open/close runes.
// If close is 0, open is used for both sides.
func quoteString(s string, openChar, closeChar rune) string {
	if openChar == 0 {
		return strconv.Quote(s)
	}

	if closeChar == 0 {
		closeChar = openChar
	}
	return string(openChar) + s + string(closeChar)
}

// isEmptyValue reports whether v is semantically "nothing": nil, an empty
// string, or a nil/empty slice or map.
func isEmptyValue(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)

	switch rv.Kind() { //nolint:exhaustive // only string, slice, and map are considered empty
	case reflect.String:
		return rv.Len() == 0
	case reflect.Slice, reflect.Map:
		return rv.IsNil() || rv.Len() == 0
	default:
		return false
	}
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

// isZeroValue reports whether v is the zero value for its type. This is a
// superset of [isEmptyValue] — it additionally covers 0, false, 0.0, zero
// duration, and any other typed zero.
func isZeroValue(v any) bool {
	if v == nil {
		return true
	}

	rv := reflect.ValueOf(v)

	// Empty slices and maps are considered zero even when non-nil.
	switch rv.Kind() { //nolint:exhaustive // only slice and map need the length check
	case reflect.Slice, reflect.Map:
		return rv.Len() == 0
	default:
		return rv.IsZero()
	}
}

// mergeFields merges base fields with overrides, replacing existing keys.
// Keys in overrides replace matching keys in base while preserving order.
func mergeFields(base, overrides []Field) []Field {
	if len(overrides) == 0 {
		return base
	}

	overrideMap := make(map[string]any)
	for _, f := range overrides {
		overrideMap[f.Key] = f.Value
	}

	result := make([]Field, 0, len(base)+len(overrides))
	usedKeys := make(map[string]bool)

	for _, f := range base {
		if val, ok := overrideMap[f.Key]; ok {
			result = append(result, Field{Key: f.Key, Value: val})
			usedKeys[f.Key] = true
		} else {
			result = append(result, f)
		}
	}

	for _, f := range overrides {
		if !usedKeys[f.Key] {
			result = append(result, f)
		}
	}
	return result
}

// reflectValueKind uses reflection to classify a value for styling.
// This handles types not covered by the formatValue type switch (e.g. int64,
// float32, uint, custom named types with numeric underlying kinds).
func reflectValueKind(v any) valueKind {
	if v == nil {
		return kindDefault
	}

	if _, ok := v.(error); ok {
		return kindError
	}

	rv := reflect.ValueOf(v)

	switch rv.Kind() { //nolint:exhaustive // only string, numeric and bool kinds need special styling
	case reflect.String:
		return kindString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return kindNumber
	case reflect.Bool:
		return kindBool
	default:
		return kindDefault
	}
}

// clampPercent restricts val to the 0–100 range.
// NaN and negative infinity clamp to 0; positive infinity clamps to 100.
func clampPercent(val float64) float64 {
	if math.IsNaN(val) || math.IsInf(val, -1) {
		return 0
	}
	if math.IsInf(val, 1) {
		return percentMax
	}
	return max(0, min(percentMax, val))
}

// interpolateGradient computes the color at position t (0.0–1.0) along the
// given gradient stops using CIE-LCh blending for perceptually uniform
// transitions. Edge cases: empty -> white, single stop -> that color,
// t outside range -> clamp to nearest stop.
func interpolateGradient(t float64, stops []ColorStop) colorful.Color {
	if len(stops) == 0 {
		return colorful.Color{R: 1, G: 1, B: 1} // white fallback
	}

	if len(stops) == 1 {
		return stops[0].Color
	}

	// Clamp t to the range of the gradient.
	if t <= stops[0].Position {
		return stops[0].Color
	}

	if t >= stops[len(stops)-1].Position {
		return stops[len(stops)-1].Color
	}

	// Find the two bracketing stops.
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].Position {
			segLen := stops[i].Position - stops[i-1].Position
			if segLen <= 0 {
				return stops[i].Color
			}

			localT := (t - stops[i-1].Position) / segLen
			return stops[i-1].Color.BlendLuvLCh(stops[i].Color, localT)
		}
	}
	return stops[len(stops)-1].Color
}

// lookupValueStyle safely looks up a typed value in the Values map.
// Returns nil for unhashable types (slices, maps, functions) that would panic.
func lookupValueStyle(v any, values ValueStyleMap) Style {
	if len(values) == 0 {
		return nil
	}

	if t := reflect.TypeOf(v); t != nil && !t.Comparable() {
		return nil
	}
	return values[v]
}

// resolveSegmentStyles determines the effective number and unit styles for a
// single number+unit pair, applying threshold overrides when the numeric value
// meets or exceeds a configured threshold.
func resolveSegmentStyles(
	num, unit string,
	numStyle, unitStyle Style,
	unitOverrides StyleMap,
	thresholds ThresholdMap,
	ignoreCase bool,
) (Style, Style) {
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
func renderPendingNum(buf *strings.Builder, num, spaces string, style Style) {
	if num == "" {
		return
	}

	if style != nil {
		buf.WriteString(style.Render(num))
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
func thresholdForUnit(unit string, thresholds ThresholdMap, ignoreCase bool) []Threshold {
	return lookupMapKey(
		unit,
		thresholds,
		ignoreCase,
		func(ts []Threshold) bool {
			return len(ts) > 0
		},
	)
}

// unitOverrideStyle looks up a per-unit style from the given overrides map.
// When ignoreCase is true, keys are matched case-insensitively.
func unitOverrideStyle(unit string, overrides StyleMap, ignoreCase bool) Style {
	return lookupMapKey(
		unit,
		overrides,
		ignoreCase,
		func(s Style) bool {
			return s != nil
		},
	)
}
