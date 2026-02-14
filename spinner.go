package clog

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/x/ansi"
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

	title     string
	titlePtr  *atomic.Pointer[string]
	fieldsPtr *atomic.Pointer[[]Field]
	base      []Field
}

// Title sets the spinner's displayed title.
func (u *ProgressUpdate) Title(title string) *ProgressUpdate {
	u.title = title
	return u
}

// Send applies the accumulated title and field changes to the spinner atomically.
func (u *ProgressUpdate) Send() {
	u.titlePtr.Store(&u.title)
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
// Uses the [Default] logger's [ColorMode] setting.
func (b *SpinnerBuilder) Path(key, path string) *SpinnerBuilder {
	Default.mu.Lock()
	mode := Default.colorMode
	Default.mu.Unlock()

	b.fields = append(b.fields, Field{Key: key, Value: pathLinkWithMode(path, 0, 0, mode)})

	return b
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Uses the [Default] logger's [ColorMode] setting.
func (b *SpinnerBuilder) Line(key, path string, lineNumber int) *SpinnerBuilder {
	Default.mu.Lock()
	mode := Default.colorMode
	Default.mu.Unlock()

	if lineNumber < 1 {
		lineNumber = 1
	}

	b.fields = append(b.fields, Field{Key: key, Value: pathLinkWithMode(path, lineNumber, 0, mode)})

	return b
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Uses the [Default] logger's [ColorMode] setting.
func (b *SpinnerBuilder) Column(key, path string, lineNumber, column int) *SpinnerBuilder {
	Default.mu.Lock()
	mode := Default.colorMode
	Default.mu.Unlock()

	if lineNumber < 1 {
		lineNumber = 1
	}

	if column < 1 {
		column = 1
	}

	b.fields = append(
		b.fields,
		Field{Key: key, Value: pathLinkWithMode(path, lineNumber, column, mode)},
	)

	return b
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Uses the [Default] logger's [ColorMode] setting.
func (b *SpinnerBuilder) Link(key, url, text string) *SpinnerBuilder {
	Default.mu.Lock()
	mode := Default.colorMode
	Default.mu.Unlock()

	b.fields = append(b.fields, Field{Key: key, Value: hyperlinkWithMode(url, text, mode)})

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

	w := &WaitResult{title: *titlePtr.Load(), err: err}
	w.fields = *fieldsPtr.Load()
	w.initSelf(w)

	return w
}

// WaitResult holds the result of a [SpinnerBuilder.Wait] operation and
// allows chaining additional fields before finalising the log output.
type WaitResult struct {
	fieldBuilder[WaitResult]

	err    error
	prefix *string // nil = use default emoji for level
	title  string
}

// Prefix sets a custom emoji prefix for the completion log message.
func (w *WaitResult) Prefix(prefix string) *WaitResult {
	w.prefix = new(prefix)

	return w
}

// Msg logs at info level with the given message on success, or at error
// level with the original spinner title on failure. Returns the error.
func (w *WaitResult) Msg(msg string) error {
	if w.err == nil {
		w.event(InfoLevel).Msg(msg)
	} else {
		w.event(ErrorLevel).Err(w.err).Msg(w.title)
	}

	return w.err
}

// Debug logs at debug level with the given message on success, or at
// error level on failure. Returns the error.
func (w *WaitResult) Debug(msg string) error {
	if w.err == nil {
		w.event(DebugLevel).Msg(msg)
	} else {
		w.event(ErrorLevel).Err(w.err).Msg(w.title)
	}

	return w.err
}

// Err returns the error, logging success at info level or failure at error
// level using the original spinner title.
func (w *WaitResult) Err() error {
	if w.err == nil {
		w.event(InfoLevel).Msg(w.title)
	} else {
		w.event(ErrorLevel).Err(w.err).Msg(w.title)
	}

	return w.err
}

// Silent returns just the error without logging anything.
func (w *WaitResult) Silent() error {
	return w.err
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
	noColor := Default.colorsDisabled()
	out := Default.out
	styles := Default.styles
	label := Default.formatLabel(InfoLevel)
	reportTS := Default.reportTimestamp
	timeFmt := Default.timeFormat
	timeLoc := Default.timeLocation
	order := Default.parts
	quoteMode := Default.quoteMode
	quoteOpen := Default.quoteOpen
	quoteClose := Default.quoteClose
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
						quoteMode:  quoteMode,
						quoteOpen:  quoteOpen,
						quoteClose: quoteClose,
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
	_, _ = io.WriteString(out, ansi.HideCursor)
	defer func() {
		_, _ = io.WriteString(out, ansi.ShowCursor)
	}()

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
			_, _ = io.WriteString(out, "\r"+ansi.EraseLineRight)

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
						styles:     styles,
						level:      InfoLevel,
						quoteMode:  quoteMode,
						quoteOpen:  quoteOpen,
						quoteClose: quoteClose,
					}), " ")
				}

				if part != "" {
					parts = append(parts, part)
				}
			}

			line := "\r" + ansi.EraseLineRight + strings.Join(parts, " ")
			_, _ = io.WriteString(out, line)
			frame++
		case <-ctx.Done():
			_, _ = io.WriteString(out, "\r"+ansi.EraseLineRight)

			return ctx.Err()
		}
	}
}
