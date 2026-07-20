package clog

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gechr/clog/field/deadline"
	"github.com/gechr/clog/field/duration"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/field/fraction"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/internal/core"
	xmath "github.com/gechr/x/math"
)

// Event represents a log event being constructed. All methods are safe
// to call on a nil receiver - disabled events (when the log level is
// below the logger's minimum) are no-ops.
type Event struct {
	logger *Logger

	deadlineStart time.Time // set by Deadline(); zero means no deadline field
	dict          bool      // true for events created by Dict() (must not call Msg/Send)
	elapsedStart  time.Time // set by Elapsed(); zero means no elapsed field
	err           error     // set by Err(); used as message by Send(), or as error= field by Msg()
	exitCode      int       // exit code for Fatal-level events; 0 means default (1)
	fields        []Field
	level         Level
	msgStyle      Style     // nil = use logger/global message style
	noExit        bool      // if true, skip exit even for LevelFatal (used by adapters)
	noTimestamp   bool      // if true, render no timestamp even when reporting is enabled (adapter passed a zero timestamp)
	omitEmpty     *bool     // nil = use logger's omitEmpty
	omitZero      *bool     // nil = use logger's omitZero
	parts         *[]Part   // nil = use logger's parts
	sort          *Sort     // nil = use logger's fieldSort
	symbol        *string   // nil = use logger/default symbol
	timestamp     time.Time // if non-zero, overrides the time.Now() value in Logger.log(); rendered only when timestamp reporting is enabled
}

// NOTE: The field methods below intentionally duplicate FieldBuilder[T] methods.
// Event cannot embed FieldBuilder because it needs nil-receiver no-ops and
// returns *Event (not *T). Keep both sets in sync when adding new field types.

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

// Base64 adds a []byte field encoded as a base64 string.
func (e *Event) Base64(key string, val []byte) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: base64.StdEncoding.EncodeToString(val)})
	return e
}

// Bytes adds a []byte field. If val is valid JSON it is stored as [RawJSON]
// with syntax highlighting; otherwise it is stored as a plain string.
func (e *Event) Bytes(key string, val []byte) *Event {
	if e == nil {
		return e
	}

	if json.Valid(val) {
		e.fields = append(e.fields, Field{Key: key, Value: core.RawJSON(val)})
	} else {
		e.fields = append(e.fields, Field{Key: key, Value: string(val)})
	}
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

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, line, column)},
	)
	return e
}

// Column represents a file path with a line and column number for use with [Event.Columns].
type Column struct {
	Path   string
	Line   int
	Column int
}

// Columns adds a string slice field where each element is a path:line:column
// hyperlink. Respects the logger's [ColorMode] setting.
func (e *Event) Columns(key string, items []Column) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	vals := make([]string, len(items))
	for i, item := range items {
		line, column := item.Line, item.Column
		if line < 1 {
			line = 1
		}
		if column < 1 {
			column = 1
		}
		vals[i] = output.pathLink(item.Path, line, column)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
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

// Duration adds a [time.Duration] field. Use options from the [duration]
// package (e.g. [duration.WithGradientMax]) to override the logger's
// gradient settings for this field only:
//
//	clog.Info().Duration("latency", d, duration.WithGradientMax(3*time.Second))
func (e *Event) Duration(key string, val time.Duration, opts ...duration.Option) *Event {
	if e == nil {
		return e
	}

	f := core.DurationField{Value: val}
	duration.Apply(&f, opts...)
	e.fields = append(e.fields, Field{Key: key, Value: f})
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

// Errs adds an error slice field. Each error is converted to its message
// string; nil errors are rendered as [Nil] ("<nil>").
func (e *Event) Errs(key string, vals []error) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: core.ErrSliceToStrings(vals)})
	return e
}

// AnErr adds an error as a keyed field. No-op if err is nil.
// Unlike [Event.Err], this does not interact with [Event.Send] or [Event.Msg]
// semantics - the error is simply added as a regular field with the given key.
func (e *Event) AnErr(key string, err error) *Event {
	if e == nil || err == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: err})
	return e
}

// Err attaches an error to the event. No-op if err is nil.
//
// If the event is finalised with [Event.Send], the error message becomes the
// log message with no extra fields. If finalised with [Event.Msg] or
// [Event.Msgf], the error is added as an "error" field alongside the message.
func (e *Event) Err(err error) *Event {
	if e == nil || err == nil {
		return e
	}

	e.err = err
	return e
}

