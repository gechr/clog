package clog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// --- Group types ---

// Group manages a set of concurrent animations rendered as a multi-line
// block. Create one with [Group] or [Logger.Group], add animations with
// [Group.Add], then call [Group.Wait] to run the render loop.
type Group struct {
	ctx context.Context //nolint:containedctx // Group shares a single ctx with all child goroutines
	mu  sync.Mutex

	logger *Logger
	tasks  []*groupTask
}

// NewGroup creates a new animation group using the [Default] logger.
func NewGroup(ctx context.Context) *Group {
	return Default.Group(ctx)
}

// Group creates a new animation group.
func (l *Logger) Group(ctx context.Context) *Group {
	return &Group{ctx: ctx, logger: l}
}

// Add registers an animation builder with the group and returns a
// [GroupEntry] for starting the task.
func (g *Group) Add(b *AnimationBuilder) *GroupEntry {
	if b.logger == nil {
		b.logger = g.logger
	}

	msgPtr := new(atomic.Pointer[string])
	fieldsPtr := new(atomic.Pointer[[]Field])
	msgPtr.Store(&b.msg)
	fieldsPtr.Store(&b.fields)

	gt := &groupTask{
		builder:   b,
		doneErr:   make(chan error, 1),
		fieldsPtr: fieldsPtr,
		msgPtr:    msgPtr,
		startTime: time.Now(),
	}
	captureTaskConfig(gt)

	g.mu.Lock()
	g.tasks = append(g.tasks, gt)
	g.mu.Unlock()

	return &GroupEntry{task: gt, group: g}
}

// Wait runs the render loop, blocking until all tasks complete or the context
// is cancelled. After Wait returns, each task's err field is populated.
// The returned [GroupResult] can be used to log a single summary line;
// alternatively, use individual [TaskResult] values for per-task messages.
func (g *Group) Wait() *GroupResult {
	g.mu.Lock()
	tasks := g.tasks
	g.mu.Unlock()

	result := &GroupResult{
		group:        g,
		logger:       g.logger,
		successLevel: InfoLevel,
		errorLevel:   ErrorLevel,
	}
	result.initSelf(result)

	if len(tasks) == 0 {
		return result
	}

	// Non-TTY: print each task's initial line, then block on all results.
	// Dynamic fields (elapsed, bar percent) are stripped because their
	// initial zero values are meaningless without live updates.
	if !tasks[0].cfg.isTTY {
		for _, t := range tasks {
			fieldsStr := strings.TrimLeft(
				formatFields(t.builder.stripDynamicFields(*t.fieldsPtr.Load()), t.fieldOpts), " ",
			)
			line := buildLine(t.cfg.order, t.cfg.reportTS,
				time.Now().In(t.cfg.timeLoc).Format(t.cfg.timeFmt),
				t.cfg.label, t.prefix, t.cfg.indentation+*t.msgPtr.Load(), fieldsStr)
			writeString(t.cfg.out, line+"\n")
		}
		for _, t := range tasks {
			select {
			case t.err = <-t.doneErr:
			case <-g.ctx.Done():
				for _, t2 := range tasks {
					if t2.err == nil {
						select {
						case t2.err = <-t2.doneErr:
						default:
							t2.err = g.ctx.Err()
						}
					}
				}
				return result
			}
		}
		return result
	}

	// Tick rate = fastest task's rate.
	tickRate := tasks[0].tickRate
	for _, t := range tasks[1:] {
		tickRate = min(tickRate, t.tickRate)
	}

	termOut := tasks[0].cfg.termOut
	termOut.HideCursor()
	defer termOut.ShowCursor()

	out := tasks[0].cfg.out
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	numLines := 0
	done := make([]bool, len(tasks))
	remaining := len(tasks)
	var frameBuf strings.Builder

	for remaining > 0 {
		select {
		case <-g.ctx.Done():
			clearBlock(out, numLines)
			for i, t := range tasks {
				if !done[i] {
					t.err = g.ctx.Err()
				}
			}
			return result
		case <-ticker.C:
			now := time.Now()
			// Drain completed tasks.
			for i, t := range tasks {
				if done[i] {
					continue
				}
				select {
				case err := <-t.doneErr:
					t.err = err
					done[i] = true
					remaining--
				default:
				}
			}
			// Batch all writes into a single string.
			frameBuf.Reset()
			if numLines > 1 {
				fmt.Fprintf(&frameBuf, "\x1b[%dA", numLines-1)
			}
			for i, t := range tasks {
				line := renderTaskLine(t, done[i], now)
				if i < len(tasks)-1 {
					fmt.Fprintf(&frameBuf, "\x1b[2K\r%s\n", line)
				} else {
					fmt.Fprintf(&frameBuf, "\x1b[2K\r%s", line)
				}
			}
			writeString(out, frameBuf.String())
			numLines = len(tasks)
			// If all done, break out after one final render.
			if remaining == 0 {
				break
			}
		}
	}

	clearBlock(out, numLines)
	return result
}

