package clog

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/printer/json"
	"github.com/gechr/clog/style"
	"github.com/gechr/x/human"
)

// formatFieldsOpts configures field formatting behaviour.
type formatFieldsOpts struct {
	fieldSort       Sort
	fieldStyleLevel Level
	formats         *FieldFormats // nil means DefaultFieldFormats
	level           Level
	noColor         bool
	quoteOpen       rune // 0 means default ('"' via strconv.Quote)
	quoteClose      rune // 0 means same as quoteOpen (or default)
	quoteMode       Quote
	quoteSmart      []QuotePair // non-empty enables content-adaptive quoting
	separatorText   string
	sliceClose      rune // 0 means default (']')
	sliceOpen       rune // 0 means default ('[')
	sliceSep        string
	styles          *style.Config
	timeFormat      string
}

// fieldFormats returns the field-format configuration to use, falling back
// to the package default when opts carries none (zero-value opts in tests).
func (opts formatFieldsOpts) fieldFormats() *FieldFormats {
	if opts.formats != nil {
		return opts.formats
	}
	return &defaultFieldFormats
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

	fmts := opts.fieldFormats()

	var buf strings.Builder

	for i := range fields {
		f := fields[i]

		// Elapsed pre-processing: round, apply minimum threshold, update value.
		if val, ok := f.Value.(core.ElapsedField); ok {
			d := val.Value
			if r := fmts.ElapsedRound; r > 0 {
				d = d.Round(r)
			}
			if d < fmts.ElapsedMinimum {
				continue
			}
			val.Value = d
			f.Value = val
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

		var valStr string
		var kind valueKind
		var customFormatted bool
		switch val := f.Value.(type) {
		case core.ElapsedField:
			fn := fmts.ElapsedFormat
			if fn == nil {
				fn = fmts.DurationFormat
			}
			if fn != nil {
				valStr = fn(val.Value)
				kind = kindElapsed
				customFormatted = true
			}
		case core.DurationField:
			if fn := fmts.DurationFormat; fn != nil {
				valStr = fn(val.Value)
				kind = kindDuration
				customFormatted = true
			}
		case time.Duration:
			if fn := fmts.DurationFormat; fn != nil {
				valStr = fn(val)
				kind = kindDuration
				customFormatted = true
			}
		case core.Percent:
			if fn := fmts.PercentFormat; fn != nil {
				valStr = fn(percentFraction(val, fmts.PercentMaximum) * percentDisplayMax)
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
				opts.quoteSmart,
				opts.timeFormat,
				fmts,
			)
		}
		quoted := opts.quoteMode != QuoteNever &&
			(kind == kindDefault || kind == kindString || kind == kindError || kind == kindTime) &&
			(opts.quoteMode == QuoteAlways || needsQuoting(valStr))

		// When a quote-delimiter style is configured, style the delimiters
		// separately from the body. Otherwise wrap first and style the whole
		// quoted string as one unit (legacy behaviour: delimiters inherit the
		// value's style).
		if cfg := quoteDelim(opts); quoted && cfg != nil {
			open, body, closing := quoteParts(
				valStr,
				opts.quoteOpen,
				opts.quoteClose,
				opts.quoteSmart,
			)
			delim := cfg.Resolve(valueBaseStyle(f.Value, f.Key, kind, opts.styles))
			styled := delim.Render(open) +
				styledFieldValue(f, body, kind, opts) +
				delim.Render(closing)
			buf.WriteString(styled)
			continue
		}

		if quoted {
			valStr = quoteString(valStr, opts.quoteOpen, opts.quoteClose, opts.quoteSmart)
		}
		buf.WriteString(styledFieldValue(f, valStr, kind, opts))
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

// formatNumber renders n according to mode, drawing the group separator and
// compact minimum from fmts. NumberPlain (and any unrecognised mode) returns
// the plain base-10 string.
func formatNumber(n int64, mode NumberFormat, fmts *FieldFormats) string {
	switch mode {
	case NumberGrouped:
		return groupNumber(n, fmts)
	case NumberCompact:
		if n > -fmts.NumberCompactMinimum && n < fmts.NumberCompactMinimum {
			// Below the abbreviation threshold, render with the configured
			// fallback (grouped by default) so smaller values still read
			// nicely (e.g. "9,999" before "10K"). Guard against recursing
			// back into compact.
			if fmts.NumberCompactFallback == NumberGrouped {
				return groupNumber(n, fmts)
			}
			return strconv.FormatInt(n, 10)
		}
		return human.FormatNumberCompact(n)
	case NumberPlain:
		return strconv.FormatInt(n, 10)
	default:
		return strconv.FormatInt(n, 10)
	}
}

// groupNumber renders n with the configured digit-group separator, defaulting
// to "," when none is set.
func groupNumber(n int64, fmts *FieldFormats) string {
	sep := fmts.NumberGroupSeparator
	if sep == "" {
		sep = ","
	}
	return human.FormatNumber(n, sep)
}

// formatUnsignedNumber renders n like [formatNumber], falling back to a plain
// decimal string for values too large to represent as an int64 (the helpers
// in x/human operate on int64).
func formatUnsignedNumber(n uint64, mode NumberFormat, fmts *FieldFormats) string {
	if n > math.MaxInt64 {
		return strconv.FormatUint(n, 10)
	}
	return formatNumber(int64(n), mode, fmts)
}

// fractionNumberFormat resolves the effective numeric format for a fraction
// field: an explicit per-field override wins, then the logger's
// FractionFormat, then its general NumberFormat.
func fractionNumberFormat(f core.Fraction, fmts *FieldFormats) NumberFormat {
	switch {
	case f.Format != nil:
		return *f.Format
	case fmts.FractionFormat != nil:
		return *fmts.FractionFormat
	default:
		return fmts.NumberFormat
	}
}

// formatValue converts a field value to its string representation.
// The returned valueKind indicates the type category for styling and quoting.
func formatValue(
	v any,
	sf sliceFormat,
	quoteMode Quote,
	quoteOpen, quoteClose rune,
	quoteSmart []QuotePair,
	timeFormat string,
	fmts *FieldFormats,
) (string, valueKind) {
	switch val := v.(type) {
	case core.ElapsedField:
		return formatElapsed(val.Value, fmts.ElapsedPrecision), kindElapsed
	case core.Fraction:
		mode := fractionNumberFormat(val, fmts)
		return formatNumber(int64(val.Current), mode, fmts) +
			"/" + formatNumber(int64(val.Total), mode, fmts), kindFraction
	case error:
		return val.Error(), kindError
	case core.RawJSON:
		return string(val), kindJSON
	case string:
		return val, kindString
	case int:
		return formatNumber(int64(val), fmts.NumberFormat, fmts), kindNumber
	case int64:
		return formatNumber(val, fmts.NumberFormat, fmts), kindNumber
	case uint:
		return formatUnsignedNumber(uint64(val), fmts.NumberFormat, fmts), kindNumber
	case uint64:
		return formatUnsignedNumber(val, fmts.NumberFormat, fmts), kindNumber
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64), kindNumber
	case bool:
		return strconv.FormatBool(val), kindBool
	case core.Percent:
		display := percentFraction(val, fmts.PercentMaximum) * percentDisplayMax
		return strconv.FormatFloat(display, 'f', fmts.PercentPrecision, 64) + "%", kindPercent
	case core.QuantityField:
		return string(val), kindQuantity
	case core.DurationField:
		return val.Value.String(), kindDuration
	case time.Duration:
		return val.String(), kindDuration
	case time.Time:
		if timeFormat == "" {
			timeFormat = time.DateTime
		}
		return val.Format(timeFormat), kindTime
	case []time.Duration:
		return formatDurationSlice(val, sf, nil, fmts), kindSlice
	case []core.QuantityField:
		return formatQuantitySlice(val, sf, nil, fmts.QuantityUnitsIgnoreCase), kindSlice
	case []string:
		return formatStringSlice(
			val,
			sf,
			nil,
			quoteMode,
			quoteOpen,
			quoteClose,
			quoteSmart,
		), kindSlice
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
			quoteMode,
			quoteOpen,
			quoteClose,
			quoteSmart,
			fmts,
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

	if s, ok := formatCompositeDuration(d); ok {
		return s
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

	if s, ok := formatCompositeDuration(d); ok {
		return s
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

func formatCompositeDuration(d time.Duration) (string, bool) {
	if d >= time.Hour {
		h := int(d / time.Hour)
		remainder := d - time.Duration(h)*time.Hour
		m := int(remainder / time.Minute)
		if m == 0 {
			return strconv.Itoa(h) + "h", true
		}
		return strconv.Itoa(h) + "h" + strconv.Itoa(m) + "m", true
	}

	if d >= time.Minute {
		m := int(d / time.Minute)
		remainder := d - time.Duration(m)*time.Minute
		s := int(remainder / time.Second)
		if s == 0 {
			return strconv.Itoa(m) + "m", true
		}
		return strconv.Itoa(m) + "m" + strconv.Itoa(s) + "s", true
	}

	return "", false
}

func percentFraction(p core.Percent, maximum float64) float64 {
	return p.Value / percent.EffectiveMaximum(p, maximum)
}

// quoteDelim returns the configured quote-delimiter style, or nil when styling
// is inactive (no color, level below the style threshold) or no FieldQuote
// style is set.
func quoteDelim(opts formatFieldsOpts) *style.QuoteStyle {
	if opts.noColor || opts.level < opts.fieldStyleLevel || opts.styles == nil {
		return nil
	}
	return opts.styles.FieldQuote
}

// valueBaseStyle returns the style that styleValue would apply to a value of
// the given kind (key > value > type priority), or nil when none applies. Only
// the quotable kinds (string, error, time, default) are resolved; other kinds
// never reach the quote-delimiter path. It is the base for QuoteStyle.Resolve.
func valueBaseStyle(originalValue any, key string, kind valueKind, styles *style.Config) Style {
	if styles == nil {
		return nil
	}
	if key != "" {
		if st := styles.Keys[key]; st != nil {
			return st
		}
	}
	if st := lookupValueStyle(originalValue, styles.Values); st != nil {
		return st
	}
	switch kind { //nolint:exhaustive // only quotable kinds have a base style here
	case kindString:
		return styles.FieldString
	case kindError:
		return styles.FieldError
	case kindTime:
		return styles.FieldTime
	default:
		return nil
	}
}

// styledFieldValue applies styling to a formatted field value.
// Returns the styled string, or the plain valStr if no styling applies.
func styledFieldValue(f Field, valStr string, kind valueKind, opts formatFieldsOpts) string {
	if opts.noColor || opts.level < opts.fieldStyleLevel {
		return valStr
	}

	fmts := opts.fieldFormats()

	// KeyStyles takes priority over per-element styling for slices.
	if kind == kindSlice {
		if style := opts.styles.Keys[f.Key]; style != nil {
			return style.Render(valStr)
		}
		return styledSlice(
			f.Value,
			opts.sliceFmt(),
			opts.styles,
			opts.quoteMode,
			opts.quoteOpen,
			opts.quoteClose,
			opts.quoteSmart,
			fmts,
		)
	}

	// String values may carry `code` spans; render them so backticked text takes
	// the Backtick style over the value's own resolved base style. Errors are left
	// verbatim in their own style - a backtick in an error message is content.
	if kind == kindString && opts.styles != nil && opts.styles.Backtick != nil {
		base := valueBaseStyle(f.Value, f.Key, kind, opts.styles)
		return opts.styles.BacktickMode.Render(valStr, base, opts.styles.Backtick)
	}

	if styled := styleValue(valStr, f.Value, f.Key, kind, opts.styles, fmts); styled != "" {
		return styled
	}
	return valStr
}

// styledSlice re-formats a slice value with per-element styling.
func styledSlice(
	v any,
	sf sliceFormat,
	styles *style.Config,
	quoteMode Quote,
	quoteOpen, quoteClose rune,
	quoteSmart []QuotePair,
	fmts *FieldFormats,
) string {
	switch vals := v.(type) {
	case []bool:
		return formatBoolSlice(vals, sf, styles)
	case []time.Duration:
		return formatDurationSlice(vals, sf, styles, fmts)
	case []core.QuantityField:
		return formatQuantitySlice(vals, sf, styles, fmts.QuantityUnitsIgnoreCase)
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
		return formatStringSlice(vals, sf, styles, quoteMode, quoteOpen, quoteClose, quoteSmart)
	case []any:
		return formatAnySlice(
			vals,
			sf,
			styles,
			quoteMode,
			quoteOpen,
			quoteClose,
			quoteSmart,
			fmts,
		)
	default:
		s, _ := formatValue(v, sf, quoteMode, quoteOpen, quoteClose, quoteSmart, "", fmts)
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
	fmts *FieldFormats,
) string {
	// Per-key styling takes priority. Slice elements pass no key.
	if key != "" {
		if style := styles.Keys[key]; style != nil {
			return style.Render(valStr)
		}
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
		if styled := styleDuration(
			valStr,
			originalValue,
			styles,
			fmts.DurationGradientMax,
		); styled != "" {
			return styled
		}
	case kindElapsed:
		if styled := styleElapsed(
			valStr,
			originalValue,
			styles,
			fmts.ElapsedGradientMax,
		); styled != "" {
			return styled
		}
	case kindFraction:
		if styled := styleFraction(
			valStr,
			originalValue,
			styles,
			fmts.PercentReverseGradient,
		); styled != "" {
			return styled
		}
	case kindPercent:
		styled := stylePercent(
			valStr,
			originalValue,
			styles,
			fmts.PercentReverseGradient,
			fmts.PercentMaximum,
		)
		if styled != "" {
			return styled
		}
	case kindQuantity:
		if styled := styleQuantity(valStr, styles, fmts.QuantityUnitsIgnoreCase); styled != "" {
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

// quoteString wraps s in quotes (see [quoteParts] for delimiter selection).
func quoteString(s string, openChar, closeChar rune, smart []QuotePair) string {
	open, body, closing := quoteParts(s, openChar, closeChar, smart)
	return open + body + closing
}

// quoteParts splits the quoting of s into its opening delimiter, body, and
// closing delimiter, so callers can style the delimiters independently. The
// body is the (possibly escaped) inner text. When smart is non-empty, it
// selects the first pair whose delimiters do not occur in s (see
// [smartQuoteParts]), ignoring open/close. Otherwise, when open is 0 it uses
// [strconv.Quote] (Go-style escaping), and when open is non-zero it wraps with
// the open/close runes (open is used for both sides when close is 0).
func quoteParts(
	s string,
	openChar, closeChar rune,
	smart []QuotePair,
) (string, string, string) {
	if len(smart) > 0 {
		return smartQuoteParts(s, smart)
	}

	if openChar == 0 {
		return escapedQuoteParts(s)
	}

	if closeChar == 0 {
		closeChar = openChar
	}
	return string(openChar), s, string(closeChar)
}

// smartQuoteParts selects the first pair from prefs whose delimiters do not
// occur in s, so the body needs no escaping. A pair with a zero Close uses its
// Open for both sides. Since literal wrapping cannot escape anything, this only
// applies when s is raw-quotable (no backslash or non-printable runes);
// otherwise, and when every pair collides with s, it falls back to
// [escapedQuoteParts] (Go-style escaping).
func smartQuoteParts(s string, prefs []QuotePair) (string, string, string) {
	if rawQuotable(s) {
		for _, p := range prefs {
			openChar, closeChar := p.Open, p.Close
			if closeChar == 0 {
				closeChar = openChar
			}
			if strings.ContainsRune(s, openChar) || strings.ContainsRune(s, closeChar) {
				continue
			}
			return string(openChar), s, string(closeChar)
		}
	}
	return escapedQuoteParts(s)
}

// escapedQuoteParts returns the parts of [strconv.Quote]: the surrounding
// double quotes and the escaped body between them.
func escapedQuoteParts(s string) (string, string, string) {
	q := strconv.Quote(s)
	return `"`, q[1 : len(q)-1], `"`
}

// rawQuotable reports whether s can be wrapped in literal delimiters without
// escaping: it must contain no backslash and no non-printable runes (control
// characters, tabs, newlines). Mirrors the printability check in needsQuoting.
func rawQuotable(s string) bool {
	for _, r := range s {
		if r == '\\' || !strconv.IsPrint(r) {
			return false
		}
	}
	return true
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
