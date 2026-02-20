package clog

import (
	"context"
	"fmt"
	"sync/atomic"

	"github.com/charmbracelet/bubbles/spinner"
)

// DefaultSpinner is the default spinner animation.
var DefaultSpinner = spinner.Spinner{
	Frames: []string{"🌔", "🌓", "🌒", "🌑", "🌘", "🌗", "🌖", "🌕"},
	FPS:    spinner.Moon.FPS,
}

// ProgressUpdate is a fluent builder for updating a spinner's title and fields
// during a [ProgressTask]. Call [ProgressUpdate.Title] and field methods to
// build the update, then [ProgressUpdate.Send] to apply it atomically.
type ProgressUpdate struct {
	fieldBuilder[ProgressUpdate]

	base      []Field
	fieldsPtr *atomic.Pointer[[]Field]
	title     string
	titlePtr  *atomic.Pointer[string]
}

// Title sets the spinner's displayed title.
func (u *ProgressUpdate) Title(title string) *ProgressUpdate {
	u.title = title
	return u
}

// Send applies the accumulated title and field changes to the spinner atomically.
func (u *ProgressUpdate) Send() {
	title := u.title
	u.titlePtr.Store(&title)
	merged := mergeFields(u.base, u.fields)
	u.fieldsPtr.Store(&merged)
	u.fields = nil // reset for reuse
}

// Err adds an error field. If err is nil, no field is added.
func (u *ProgressUpdate) Err(err error) *ProgressUpdate {
	if err == nil {
		return u
	}
	u.fields = append(u.fields, Field{Key: ErrorKey, Value: err})
	return u
}

// Stringer adds a field by calling the value's String method. No-op if val is nil.
func (u *ProgressUpdate) Stringer(key string, val fmt.Stringer) *ProgressUpdate {
	if isNilStringer(val) {
		return u
	}

	u.fields = append(u.fields, Field{Key: key, Value: val.String()})
	return u
}

// Stringers adds a field with a slice of [fmt.Stringer] values.
func (u *ProgressUpdate) Stringers(key string, vals []fmt.Stringer) *ProgressUpdate {
	strs := make([]string, len(vals))
	for i, v := range vals {
		if isNilStringer(v) {
			strs[i] = Nil
		} else {
			strs[i] = v.String()
		}
	}

	u.fields = append(u.fields, Field{Key: key, Value: strs})
	return u
}

// Task is a function executed by [SpinnerBuilder.Wait].
type Task func(context.Context) error

// ProgressTask is a function executed by [SpinnerBuilder.Progress].
// The [ProgressUpdate] allows updating the spinner's title and fields.
type ProgressTask func(context.Context, *ProgressUpdate) error

// SpinnerBuilder configures a spinner before execution.
type SpinnerBuilder struct {
	fieldBuilder[SpinnerBuilder]

	pulseStops   []ColorStop // nil = no pulse
	shimmerDir   Direction   // shimmer wave direction
	shimmerStops []ColorStop // nil = no shimmer
	spinner      spinner.Spinner
	title        string
}

// Spinner creates a new [SpinnerBuilder] with the given title.
func Spinner(title string) *SpinnerBuilder {
	b := &SpinnerBuilder{
		spinner: DefaultSpinner,
		title:   title,
	}
	b.initSelf(b)
	return b
}

// Type sets the spinner animation type.
func (b *SpinnerBuilder) Type(s spinner.Spinner) *SpinnerBuilder {
	b.spinner = s
	return b
}

// Shimmer enables animated gradient shimmer on the spinner's message text.
// With no arguments, the default shimmer gradient is used. Custom gradient
// stops can be passed to override the default.
func (b *SpinnerBuilder) Shimmer(stops ...ColorStop) *SpinnerBuilder {
	if len(stops) == 0 {
		b.shimmerStops = DefaultShimmerGradient()
	} else {
		b.shimmerStops = stops
	}
	return b
}

// ShimmerDirection sets the direction the shimmer wave travels.
// Defaults to [DirectionRight]. Use [DirectionLeft] to reverse
// or [DirectionMiddleIn] for a wave entering from both edges.
func (b *SpinnerBuilder) ShimmerDirection(d Direction) *SpinnerBuilder {
	b.shimmerDir = d
	return b
}

// Pulse enables an animated color pulse on the spinner's message text.
// All characters fade uniformly between colors in the gradient.
// With no arguments, the default pulse gradient is used. Custom gradient
// stops can be passed to override the default.
func (b *SpinnerBuilder) Pulse(stops ...ColorStop) *SpinnerBuilder {
	if len(stops) == 0 {
		b.pulseStops = DefaultPulseGradient()
	} else {
		b.pulseStops = stops
	}
	return b
}

