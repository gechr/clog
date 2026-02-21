package clog

import (
	"context"
	"io"
	"math"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

// clearLine is the ANSI escape to erase the entire current line (EL2),
// followed by a carriage return to reset the cursor to column 0.
const clearLine = "\x1b[2K\r"

// animMode is the animation rendering mode for an [AnimationBuilder].
type animMode int

const (
	animModeSpinner animMode = iota
	animModePulse
	animModeShimmer
)

// Task is a function executed by [AnimationBuilder.Wait].
type Task func(context.Context) error

// ProgressTask is a function executed by [AnimationBuilder.Progress].
// The [ProgressUpdate] allows updating the animation's message and fields.
type ProgressTask func(context.Context, *ProgressUpdate) error

// ProgressUpdate is a fluent builder for updating an animation's message and fields
// during a [ProgressTask]. Call [ProgressUpdate.Msg] and field methods to
// build the update, then [ProgressUpdate.Send] to apply it atomically.
type ProgressUpdate struct {
	fieldBuilder[ProgressUpdate]

	base      []Field
	fieldsPtr *atomic.Pointer[[]Field]
	msg       string
	msgPtr    *atomic.Pointer[string]
}

// Msg sets the animation's displayed message.
func (u *ProgressUpdate) Msg(msg string) *ProgressUpdate {
	u.msg = msg
	return u
}

// Send applies the accumulated message and field changes to the animation atomically.
func (u *ProgressUpdate) Send() {
	msg := u.msg
	u.msgPtr.Store(&msg)
	merged := mergeFields(u.base, u.fields)
	u.fieldsPtr.Store(&merged)
	u.fields = nil // reset for reuse
}

// AnimationBuilder configures an animation before execution.
// Create one with [Spinner], [Pulse], or [Shimmer].
type AnimationBuilder struct {
	fieldBuilder[AnimationBuilder]

	level        Level // log level used during animation rendering (default: InfoLevel)
	mode         animMode
	msg          string
	prefix       string // icon shown during animation; defaults to "⏳" for pulse/shimmer
	pulseStops   []ColorStop
	shimmerDir   Direction
	shimmerStops []ColorStop
	spinner      SpinnerType
}

// Pulse creates a new [AnimationBuilder] with an animated color pulse on the message text.
// All characters fade uniformly between colors in the gradient.
// With no arguments, the default pulse gradient is used. Custom gradient
// stops can be passed to override the default.
func Pulse(msg string, stops ...ColorStop) *AnimationBuilder {
	if len(stops) == 0 {
		stops = DefaultPulseGradient()
	}
	b := &AnimationBuilder{
		level:      InfoLevel,
		mode:       animModePulse,
		msg:        msg,
		pulseStops: stops,
		spinner:    DefaultSpinner,
	}
	b.initSelf(b)
	return b
}

// Shimmer creates a new [AnimationBuilder] with an animated gradient shimmer on the message text.
// Each character is coloured independently based on its position in the wave.
// With no arguments, the default shimmer gradient is used. Custom gradient
// stops can be passed to override the default.
func Shimmer(msg string, stops ...ColorStop) *AnimationBuilder {
	if len(stops) == 0 {
		stops = DefaultShimmerGradient()
	}
	b := &AnimationBuilder{
		level:        InfoLevel,
		mode:         animModeShimmer,
		msg:          msg,
		shimmerStops: stops,
		spinner:      DefaultSpinner,
	}
	b.initSelf(b)
	return b
}

// Prefix sets the icon displayed beside the message during animation.
// For [Pulse] and [Shimmer] this defaults to "⏳".
// For [Spinner] the prefix is the current spinner frame and this setting is ignored.
func (b *AnimationBuilder) Prefix(prefix string) *AnimationBuilder {
	b.prefix = prefix
	return b
}

// Type sets the spinner animation type.
// Only meaningful when the builder was created with [Spinner].
func (b *AnimationBuilder) Type(s SpinnerType) *AnimationBuilder {
	b.spinner = s
	return b
}

// ShimmerDirection sets the direction the shimmer wave travels.
// Defaults to [DirectionRight]. Use [DirectionLeft] to reverse
// or [DirectionMiddleIn] for a wave entering from both edges.
// Only meaningful when the builder was created with [Shimmer].
func (b *AnimationBuilder) ShimmerDirection(d Direction) *AnimationBuilder {
	b.shimmerDir = d
	return b
}

// Path adds a file path field as a clickable terminal hyperlink.
// Uses the [Default] logger's [Output] setting.
func (b *AnimationBuilder) Path(key, path string) *AnimationBuilder {
	output := Default.Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.pathLink(path, 0, 0)})
	return b
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Uses the [Default] logger's [Output] setting.
func (b *AnimationBuilder) Line(key, path string, line int) *AnimationBuilder {
	output := Default.Output()

	if line < 1 {
		line = 1
	}

	b.fields = append(b.fields, Field{Key: key, Value: output.pathLink(path, line, 0)})
	return b
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Uses the [Default] logger's [Output] setting.
func (b *AnimationBuilder) Column(key, path string, line, column int) *AnimationBuilder {
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
func (b *AnimationBuilder) URL(key, url string) *AnimationBuilder {
	output := Default.Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.hyperlink(url, url)})
	return b
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Uses the [Default] logger's [Output] setting.
func (b *AnimationBuilder) Link(key, url, text string) *AnimationBuilder {
	output := Default.Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.hyperlink(url, text)})
	return b
}