// ExitCode sets the exit code for [LevelFatal] events.
// The default exit code is 1. This has no effect on non-fatal events.
func (e *Event) ExitCode(code int) *Event {
	if e == nil {
		return e
	}
	e.exitCode = code
	return e
}

// Func executes fn with the event if the event is enabled (non-nil).
// This is useful for computing expensive fields lazily - the callback
// is skipped entirely when the log level is disabled.
func (e *Event) Func(fn func(*Event)) *Event {
	if e == nil {
		return e
	}
	fn(e)
	return e
}

// When calls fn with the event if condition is true and the event is
// enabled (non-nil). This is useful for conditionally adding fields
// without breaking the chain.
func (e *Event) When(condition bool, fn func(*Event)) *Event {
	if e == nil {
		return e
	}
	if condition && fn != nil {
		fn(e)
	}
	return e
}

// Discard disables the event so Msg/Msgf/Send won't produce output.
// Returns nil to short-circuit subsequent field methods.
func (e *Event) Discard() *Event {
	return nil
}

// Disabled returns true if the event is disabled (nil).
func (e *Event) Disabled() bool {
	return e == nil
}

// Enabled returns true if the event is enabled (non-nil).
func (e *Event) Enabled() bool {
	return e != nil
}

// Float32 adds a float32 field.
func (e *Event) Float32(key string, val float32) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
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

// Floats32 adds a float32 slice field.
func (e *Event) Floats32(key string, vals []float32) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
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

// Hex adds a []byte field encoded as a hex string.
func (e *Event) Hex(key string, val []byte) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: hex.EncodeToString(val)})
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

// Int8 adds an int8 field.
func (e *Event) Int8(key string, val int8) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Int16 adds an int16 field.
func (e *Event) Int16(key string, val int16) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Int32 adds an int32 field.
func (e *Event) Int32(key string, val int32) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
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

// Ints adds an int slice field.
func (e *Event) Ints(key string, vals []int) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Ints8 adds an int8 slice field.
func (e *Event) Ints8(key string, vals []int8) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Ints16 adds an int16 slice field.
func (e *Event) Ints16(key string, vals []int16) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Ints32 adds an int32 slice field.
func (e *Event) Ints32(key string, vals []int32) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Ints64 adds an int64 slice field.
func (e *Event) Ints64(key string, vals []int64) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting. If line < 1, the line number is
// omitted and the field is rendered as a plain path hyperlink (equivalent to
// [Event.Path]).
func (e *Event) Line(key, path string, line int) *Event {
	if e == nil {
		return e
	}

	if line < 1 {
		return e.Path(key, path)
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, line, 0)},
	)
	return e
}

// Line represents a file path with a line number for use with [Event.Lines].
type Line struct {
	Path string
	Line int
}

// Lines adds a string slice field where each element is a path:line
// hyperlink. If an item's Line < 1, that element is rendered as a plain path
// hyperlink (equivalent to [Event.Path]). Respects the logger's [ColorMode]
// setting.
func (e *Event) Lines(key string, items []Line) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	vals := make([]string, len(items))
	for i, item := range items {
		vals[i] = output.pathLink(item.Path, item.Line, 0)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Link represents a hyperlink with a URL and display text.
type Link struct {
	URL  string
	Text string
}

// Link adds a single hyperlink field.
func (e *Event) Link(key, url, text string) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.hyperlink(url, text)},
	)
	return e
}