// Path adds a file path field as a clickable terminal hyperlink.
// Uses the [Default] logger's [Output] setting.
func (b *SpinnerBuilder) Path(key, path string) *SpinnerBuilder {
	output := Default.Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.pathLink(path, 0, 0)})
	return b
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Uses the [Default] logger's [Output] setting.
func (b *SpinnerBuilder) Line(key, path string, line int) *SpinnerBuilder {
	output := Default.Output()

	if line < 1 {
		line = 1
	}

	b.fields = append(b.fields, Field{Key: key, Value: output.pathLink(path, line, 0)})
	return b
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Uses the [Default] logger's [Output] setting.
func (b *SpinnerBuilder) Column(key, path string, line, column int) *SpinnerBuilder {
	output := Default.Output()

	if line < 1 {
		line = 1
	}

	if column < 1 {
		column = 1
	}

	b.fields = append(
		b.fields,
		Field{Key: key, Value: output.pathLink(path, line, column)},
	)
	return b
}

// URL adds a field as a clickable terminal hyperlink where the URL is also the display text.
// Uses the [Default] logger's [Output] setting.
func (b *SpinnerBuilder) URL(key, url string) *SpinnerBuilder {
	output := Default.Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.hyperlink(url, url)})
	return b
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Uses the [Default] logger's [Output] setting.
func (b *SpinnerBuilder) Link(key, url, text string) *SpinnerBuilder {
	output := Default.Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.hyperlink(url, text)})
	return b
}

// Wait executes the task with a spinner and returns a [WaitResult] for chaining.
// The spinner displays as: <level> <spinner> <title> <fields>.
func (b *SpinnerBuilder) Wait(ctx context.Context, task Task) *WaitResult {
	return b.Progress(ctx, func(ctx context.Context, _ *ProgressUpdate) error {
		return task(ctx)
	})
}

// Progress executes the task with a spinner whose title and fields
// can be updated via the [ProgressUpdate] builder. This is useful for multi-step
// operations where the spinner should reflect the current step.
func (b *SpinnerBuilder) Progress(
	ctx context.Context,
	task ProgressTask,
) *WaitResult {
	var titlePtr atomic.Pointer[string]
	var fieldsPtr atomic.Pointer[[]Field]

	titlePtr.Store(&b.title)
	fieldsPtr.Store(&b.fields)

	update := &ProgressUpdate{
		title:     b.title,
		titlePtr:  &titlePtr,
		fieldsPtr: &fieldsPtr,
		base:      b.fields,
	}
	update.initSelf(update)

	wrapped := func(ctx context.Context) error {
		return task(ctx, update)
	}

	err := runAnimation(
		ctx,
		&titlePtr,
		&fieldsPtr,
		b.spinner,
		b.shimmerStops,
		b.shimmerDir,
		b.pulseStops,
		wrapped,
	)

	title := *titlePtr.Load()
	w := &WaitResult{
		err:          err,
		successLevel: InfoLevel,
		successMsg:   title,
		errorLevel:   ErrorLevel,
	}
	w.fields = *fieldsPtr.Load()
	w.initSelf(w)
	return w
}

// WaitResult holds the result of a [SpinnerBuilder.Wait] operation and
// allows chaining additional fields before finalising the log output.
type WaitResult struct {
	fieldBuilder[WaitResult]

	err          error
	errorLevel   Level
	errorMsg     *string // nil = use error string
	prefix       *string // nil = use default emoji for level
	successLevel Level
	successMsg   string
}

// Err returns the error, logging success at info level or failure at error
// level using the original spinner title.
func (w *WaitResult) Err() error {
	return w.Send()
}

func (w *WaitResult) event(level Level) *Event {
	e := Default.newEvent(level)
	if e == nil {
		return nil
	}

	e = e.withFields(w.fields)

	if w.prefix != nil {
		e = e.withPrefix(*w.prefix)
	}
	return e
}

// Msg logs at info level with the given message on success, or at error
// level with the original spinner title on failure. Returns the error.
func (w *WaitResult) Msg(msg string) error {
	w.successMsg = msg
	return w.Send()
}

// OnErrorLevel sets the log level for the error case. Defaults to [ErrorLevel].
func (w *WaitResult) OnErrorLevel(level Level) *WaitResult {
	w.errorLevel = level
	return w
}

// OnErrorMessage sets a custom message for the error case. Defaults to the
// error string.
func (w *WaitResult) OnErrorMessage(msg string) *WaitResult {
	w.errorMsg = &msg
	return w
}

// OnSuccessLevel sets the log level for the success case. Defaults to [InfoLevel].
func (w *WaitResult) OnSuccessLevel(level Level) *WaitResult {
	w.successLevel = level
	return w
}

// OnSuccessMessage sets the message for the success case. Defaults to the
// original spinner title.
func (w *WaitResult) OnSuccessMessage(msg string) *WaitResult {
	w.successMsg = msg
	return w
}

// Prefix sets a custom emoji prefix for the completion log message.
func (w *WaitResult) Prefix(prefix string) *WaitResult {
	w.prefix = new(prefix)
	return w
}

// Send finalises the result, logging at the configured success or error
// level. Returns the error from the task.
func (w *WaitResult) Send() error {
	if w.err == nil {
		w.event(w.successLevel).Msg(w.successMsg)
	} else {
		msg := w.err.Error()
		if w.errorMsg != nil {
			msg = *w.errorMsg
		}

		w.event(w.errorLevel).Err(w.err).Msg(msg)
	}
	return w.err
}

// Silent returns just the error without logging anything.
func (w *WaitResult) Silent() error {
	return w.err
}