// Wait executes the task with the animation and returns a [WaitResult] for chaining.
// The animation displays as: <level> <icon> <message> <fields>.
func (b *AnimationBuilder) Wait(ctx context.Context, task Task) *WaitResult {
	return b.Progress(ctx, func(ctx context.Context, _ *ProgressUpdate) error {
		return task(ctx)
	})
}

// Progress executes the task with the animation whose message and fields
// can be updated via the [ProgressUpdate] builder. This is useful for multi-step
// operations where the animation should reflect the current step.
func (b *AnimationBuilder) Progress(
	ctx context.Context,
	task ProgressTask,
) *WaitResult {
	var msgPtr atomic.Pointer[string]
	var fieldsPtr atomic.Pointer[[]Field]

	msgPtr.Store(&b.msg)
	fieldsPtr.Store(&b.fields)

	update := &ProgressUpdate{
		msg:       b.msg,
		msgPtr:    &msgPtr,
		fieldsPtr: &fieldsPtr,
		base:      b.fields,
	}
	update.initSelf(update)

	wrapped := func(ctx context.Context) error {
		return task(ctx, update)
	}

	err := runAnimation(ctx, &msgPtr, &fieldsPtr, b, wrapped)

	msg := *msgPtr.Load()
	w := &WaitResult{
		err:          err,
		successLevel: b.level,
		successMsg:   msg,
		errorLevel:   ErrorLevel,
	}
	w.fields = *fieldsPtr.Load()
	w.initSelf(w)
	return w
}

// WaitResult holds the result of an [AnimationBuilder.Wait] operation and
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
// level using the original animation message.
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
// level with the error string on failure. Returns the error.
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
// original animation message.
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