// GroupEntry is returned by [Group.Add] and provides [Run] and [Progress]
// methods to start a task within the group.
type GroupEntry struct {
	task  *groupTask
	group *Group
}

// Run starts a simple task (no progress updates) and returns a [TaskResult].
func (ge *GroupEntry) Run(task TaskFunc) *TaskResult {
	return ge.Progress(func(ctx context.Context, _ *ProgressUpdate) error {
		return task(ctx)
	})
}

// Progress starts a task with progress update capability and returns a [TaskResult].
func (ge *GroupEntry) Progress(task ProgressTaskFunc) *TaskResult {
	t := ge.task
	b := t.builder
	g := ge.group

	update := &ProgressUpdate{
		msg:       b.msg,
		msgPtr:    t.msgPtr,
		fieldsPtr: t.fieldsPtr,
		base:      b.fields,
	}
	if b.mode == animationBar {
		update.progressPtr = b.barProgressPtr
		update.totalPtr = b.barTotalPtr
	}
	update.initSelf(update)

	go func() {
		t.doneErr <- task(g.ctx, update)
	}()

	r := &TaskResult{
		task:         t,
		logger:       b.indentedLogger(),
		parts:        b.parts,
		successLevel: b.level,
		errorLevel:   ErrorLevel,
	}
	r.initSelf(r)
	return r
}

// TaskResult holds the result of a group animation task. It mirrors
// [WaitResult] but reads its error from the task (set by [Group.Wait]).
type TaskResult struct {
	fieldBuilder[TaskResult]

	task         *groupTask
	logger       *Logger
	successLevel Level
	errorLevel   Level
	successMsg   string // empty = use *task.msgPtr.Load()
	errorMsg     *string
	parts        *[]Part
	prefix       *string
}

// Err returns the error, logging success or failure using the original message.
func (r *TaskResult) Err() error {
	return r.Send()
}

// Msg logs at success level with the given message on success, or at error
// level with the error string on failure. Returns the error.
func (r *TaskResult) Msg(msg string) error {
	r.successMsg = msg
	return r.Send()
}

// OnErrorLevel sets the log level for the error case.
func (r *TaskResult) OnErrorLevel(level Level) *TaskResult {
	r.errorLevel = level
	return r
}

// OnErrorMessage sets a custom message for the error case.
func (r *TaskResult) OnErrorMessage(msg string) *TaskResult {
	r.errorMsg = &msg
	return r
}

// OnSuccessLevel sets the log level for the success case.
func (r *TaskResult) OnSuccessLevel(level Level) *TaskResult {
	r.successLevel = level
	return r
}

// OnSuccessMessage sets the message for the success case.
func (r *TaskResult) OnSuccessMessage(msg string) *TaskResult {
	r.successMsg = msg
	return r
}

// Parts overrides the log-line part order for the completion message.
func (r *TaskResult) Parts(parts ...Part) *TaskResult {
	r.parts = new(parts)
	return r
}

// Prefix sets a custom emoji prefix for the completion log message.
func (r *TaskResult) Prefix(prefix string) *TaskResult {
	r.prefix = new(prefix)
	return r
}

