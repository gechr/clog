package clog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
)

// sliceFormat holds the configurable open, close, and separator strings
// for rendering slice field values.
type sliceFormat struct {
	open  string // e.g. "["
	close string // e.g. "]"
	sep   string // e.g. ", "
}

// formatSlice formats any slice with configurable separation and optional per-element styling.
// stringify converts each element to its string representation.
// stylize returns a styled string, or "" to fall back to the plain string.
func formatSlice[T any](
	vals []T,
	sf sliceFormat,
	styles *style.Config,
	stringify func(T) string,
	stylize func(T, string, *style.Config) string,
) string {
	var buf strings.Builder

	buf.WriteString(sf.open)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sf.sep)
		}

		s := stringify(v)
		if styled := stylize(v, s, styles); styled != "" {
			buf.WriteString(styled)
		} else {
			buf.WriteString(s)
		}
	}

	buf.WriteString(sf.close)
	return buf.String()
}

// numberSliceStyle is a stylize function for numeric slice elements.
// It applies Styles.FieldNumber when set.
func numberSliceStyle[T any](_ T, s string, styles *style.Config) string {
	if styles != nil && styles.FieldNumber != nil {
		return styles.FieldNumber.Render(s)
	}
	return ""
}

// formatBoolSlice formats a bool slice.
// When styles is non-nil, individual elements are styled via ValueStyles.
func formatBoolSlice(vals []bool, sf sliceFormat, styles *style.Config) string {
	return formatSlice(
		vals,
		sf,
		styles,
		strconv.FormatBool,
		func(v bool, s string, st *style.Config) string {
			if st != nil {
				if style := st.Values[v]; style != nil {
					return style.Render(s)
				}
			}
			return ""
		},
	)
}

// formatDurationSlice formats a [time.Duration] slice.
// When styles is non-nil, individual elements are styled via [styleDuration].
func formatDurationSlice(
	vals []time.Duration,
	sf sliceFormat,
	styles *style.Config,
	fmts *FieldFormats,
) string {
	stringify := time.Duration.String
	if fn := fmts.DurationFormat; fn != nil {
		stringify = fn
	}
	return formatSlice(
		vals,
		sf,
		styles,
		stringify,
		func(v time.Duration, s string, st *style.Config) string {
			if st == nil {
				return ""
			}
			return styleDuration(s, v, st, fmts.DurationGradientMax)
		},
	)
}

// formatFloat64Slice formats a float64 slice.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatFloat64Slice(vals []float64, sf sliceFormat, styles *style.Config) string {
	return formatSlice(vals, sf, styles,
		func(v float64) string {
			return strconv.FormatFloat(v, 'f', -1, 64)
		},
		numberSliceStyle[float64],
	)
}

// formatIntSlice formats an int slice.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatIntSlice(vals []int, sf sliceFormat, styles *style.Config) string {
	return formatSlice(vals, sf, styles, strconv.Itoa, numberSliceStyle[int])
}

// formatInt64Slice formats an int64 slice.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatInt64Slice(vals []int64, sf sliceFormat, styles *style.Config) string {
	return formatSlice(vals, sf, styles,
		func(v int64) string {
			return strconv.FormatInt(v, 10)
		},
		numberSliceStyle[int64],
	)
}

// formatUintSlice formats a uint slice.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatUintSlice(vals []uint, sf sliceFormat, styles *style.Config) string {
	return formatSlice(vals, sf, styles,
		func(v uint) string {
			return strconv.FormatUint(uint64(v), 10)
		},
		numberSliceStyle[uint],
	)
}

// formatUint64Slice formats a uint64 slice.
// When styles is non-nil, individual elements are styled via FieldNumber.
func formatUint64Slice(vals []uint64, sf sliceFormat, styles *style.Config) string {
	return formatSlice(vals, sf, styles,
		func(v uint64) string {
			return strconv.FormatUint(v, 10)
		},
		numberSliceStyle[uint64],
	)
}

