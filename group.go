package clog

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/muesli/termenv"
)

// slotConfig is an immutable snapshot of logger settings captured under the
// logger's mutex. It stores exactly the fields needed for per-tick rendering
// so the animation loop never touches the logger after the initial capture.
type slotConfig struct {
	elapsedFormatFunc       func(time.Duration) string
	elapsedMinimum          time.Duration
	elapsedPrecision        int
	elapsedRound            time.Duration
	fieldSort               Sort
	fieldStyleLevel         Level
	fieldTimeFormat         string
	label                   string    // pre-computed padded label
	levelPrefix             string    // styled label (via styles.Levels[level])
	noColor                 bool      // output.ColorsDisabled()
	order                   []Part    // l.parts
	out                     io.Writer // output.Writer()
	output                  *Output   // for Width() in bar mode
	percentFormatFunc       func(float64) string
	percentPrecision        int
	quantityUnitsIgnoreCase bool
	termOut                 *termenv.Output // output.Renderer().Output()
	quoteClose              rune
	quoteMode               QuoteMode
	quoteOpen               rune
	reportTS                bool
	separatorText           string
	styles                  *Styles
	timeFmt                 string
	timeLoc                 *time.Location
}

// groupSlot holds per-animation mutable state for both the single-animation
// (runAnimation) and multi-animation (Group) paths.
type groupSlot struct {
	builder   *AnimationBuilder
	msgPtr    *atomic.Pointer[string]
	fieldsPtr *atomic.Pointer[[]Field]
	doneErr   chan error // buffered(1); goroutine sends result here (Group only)
	err       error      // populated by Wait() after doneErr is drained (Group only)
	startTime time.Time
	cfg       slotConfig
	tickRate  time.Duration
	prefix    string // resolved icon (builder.prefix or "⏳")

	// per-tick mutable state
	pCache          pulseCache
	hexLUT          *shimmerLUT      // shimmer only, immutable after init
	styleLUT        *shimmerStyleLUT // shimmer only, immutable after init
	cachedFieldsPtr *[]Field         // dedup: last-formatted fields pointer
	cachedFieldsStr string           // dedup: last-formatted fields string
	fieldOpts       formatFieldsOpts // pre-built from slotConfig
}

