package clog

import (
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

//go:generate go run ./internal/gen/fieldmethods

// NOTE: Event cannot embed FieldBuilder because it needs nil-receiver no-ops
// and returns *Event (not *T). The field methods shared with FieldBuilder are
// generated into event_fields.go from internal/gen/fieldmethods; only methods
// whose semantics diverge between the two receivers live here.

// output returns the logger's [Output] for hyperlink and color resolution.
// Every Event is created by a Logger constructor ([Logger.newEvent],
// [Logger.Dict]), so e.logger is never nil.
func (e *Event) output() *Output {
	return e.logger.Output()
}

// fieldFormats returns the owning logger's field-format snapshot.
func (e *Event) fieldFormats() *FieldFormats {
	return e.logger.loadFieldFormats()
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

	output := e.output()

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, line, column)},
	)
	return e
}

// Columns adds a string slice field where each element is a path:line:column
// hyperlink. Respects the logger's [ColorMode] setting.
func (e *Event) Columns(key string, items []Column) *Event {
	if e == nil {
		return e
	}

	output := e.output()

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

// Column represents a file path with a line and column number for use with [Event.Columns].
type Column struct {
	Path   string
	Line   int
	Column int
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

// Disabled returns true if the event is disabled (nil).
func (e *Event) Disabled() bool {
	return e == nil
}

// Discard disables the event so Msg/Msgf/Send won't produce output.
// Returns nil to short-circuit subsequent field methods.
func (e *Event) Discard() *Event {
	return nil
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

// Enabled returns true if the event is enabled (non-nil).
func (e *Event) Enabled() bool {
	return e != nil
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

	output := e.output()

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, line, 0)},
	)
	return e
}

// Lines adds a string slice field where each element is a path:line
// hyperlink. If an item's Line < 1, that element is rendered as a plain path
// hyperlink (equivalent to [Event.Path]). Respects the logger's [ColorMode]
// setting.
func (e *Event) Lines(key string, items []Line) *Event {
	if e == nil {
		return e
	}

	output := e.output()

	vals := make([]string, len(items))
	for i, item := range items {
		vals[i] = output.pathLink(item.Path, item.Line, 0)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Line represents a file path with a line number for use with [Event.Lines].
type Line struct {
	Path string
	Line int
}

// Link adds a single hyperlink field.
func (e *Event) Link(key, url, text string) *Event {
	if e == nil {
		return e
	}

	output := e.output()

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

	output := e.output()

	vals := make([]string, len(links))
	for i, l := range links {
		vals[i] = output.hyperlink(l.URL, l.Text)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
	return e
}

// Link represents a hyperlink with a URL and display text.
type Link struct {
	URL  string
	Text string
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

// Path adds a file path field as a clickable terminal hyperlink.
// Respects the logger's [ColorMode] setting.
func (e *Event) Path(key, path string) *Event {
	if e == nil {
		return e
	}

	output := e.output()

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLink(path, 0, 0)},
	)
	return e
}

// Paths adds a string slice field where each element is a path hyperlink.
// Respects the logger's [ColorMode] setting.
func (e *Event) Paths(key string, paths []string) *Event {
	if e == nil {
		return e
	}

	output := e.output()

	vals := make([]string, len(paths))
	for i, p := range paths {
		vals[i] = output.pathLink(p, 0, 0)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
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

	output := e.output()

	e.fields = append(
		e.fields,
		Field{Key: key, Value: output.pathLinkText(text, path, 0, 0)},
	)
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
func (e *Event) Percent(key string, val float64, opts ...percent.Option) *Event {
	if e == nil {
		return e
	}

	p := core.Percent{Value: val}
	percent.Apply(&p, opts...)
	p.Value = xmath.Clamp(
		p.Value,
		0,
		percent.EffectiveMaximum(p, e.fieldFormats().PercentMaximum),
	)
	e.fields = append(e.fields, Field{Key: key, Value: p})
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

// URL adds a field as a clickable terminal hyperlink where the URL is also the display text.
// Respects the logger's [ColorMode] setting.
func (e *Event) URL(key, url string) *Event {
	if e == nil {
		return e
	}

	output := e.output()

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

	output := e.output()

	vals := make([]string, len(urls))
	for i, u := range urls {
		vals[i] = output.hyperlink(u, u)
	}
	e.fields = append(e.fields, Field{Key: key, Value: vals})
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