// formatQuantitySlice formats a quantity slice.
// When styles is non-nil, individual elements are styled via [styleQuantity].
func formatQuantitySlice(
	vals []core.QuantityField,
	sf sliceFormat,
	styles *style.Config,
	ignoreCase bool,
) string {
	return formatSlice(
		vals,
		sf,
		styles,
		func(v core.QuantityField) string {
			return string(v)
		},
		func(_ core.QuantityField, s string, st *style.Config) string {
			if st == nil {
				return ""
			}
			return styleQuantity(s, st, ignoreCase)
		},
	)
}

// formatStringSlice formats a string slice with per-element quoting.
// When styles is non-nil, individual elements are styled via ValueStyles.
func formatStringSlice(
	vals []string,
	sf sliceFormat,
	styles *style.Config,
	quoteMode Quote,
	quoteOpen, quoteClose rune,
	quoteSmart []QuotePair,
) string {
	var buf strings.Builder

	buf.WriteString(sf.open)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sf.sep)
		}

		quoted := quoteMode != QuoteNever && (quoteMode == QuoteAlways || needsQuoting(v))

		// When a quote-delimiter style is set, style the delimiters separately
		// from the body; otherwise style the whole quoted element as one unit.
		if quoted && styles != nil && styles.FieldQuote != nil {
			open, body, closing := quoteParts(v, quoteOpen, quoteClose, quoteSmart)
			delim := styles.FieldQuote.Resolve(valueBaseStyle(v, "", kindString, styles))
			buf.WriteString(delim.Render(open))
			buf.WriteString(styleStringElem(body, v, styles))
			buf.WriteString(delim.Render(closing))

			continue
		}

		display := v
		if quoted {
			display = quoteString(v, quoteOpen, quoteClose, quoteSmart)
		}
		buf.WriteString(styleStringElem(display, v, styles))
	}

	buf.WriteString(sf.close)
	return buf.String()
}

// styleStringElem styles a string slice element s (keyed by its raw value v):
// a per-value style takes priority, then FieldString, else s is returned plain.
func styleStringElem(s, v string, styles *style.Config) string {
	if styles != nil {
		if style := styles.Values[v]; style != nil {
			return style.Render(s)
		}
		if styles.FieldString != nil {
			return styles.FieldString.Render(s)
		}
	}
	return s
}

// formatAnySlice formats a []any slice with per-element
// styling. Uses reflection to determine each element's type for highlighting.
func formatAnySlice(
	vals []any,
	sf sliceFormat,
	styles *style.Config,
	quoteMode Quote,
	quoteOpen, quoteClose rune,
	quoteSmart []QuotePair,
	fmts *FieldFormats,
) string {
	var buf strings.Builder

	buf.WriteString(sf.open)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sf.sep)
		}

		s := fmt.Sprintf("%v", v)
		kind := reflectValueKind(v)

		quoted := quoteMode != QuoteNever &&
			(kind == kindDefault || kind == kindString) &&
			(quoteMode == QuoteAlways || needsQuoting(s))

		// When a quote-delimiter style is set, style the delimiters separately
		// from the body; otherwise style the whole quoted element as one unit.
		if quoted && styles != nil && styles.FieldQuote != nil {
			open, body, closing := quoteParts(s, quoteOpen, quoteClose, quoteSmart)
			delim := styles.FieldQuote.Resolve(valueBaseStyle(v, "", kind, styles))
			styledBody := body
			if styled := styleAnyElement(body, v, kind, styles, fmts); styled != "" {
				styledBody = styled
			}
			buf.WriteString(delim.Render(open))
			buf.WriteString(styledBody)
			buf.WriteString(delim.Render(closing))

			continue
		}

		if quoted {
			s = quoteString(s, quoteOpen, quoteClose, quoteSmart)
		}

		if styles != nil {
			styled := styleAnyElement(s, v, kind, styles, fmts)
			if styled != "" {
				buf.WriteString(styled)

				continue
			}
		}

		buf.WriteString(s)
	}

	buf.WriteString(sf.close)
	return buf.String()
}

// styleAnyElement applies the appropriate style to a single element in a []any slice.
// originalValue is the pre-format typed value for typed Values map lookups.
func styleAnyElement(
	s string,
	originalValue any,
	kind valueKind,
	styles *style.Config,
	fmts *FieldFormats,
) string {
	return styleValue(s, originalValue, "", kind, styles, fmts)
}