// Send finalises the result, logging at the configured success or error level.
func (r *TaskResult) Send() error {
	t := r.task
	err := t.err

	// Resolve message.
	msg := r.successMsg
	if msg == "" {
		msg = *t.msgPtr.Load()
	}

	// Resolve final fields: animation fields + any fields added to the TaskResult.
	finalFields := t.builder.resolveDynamicFields(*t.fieldsPtr.Load(), time.Since(t.startTime))
	if len(r.fields) > 0 {
		finalFields = mergeFields(finalFields, r.fields)
	}

	return sendResult(
		r.logger,
		finalFields,
		r.parts,
		r.prefix,
		r.successLevel,
		r.errorLevel,
		msg,
		r.errorMsg,
		err,
	)
}

// Silent returns just the error without logging anything.
func (r *TaskResult) Silent() error {
	return r.task.err
}

// GroupResult holds the aggregate result of a [Group.Wait] and allows
// chaining a single summary log line instead of per-task messages.
type GroupResult struct {
	fieldBuilder[GroupResult]

	group        *Group
	logger       *Logger
	successLevel Level
	errorLevel   Level
	successMsg   string
	errorMsg     *string
	parts        *[]Part
	prefix       *string
}

// Err returns the joined error, logging success at info level or failure at
// error level using the original message.
func (r *GroupResult) Err() error {
	return r.Send()
}

// Msg logs at success level with the given message if all tasks succeeded,
// or at error level with the joined error string on failure. Returns the error.
func (r *GroupResult) Msg(msg string) error {
	r.successMsg = msg
	return r.Send()
}

// OnErrorLevel sets the log level for the error case.
func (r *GroupResult) OnErrorLevel(level Level) *GroupResult {
	r.errorLevel = level
	return r
}

// OnErrorMessage sets a custom message for the error case.
func (r *GroupResult) OnErrorMessage(msg string) *GroupResult {
	r.errorMsg = &msg
	return r
}

// OnSuccessLevel sets the log level for the success case.
func (r *GroupResult) OnSuccessLevel(level Level) *GroupResult {
	r.successLevel = level
	return r
}

// OnSuccessMessage sets the message for the success case.
func (r *GroupResult) OnSuccessMessage(msg string) *GroupResult {
	r.successMsg = msg
	return r
}

// Parts overrides the log-line part order for the completion message.
func (r *GroupResult) Parts(parts ...Part) *GroupResult {
	r.parts = new(parts)
	return r
}

// Prefix sets a custom emoji prefix for the completion log message.
func (r *GroupResult) Prefix(prefix string) *GroupResult {
	r.prefix = new(prefix)
	return r
}

// Send finalises the result, logging at the configured success or error level.
// The error is the [errors.Join] of all task errors (nil when all succeeded).
func (r *GroupResult) Send() error {
	err := r.joinErrors()
	return sendResult(
		r.logger,
		r.fields,
		r.parts,
		r.prefix,
		r.successLevel,
		r.errorLevel,
		r.successMsg,
		r.errorMsg,
		err,
	)
}

// Silent returns the joined error without logging anything.
func (r *GroupResult) Silent() error {
	return r.joinErrors()
}

// joinErrors returns the [errors.Join] of all task errors.
func (r *GroupResult) joinErrors() error {
	var errs []error
	for _, s := range r.group.tasks {
		if s.err != nil {
			errs = append(errs, s.err)
		}
	}
	return errors.Join(errs...)
}

// taskConfig is an immutable snapshot of logger settings captured under the
// logger's mutex. It stores exactly the fields needed for per-tick rendering
// so the animation loop never touches the logger after the initial capture.
type taskConfig struct {
	indentation string    // pre-computed indent string for message prefix
	isTTY       bool      // output.IsTTY()
	label       string    // pre-computed padded label
	levelPrefix string    // styled label (via styles.Levels[level])
	noColor     bool      // output.ColorsDisabled()
	order       []Part    // l.parts
	out         io.Writer // output.Writer()
	output      *Output   // for Width() in bar mode
	reportTS    bool
	styles      *Styles
	termOut     *termenv.Output // output.Renderer().Output()
	timeFmt     string
	timeLoc     *time.Location
}

