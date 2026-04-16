package clog

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/field/duration"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/field/quantity"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/printer/json"
	"github.com/gechr/clog/style"
)

// formatFieldsOpts configures field formatting behaviour.
type formatFieldsOpts struct {
	fieldSort       Sort
	fieldStyleLevel Level
	level           Level
	noColor         bool
	quoteOpen       rune // 0 means default ('"' via strconv.Quote)
	quoteClose      rune // 0 means same as quoteOpen (or default)
	quoteMode       Quote
	separatorText   string
	sliceClose      rune // 0 means default (']')
	sliceOpen       rune // 0 means default ('[')
	sliceSep        string
	styles          *style.Config
	timeFormat      string
}

// valueKind classifies a formatted value for type-based styling.
type valueKind int

const (
	kindDefault valueKind = iota
	kindBool
	kindDuration
	kindElapsed
	kindError
	kindFraction
	kindJSON
	kindNumber
	kindPercent
	kindQuantity
	kindSlice
	kindString
	kindTime
)

const percentDisplayMax = 100.0

// formatFields formats fields for display.
// Returns an empty string if fields is empty.
func formatFields(fields []Field, opts formatFieldsOpts) string {
	if len(fields) == 0 {
		return ""
	}

	if opts.fieldSort != SortNone {
		fields = slices.Clone(fields)
		slices.SortFunc(fields, func(a, b Field) int {
			cmp := strings.Compare(a.Key, b.Key)
			if opts.fieldSort == SortDescending {
				return -cmp
			}
			return cmp
		})
	}

	var buf strings.Builder

	for i := range fields {
		f := fields[i]

		// Elapsed pre-processing: round, apply minimum threshold, update value.
		if val, ok := f.Value.(core.ElapsedField); ok {
			d := time.Duration(val)
			if r := elapsed.Round(); r > 0 {
				d = d.Round(r)
			}
			if d < elapsed.Minimum() {
				continue
			}
			f.Value = core.ElapsedField(d)
		}

		buf.WriteString(" ")

		sep := opts.separatorText
		if sep == "" {
			sep = "="
		}

		if !opts.noColor && opts.styles != nil && opts.styles.KeyDefault != nil {
			buf.WriteString(opts.styles.KeyDefault.Render(f.Key))
		} else {
			buf.WriteString(f.Key)
		}

		if !opts.noColor && opts.styles != nil && opts.styles.Separator != nil {
			buf.WriteString(opts.styles.Separator.Render(sep))
		} else {
			buf.WriteString(sep)
		}

		percentPrecision := percent.Precision()
		elapsedPrecision := elapsed.Precision()

		var valStr string
		var kind valueKind
		var customFormatted bool
		switch val := f.Value.(type) {
		case core.ElapsedField:
			fn := elapsed.FormatFunc()
			if fn == nil {
				fn = duration.FormatFunc()
			}
			if fn != nil {
				valStr = fn(time.Duration(val))
				kind = kindElapsed
				customFormatted = true
			}
		case time.Duration:
			if fn := duration.FormatFunc(); fn != nil {
				valStr = fn(val)
				kind = kindDuration
				customFormatted = true
			}
		case core.Percent:
			if fn := percent.FormatFunc(); fn != nil {
				valStr = fn(val.Value / percent.EffectiveMaximum(val) * percentDisplayMax)
				kind = kindPercent
				customFormatted = true
			}
		}
		if !customFormatted {
			valStr, kind = formatValue(
				f.Value,
				opts.sliceFmt(),
				opts.quoteMode,
				opts.quoteOpen,
				opts.quoteClose,
				opts.timeFormat,
				percentPrecision,
				elapsedPrecision,
			)
		}
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

// sliceFmt builds a sliceFormat from opts, applying defaults for zero-value runes.
func (opts formatFieldsOpts) sliceFmt() sliceFormat {
	openChar, closeChar := opts.sliceOpen, opts.sliceClose
	if openChar == 0 {
		openChar = '['
	}
	if closeChar == 0 {
		closeChar = ']'
	}
	sep := opts.sliceSep
	if sep == "" {
		sep = ", "
	}
	return sliceFormat{open: string(openChar), close: string(closeChar), sep: sep}
}

// formatValue converts a field value to its string representation.
// The returned valueKind indicates the type category for styling and quoting.
func formatValue(
	v any,
	sf sliceFormat,
	quoteMode Quote,
	quoteOpen, quoteClose rune,
	timeFormat string,
	percentPrecision int,
	elapsedPrecision int,
) (string, valueKind) {
	switch val := v.(type) {
	case core.ElapsedField:
		return formatElapsed(time.Duration(val), elapsedPrecision), kindElapsed
	case core.Fraction:
		return strconv.Itoa(val.Current) + "/" + strconv.Itoa(val.Total), kindFraction
	case error:
		return val.Error(), kindError
	case core.RawJSON:
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
	case core.Percent:
		display := val.Value / percent.EffectiveMaximum(val) * percentDisplayMax
		return strconv.FormatFloat(display, 'f', percentPrecision, 64) + "%", kindPercent
	case core.QuantityField:
		return string(val), kindQuantity
	case time.Duration:
		return val.String(), kindDuration
	case time.Time:
		if timeFormat == "" {
			timeFormat = time.DateTime
		}
		return val.Format(timeFormat), kindTime
	case []time.Duration:
		return formatDurationSlice(val, sf, nil), kindSlice
	case []core.QuantityField:
		return formatQuantitySlice(val, sf, nil, false), kindSlice
	case []string:
		return formatStringSlice(val, sf, nil, quoteMode, quoteOpen, quoteClose), kindSlice
	case []int:
		return formatIntSlice(val, sf, nil), kindSlice
	case []int64:
		return formatInt64Slice(val, sf, nil), kindSlice
	case []uint:
		return formatUintSlice(val, sf, nil), kindSlice
	case []uint64:
		return formatUint64Slice(val, sf, nil), kindSlice
	case []float64:
		return formatFloat64Slice(val, sf, nil), kindSlice
	case []bool:
		return formatBoolSlice(val, sf, nil), kindSlice
	case []any:
		return formatAnySlice(
			val,
			sf,
			nil,
			false,
			quoteMode,
			quoteOpen,
			quoteClose,
			false,
		), kindSlice
	default:
		return fmt.Sprintf("%v", v), kindDefault
	}
}

// formatETA formats a duration as a compact ETA string, always rounded to
// whole seconds. Uses the same composite format as [formatElapsed]:
//   - >= 1h: "1h2m"
//   - >= 1m: "2m30s"
//   - < 1m: "5s", minimum "1s" (never "0s")
func formatETA(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	// Round to whole seconds.
	d = d.Round(time.Second)

	if d >= time.Hour {
		h := int(d / time.Hour)
		remainder := d - time.Duration(h)*time.Hour
		m := int(remainder / time.Minute)
		if m == 0 {
			return strconv.Itoa(h) + "h"
		}
		return strconv.Itoa(h) + "h" + strconv.Itoa(m) + "m"
	}

	if d >= time.Minute {
		m := int(d / time.Minute)
		remainder := d - time.Duration(m)*time.Minute
		s := int(remainder / time.Second)
		if s == 0 {
			return strconv.Itoa(m) + "m"
		}
		return strconv.Itoa(m) + "m" + strconv.Itoa(s) + "s"
	}

	s := max(int(d/time.Second), 1)
	return strconv.Itoa(s) + "s"
}

// formatElapsed formats a duration for display. For durations >= 1 hour it
// uses composite "XhYm" format (omitting Ym when Y=0). For durations >= 1
// minute it uses "XmYs" (omitting Ys when Y=0). For shorter durations it
// picks the largest unit where the value is >= 1 and formats with the given
// decimal precision (no trailing zero trimming).
func formatElapsed(d time.Duration, precision int) string {
	if d < 0 {
		d = -d
	}

	// Composite format for >= 1h: "XhYm"
	if d >= time.Hour {
		h := int(d / time.Hour)
		remainder := d - time.Duration(h)*time.Hour
		m := int(remainder / time.Minute)
		if m == 0 {
			return strconv.Itoa(h) + "h"
		}
		return strconv.Itoa(h) + "h" + strconv.Itoa(m) + "m"
	}

	// Composite format for >= 1m: "XmYs"
	if d >= time.Minute {
		m := int(d / time.Minute)
		remainder := d - time.Duration(m)*time.Minute
		s := int(remainder / time.Second)
		if s == 0 {
			return strconv.Itoa(m) + "m"
		}
		return strconv.Itoa(m) + "m" + strconv.Itoa(s) + "s"
	}

	// Single unit with precision, no trailing zero trimming.
	type unit struct {
		suffix string
		div    time.Duration
	}

	units := [...]unit{
		{"s", time.Second},
		{"ms", time.Millisecond},
		{"µs", time.Microsecond},
		{"ns", time.Nanosecond},
	}

	for _, u := range units {
		if d >= u.div {
			val := float64(d) / float64(u.div)
			return strconv.FormatFloat(val, 'f', precision, 64) + u.suffix
		}
	}
	return "0s"
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
		return styledSlice(
			f.Value,
			opts.sliceFmt(),
			opts.styles,
			quantity.UnitsIgnoreCase(),
			opts.quoteMode,
			opts.quoteOpen,
			opts.quoteClose,
			percent.ReverseGradient(),
		)
	}

	if styled := styleValue(
		valStr,
		f.Value,
		f.Key,
		kind,
		opts.styles,
		quantity.UnitsIgnoreCase(),
		percent.ReverseGradient(),
	); styled != "" {
		return styled
	}
	return valStr
}

// styledSlice re-formats a slice value with per-element styling.
func styledSlice(
	v any,
	sf sliceFormat,
	styles *style.Config,
	ignoreCase bool,
	quoteMode Quote,
	quoteOpen, quoteClose rune,
	percentReverse bool,
) string {
	switch vals := v.(type) {
	case []bool:
		return formatBoolSlice(vals, sf, styles)
	case []time.Duration:
		return formatDurationSlice(vals, sf, styles)
	case []core.QuantityField:
		return formatQuantitySlice(vals, sf, styles, ignoreCase)
	case []int:
		return formatIntSlice(vals, sf, styles)
	case []int64:
		return formatInt64Slice(vals, sf, styles)
	case []uint:
		return formatUintSlice(vals, sf, styles)
	case []uint64:
		return formatUint64Slice(vals, sf, styles)
	case []float64:
		return formatFloat64Slice(vals, sf, styles)
	case []string:
		return formatStringSlice(vals, sf, styles, quoteMode, quoteOpen, quoteClose)
	case []any:
		return formatAnySlice(
			vals,
			sf,
			styles,
			ignoreCase,
			quoteMode,
			quoteOpen,
			quoteClose,
			percentReverse,
		)
	default:
		s, _ := formatValue(v, sf, quoteMode, quoteOpen, quoteClose, "", 0, 1)
		return s
	}
}

// styleValue applies the appropriate style to a formatted value.
// Priority: key style -> value style -> type style. Returns "" if no style applies.
// originalValue is the pre-format typed value for typed Values map lookups.
func styleValue(
	valStr string,
	originalValue any,
	key string,
	kind valueKind,
	styles *style.Config,
	ignoreCase bool,
	percentReverse bool,
) string {
	// Per-key styling takes priority.
	if style := styles.Keys[key]; style != nil {
		return style.Render(valStr)
	}

	// Per-value styling (typed key lookup - bool true ≠ string "true").
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
		if styled := styleDuration(valStr, originalValue, styles); styled != "" {
			return styled
		}
	case kindElapsed:
		if styled := styleElapsed(valStr, originalValue, styles); styled != "" {
			return styled
		}
	case kindFraction:
		if styled := styleFraction(valStr, originalValue, styles, percentReverse); styled != "" {
			return styled
		}
	case kindPercent:
		if styled := stylePercent(valStr, originalValue, styles, percentReverse); styled != "" {
			return styled
		}
	case kindQuantity:
		if styled := styleQuantity(valStr, styles, ignoreCase); styled != "" {
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
		return json.Highlight(valStr, styles.JSON)
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

// isZeroValue reports whether v is the zero value for its type. This is a
// superset of [isEmptyValue] - it additionally covers 0, false, 0.0, zero
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
func lookupValueStyle(v any, values style.ValueMap) *lipgloss.Style {
	if len(values) == 0 {
		return nil
	}

	if t := reflect.TypeOf(v); t != nil && !t.Comparable() {
		return nil
	}
	return values[v]
}

// computeLabelWidth returns the length of the longest label in the map.
func computeLabelWidth(labels LabelMap) int {
	maxWidth := 0
	for _, lbl := range labels {
		if len(lbl) > maxWidth {
			maxWidth = len(lbl)
		}
	}
	return maxWidth
}

// centerPad centres s within width, padding with spaces.
func centerPad(s string, width int) string {
	pad := width - len(s)
	left := pad / 2 //nolint:mnd // half the padding goes left
	right := pad - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
