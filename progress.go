package clog

import (
	"context"
	"io"
	"math"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

// clearLine is the ANSI escape to erase the entire current line (EL2),
// followed by a carriage return to reset the cursor to column 0.
const clearLine = "\x1b[2K\r"

// animation is the animation rendering mode for an [AnimationBuilder].
type animation int

const (
	animationBar animation = iota
	animationPulse
	animationShimmer
	animationSpinner
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

	base        []Field
	fieldsPtr   *atomic.Pointer[[]Field]
	msg         string
	msgPtr      *atomic.Pointer[string]
	progressPtr *atomic.Int64 // bar mode: current progress value; nil for non-bar modes
	totalPtr    *atomic.Int64 // bar mode: total progress value; nil for non-bar modes
}

// SetProgress sets the current progress value for a bar animation.
// Values are clamped to [0, total]. No-op if this is not a bar animation.
func (p *ProgressUpdate) SetProgress(current int) *ProgressUpdate {
	if p.progressPtr != nil {
		if current < 0 {
			current = 0
		}
		if p.totalPtr != nil {
			if total := int(p.totalPtr.Load()); current > total {
				current = total
			}
		}
		p.progressPtr.Store(int64(current))
	}
	return p
}

// SetTotal updates the total progress value for a bar animation.
// No-op if this is not a bar animation.
func (p *ProgressUpdate) SetTotal(total int) *ProgressUpdate {
	if p.totalPtr != nil {
		if total <= 0 {
			total = 1
		}
		p.totalPtr.Store(int64(total))
	}
	return p
}

// Msg sets the animation's displayed message.
func (p *ProgressUpdate) Msg(msg string) *ProgressUpdate {
	p.msg = msg
	return p
}

// Send applies the accumulated message and field changes to the animation atomically.
func (p *ProgressUpdate) Send() {
	msg := p.msg
	p.msgPtr.Store(&msg)
	merged := mergeFields(p.base, p.fields)
	p.fieldsPtr.Store(&merged)
	p.fields = nil // reset for reuse
}

// AnimationBuilder configures an animation before execution.
// Create one with [Spinner], [Pulse], [Shimmer], or [Bar], or their [Logger] method equivalents.
type AnimationBuilder struct {
	fieldBuilder[AnimationBuilder]

	barProgressPtr *atomic.Int64 // bar mode: current progress; nil for non-bar modes
	barStyle       BarStyle      // bar mode: visual style
	barTotalPtr    *atomic.Int64 // bar mode: total progress; nil for non-bar modes
	delay          time.Duration // when set, suppresses animation until this duration elapses
	elapsedKey     string        // when set, a formatted elapsed-time field is injected each tick
	level          Level         // log level used during animation rendering (default: InfoLevel)
	logger         *Logger
	mode           animation
	msg            string
	prefix         string // icon shown during animation; defaults to "⏳" for pulse/shimmer/bar
	pulseStops     []ColorStop
	shimmerDir     Direction
	shimmerStops   []ColorStop
	spinner        SpinnerType
}

// resolveLogger returns the builder's logger, falling back to [Default].
func (b *AnimationBuilder) resolveLogger() *Logger {
	if b.logger != nil {
		return b.logger
	}
	return Default
}

// Pulse creates a new [AnimationBuilder] using the [Default] logger with an
// animated color pulse on the message text.
// All characters fade uniformly between colors in the gradient.
// With no arguments, the default pulse gradient is used. Custom gradient
// stops can be passed to override the default.
func Pulse(msg string, stops ...ColorStop) *AnimationBuilder { return Default.Pulse(msg, stops...) }

// Pulse creates a new [AnimationBuilder] with an animated color pulse on the message text.
// All characters fade uniformly between colors in the gradient.
// With no arguments, the default pulse gradient is used. Custom gradient
// stops can be passed to override the default.
func (l *Logger) Pulse(msg string, stops ...ColorStop) *AnimationBuilder {
	if len(stops) == 0 {
		stops = DefaultPulseGradient()
	}
	b := &AnimationBuilder{
		level:      InfoLevel,
		logger:     l,
		mode:       animationPulse,
		msg:        msg,
		pulseStops: stops,
		spinner:    DefaultSpinner,
	}
	b.initSelf(b)
	return b
}

// Shimmer creates a new [AnimationBuilder] using the [Default] logger with an
// animated gradient shimmer on the message text.
// Each character is coloured independently based on its position in the wave.
// With no arguments, the default shimmer gradient is used. Custom gradient
// stops can be passed to override the default.
func Shimmer(msg string, stops ...ColorStop) *AnimationBuilder {
	return Default.Shimmer(msg, stops...)
}

// Shimmer creates a new [AnimationBuilder] with an animated gradient shimmer on the message text.
// Each character is coloured independently based on its position in the wave.
// With no arguments, the default shimmer gradient is used. Custom gradient
// stops can be passed to override the default.
func (l *Logger) Shimmer(msg string, stops ...ColorStop) *AnimationBuilder {
	if len(stops) == 0 {
		stops = DefaultShimmerGradient()
	}
	b := &AnimationBuilder{
		level:        InfoLevel,
		logger:       l,
		mode:         animationShimmer,
		msg:          msg,
		shimmerStops: stops,
		spinner:      DefaultSpinner,
	}
	b.initSelf(b)
	return b
}

// After sets a delay before the animation becomes visible. If the task
// completes before the delay elapses, no animation is shown at all.
// This is useful for operations that are usually fast but occasionally slow —
// the animation only appears when needed, avoiding visual noise.
func (b *AnimationBuilder) After(d time.Duration) *AnimationBuilder {
	b.delay = d
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

// Elapsed enables an auto-updating elapsed-time field that is injected on
// each animation tick and included in the final completion log. The key
// parameter is the field name (e.g. "elapsed"). The value is formatted
// using [formatElapsed] with [Styles.ElapsedPrecision].
//
// The field respects the position where Elapsed is called relative to other
// field methods (e.g. Str, Int) on the builder.
func (b *AnimationBuilder) Elapsed(key string) *AnimationBuilder {
	b.elapsedKey = key
	b.fields = append(b.fields, Field{Key: key, Value: elapsed(0)})
	return b
}

// Path adds a file path field as a clickable terminal hyperlink.
// Uses the builder's logger's [Output] setting.
func (b *AnimationBuilder) Path(key, path string) *AnimationBuilder {
	output := b.resolveLogger().Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.pathLink(path, 0, 0)})
	return b
}

