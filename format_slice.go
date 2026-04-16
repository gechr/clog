package clog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gechr/clog/field/duration"
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
func formatDurationSlice(vals []time.Duration, sf sliceFormat, styles *style.Config) string {
	stringify := time.Duration.String
	if fn := duration.FormatFunc(); fn != nil {
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
			return styleDuration(s, v, st)
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
) string {
	var buf strings.Builder

	buf.WriteString(sf.open)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sf.sep)
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

	buf.WriteString(sf.close)
	return buf.String()
}

// formatAnySlice formats a []any slice with per-element
// styling. Uses reflection to determine each element's type for highlighting.
func formatAnySlice(
	vals []any,
	sf sliceFormat,
	styles *style.Config,
	ignoreCase bool,
	quoteMode Quote,
	quoteOpen, quoteClose rune,
	percentReverse bool,
) string {
	var buf strings.Builder

	buf.WriteString(sf.open)

	for i, v := range vals {
		if i > 0 {
			buf.WriteString(sf.sep)
		}

		s := fmt.Sprintf("%v", v)
		kind := reflectValueKind(v)

		if quoteMode != QuoteNever &&
			(kind == kindDefault || kind == kindString) &&
			(quoteMode == QuoteAlways || needsQuoting(s)) {
			s = quoteString(s, quoteOpen, quoteClose)
		}

		if styles != nil {
			styled := styleAnyElement(s, v, kind, styles, ignoreCase, percentReverse)
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
	ignoreCase bool,
	percentReverse bool,
) string {
	// Per-value styling (typed key lookup - bool true ≠ string "true").
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
		if styled := styleDuration(s, originalValue, styles); styled != "" {
			return styled
		}
	case kindElapsed:
		if styled := styleElapsed(s, originalValue, styles); styled != "" {
			return styled
		}
	case kindPercent:
		if styled := stylePercent(s, originalValue, styles, percentReverse); styled != "" {
			return styled
		}
	case kindQuantity:
		if styled := styleQuantity(s, styles, ignoreCase); styled != "" {
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