// groupTask holds per-animation mutable state for both the single-animation
// (runAnimation) and multi-animation (Group) paths.
type groupTask struct {
	builder   *AnimationBuilder
	cfg       taskConfig
	doneErr   chan error // buffered(1); goroutine sends result here (Group only)
	err       error      // populated by Wait() after doneErr is drained (Group only)
	fieldsPtr *atomic.Pointer[[]Field]
	msgPtr    *atomic.Pointer[string]
	prefix    string // resolved icon (builder.prefix or "⏳")
	startTime time.Time
	tickRate  time.Duration

	// per-tick mutable state
	cachedFieldsPtr *[]Field         // dedup: last-formatted fields pointer
	cachedFieldsStr string           // dedup: last-formatted fields string
	fieldOpts       formatFieldsOpts // pre-built from taskConfig
	hexLUT          *shimmerLUT      // shimmer only, immutable after init
	pCache          pulseCache
	styleLUT        *shimmerStyleLUT // shimmer only, immutable after init

	// gradient cache (bar mode with ProgressGradient only)
	gradientProgress float64
	gradientStyle    lipgloss.Style
	gradientValid    bool
}

// captureTaskConfig locks the builder's logger, snapshots all fields into
// s.cfg, and pre-computes s.tickRate, s.prefix, s.fieldOpts, s.cfg.levelPrefix,
// and shimmer LUTs.
func captureTaskConfig(gt *groupTask) {
	b := gt.builder
	l := b.resolveLogger()
	l.mu.Lock()
	animInterval := l.animationInterval
	order := l.parts
	if b.parts != nil {
		order = *b.parts
	}
	combinedTree := l.tree
	if len(b.tree) > 0 {
		combinedTree = append(append([]TreePos{}, l.tree...), b.tree...)
	}
	gt.cfg = taskConfig{
		indentation: computeIndent(
			l.indent+b.depth,
			l.indentWidth,
			l.indentPrefixes,
			l.indentPrefixSep,
		) + computeTreeIndent(combinedTree, l.treeChars),
		isTTY:    l.output.IsTTY(),
		label:    l.formatLabel(b.level),
		noColor:  l.output.ColorsDisabled(),
		order:    order,
		out:      l.output.Writer(),
		output:   l.output,
		reportTS: l.reportTimestamp,
		styles:   l.styles,
		termOut:  l.output.Renderer().Output(),
		timeFmt:  l.timeFormat,
		timeLoc:  l.timeLocation,
	}
	gt.fieldOpts = formatFieldsOpts{
		elapsedFormatFunc:       l.elapsedFormatFunc,
		elapsedMinimum:          l.elapsedMinimum,
		elapsedPrecision:        l.elapsedPrecision,
		elapsedRound:            l.elapsedRound,
		fieldSort:               l.fieldSort,
		fieldStyleLevel:         l.fieldStyleLevel,
		level:                   b.level,
		noColor:                 l.output.ColorsDisabled(),
		percentFormatFunc:       l.percentFormatFunc,
		percentPrecision:        l.percentPrecision,
		quantityUnitsIgnoreCase: l.quantityUnitsIgnoreCase,
		quoteOpen:               l.quoteOpen,
		quoteClose:              l.quoteClose,
		quoteMode:               l.quoteMode,
		separatorText:           l.separatorText,
		styles:                  l.styles,
		timeFormat:              l.fieldTimeFormat,
	}
	l.mu.Unlock()

	// Styled level prefix.
	if style := gt.cfg.styles.Levels[b.level]; style != nil && !gt.cfg.noColor {
		gt.cfg.levelPrefix = style.Render(gt.cfg.label)
	} else {
		gt.cfg.levelPrefix = gt.cfg.label
	}

	// Resolve the prefix icon.
	gt.prefix = b.prefix
	if gt.prefix == "" {
		gt.prefix = "⏳"
	}

	// Determine tick rate and pre-compute mode-specific resources.
	switch b.mode {
	case animationSpinner:
		gt.tickRate = b.spinner.FPS
	case animationPulse:
		gt.tickRate = pulseTickRate
	case animationShimmer:
		gt.tickRate = shimmerTickRate
		gt.hexLUT = buildShimmerLUT(b.shimmerStops)
		gt.styleLUT = buildShimmerStyleLUT(gt.hexLUT, gt.cfg.output.Renderer())
	case animationBar:
		gt.tickRate = barTickRate
	}

	// Guard against invalid SpinnerStyle values.
	if b.mode == animationSpinner && len(b.spinner.Frames) == 0 {
		b.spinner.Frames = DefaultSpinnerStyle().Frames
	}
	if b.mode == animationSpinner && b.spinner.Boomerang {
		b.spinner.Frames = boomerangFrames(b.spinner.Frames)
	}
	if gt.tickRate <= 0 {
		gt.tickRate = DefaultSpinnerStyle().FPS
	}
	if animInterval > 0 && gt.tickRate < animInterval {
		gt.tickRate = animInterval
	}
}

