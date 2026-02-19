package clog

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

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

// Task is a function executed by [SpinnerBuilder.Wait].
type Task func(context.Context) error

// ProgressTask is a function executed by [SpinnerBuilder.Progress].
// The [ProgressUpdate] allows updating the spinner's title and fields.
type ProgressTask func(context.Context, *ProgressUpdate) error

// SpinnerBuilder configures a spinner before execution.
type SpinnerBuilder struct {
	fieldBuilder[SpinnerBuilder]

	spinner spinner.Spinner
	title   string
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

	err := runSpinner(ctx, &titlePtr, &fieldsPtr, b.spinner, wrapped)

	title := *titlePtr.Load()
	w := &WaitResult{
		title:        title,
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
	title        string
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

//nolint:cyclop // spinner loop has inherent complexity
func runSpinner(
	ctx context.Context,
	title *atomic.Pointer[string],
	fields *atomic.Pointer[[]Field],
	s spinner.Spinner,
	task Task,
) error {
	// Run the task in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- task(ctx)
	}()

	// Snapshot Default's settings under the mutex to avoid data races.
	Default.mu.Lock()
	fieldStyleLevel := Default.fieldStyleLevel
	fieldTimeFormat := Default.fieldTimeFormat
	label := Default.formatLabel(InfoLevel)
	noColor := Default.output.ColorsDisabled()
	order := Default.parts
	out := Default.output.Writer()
	termOut := Default.output.Renderer().Output()
	quoteClose := Default.quoteClose
	quoteMode := Default.quoteMode
	quoteOpen := Default.quoteOpen
	reportTS := Default.reportTimestamp
	styles := Default.styles
	timeFmt := Default.timeFormat
	timeLoc := Default.timeLocation
	Default.mu.Unlock()

	// Don't animate if colours are disabled (CI, piped output, etc.).
	// Print the initial title so the user knows something is in progress.
	if noColor {
		parts := make([]string, 0, len(order))

		for _, p := range order {
			var part string

			switch p {
			case PartTimestamp:
				if !reportTS {
					continue
				}

				part = time.Now().In(timeLoc).Format(timeFmt)
			case PartLevel:
				part = label
			case PartPrefix:
				part = "⏳"
			case PartMessage:
				part = *title.Load()
			case PartFields:
				part = strings.TrimLeft(
					formatFields(*fields.Load(), formatFieldsOpts{
						noColor:    true,
						quoteClose: quoteClose,
						quoteMode:  quoteMode,
						quoteOpen:  quoteOpen,
						timeFormat: fieldTimeFormat,
					}), " ",
				)
			}

			if part != "" {
				parts = append(parts, part)
			}
		}

		_, _ = fmt.Fprintf(out, "%s\n", strings.Join(parts, " "))
		return <-done
	}

	// Hide cursor during animation.
	termOut.HideCursor()
	defer termOut.ShowCursor()

	var levelPrefix string
	if style := styles.Levels[InfoLevel]; style != nil {
		levelPrefix = style.Render(label)
	} else {
		levelPrefix = label
	}

	ticker := time.NewTicker(s.FPS)
	defer ticker.Stop()

	frame := 0

	for {
		select {
		case err := <-done:
			termOut.ClearLine()
			_, _ = fmt.Fprint(out, "\r")
			return err
		case <-ticker.C:
			char := s.Frames[frame%len(s.Frames)]

			parts := make([]string, 0, len(order))

			for _, p := range order {
				var part string

				switch p {
				case PartTimestamp:
					if !reportTS {
						continue
					}

					ts := time.Now().In(timeLoc).Format(timeFmt)
					if styles.Timestamp != nil {
						part = styles.Timestamp.Render(ts)
					} else {
						part = ts
					}
				case PartLevel:
					part = levelPrefix
				case PartPrefix:
					part = char
				case PartMessage:
					part = *title.Load()
				case PartFields:
					part = strings.TrimLeft(formatFields(*fields.Load(), formatFieldsOpts{
						fieldStyleLevel: fieldStyleLevel,
						styles:          styles,
						level:           InfoLevel,
						quoteMode:       quoteMode,
						quoteOpen:       quoteOpen,
						quoteClose:      quoteClose,
						timeFormat:      fieldTimeFormat,
					}), " ")
				}

				if part != "" {
					parts = append(parts, part)
				}
			}

			termOut.ClearLine()
			_, _ = fmt.Fprintf(out, "\r%s", strings.Join(parts, " "))
			frame++
		case <-ctx.Done():
			termOut.ClearLine()
			_, _ = fmt.Fprint(out, "\r")
			return ctx.Err()
		}
	}
}