// captureSlotConfig locks the builder's logger, snapshots all fields into
// s.cfg, and pre-computes s.tickRate, s.prefix, s.fieldOpts, s.cfg.levelPrefix,
// and shimmer LUTs.
func captureSlotConfig(s *groupSlot) {
	b := s.builder
	l := b.resolveLogger()
	l.mu.Lock()
	s.cfg = slotConfig{
		elapsedFormatFunc:       l.elapsedFormatFunc,
		elapsedMinimum:          l.elapsedMinimum,
		elapsedPrecision:        l.elapsedPrecision,
		elapsedRound:            l.elapsedRound,
		fieldSort:               l.fieldSort,
		fieldStyleLevel:         l.fieldStyleLevel,
		fieldTimeFormat:         l.fieldTimeFormat,
		label:                   l.formatLabel(b.level),
		noColor:                 l.output.ColorsDisabled(),
		order:                   l.parts,
		out:                     l.output.Writer(),
		output:                  l.output,
		percentFormatFunc:       l.percentFormatFunc,
		percentPrecision:        l.percentPrecision,
		quantityUnitsIgnoreCase: l.quantityUnitsIgnoreCase,
		termOut:                 l.output.Renderer().Output(),
		quoteClose:              l.quoteClose,
		quoteMode:               l.quoteMode,
		quoteOpen:               l.quoteOpen,
		reportTS:                l.reportTimestamp,
		separatorText:           l.separatorText,
		styles:                  l.styles,
		timeFmt:                 l.timeFormat,
		timeLoc:                 l.timeLocation,
	}
	l.mu.Unlock()

	// Styled level prefix.
	if style := s.cfg.styles.Levels[b.level]; style != nil && !s.cfg.noColor {
		s.cfg.levelPrefix = style.Render(s.cfg.label)
	} else {
		s.cfg.levelPrefix = s.cfg.label
	}

	// Resolve the prefix icon.
	s.prefix = b.prefix
	if s.prefix == "" {
		s.prefix = "⏳"
	}

	// Determine tick rate and pre-compute mode-specific resources.
	switch b.mode {
	case animationSpinner:
		s.tickRate = b.spinner.FPS
	case animationPulse:
		s.tickRate = pulseTickRate
	case animationShimmer:
		s.tickRate = shimmerTickRate
		s.hexLUT = buildShimmerLUT(b.shimmerStops)
		s.styleLUT = buildShimmerStyleLUT(s.hexLUT)
	case animationBar:
		s.tickRate = barTickRate
	}

	// Guard against invalid SpinnerStyle values.
	if b.mode == animationSpinner && len(b.spinner.Frames) == 0 {
		b.spinner.Frames = DefaultSpinnerStyle().Frames
	}
	if s.tickRate <= 0 {
		s.tickRate = DefaultSpinnerStyle().FPS
	}

	// Pre-build the field formatting options.
	s.fieldOpts = formatFieldsOpts{
		elapsedFormatFunc:       s.cfg.elapsedFormatFunc,
		elapsedMinimum:          s.cfg.elapsedMinimum,
		elapsedPrecision:        s.cfg.elapsedPrecision,
		elapsedRound:            s.cfg.elapsedRound,
		fieldSort:               s.cfg.fieldSort,
		fieldStyleLevel:         s.cfg.fieldStyleLevel,
		level:                   b.level,
		percentFormatFunc:       s.cfg.percentFormatFunc,
		percentPrecision:        s.cfg.percentPrecision,
		quantityUnitsIgnoreCase: s.cfg.quantityUnitsIgnoreCase,
		quoteClose:              s.cfg.quoteClose,
		quoteMode:               s.cfg.quoteMode,
		quoteOpen:               s.cfg.quoteOpen,
		separatorText:           s.cfg.separatorText,
		styles:                  s.cfg.styles,
		timeFormat:              s.cfg.fieldTimeFormat,
	}
	if s.cfg.noColor {
		s.fieldOpts.noColor = true
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

// renderSlotFields formats the fields for a slot, caching the result when
// the atomic pointer has not changed.
func renderSlotFields(s *groupSlot, dur time.Duration) string {
	b := s.builder
	fp := s.fieldsPtr.Load()
	stylePercent := b.barStyle.percentFieldKey() != "" && b.barPercentKey == "" &&
		!b.barStyle.HidePercent
	if b.elapsedKey != "" || b.barPercentKey != "" || stylePercent {
		clone := slices.Clone(*fp)
		for i := range clone {
			switch clone[i].Key {
			case b.elapsedKey:
				clone[i].Value = elapsed(dur)
			case b.barPercentKey:
				cur := int(b.barProgressPtr.Load())
				tot := int(b.barTotalPtr.Load())
				pct := float64(cur) / float64(max(tot, 1)) * percentMax
				clone[i].Value = percent(min(pct, percentMax))
			}
		}
		if stylePercent {
			cur := int(b.barProgressPtr.Load())
			tot := int(b.barTotalPtr.Load())
			pct := float64(cur) / float64(max(tot, 1)) * percentMax
			clone = append(
				clone,
				Field{Key: b.barStyle.percentFieldKey(), Value: percent(min(pct, percentMax))},
			)
		}
		s.cachedFieldsStr = strings.TrimLeft(formatFields(clone, s.fieldOpts), " ")
	} else if fp != s.cachedFieldsPtr {
		s.cachedFieldsStr = strings.TrimLeft(formatFields(*fp, s.fieldOpts), " ")
	}
	s.cachedFieldsPtr = fp
	return s.cachedFieldsStr
}

// renderSlotTimestamp returns the styled timestamp string for a slot.
func renderSlotTimestamp(s *groupSlot) string {
	if !s.cfg.reportTS {
		return ""
	}
	ts := time.Now().In(s.cfg.timeLoc).Format(s.cfg.timeFmt)
	if s.cfg.styles.Timestamp != nil && !s.cfg.noColor {
		return s.cfg.styles.Timestamp.Render(ts)
	}
	return ts
}

// renderSlotLine renders a single animation frame line for a slot.
// For done slots, it renders the frozen final state with the level's default prefix.
// For active slots, it renders the current animation frame.
// It does not perform any I/O.
func renderSlotLine(s *groupSlot, isDone bool, now time.Time) string {
	b := s.builder
	dur := now.Sub(s.startTime)
	fieldsStr := renderSlotFields(s, dur)
	tsStr := renderSlotTimestamp(s)

	if isDone {
		// Show the frozen final line with the level's default prefix.
		msg := *s.msgPtr.Load()
		if msgStyle := s.cfg.styles.Messages[b.level]; msgStyle != nil && !s.cfg.noColor {
			msg = msgStyle.Render(msg)
		}
		levelPrefix := s.cfg.levelPrefix
		// Use a checkmark or the builder prefix for completed items.
		donePrefix := s.prefix
		return buildLine(
			s.cfg.order,
			s.cfg.reportTS,
			tsStr,
			levelPrefix,
			donePrefix,
			msg,
			fieldsStr,
		)
	}

	// Bar mode has its own rendering path.
	if b.mode == animationBar {
		return renderSlotBarLine(s, dur, fieldsStr, tsStr)
	}

	msg := *s.msgPtr.Load()
	var char string

	switch b.mode { //nolint:exhaustive // animationBar handled above
	case animationSpinner:
		n := len(b.spinner.Frames)
		i := int(dur/b.spinner.FPS) % n
		if b.spinner.Reverse {
			i = n - 1 - i
		}
		char = b.spinner.Frames[i]
		if msgStyle := s.cfg.styles.Messages[b.level]; msgStyle != nil && !s.cfg.noColor {
			msg = msgStyle.Render(msg)
		}
	case animationPulse:
		char = s.prefix
		t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*pulseSpeed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
		msg = pulseTextCached(msg, t, b.pulseStops, &s.pCache)
	case animationShimmer:
		char = s.prefix
		phase := math.Mod(dur.Seconds()*shimmerSpeed, 1.0)
		msg = shimmerText(msg, phase, b.shimmerDir, s.hexLUT, s.styleLUT)
	}

	return buildLine(s.cfg.order, s.cfg.reportTS, tsStr, s.cfg.levelPrefix, char, msg, fieldsStr)
}

// renderSlotBarLine renders a bar-animation frame for a slot. Factored out to
// keep renderSlotLine focused.
func renderSlotBarLine(s *groupSlot, dur time.Duration, fieldsStr, tsStr string) string {
	b := s.builder
	msg := *s.msgPtr.Load()
	if msgStyle := s.cfg.styles.Messages[b.level]; msgStyle != nil && !s.cfg.noColor {
		msg = msgStyle.Render(msg)
	}

	current := int(b.barProgressPtr.Load())
	total := int(b.barTotalPtr.Load())
	barStr := renderBar(current, total, b.barStyle, s.cfg.output.Width())
	sep := b.barStyle.Separator
	if sep == "" {
		sep = " "
	}

	barFull := barStr
	if !b.barStyle.HidePercent && b.barPercentKey == "" && b.barStyle.percentFieldKey() == "" {
		pct := barPercent(
			current,
			total,
			b.barStyle.PercentPrecision,
			!b.barStyle.NoPadPercent,
		)
		if b.barStyle.PercentPosition == PercentLeft {
			barFull = pct + sep + barStr
		} else {
			barFull = barStr + sep + pct
		}
	}

	// writeFrame equivalent: build the complete line string.
	_ = dur // dur already used in fieldsStr computation

	if b.barStyle.Align == BarAlignInline {
		parts := buildLine(
			s.cfg.order,
			s.cfg.reportTS,
			tsStr,
			s.cfg.levelPrefix,
			s.prefix,
			msg+sep+barFull,
			fieldsStr,
		)
		return parts
	}
	parts := buildLine(
		s.cfg.order,
		s.cfg.reportTS,
		tsStr,
		s.cfg.levelPrefix,
		s.prefix,
		msg,
		fieldsStr,
	)
	return alignBarLine(parts, barFull, sep, b.barStyle.Align, s.cfg.output.Width())
}

// --- Group types ---

// Group manages a set of concurrent animations rendered as a multi-line
// block. Create one with [Group] or [Logger.Group], add animations with
// [Group.Add], then call [Group.Wait] to run the render loop.
type Group struct {
	ctx    context.Context //nolint:containedctx // Group shares a single ctx with all child goroutines
	logger *Logger
	mu     sync.Mutex
	slots  []*groupSlot
}

// NewGroup creates a new animation group using the [Default] logger.
func NewGroup(ctx context.Context) *Group {
	return Default.Group(ctx)
}

// Group creates a new animation group.
func (l *Logger) Group(ctx context.Context) *Group {
	return &Group{ctx: ctx, logger: l}
}

// GroupEntry is returned by [Group.Add] and provides [Run] and [Progress]
// methods to start a task within the group.
type GroupEntry struct {
	slot  *groupSlot
	group *Group
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

	s := &groupSlot{
		builder:   b,
		msgPtr:    msgPtr,
		fieldsPtr: fieldsPtr,
		doneErr:   make(chan error, 1),
		startTime: time.Now(),
	}
	captureSlotConfig(s)

	g.mu.Lock()
	g.slots = append(g.slots, s)
	g.mu.Unlock()

	return &GroupEntry{slot: s, group: g}
}

// Run starts a simple task (no progress updates) and returns a [SlotResult].
func (ge *GroupEntry) Run(task Task) *SlotResult {
	return ge.Progress(func(ctx context.Context, _ *ProgressUpdate) error {
		return task(ctx)
	})
}

// Progress starts a task with progress update capability and returns a [SlotResult].
func (ge *GroupEntry) Progress(task ProgressTask) *SlotResult {
	s := ge.slot
	b := s.builder
	g := ge.group

	update := &ProgressUpdate{
		msg:       b.msg,
		msgPtr:    s.msgPtr,
		fieldsPtr: s.fieldsPtr,
		base:      b.fields,
	}
	if b.mode == animationBar {
		update.progressPtr = b.barProgressPtr
		update.totalPtr = b.barTotalPtr
	}
	update.initSelf(update)

	go func() {
		s.doneErr <- task(g.ctx, update)
	}()

	r := &SlotResult{
		slot:         s,
		logger:       g.logger,
		successLevel: b.level,
		errorLevel:   ErrorLevel,
	}
	r.initSelf(r)
	return r
}

// SlotResult holds the result of a group animation task. It mirrors
// [WaitResult] but reads its error from the slot (set by [Group.Wait]).
type SlotResult struct {
	fieldBuilder[SlotResult]

	slot         *groupSlot
	logger       *Logger
	successLevel Level
	errorLevel   Level
	successMsg   string // empty = use *slot.msgPtr.Load()
	errorMsg     *string
	prefix       *string
}

// Err returns the error, logging success or failure using the original message.
func (r *SlotResult) Err() error {
	return r.Send()
}

// Msg logs at success level with the given message on success, or at error
// level with the error string on failure. Returns the error.
func (r *SlotResult) Msg(msg string) error {
	r.successMsg = msg
	return r.Send()
}

// OnErrorLevel sets the log level for the error case.
func (r *SlotResult) OnErrorLevel(level Level) *SlotResult {
	r.errorLevel = level
	return r
}

// OnErrorMessage sets a custom message for the error case.
func (r *SlotResult) OnErrorMessage(msg string) *SlotResult {
	r.errorMsg = &msg
	return r
}

// OnSuccessLevel sets the log level for the success case.
func (r *SlotResult) OnSuccessLevel(level Level) *SlotResult {
	r.successLevel = level
	return r
}

// OnSuccessMessage sets the message for the success case.
func (r *SlotResult) OnSuccessMessage(msg string) *SlotResult {
	r.successMsg = msg
	return r
}

// Prefix sets a custom emoji prefix for the completion log message.
func (r *SlotResult) Prefix(prefix string) *SlotResult {
	r.prefix = new(prefix)
	return r
}

// Send finalises the result, logging at the configured success or error level.
func (r *SlotResult) Send() error {
	s := r.slot
	err := s.err

	// Resolve message.
	msg := r.successMsg
	if msg == "" {
		msg = *s.msgPtr.Load()
	}

	// Resolve final fields: animation fields + any fields added to the SlotResult.
	finalFields := *s.fieldsPtr.Load()
	b := s.builder
	stylePercent := b.barStyle.percentFieldKey() != "" && b.barPercentKey == "" &&
		!b.barStyle.HidePercent
	if b.elapsedKey != "" || b.barPercentKey != "" || stylePercent {
		finalFields = slices.Clone(finalFields)
		for i := range finalFields {
			switch finalFields[i].Key {
			case b.elapsedKey:
				finalFields[i].Value = elapsed(time.Since(s.startTime))
			case b.barPercentKey:
				cur := int(b.barProgressPtr.Load())
				tot := int(b.barTotalPtr.Load())
				pct := float64(cur) / float64(max(tot, 1)) * percentMax
				finalFields[i].Value = percent(min(pct, percentMax))
			}
		}
		if stylePercent {
			cur := int(b.barProgressPtr.Load())
			tot := int(b.barTotalPtr.Load())
			pct := float64(cur) / float64(max(tot, 1)) * percentMax
			finalFields = append(
				finalFields,
				Field{Key: b.barStyle.percentFieldKey(), Value: percent(min(pct, percentMax))},
			)
		}
	}
	if len(r.fields) > 0 {
		finalFields = mergeFields(finalFields, r.fields)
	}

	l := r.logger
	if l == nil {
		l = Default
	}

	switch {
	case err == nil:
		e := l.newEvent(r.successLevel)
		if e == nil {
			break
		}
		e = e.withFields(finalFields)
		if r.prefix != nil {
			e = e.withPrefix(*r.prefix)
		}
		e.Msg(msg)
	case r.errorMsg != nil:
		e := l.newEvent(r.errorLevel)
		if e == nil {
			break
		}
		e = e.withFields(finalFields)
		if r.prefix != nil {
			e = e.withPrefix(*r.prefix)
		}
		e.Err(err).Msg(*r.errorMsg)
	default:
		e := l.newEvent(r.errorLevel)
		if e == nil {
			break
		}
		e = e.withFields(finalFields)
		if r.prefix != nil {
			e = e.withPrefix(*r.prefix)
		}
		e.Msg(err.Error())
	}

	return err
}

// Silent returns just the error without logging anything.
func (r *SlotResult) Silent() error {
	return r.slot.err
}

// GroupResult holds the aggregate result of a [Group.Wait] and allows
// chaining a single summary log line instead of per-slot messages.
type GroupResult struct {
	fieldBuilder[GroupResult]

	group        *Group
	logger       *Logger
	successLevel Level
	errorLevel   Level
	successMsg   string
	errorMsg     *string
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

// Prefix sets a custom emoji prefix for the completion log message.
func (r *GroupResult) Prefix(prefix string) *GroupResult {
	r.prefix = new(prefix)
	return r
}

// Send finalises the result, logging at the configured success or error level.
// The error is the [errors.Join] of all slot errors (nil when all succeeded).
func (r *GroupResult) Send() error {
	err := r.joinErrors()

	l := r.logger
	if l == nil {
		l = Default
	}

	msg := r.successMsg

	switch {
	case err == nil:
		e := l.newEvent(r.successLevel)
		if e == nil {
			break
		}
		e = e.withFields(r.fields)
		if r.prefix != nil {
			e = e.withPrefix(*r.prefix)
		}
		e.Msg(msg)
	case r.errorMsg != nil:
		e := l.newEvent(r.errorLevel)
		if e == nil {
			break
		}
		e = e.withFields(r.fields)
		if r.prefix != nil {
			e = e.withPrefix(*r.prefix)
		}
		e.Err(err).Msg(*r.errorMsg)
	default:
		e := l.newEvent(r.errorLevel)
		if e == nil {
			break
		}
		e = e.withFields(r.fields)
		if r.prefix != nil {
			e = e.withPrefix(*r.prefix)
		}
		e.Msg(err.Error())
	}

	return err
}

// Silent returns the joined error without logging anything.
func (r *GroupResult) Silent() error {
	return r.joinErrors()
}

// joinErrors returns the [errors.Join] of all slot errors.
func (r *GroupResult) joinErrors() error {
	var errs []error
	for _, s := range r.group.slots {
		if s.err != nil {
			errs = append(errs, s.err)
		}
	}
	return errors.Join(errs...)
}

// Wait runs the render loop, blocking until all slots complete or the context
// is cancelled. After Wait returns, each slot's err field is populated.
// The returned [GroupResult] can be used to log a single summary line;
// alternatively, use individual [SlotResult] values for per-slot messages.
func (g *Group) Wait() *GroupResult {
	g.mu.Lock()
	slots := g.slots
	g.mu.Unlock()

	result := &GroupResult{
		group:        g,
		logger:       g.logger,
		successLevel: InfoLevel,
		errorLevel:   ErrorLevel,
	}
	result.initSelf(result)

	if len(slots) == 0 {
		return result
	}

	// Non-TTY: print each slot's initial line, then block on all results.
	if slots[0].cfg.noColor {
		for _, s := range slots {
			fieldsStr := strings.TrimLeft(
				formatFields(*s.fieldsPtr.Load(), s.fieldOpts), " ",
			)
			line := buildLine(s.cfg.order, s.cfg.reportTS,
				time.Now().In(s.cfg.timeLoc).Format(s.cfg.timeFmt),
				s.cfg.label, s.prefix, *s.msgPtr.Load(), fieldsStr)
			_, _ = io.WriteString(s.cfg.out, line+"\n")
		}
		for _, s := range slots {
			s.err = <-s.doneErr
		}
		return result
	}

	// Tick rate = fastest slot's rate.
	tickRate := slots[0].tickRate
	for _, s := range slots[1:] {
		tickRate = min(tickRate, s.tickRate)
	}

	termOut := slots[0].cfg.termOut
	termOut.HideCursor()
	defer termOut.ShowCursor()

	out := slots[0].cfg.out
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	numLines := 0
	done := make([]bool, len(slots))
	remaining := len(slots)

	for remaining > 0 {
		select {
		case <-g.ctx.Done():
			clearBlock(out, numLines)
			for i, s := range slots {
				if !done[i] {
					s.err = g.ctx.Err()
				}
			}
			return result
		case <-ticker.C:
			now := time.Now()
			// Drain completed tasks.
			for i, s := range slots {
				if done[i] {
					continue
				}
				select {
				case err := <-s.doneErr:
					s.err = err
					done[i] = true
					remaining--
				default:
				}
			}
			// Move cursor up to overwrite the previous block.
			if numLines > 0 {
				fmt.Fprintf(out, "\x1b[%dA", numLines)
			}
			// Render each slot's line.
			for i, s := range slots {
				line := renderSlotLine(s, done[i], now)
				fmt.Fprintf(out, "\x1b[2K\r%s\n", line)
			}
			numLines = len(slots)
			// If all done, break out after one final render.
			if remaining == 0 {
				break
			}
		}
	}

	clearBlock(out, numLines)
	return result
}

// clearBlock erases n lines above the cursor and repositions the cursor.
func clearBlock(out io.Writer, n int) {
	if n == 0 {
		return
	}
	fmt.Fprintf(out, "\x1b[%dA", n)
	for range n {
		fmt.Fprint(out, "\x1b[2K\r\n")
	}
	fmt.Fprintf(out, "\x1b[%dA", n)
}
