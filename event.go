package clog

import (
	"fmt"
	"reflect"
	"time"
)

// Event represents a log event being constructed. All methods are safe
// to call on a nil receiver — disabled events (when the log level is
// below the logger's minimum) are no-ops.
type Event struct {
	logger *Logger
	level  Level
	fields []Field
	prefix *string // nil = use logger/default prefix
}

// Any adds a field with an arbitrary value.
func (e *Event) Any(key string, val any) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Anys adds a slice of arbitrary values. Individual elements are
// highlighted using reflection to determine their type.
func (e *Event) Anys(key string, vals []any) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Bool adds a bool field.
func (e *Event) Bool(key string, val bool) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Bools adds a bool slice field.
func (e *Event) Bools(key string, vals []bool) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (e *Event) Column(key, path string, line, column int) *Event {
	if e == nil {
		return e
	}

	if line < 1 {
		line = 1
	}

	if column < 1 {
		column = 1
	}

	output := Default.Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, line, column)},
	)
	return e
}

// Dict adds a group of fields under a key prefix using dot notation.
// Build the nested fields using [Dict] to create a field-only Event:
//
//	clog.Info().Dict("request", clog.Dict().
//	    Str("method", "GET").
//	    Int("status", 200),
//	).Msg("handled")
//	// Output: INF ℹ️ handled request.method=GET request.status=200
func (e *Event) Dict(key string, dict *Event) *Event {
	if e == nil || dict == nil {
		return e
	}

	for _, f := range dict.fields {
		e.fields = append(e.fields, Field{Key: key + "." + f.Key, Value: f.Value})
	}
	return e
}

// Duration adds a [time.Duration] field.
func (e *Event) Duration(key string, val time.Duration) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Durations adds a [time.Duration] slice field.
func (e *Event) Durations(key string, vals []time.Duration) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Err adds an error field with key "error". No-op if err is nil.
func (e *Event) Err(err error) *Event {
	if e == nil || err == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: ErrorKey, Value: err})
	return e
}

// Float64 adds a float64 field.
func (e *Event) Float64(key string, val float64) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Floats64 adds a float64 slice field.
func (e *Event) Floats64(key string, vals []float64) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Int adds an int field.
func (e *Event) Int(key string, val int) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Ints adds an int slice field.
func (e *Event) Ints(key string, vals []int) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Int64 adds an int64 field.
func (e *Event) Int64(key string, val int64) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (e *Event) Line(key, path string, line int) *Event {
	if e == nil {
		return e
	}

	if line < 1 {
		line = 1
	}

	output := Default.Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, line, 0)},
	)
	return e
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Respects the logger's [ColorMode] setting.
func (e *Event) Link(key, url, text string) *Event {
	if e == nil {
		return e
	}

	output := Default.Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.hyperlink(url, text)},
	)
	return e
}

// Msg finalises the event and writes the log entry.
// For [FatalLevel] events, Msg calls [os.Exit](1) after writing.
func (e *Event) Msg(msg string) {
	if e == nil {
		return
	}

	if e.logger == nil {
		panic("clog: Msg/Msgf/Send called on a Dict() event -- pass it to Event.Dict() instead")
	}

	e.logger.log(e, msg)

	if e.level == FatalLevel {
		e.logger.exit(1)
	}
}

// Msgf finalises the event with a formatted message.
func (e *Event) Msgf(format string, args ...any) {
	if e == nil {
		return
	}

	e.Msg(fmt.Sprintf(format, args...))
}

// Percent adds a percentage field (0–100) with gradient color styling.
// Values are clamped to the 0–100 range. The color is interpolated from
// the [Styles.PercentGradient] stops (default: red → yellow → green).
func (e *Event) Percent(key string, val float64) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: percent(clampPercent(val))})
	return e
}

// Path adds a file path field as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (e *Event) Path(key, path string) *Event {
	if e == nil {
		return e
	}

	output := Default.Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, 0, 0)},
	)
	return e
}

// Prefix overrides the default emoji prefix for this entry.
func (e *Event) Prefix(prefix string) *Event {
	if e == nil {
		return e
	}

	e.prefix = new(prefix)
	return e
}

// Quantities adds a quantity string slice field. Each element is styled
// with [Styles.FieldQuantityNumber] and [Styles.FieldQuantityUnit].
func (e *Event) Quantities(key string, vals []string) *Event {
	if e == nil {
		return e
	}

	q := make([]quantity, len(vals))
	for i, v := range vals {
		q[i] = quantity(v)
	}
	e.fields = append(e.fields, Field{Key: key, Value: q})
	return e
}

// Quantity adds a quantity string field where numeric and unit segments are
// styled independently (e.g. "5m", "5.1km", "100MB").
// The value is styled with [Styles.FieldQuantityNumber] and [Styles.FieldQuantityUnit].
func (e *Event) Quantity(key, val string) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: quantity(val)})
	return e
}

// Send finalises the event with an empty message.
func (e *Event) Send() {
	e.Msg("")
}

// Str adds a string field.
func (e *Event) Str(key, val string) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Stringer adds a field by calling the value's String method. No-op if val is nil.
func (e *Event) Stringer(key string, val fmt.Stringer) *Event {
	if e == nil || isNilStringer(val) {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val.String()})
	return e
}

// Stringers adds a field with a slice of [fmt.Stringer] values.
func (e *Event) Stringers(key string, vals []fmt.Stringer) *Event {
	if e == nil {
		return e
	}

	strs := make([]string, len(vals))
	for i, v := range vals {
		if isNilStringer(v) {
			strs[i] = Nil
		} else {
			strs[i] = v.String()
		}
	}

	e.fields = append(e.fields, Field{Key: key, Value: strs})
	return e
}

// Strs adds a string slice field.
func (e *Event) Strs(key string, vals []string) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Time adds a [time.Time] field.
func (e *Event) Time(key string, val time.Time) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Uint adds a uint field.
func (e *Event) Uint(key string, val uint) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Uint64 adds a uint64 field.
func (e *Event) Uint64(key string, val uint64) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Uints64 adds a uint64 slice field.
func (e *Event) Uints64(key string, vals []uint64) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// URL adds a field as a clickable terminal hyperlink where the URL is also the display text.
// Respects the logger's [ColorMode] setting.
func (e *Event) URL(key, url string) *Event {
	if e == nil {
		return e
	}

	output := Default.Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.hyperlink(url, url)},
	)
	return e
}

// withFields appends pre-existing fields to the event (used internally).
func (e *Event) withFields(fields []Field) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, fields...)
	return e
}

// withPrefix sets the prefix on the event (used internally).
func (e *Event) withPrefix(prefix string) *Event {
	if e == nil {
		return e
	}

	e.prefix = new(prefix)
	return e
}

// isNilStringer reports whether val is nil, either as an untyped nil interface
// or as a typed nil whose underlying kind supports IsNil.
func isNilStringer(val fmt.Stringer) bool {
	if val == nil {
		return true
	}

	rv := reflect.ValueOf(val)
	//nolint:exhaustive // only nilable kinds need checking
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func:
		return rv.IsNil()
	default:
		return false
	}
}