//nolint:cyclop // animation loop has inherent complexity
func runAnimation(
	ctx context.Context,
	msgPtr *atomic.Pointer[string],
	fields *atomic.Pointer[[]Field],
	b *AnimationBuilder,
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
	label := Default.formatLabel(b.level)
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

	// buildParts assembles the display parts slice from the configured order.
	buildParts := func(order []Part, reportTS bool, tsStr, levelStr, prefix, msg, fieldsStr string) string {
		parts := make([]string, 0, len(order))
		for _, p := range order {
			var part string
			switch p {
			case PartTimestamp:
				if !reportTS {
					continue
				}
				part = tsStr
			case PartLevel:
				part = levelStr
			case PartPrefix:
				part = prefix
			case PartMessage:
				part = msg
			case PartFields:
				part = fieldsStr
			}
			if part != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, " ")
	}

	// Resolve the prefix icon for pulse/shimmer modes.
	prefix := b.prefix
	if prefix == "" {
		prefix = "⏳"
	}

	// Don't animate if colours are disabled (CI, piped output, etc.).
	// Print the initial message so the user knows something is in progress.
	if noColor {
		fieldsStr := strings.TrimLeft(
			formatFields(*fields.Load(), formatFieldsOpts{
				noColor:    true,
				quoteClose: quoteClose,
				quoteMode:  quoteMode,
				quoteOpen:  quoteOpen,
				timeFormat: fieldTimeFormat,
			}), " ",
		)
		line := buildParts(order, reportTS,
			time.Now().In(timeLoc).Format(timeFmt),
			label, prefix, *msgPtr.Load(), fieldsStr)
		_, _ = io.WriteString(out, line+"\n")
		return <-done
	}

	// Hide cursor during animation.
	termOut.HideCursor()
	defer termOut.ShowCursor()

	var levelPrefix string
	if style := styles.Levels[b.level]; style != nil {
		levelPrefix = style.Render(label)
	} else {
		levelPrefix = label
	}

	// Determine the tick rate and pre-compute any mode-specific resources.
	var tickRate time.Duration
	var hexLUT *shimmerLUT
	var styleLUT *shimmerStyleLUT
	switch b.mode {
	case animModeSpinner:
		tickRate = b.spinner.FPS
	case animModePulse:
		tickRate = pulseTickRate
	case animModeShimmer:
		tickRate = shimmerTickRate
		hexLUT = buildShimmerLUT(b.shimmerStops)
		styleLUT = buildShimmerStyleLUT(hexLUT)
	}

	// Cache formatted fields — only re-format when the atomic pointer changes.
	var cachedFieldsPtr unsafe.Pointer
	var cachedFieldsStr string
	fieldOpts := formatFieldsOpts{
		fieldStyleLevel: fieldStyleLevel,
		styles:          styles,
		level:           b.level,
		quoteMode:       quoteMode,
		quoteOpen:       quoteOpen,
		quoteClose:      quoteClose,
		timeFormat:      fieldTimeFormat,
	}

	startTime := time.Now()
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	frame := 0            // spinner frame counter
	var pCache pulseCache // reused across pulse frames to avoid style re-creation
	var frameBuf strings.Builder

	for {
		select {
		case err := <-done:
			_, _ = io.WriteString(out, clearLine)
			return err
		case now := <-ticker.C:
			elapsed := now.Sub(startTime)

			msg := *msgPtr.Load()
			var char string

			switch b.mode {
			case animModeSpinner:
				n := len(b.spinner.Frames)
				i := frame % n
				if b.spinner.Reverse {
					i = n - 1 - i
				}
				char = b.spinner.Frames[i]
				frame++
				if msgStyle := styles.Messages[b.level]; msgStyle != nil {
					msg = msgStyle.Render(msg)
				}
			case animModePulse:
				char = prefix
				t := (1.0 + math.Sin(2*math.Pi*elapsed.Seconds()*pulseSpeed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
				msg = pulseTextCached(msg, t, b.pulseStops, &pCache)
			case animModeShimmer:
				char = prefix
				phase := math.Mod(elapsed.Seconds()*shimmerSpeed, 1.0)
				msg = shimmerText(msg, phase, b.shimmerDir, hexLUT, styleLUT)
			}

			// Only re-format fields when the pointer changes.
			fp := fields.Load()
			fpRaw := unsafe.Pointer(fp)
			if fpRaw != cachedFieldsPtr {
				cachedFieldsPtr = fpRaw
				cachedFieldsStr = strings.TrimLeft(formatFields(*fp, fieldOpts), " ")
			}

			var tsStr string
			if reportTS {
				ts := now.In(timeLoc).Format(timeFmt)
				if styles.Timestamp != nil {
					tsStr = styles.Timestamp.Render(ts)
				} else {
					tsStr = ts
				}
			}

			line := buildParts(order, reportTS, tsStr, levelPrefix, char, msg, cachedFieldsStr)
			frameBuf.Reset()
			frameBuf.WriteString(clearLine)
			frameBuf.WriteString(line)
			_, _ = io.WriteString(out, frameBuf.String())
		case <-ctx.Done():
			_, _ = io.WriteString(out, clearLine)
			return ctx.Err()
		}
	}
}
