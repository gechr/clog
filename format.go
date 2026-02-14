package clog

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

// formatFieldsOpts configures field formatting behaviour.
type formatFieldsOpts struct {
	noColor    bool
	level      Level
	styles     *Styles
	quoteMode  QuoteMode
	quoteOpen  rune // 0 means default ('"' via strconv.Quote)
	quoteClose rune // 0 means same as quoteOpen (or default)
}

// valueKind classifies a formatted value for type-based styling.
type valueKind int

const (
	kindDefault valueKind = iota
	kindBool
	kindError
	kindNumber
	kindSlice
	kindString
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

		if opts.noColor {
			buf.WriteString(f.Key)
			buf.WriteString(sep)
		} else {
			buf.WriteString(opts.styles.Key.Render(f.Key))
			buf.WriteString(opts.styles.Separator.Render(sep))
		}

		valStr, kind := formatValue(f.Value, opts.quoteMode, opts.quoteOpen, opts.quoteClose)
		if opts.quoteMode != QuoteNever &&
			(kind == kindDefault || kind == kindString || kind == kindError) &&
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
func formatValue(v any, quoteMode QuoteMode, quoteOpen, quoteClose rune) (string, valueKind) {
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

// styledFieldValue applies styling to a formatted field value.
// Returns the styled string, or the plain valStr if no styling applies.
func styledFieldValue(f Field, valStr string, kind valueKind, opts formatFieldsOpts) string {
	if opts.noColor || opts.level < InfoLevel {
		return valStr
	}

	// KeyStyles takes priority over per-element styling for slices.
	if kind == kindSlice {
		if style, ok := opts.styles.KeyStyles[f.Key]; ok {
			return style.Render(valStr)
		}

		return styledSlice(f.Value, opts.styles, opts.quoteMode, opts.quoteOpen, opts.quoteClose)
	}

	if styled := styleValue(valStr, f.Key, kind, opts.styles); styled != "" {
		return styled
	}

	return valStr
}

// styleValue applies the appropriate style to a formatted value.
// Priority: key style → value style → type style. Returns "" if no style applies.
func styleValue(valStr, key string, kind valueKind, styles *Styles) string {
	// Per-key styling takes priority.
	if style, ok := styles.KeyStyles[key]; ok {
		return style.Render(valStr)
	}

	// Per-value styling (exact match on formatted string).
	if style, ok := styles.ValueStyles[valStr]; ok {
		return style.Render(valStr)
	}

	// Type-based styling.
	switch kind {
	case kindString:
		if styles.String != nil {
			return styles.String.Render(valStr)
		}
	case kindNumber:
		if styles.Number != nil {
			return styles.Number.Render(valStr)
		}
	case kindError:
		if styles.Error != nil {
			return styles.Error.Render(valStr)
		}
	case kindBool, kindSlice, kindDefault:
		// No type-based style for these.
	}

	return ""
}

// styledSlice re-formats a slice value with per-element styling.
func styledSlice(v any, styles *Styles, quoteMode QuoteMode, quoteOpen, quoteClose rune) string {
	switch vals := v.(type) {
	case []bool:
		return formatBoolSlice(vals, styles)
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
		s, _ := formatValue(v, quoteMode, quoteOpen, quoteClose)
		return s
	}
}

// styleAnyElement applies the appropriate style to a single element in a []any slice.
func styleAnyElement(s string, kind valueKind, styles *Styles) string {
	// Per-value styling (exact match on formatted string).
	if style, ok := styles.ValueStyles[s]; ok {
		return style.Render(s)
	}

	switch kind { //nolint:exhaustive // slices don't appear as individual elements
	case kindString:
		if styles.String != nil {
			return styles.String.Render(s)
		}
	case kindNumber:
		if styles.Number != nil {
			return styles.Number.Render(s)
		}
	case kindError:
		if styles.Error != nil {
			return styles.Error.Render(s)
		}
	case kindBool, kindDefault:
		// No type-based style for these.
	}

	return ""
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

// formatStringSlice formats a string slice with comma separation and per-element quoting.
// When styles is non-nil, individual elements are styled via ValueStyles.
func formatStringSlice(
	vals []string,
	styles *Styles,
	quoteMode QuoteMode,
	quoteOpen, quoteClose rune,
) string {
	var buf strings.Builder

	buf.WriteByte('[')

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(", ")
		}

		display := v
		if quoteMode != QuoteNever && (quoteMode == QuoteAlways || needsQuoting(v)) {
			display = quoteString(v, quoteOpen, quoteClose)
		}

		if styles != nil {
			if style, ok := styles.ValueStyles[v]; ok {
				buf.WriteString(style.Render(display))

				continue
			}

			if styles.String != nil {
				buf.WriteString(styles.String.Render(display))

				continue
			}
		}

		buf.WriteString(display)
	}

	buf.WriteByte(']')

	return buf.String()
}

// formatIntSlice formats an int slice with comma separation.
// When styles is non-nil, individual elements are styled via Number style.
func formatIntSlice(vals []int, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte('[')

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(", ")
		}

		s := strconv.Itoa(v)
		if styles != nil && styles.Number != nil {
			buf.WriteString(styles.Number.Render(s))
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteByte(']')

	return buf.String()
}

// formatUint64Slice formats a uint64 slice with comma separation.
// When styles is non-nil, individual elements are styled via Number style.
func formatUint64Slice(vals []uint64, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte('[')

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(", ")
		}

		s := strconv.FormatUint(v, 10)
		if styles != nil && styles.Number != nil {
			buf.WriteString(styles.Number.Render(s))
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteByte(']')

	return buf.String()
}

// formatFloat64Slice formats a float64 slice with comma separation.
// When styles is non-nil, individual elements are styled via Number style.
func formatFloat64Slice(vals []float64, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte('[')

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(", ")
		}

		s := strconv.FormatFloat(v, 'f', -1, 64)
		if styles != nil && styles.Number != nil {
			buf.WriteString(styles.Number.Render(s))
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteByte(']')

	return buf.String()
}

// formatBoolSlice formats a bool slice with comma separation.
// When styles is non-nil, individual elements are styled via ValueStyles.
func formatBoolSlice(vals []bool, styles *Styles) string {
	var buf strings.Builder

	buf.WriteByte('[')

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(", ")
		}

		s := strconv.FormatBool(v)
		if styles != nil {
			if style, ok := styles.ValueStyles[s]; ok {
				buf.WriteString(style.Render(s))

				continue
			}
		}

		buf.WriteString(s)
	}

	buf.WriteByte(']')

	return buf.String()
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

	buf.WriteByte('[')

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(", ")
		}

		s := fmt.Sprintf("%v", v)
		kind := reflectValueKind(v)

		if quoteMode != QuoteNever &&
			(kind == kindDefault || kind == kindString) &&
			(quoteMode == QuoteAlways || needsQuoting(s)) {
			s = quoteString(s, quoteOpen, quoteClose)
		}

		if styles != nil {
			styled := styleAnyElement(s, kind, styles)
			if styled != "" {
				buf.WriteString(styled)

				continue
			}
		}

		buf.WriteString(s)
	}

	buf.WriteByte(']')

	return buf.String()
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