// Links adds a string slice field where each element is a hyperlink.
func (e *Event) Links(key string, links []Link) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	vals := make([]string, len(links))
	for i, l := range links {
		vals[i] = output.hyperlink(l.URL, l.Text)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Msg finalises the event and writes the log entry.
// If [Event.Err] was called, the error is included as an "error" field.
// For [LevelFatal] events, Msg calls [os.Exit] after writing.
// The exit code defaults to 1, but can be changed with [Event.ExitCode]
// or [Logger.SetExitCode].
func (e *Event) Msg(msg string) {
	if e == nil {
		return
	}

	if e.dict {
		panic("clog: Msg/Msgf/Send called on a Dict() event -- pass it to Event.Dict() instead")
	}

	e.resolveElapsed()
	e.resolveDeadline()

	if e.err != nil {
		e.fields = append(e.fields, Field{Key: ErrorKey, Value: e.err})
	}

	e.logger.log(e, msg)

	if e.level == LevelFatal && !e.noExit {
		code := e.exitCode
		if code == 0 {
			code = 1
		}
		e.logger.exit(code)
	}
}

// Msgf finalises the event with a formatted message.
func (e *Event) Msgf(format string, args ...any) {
	if e == nil {
		return
	}

	e.Msg(fmt.Sprintf(format, args...))
}

// MsgFunc finalises the event with a lazily-computed message.
// The function is only called if the event is enabled (non-nil).
func (e *Event) MsgFunc(createMsg func() string) {
	if e == nil {
		return
	}

	e.Msg(createMsg())
}

// Fraction adds a current/total field with gradient color styling.
// The color is interpolated from the [style.Config.PercentGradient] stops
// (default: red → yellow → green) based on current/total progress.
// Current is clamped to [0, total].
func (e *Event) Fraction(key string, current, total int, opts ...fraction.Option) *Event {
	if e == nil {
		return e
	}
	current = xmath.Clamp(current, 0, total)
	f := core.Fraction{Current: current, Total: total}
	fraction.Apply(&f, opts...)
	e.fields = append(e.fields, Field{Key: key, Value: f})
	return e
}

// Percent adds a percentage field with gradient color styling.
// Values are clamped to [0, Maximum] (default maximum is 1, so 0.75 → "75%").
// The color is interpolated from the [style.Config.PercentGradient] stops
// (default: red → yellow → green).
//
// Use [percent.WithReverseGradient] to flip the gradient for this field:
//
//	e.Percent("cpu", usage, percent.WithReverseGradient())
//
// Use [percent.WithMaximum] to override the input range for this field:
//
//	e.Percent("progress", 75, percent.WithMaximum(100))
//
// fieldFormats returns the owning logger's field-format snapshot,
// falling back to the [Default] logger.
func (e *Event) fieldFormats() *FieldFormats {
	if e.logger != nil {
		return e.logger.loadFieldFormats()
	}
	return Default().loadFieldFormats()
}

func (e *Event) Percent(key string, val float64, opts ...percent.Option) *Event {
	if e == nil {
		return e
	}

	p := core.Percent{Value: val}
	percent.Apply(&p, opts...)
	p.Value = core.ClampPercent(
		p.Value,
		percent.EffectiveMaximum(p, e.fieldFormats().PercentMaximum),
	)
	e.fields = append(e.fields, Field{Key: key, Value: p})
	return e
}

// Path adds a file path field as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (e *Event) Path(key, path string) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, 0, 0)},
	)
	return e
}

// PathText adds a file path field as a clickable terminal hyperlink whose
// visible label is text rather than path. The link still targets path, so a
// caller can show an abbreviated or home-contracted path (e.g. ~/bin/foo)
// while linking to its full location. Respects the logger's [ColorMode]
// setting.
func (e *Event) PathText(key, text, path string) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLinkText(text, path, 0, 0)},
	)
	return e
}

// Paths adds a string slice field where each element is a path hyperlink.
// Respects the logger's [ColorMode] setting.
func (e *Event) Paths(key string, paths []string) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	vals := make([]string, len(paths))
	for i, p := range paths {
		vals[i] = output.pathLink(p, 0, 0)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// RawJSON adds a field with pre-serialized JSON bytes, emitted verbatim
// without quoting or escaping. The bytes must be valid JSON.
func (e *Event) RawJSON(key string, val []byte) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: core.RawJSON(val)})
	return e
}

// JSON marshals val to JSON and adds it as a highlighted field.
// On marshal error the field value is the error string.
func (e *Event) JSON(key string, val any) *Event {
	if e == nil {
		return e
	}

	b, err := json.Marshal(val)
	if err != nil {
		e.fields = append(e.fields, Field{Key: key, Value: err.Error()})
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: core.RawJSON(b)})
	return e
}

// OmitEmpty overrides the logger's omit-empty setting for this entry.
// Empty means nil, empty strings, and nil or empty slices/maps.
func (e *Event) OmitEmpty(omit bool) *Event {
	if e == nil {
		return e
	}

	e.omitEmpty = new(omit)
	return e
}

