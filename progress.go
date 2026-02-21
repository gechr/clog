package clog

import (
	"context"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
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

	barPercentKey  string        // when set, a formatted percent field is injected each tick
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
	spinner        SpinnerStyle
}

// resolveLogger returns the builder's logger, falling back to [Default].
func (b *AnimationBuilder) resolveLogger() *Logger {
	if b.logger != nil {
		return b.logger
	}
	return Default
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

// AnimationStyle is an animation style that can be passed to [AnimationBuilder.Style].
// Valid implementations are [SpinnerStyle] and [BarStyle].
type AnimationStyle interface {
	applyAnimation(*AnimationBuilder)
}

// Style sets the animation style.
// Pass a [SpinnerStyle] for spinner animations or a [BarStyle] for bar animations.
func (b *AnimationBuilder) Style(s AnimationStyle) *AnimationBuilder {
	s.applyAnimation(b)
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

// BarPercent enables an auto-updating percentage field that is injected on
// each animation tick for [Bar] animations. The key parameter is the field
// name (e.g. "progress"). This is useful with [BarStyle.HidePercent] to move
// the percentage from beside the bar into the structured fields.
//
// The field respects the position where BarPercent is called relative to other
// field methods (e.g. Str, Int) on the builder. No-op for non-bar animations.
func (b *AnimationBuilder) BarPercent(key string) *AnimationBuilder {
	b.barPercentKey = key
	b.fields = append(b.fields, Field{Key: key, Value: percent(0)})
	return b
}

// barPercentValue returns the current progress as a percent value.
func (b *AnimationBuilder) barPercentValue() percent {
	cur := int(b.barProgressPtr.Load())
	tot := int(b.barTotalPtr.Load())
	pct := float64(cur) / float64(max(tot, 1)) * percentMax
	return percent(min(pct, percentMax))
}

// resolveDynamicFields clones fields and injects elapsed/percent values
// for any dynamic field keys configured on the builder. Returns the
// original slice unmodified when no dynamic keys are configured.
func (b *AnimationBuilder) resolveDynamicFields(fields []Field, dur time.Duration) []Field {
	stylePercent := b.barStyle.percentFieldKey() != "" && b.barPercentKey == "" &&
		!b.barStyle.HidePercent
	if b.elapsedKey == "" && b.barPercentKey == "" && !stylePercent {
		return fields
	}
	fields = slices.Clone(fields)
	for i := range fields {
		switch fields[i].Key {
		case b.elapsedKey:
			fields[i].Value = elapsed(dur)
		case b.barPercentKey:
			fields[i].Value = b.barPercentValue()
		}
	}
	if stylePercent {
		fields = append(fields, Field{
			Key:   b.barStyle.percentFieldKey(),
			Value: b.barPercentValue(),
		})
	}
	return fields
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
	w.fields = b.resolveDynamicFields(*fieldsPtr.Load(), time.Since(startTime))
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

	// Build the slot and snapshot the logger's settings.
	slot := &groupSlot{builder: b, msgPtr: msgPtr, fieldsPtr: fields, startTime: startTime}
	captureSlotConfig(slot)

	// Don't animate if not a TTY (CI, piped output, etc.).
	// Print the initial message so the user knows something is in progress.
	if !slot.cfg.isTTY {
		fieldsStr := strings.TrimLeft(
			formatFields(*fields.Load(), slot.fieldOpts), " ",
		)
		line := buildLine(slot.cfg.order, slot.cfg.reportTS,
			time.Now().In(slot.cfg.timeLoc).Format(slot.cfg.timeFmt),
			slot.cfg.label, slot.prefix, *msgPtr.Load(), fieldsStr)
		_, _ = io.WriteString(slot.cfg.out, line+"\n")
		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Hide cursor during animation.
	slot.cfg.termOut.HideCursor()
	defer slot.cfg.termOut.ShowCursor()

	ticker := time.NewTicker(slot.tickRate)
	defer ticker.Stop()

	var frameBuf strings.Builder

	for {
		select {
		case err := <-done:
			// For bar animations, render one final frame so 100% is visible
			// before the line is cleared and replaced with the completion message.
			if b.mode == animationBar && err == nil {
				line := renderSlotLine(slot, false, time.Now())
				frameBuf.Reset()
				frameBuf.WriteString(clearLine)
				frameBuf.WriteString(line)
				_, _ = io.WriteString(slot.cfg.out, frameBuf.String())
			}
			_, _ = io.WriteString(slot.cfg.out, clearLine)
			return err
		case now := <-ticker.C:
			line := renderSlotLine(slot, false, now)
			frameBuf.Reset()
			frameBuf.WriteString(clearLine)
			frameBuf.WriteString(line)
			_, _ = io.WriteString(slot.cfg.out, frameBuf.String())
		case <-ctx.Done():
			_, _ = io.WriteString(slot.cfg.out, clearLine)
			return ctx.Err()
		}
	}
}

// alignBarLine positions barPart relative to msgParts according to the
// alignment mode and terminal width. sep is the fallback separator used
// when the terminal is too narrow for padding.
func alignBarLine(msgParts, barPart, sep string, align BarAlign, tw int) string {
	switch align {
	case BarAlignRightPad:
		gap := tw - lipgloss.Width(msgParts) - lipgloss.Width(barPart)
		if gap > 0 {
			return msgParts + strings.Repeat(" ", gap) + barPart
		}
		return msgParts + sep + barPart
	case BarAlignLeftPad:
		gap := tw - lipgloss.Width(barPart) - lipgloss.Width(msgParts)
		if gap > 0 {
			return barPart + strings.Repeat(" ", gap) + msgParts
		}
		return barPart + sep + msgParts
	case BarAlignRight:
		return msgParts + sep + barPart
	case BarAlignLeft:
		return barPart + sep + msgParts
	case BarAlignInline:
		return msgParts
	default:
		return msgParts
	}
}