// buildLine assembles a log line from the configured parts order.
func buildLine(order []Part, reportTS bool, tsStr, levelStr, prefix, msg, fieldsStr string) string {
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

// styledMsg applies the message style for the given level, if any.
func styledMsg(msg string, level Level, styles *Styles, noColor bool) string {
	if s := styles.Messages[level]; s != nil && !noColor {
		return s.Render(msg)
	}
	return msg
}

// renderTaskFields formats the fields for a task, caching the result when
// the atomic pointer has not changed.
func renderTaskFields(gt *groupTask, dur time.Duration) string {
	b := gt.builder
	fp := gt.fieldsPtr.Load()
	if b.elapsedKey != "" || b.barPercentKey != "" {
		resolved := b.resolveDynamicFields(*fp, dur)
		gt.cachedFieldsStr = strings.TrimLeft(formatFields(resolved, gt.fieldOpts), " ")
	} else if fp != gt.cachedFieldsPtr {
		gt.cachedFieldsStr = strings.TrimLeft(formatFields(*fp, gt.fieldOpts), " ")
	}
	gt.cachedFieldsPtr = fp
	return gt.cachedFieldsStr
}

// renderTaskTimestamp returns the styled timestamp string for a task.
func renderTaskTimestamp(gt *groupTask) string {
	if !gt.cfg.reportTS {
		return ""
	}
	ts := time.Now().In(gt.cfg.timeLoc).Format(gt.cfg.timeFmt)
	if gt.cfg.styles.Timestamp != nil && !gt.cfg.noColor {
		return gt.cfg.styles.Timestamp.Render(ts)
	}
	return ts
}

// renderTaskLine renders a single animation frame line for a task.
// For done tasks, it renders the frozen final state with the level's default prefix.
// For active tasks, it renders the current animation frame.
// It does not perform any I/O.
func renderTaskLine(gt *groupTask, isDone bool, now time.Time) string {
	b := gt.builder
	dur := now.Sub(gt.startTime)
	fieldsStr := renderTaskFields(gt, dur)
	tsStr := renderTaskTimestamp(gt)

	if isDone {
		// Show the frozen final line with the level's default prefix.
		msg := gt.cfg.indentation + styledMsg(
			*gt.msgPtr.Load(),
			b.level,
			gt.cfg.styles,
			gt.cfg.noColor,
		)
		levelPrefix := gt.cfg.levelPrefix
		// Use a checkmark or the builder prefix for completed items.
		donePrefix := gt.prefix
		return buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			levelPrefix,
			donePrefix,
			msg,
			fieldsStr,
		)
	}

	// Bar mode has its own rendering path.
	if b.mode == animationBar {
		return renderTaskBarLine(gt, fieldsStr, tsStr, now)
	}

	msg := *gt.msgPtr.Load()
	var char string

	switch b.mode { //nolint:exhaustive // animationBar handled above
	case animationSpinner:
		n := len(b.spinner.Frames)
		i := int(dur/b.spinner.FPS) % n
		if b.spinner.Reverse {
			i = n - 1 - i
		}
		char = b.spinner.Frames[i]
		msg = styledMsg(msg, b.level, gt.cfg.styles, gt.cfg.noColor)
	case animationPulse:
		char = gt.prefix
		t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*b.speed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
		msg = pulseTextCached(msg, t, b.pulseStops, &gt.pCache, gt.cfg.output.Renderer())
	case animationShimmer:
		char = gt.prefix
		phase := math.Mod(dur.Seconds()*b.speed, 1.0)
		msg = shimmerText(msg, phase, b.shimmerDir, gt.hexLUT, gt.styleLUT)
	}

	return buildLine(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelPrefix,
		char,
		gt.cfg.indentation+msg,
		fieldsStr,
	)
}