// OmitZero overrides the logger's omit-zero setting for this entry.
// Zero means the zero value for any type (0, false, "", nil, etc.).
// This is a superset of [Event.OmitEmpty].
func (e *Event) OmitZero(omit bool) *Event {
	if e == nil {
		return e
	}

	e.omitZero = new(omit)
	return e
}

// Parts overrides the log-line part order for this entry.
// Parts not included are hidden. This does not affect the logger's global parts.
func (e *Event) Parts(parts ...Part) *Event {
	if e == nil {
		return e
	}

	e.parts = new(parts)
	return e
}

// Sort overrides the logger's field sort order for this entry.
// Default [SortNone] preserves insertion order.
func (e *Event) Sort(sort Sort) *Event {
	if e == nil {
		return e
	}

	e.sort = new(sort)
	return e
}

// Symbol overrides the default emoji symbol for this entry.
func (e *Event) Symbol(symbol string) *Event {
	if e == nil {
		return e
	}

	e.symbol = new(symbol)
	return e
}

// MessageStyle overrides the message text style for this entry, taking
// precedence over the global [style.Config.Message] and per-level
// [style.Config.Messages]. It lets a caller style one line without mutating
// global styles; a nil style falls back to those globals, and an empty
// [lipgloss.NewStyle] renders the message plain (the level colour does not
// leak, since this replaces the level style rather than nesting inside it).
func (e *Event) MessageStyle(s Style) *Event {
	if e == nil {
		return e
	}

	e.msgStyle = s
	return e
}

// Quantities adds a quantity string slice field. Each element is styled
// with [style.Config.FieldQuantityNumber] and [style.Config.FieldQuantityUnit].
func (e *Event) Quantities(key string, vals []string) *Event {
	if e == nil {
		return e
	}

	q := make([]core.QuantityField, len(vals))
	for i, v := range vals {
		q[i] = core.QuantityField(v)
	}
	e.fields = append(e.fields, Field{Key: key, Value: q})
	return e
}

// Quantity adds a quantity string field where numeric and unit segments are
// styled independently (e.g. "5m", "5.1km", "100MB").
// The value is styled with [style.Config.FieldQuantityNumber] and [style.Config.FieldQuantityUnit].
func (e *Event) Quantity(key, val string) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: core.QuantityField(val)})
	return e
}

// Send finalises the event. If [Event.Err] was called, the error message is
// used as the log message (no "error" field is added). Any other fields on the
// event are preserved. If [Event.Err] was not called, the message is empty.
func (e *Event) Send() {
	if e == nil {
		return
	}

	if e.err != nil {
		msg := e.err.Error()
		e.err = nil // prevent Msg from also adding it as a field
		e.Msg(msg)
		return
	}

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
	if e == nil || core.IsNilStringer(val) {
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
		if core.IsNilStringer(v) {
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

// Times adds a [time.Time] slice field.
func (e *Event) Times(key string, vals []time.Time) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// TimeDiff adds the field key with the duration between t and start.
// If t is not after start, the duration is zero.
func (e *Event) TimeDiff(key string, t, start time.Time) *Event {
	if e == nil {
		return e
	}

	var d time.Duration
	if t.After(start) {
		d = t.Sub(start)
	}

	e.fields = append(e.fields, Field{Key: key, Value: d})
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

// Uint8 adds a uint8 field.
func (e *Event) Uint8(key string, val uint8) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Uint16 adds a uint16 field.
func (e *Event) Uint16(key string, val uint16) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: val})
	return e
}

// Uint32 adds a uint32 field.
func (e *Event) Uint32(key string, val uint32) *Event {
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

// Uints adds a uint slice field.
func (e *Event) Uints(key string, vals []uint) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Uints8 adds a uint8 slice field.
func (e *Event) Uints8(key string, vals []uint8) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Uints16 adds a uint16 slice field.
func (e *Event) Uints16(key string, vals []uint16) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Uints32 adds a uint32 slice field.
func (e *Event) Uints32(key string, vals []uint32) *Event {
	if e == nil {
		return e
	}

	e.fields = append(e.fields, Field{Key: key, Value: vals})
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

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.hyperlink(url, url)},
	)
	return e
}

