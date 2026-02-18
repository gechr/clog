package clog

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"
)

// quantity wraps a string value with numeric and unit segments (e.g. "5m",
// "5.1km", "100MB") so [formatValue] can identify it for quantity styling.
type quantity string

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
	kindNumber
	kindQuantity
	kindSlice
	kindString
	kindTime
)

const (
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

		valStr, kind := formatValue(
			f.Value,
			opts.quoteMode,
			opts.quoteOpen,
			opts.quoteClose,
			opts.timeFormat,
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
) (string, valueKind) {
	switch val := v.(type) {
	case error:
		return val.Error(), kindError
	case string:
		return val, kindString
	case int:
		return strconv.Itoa(val), kindNumber
	case uint64:
		return strconv.FormatUint(val, 10), kindNumber
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), kindNumber
	case bool:
		return strconv.FormatBool(val), kindBool
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

// formatBoolSlice formats a bool slice with comma separation.
// When styles is non-nil, individual elements are styled via ValueStyles.
func formatBoolSlice(vals []bool, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := strconv.FormatBool(v)
		if styles != nil {
			if style := styles.Values[v]; style != nil {
				buf.WriteString(style.Render(s))

				continue
			}
		}

		buf.WriteString(s)
	}

	buf.WriteByte(sliceClose)

	return buf.String()
}

// formatDurationSlice formats a [time.Duration] slice with comma separation.
// When styles is non-nil, individual elements are styled via [styleDuration].
func formatDurationSlice(vals []time.Duration, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := v.String()
		if styles != nil {
			if styled := styleDuration(s, styles); styled != "" {
				buf.WriteString(styled)

				continue
			}
		}

		buf.WriteString(s)
	}

	buf.WriteByte(sliceClose)

	return buf.String()
}

// formatFloat64Slice formats a float64 slice with comma separation.
// When styles is non-nil, individual elements are styled via Number style.
func formatFloat64Slice(vals []float64, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := strconv.FormatFloat(v, 'f', -1, 64)
		if styles != nil && styles.FieldNumber != nil {
			buf.WriteString(styles.FieldNumber.Render(s))
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteByte(sliceClose)

	return buf.String()
}

// formatIntSlice formats an int slice with comma separation.
// When styles is non-nil, individual elements are styled via Number style.
func formatIntSlice(vals []int, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := strconv.Itoa(v)
		if styles != nil && styles.FieldNumber != nil {
			buf.WriteString(styles.FieldNumber.Render(s))
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteByte(sliceClose)

	return buf.String()
}

// formatQuantitySlice formats a quantity slice with comma separation.
// When styles is non-nil, individual elements are styled via [styleQuantity].
func formatQuantitySlice(vals []quantity, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := string(v)
		if styles != nil {
			if styled := styleQuantity(s, styles); styled != "" {
				buf.WriteString(styled)

				continue
			}
		}

		buf.WriteString(s)
	}

	buf.WriteByte(sliceClose)

	return buf.String()
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
// When styles is non-nil, individual elements are styled via Number style.
func formatUint64Slice(vals []uint64, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte(sliceOpen)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sliceSep)
		}

		s := strconv.FormatUint(v, 10)
		if styles != nil && styles.FieldNumber != nil {
			buf.WriteString(styles.FieldNumber.Render(s))
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteByte(sliceClose)

	return buf.String()
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
	case kindBool, kindDefault:
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
	case []uint64:
		return formatUint64Slice(vals, styles)
	case []float64:
		return formatFloat64Slice(vals, styles)
	case []string:
		return formatStringSlice(vals, styles, quoteMode, quoteOpen, quoteClose)
	case []any:
		return formatAnySlice(vals, styles, quoteMode, quoteOpen, quoteClose)
	default:
		s, _ := formatValue(v, quoteMode, quoteOpen, quoteClose, "")
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
	numStyle, unitStyle *lipgloss.Style,
	unitOverrides map[string]*lipgloss.Style,
	thresholds map[string][]QuantityThreshold,
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
// Priority: key style → value style → type style. Returns "" if no style applies.
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

// lookupValueStyle safely looks up a typed value in the Values map.
// Returns nil for unhashable types (slices, maps, functions) that would panic.
func lookupValueStyle(v any, values map[any]*lipgloss.Style) *lipgloss.Style {
	if len(values) == 0 || v == nil {
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
	numStyle, unitStyle *lipgloss.Style,
	unitOverrides map[string]*lipgloss.Style,
	thresholds map[string][]QuantityThreshold,
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
			if t.Number != nil {
				effNumStyle = t.Number
			}

			if t.Unit != nil {
				effUnitStyle = t.Unit
			}

			break
		}
	}

	return effNumStyle, effUnitStyle
}

// renderPendingNum renders a buffered number segment with optional trailing
// spaces. This is a no-op when num is empty.
func renderPendingNum(buf *strings.Builder, num, spaces string, style *lipgloss.Style) {
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

// thresholdForUnit looks up quantity thresholds for a unit string.
// When ignoreCase is true, keys are matched case-insensitively.
func thresholdForUnit(
	unit string,
	thresholds map[string][]QuantityThreshold,
	ignoreCase bool,
) []QuantityThreshold {
	if len(thresholds) == 0 {
		return nil
	}

	if ts := thresholds[unit]; len(ts) > 0 {
		return ts
	}

	if ignoreCase {
		lower := strings.ToLower(unit)
		for k, ts := range thresholds {
			if strings.ToLower(k) == lower {
				return ts
			}
		}
	}

	return nil
}

// unitOverrideStyle looks up a per-unit style from the given overrides map.
// When ignoreCase is true, keys are matched case-insensitively.
func unitOverrideStyle(
	unit string,
	overrides map[string]*lipgloss.Style,
	ignoreCase bool,
) *lipgloss.Style {
	if len(overrides) == 0 {
		return nil
	}

	if style := overrides[unit]; style != nil {
		return style
	}

	if ignoreCase {
		lower := strings.ToLower(unit)
		for k, style := range overrides {
			if strings.ToLower(k) == lower {
				return style
			}
		}
	}

	return nil
}