// renderTaskBarLine renders a bar-animation frame for a task. Factored out to
// keep renderTaskLine focused.
func renderTaskBarLine(gt *groupTask, fieldsStr, tsStr string, now time.Time) string {
	b := gt.builder
	msg := gt.cfg.indentation + styledMsg(*gt.msgPtr.Load(), b.level, gt.cfg.styles, gt.cfg.noColor)

	current := int(b.barProgressPtr.Load())
	total := int(b.barTotalPtr.Load())

	// Cache the gradient style to avoid lipgloss.NewStyle() per frame.
	barStyle := b.barStyle
	if len(barStyle.ProgressGradient) > 0 {
		progress := float64(current) / float64(max(total, 1))
		if !gt.gradientValid || gt.gradientProgress != progress {
			c := interpolateGradient(progress, barStyle.ProgressGradient)
			gt.gradientStyle = gt.cfg.output.Renderer().
				NewStyle().
				Foreground(lipgloss.Color(c.Clamped().Hex()))
			gt.gradientProgress = progress
			gt.gradientValid = true
		}
		barStyle.StyleFill = &gt.gradientStyle
		barStyle.ProgressGradient = nil // prevent renderBar from recomputing
	}
	barStr := renderBar(current, total, barStyle, gt.cfg.output.Width())
	sep := b.barStyle.Separator
	if sep == "" {
		sep = " "
	}

	elapsed := now.Sub(gt.startTime)
	var rate float64
	if secs := elapsed.Seconds(); secs > 0 && current > 0 {
		rate = float64(current) / secs
	}
	state := BarState{Current: current, Total: total, Elapsed: elapsed, Rate: rate}

	var leftText, rightText string
	if b.barStyle.WidgetLeft != nil {
		leftText = b.barStyle.WidgetLeft(state)
	}
	if b.barStyle.WidgetRight != nil && b.barPercentKey == "" {
		rightText = b.barStyle.WidgetRight(state)
	} else if b.barStyle.WidgetLeft == nil && b.barStyle.WidgetRight == nil && b.barPercentKey == "" {
		// Default: padded percent on the right when no widgets are configured
		// and no BarPercent field is set.
		rightText = barPercent(current, total, 0, true)
	}

	barFull := barStr
	if leftText != "" {
		barFull = leftText + sep + barFull
	}
	if rightText != "" {
		barFull = barFull + sep + rightText
	}

	// writeFrame equivalent: build the complete line string.
	if b.barStyle.Align == BarAlignInline {
		parts := buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			gt.cfg.levelPrefix,
			gt.prefix,
			msg+sep+barFull,
			fieldsStr,
		)
		return parts
	}
	parts := buildLine(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelPrefix,
		gt.prefix,
		msg,
		fieldsStr,
	)
	return alignBarLine(parts, barFull, sep, b.barStyle.Align, gt.cfg.output.Width())
}

// sendResult logs a success or error event and returns the error.
func sendResult(
	l *Logger,
	fields []Field,
	parts *[]Part,
	prefix *string,
	successLevel, errorLevel Level,
	successMsg string,
	errorMsg *string,
	err error,
) error {
	if l == nil {
		l = Default
	}

	var level Level
	var msg string
	var errField error

	switch {
	case err == nil:
		level = successLevel
		msg = successMsg
	case errorMsg != nil:
		level = errorLevel
		msg = *errorMsg
		errField = err
	default:
		level = errorLevel
		msg = err.Error()
	}

	e := l.newEvent(level)
	if e == nil {
		return err
	}
	e = e.withFields(fields)
	if parts != nil {
		e = e.withParts(parts)
	}
	if prefix != nil {
		e = e.withPrefix(*prefix)
	}
	if errField != nil {
		e.Err(errField).Msg(msg)
	} else {
		e.Msg(msg)
	}
	return err
}

// clearBlock erases n lines above the cursor and repositions the cursor.
func clearBlock(out io.Writer, n int) {
	if n == 0 {
		return
	}
	var buf strings.Builder
	fmt.Fprintf(&buf, "\x1b[%dA", n)
	for range n {
		buf.WriteString("\x1b[2K\r\n")
	}
	fmt.Fprintf(&buf, "\x1b[%dA", n)
	writeString(out, buf.String())
}