// URLs adds a string slice field where each element is a hyperlink
// with the URL as the display text.
func (e *Event) URLs(key string, urls []string) *Event {
	if e == nil {
		return e
	}

	output := Default().Output()
	if e.logger != nil {
		output = e.logger.Output()
	}

	vals := make([]string, len(urls))
	for i, u := range urls {
		vals[i] = output.hyperlink(u, u)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
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

// withParts sets the parts override on the event (used internally).
func (e *Event) withParts(parts *[]Part) *Event {
	if e == nil {
		return e
	}

	e.parts = parts
	return e
}

// withSymbol sets the symbol on the event (used internally).
func (e *Event) withSymbol(symbol string) *Event {
	if e == nil {
		return e
	}

	e.symbol = new(symbol)
	return e
}

// Elapsed adds an elapsed-time field at the current position in the field
// list. The duration is measured from the first Elapsed call on this event
// until the event is finalised with [Event.Send], [Event.Msg], or
// [Event.Msgf].
//
// The key parameter is the field name (e.g. "elapsed"). The field uses the
// same formatting and styling as [fx.Builder.Elapsed]. Use options from the
// [elapsed] package (e.g. [elapsed.WithGradientMax]) to override the
// logger's gradient settings for this field only:
//
//	e := clog.Info().Str("step", "migrate").
//	    Elapsed("elapsed", elapsed.WithGradientMax(3*time.Second))
//	runMigrations()
//	e.Msg("done")
//	// Output: INF ℹ️ done step=migrate elapsed=2s
func (e *Event) Elapsed(key string, opts ...elapsed.Option) *Event {
	if e == nil {
		return e
	}
	if e.elapsedStart.IsZero() {
		e.elapsedStart = time.Now()
	}
	f := core.ElapsedField{}
	elapsed.Apply(&f, opts...)
	e.fields = append(e.fields, Field{Key: key, Value: f})
	return e
}

// Deadline adds a countdown field at the current position in the field list.
// The field displays the time remaining until from has elapsed (clamped at 0),
// measured from the first Deadline call on this event until the event is
// finalised with [Event.Send], [Event.Msg], or [Event.Msgf].
//
// The key parameter is the field name (e.g. "timeout"). The field uses the
// same formatting and styling as [fx.Builder.Deadline]: it is colored by the
// consumed time against the logger's deadline gradient, so a fresh deadline
// uses the gradient's first stop (green) and an expiring one the last (red).
// Use options from the [deadline] package (e.g. [deadline.WithGradient]) to
// override the gradient settings for this field only:
//
//	e := clog.Info().Str("job", "upload").
//	    Deadline("timeout", 15*time.Second)
//	waitForCompletion()
//	e.Msg("done")
//	// Output: INF ℹ️ done job=upload timeout=12s
func (e *Event) Deadline(key string, from time.Duration, opts ...deadline.Option) *Event {
	if e == nil {
		return e
	}
	if e.deadlineStart.IsZero() {
		e.deadlineStart = time.Now()
	}
	f := core.DeadlineField{From: from}
	deadline.Apply(&f, opts...)
	e.fields = append(e.fields, Field{Key: key, Value: f})
	return e
}

// resolveDeadline replaces any unresolved core.DeadlineField placeholders in
// the event's fields with the remaining time since the first [Event.Deadline]
// call, preserving any per-field overrides set on them.
func (e *Event) resolveDeadline() {
	if e.deadlineStart.IsZero() {
		return
	}
	dur := time.Since(e.deadlineStart)
	for i := range e.fields {
		if v, ok := e.fields[i].Value.(core.DeadlineField); ok && v.Remaining == 0 {
			v.Remaining = max(v.From-dur, 0)
			e.fields[i].Value = v
		}
	}
}

// resolveElapsed replaces any zero-value core.ElapsedField placeholders in the
// event's fields with the actual elapsed duration since the first
// [Event.Elapsed] call, preserving any per-field overrides set on them.
func (e *Event) resolveElapsed() {
	if e.elapsedStart.IsZero() {
		return
	}
	dur := time.Since(e.elapsedStart)
	for i := range e.fields {
		if v, ok := e.fields[i].Value.(core.ElapsedField); ok && v.Value == 0 {
			v.Value = dur
			e.fields[i].Value = v
		}
	}
}