// Line adds a file path field with a line number as a clickable terminal hyperlink.
// Uses the builder's logger's [Output] setting.
func (b *AnimationBuilder) Line(key, path string, line int) *AnimationBuilder {
	output := b.resolveLogger().Output()

	if line < 1 {
		line = 1
	}

	b.fields = append(b.fields, Field{Key: key, Value: output.pathLink(path, line, 0)})
	return b
}

// Column adds a file path field with a line and column number as a clickable terminal hyperlink.
// Uses the builder's logger's [Output] setting.
func (b *AnimationBuilder) Column(key, path string, line, column int) *AnimationBuilder {
	output := b.resolveLogger().Output()

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
// Uses the builder's logger's [Output] setting.
func (b *AnimationBuilder) URL(key, url string) *AnimationBuilder {
	output := b.resolveLogger().Output()
	b.fields = append(b.fields, Field{Key: key, Value: output.hyperlink(url, url)})
	return b
}

// Link adds a field as a clickable terminal hyperlink with custom URL and display text.
// Uses the builder's logger's [Output] setting.
func (b *AnimationBuilder) Link(key, url, text string) *AnimationBuilder {
	output := b.resolveLogger().Output()
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
	if b.mode == animationBar {
		update.progressPtr = b.barProgressPtr
		update.totalPtr = b.barTotalPtr
	}
	update.initSelf(update)

	wrapped := func(ctx context.Context) error {
		return task(ctx, update)
	}

	startTime := time.Now()
	err := runAnimation(ctx, &msgPtr, &fieldsPtr, b, wrapped, startTime)

	msg := *msgPtr.Load()
	w := &WaitResult{
		err:          err,
		logger:       b.logger,
		successLevel: b.level,
		successMsg:   msg,
		errorLevel:   ErrorLevel,
	}
	w.fields = *fieldsPtr.Load()
	if b.elapsedKey != "" {
		w.fields = slices.Clone(w.fields)
		for i := range w.fields {
			if w.fields[i].Key == b.elapsedKey {
				w.fields[i].Value = elapsed(time.Since(startTime))
				break
			}
		}
	}
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
	logger       *Logger // nil = Default
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
	l := w.logger
	if l == nil {
		l = Default
	}
	e := l.newEvent(level)
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

// OnErrorMessage sets a custom message for the error case. When set, the
// original error is added as an [ErrorKey] field alongside the custom message.
// Defaults to using the error string as the message with no extra field.
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
// level. On failure, the error string is used as the message. If a custom
// error message was set via [WaitResult.OnErrorMessage], the original error
// is included as an [ErrorKey] field. Returns the error from the task.
func (w *WaitResult) Send() error {
	switch {
	case w.err == nil:
		w.event(w.successLevel).Msg(w.successMsg)
	case w.errorMsg != nil:
		w.event(w.errorLevel).Err(w.err).Msg(*w.errorMsg)
	default:
		w.event(w.errorLevel).Msg(w.err.Error())
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
	startTime time.Time,
) error {
	// Run the task in a goroutine.
	done := make(chan error, 1)
	go func() {
		done <- task(ctx)
	}()

	// If a delay is configured, wait for it to elapse before showing
	// any animation. If the task completes first, return immediately.
	if b.delay > 0 {
		timer := time.NewTimer(b.delay)
		select {
		case err := <-done:
			timer.Stop()
			return err
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	// Snapshot the logger's settings under the mutex to avoid data races.
	l := b.resolveLogger()
	l.mu.Lock()
	elapsedFormatFunc := l.elapsedFormatFunc
	elapsedMinimum := l.elapsedMinimum
	elapsedPrecision := l.elapsedPrecision
	elapsedRound := l.elapsedRound
	fieldSort := l.fieldSort
	fieldStyleLevel := l.fieldStyleLevel
	fieldTimeFormat := l.fieldTimeFormat
	label := l.formatLabel(b.level)
	noColor := l.output.ColorsDisabled()
	order := l.parts
	out := l.output.Writer()
	output := l.output // captured for per-tick width queries in bar mode
	percentFormatFunc := l.percentFormatFunc
	percentPrecision := l.percentPrecision
	quantityUnitsIgnoreCase := l.quantityUnitsIgnoreCase
	termOut := l.output.Renderer().Output()
	quoteClose := l.quoteClose
	quoteMode := l.quoteMode
	quoteOpen := l.quoteOpen
	reportTS := l.reportTimestamp
	separatorText := l.separatorText
	styles := l.styles
	timeFmt := l.timeFormat
	timeLoc := l.timeLocation
	l.mu.Unlock()

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
				elapsedFormatFunc:       elapsedFormatFunc,
				elapsedMinimum:          elapsedMinimum,
				elapsedPrecision:        elapsedPrecision,
				elapsedRound:            elapsedRound,
				fieldSort:               fieldSort,
				noColor:                 true,
				percentFormatFunc:       percentFormatFunc,
				percentPrecision:        percentPrecision,
				quantityUnitsIgnoreCase: quantityUnitsIgnoreCase,
				quoteClose:              quoteClose,
				quoteMode:               quoteMode,
				quoteOpen:               quoteOpen,
				separatorText:           separatorText,
				timeFormat:              fieldTimeFormat,
			}), " ",
		)
		line := buildParts(order, reportTS,
			time.Now().In(timeLoc).Format(timeFmt),
			label, prefix, *msgPtr.Load(), fieldsStr)
		_, _ = io.WriteString(out, line+"\n")
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
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
	case animationSpinner:
		tickRate = b.spinner.FPS
	case animationPulse:
		tickRate = pulseTickRate
	case animationShimmer:
		tickRate = shimmerTickRate
		hexLUT = buildShimmerLUT(b.shimmerStops)
		styleLUT = buildShimmerStyleLUT(hexLUT)
	case animationBar:
		tickRate = barTickRate
	}

	// Cache formatted fields — only re-format when the atomic pointer changes.
	var cachedFieldsPtr *[]Field
	var cachedFieldsStr string
	fieldOpts := formatFieldsOpts{
		elapsedFormatFunc:       elapsedFormatFunc,
		elapsedMinimum:          elapsedMinimum,
		elapsedPrecision:        elapsedPrecision,
		elapsedRound:            elapsedRound,
		fieldSort:               fieldSort,
		fieldStyleLevel:         fieldStyleLevel,
		level:                   b.level,
		percentFormatFunc:       percentFormatFunc,
		percentPrecision:        percentPrecision,
		quantityUnitsIgnoreCase: quantityUnitsIgnoreCase,
		quoteClose:              quoteClose,
		quoteMode:               quoteMode,
		quoteOpen:               quoteOpen,
		separatorText:           separatorText,
		styles:                  styles,
		timeFormat:              fieldTimeFormat,
	}

	// Guard against invalid SpinnerType values that would cause panics.
	if b.mode == animationSpinner && len(b.spinner.Frames) == 0 {
		b.spinner.Frames = DefaultSpinner.Frames
	}
	if tickRate <= 0 {
		tickRate = DefaultSpinner.FPS
	}

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
			dur := now.Sub(startTime)

			msg := *msgPtr.Load()
			var char string

			switch b.mode {
			case animationSpinner:
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
			case animationPulse:
				char = prefix
				t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*pulseSpeed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
				msg = pulseTextCached(msg, t, b.pulseStops, &pCache)
			case animationShimmer:
				char = prefix
				phase := math.Mod(dur.Seconds()*shimmerSpeed, 1.0)
				msg = shimmerText(msg, phase, b.shimmerDir, hexLUT, styleLUT)
			case animationBar:
				char = prefix
				if msgStyle := styles.Messages[b.level]; msgStyle != nil {
					msg = msgStyle.Render(msg)
				}
				current := int(b.barProgressPtr.Load())
				total := int(b.barTotalPtr.Load())
				barStr := renderBar(current, total, b.barStyle, output.Width())
				pctStr := barPercent(current, total)
				sep := b.barStyle.Separator
				if sep == "" {
					sep = " "
				}
				msg = msg + sep + barStr + sep + pctStr
			}

			// Re-format fields when the pointer changes, or every tick if
			// an elapsed field is present (its value changes each tick).
			fp := fields.Load()
			if b.elapsedKey != "" {
				clone := slices.Clone(*fp)
				for i := range clone {
					if clone[i].Key == b.elapsedKey {
						clone[i].Value = elapsed(dur)
						break
					}
				}
				cachedFieldsStr = strings.TrimLeft(formatFields(clone, fieldOpts), " ")
			} else if fp != cachedFieldsPtr {
				cachedFieldsStr = strings.TrimLeft(formatFields(*fp, fieldOpts), " ")
			}
			cachedFieldsPtr = fp

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
