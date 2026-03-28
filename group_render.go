package clog

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/pulse"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
)

// taskConfig is an immutable snapshot of logger settings captured under the
// logger's mutex. It stores exactly the fields needed for per-tick rendering
// so the animation loop never touches the logger after the initial capture.
type taskConfig struct {
	indentation  string    // pre-computed indent string before message
	isTTY        bool      // output.IsTTY()
	label        string    // pre-computed padded label
	labels       LabelMap  // all padded labels, for level overrides
	levelSymbol  string    // styled label (via styles.Levels[level])
	noColor      bool      // output.ColorsDisabled()
	nonTTYSilent bool      // builder.SuppressNonTTY || level < logger.nonTTYLevel
	order        []Part    // l.parts
	out          io.Writer // output.Writer()
	output       *Output   // for Width() in bar mode
	reportTS     bool
	styles       *style.Config
	timeFmt      string
	timeLoc      *time.Location
}

// groupTask holds per-animation mutable state for both the single-animation
// (runAnimation) and multi-animation (Group) paths. It embeds *fx.GroupTask
// for shared state and adds rendering-specific fields.
type groupTask struct {
	*fx.GroupTask

	cfg            taskConfig
	maxBarProgress float64
	maxBarTotal    int
	monotonic      bool
	syncEpoch      time.Time
	tickRate       time.Duration
	visible        bool

	// per-tick mutable state
	cachedFieldsPtr *[]core.Field    // dedup: last-formatted fields pointer
	cachedFieldsStr string           // dedup: last-formatted fields string
	fieldOpts       formatFieldsOpts // pre-built from taskConfig
	hexLUT          *shimmer.LUT     // shimmer only, immutable after init
	pCache          pulse.Cache
	styleLUT        *shimmer.StyleLUT // shimmer only, immutable after init

	// gradient cache (bar mode with ProgressGradient only)
	gradientProgress float64
	gradientStyle    lipgloss.Style
	gradientValid    bool

	// smoothing state (bar mode with SmoothEase only)
	smoothedProgress float64
	smoothedTime     time.Time
	smoothedInit     bool

	// throttled timing snapshot (bar mode with UpdateInterval only)
	barWidgetState barWidgetState
	barWidgetValid bool
}

// resolveLevel returns the effective level and styled level symbol for a
// completed task. If SetLevel was called on the Update, the overridden
// level is used; otherwise the builder's original level applies.
func (gt *groupTask) resolveLevel() (Level, string) {
	b := gt.Builder
	if gt.LevelPtr != nil {
		if override := Level(gt.LevelPtr.Load()); override != UnsetLevel {
			label := gt.cfg.labels[override]
			if s := gt.cfg.styles.Levels[override]; s != nil && !gt.cfg.noColor {
				return override, s.Render(label)
			}
			return override, label
		}
	}
	return b.Level, gt.cfg.levelSymbol
}

type barWidgetState struct {
	elapsed  time.Duration
	rate     float64
	renderAt time.Time
}

type groupBarColumns struct {
	hasLeft  bool
	hasRight bool
	maxBar   int
	maxLeft  int
	maxParts int
	maxRight int
}

// groupBarAligned tracks the maximum message-parts width for PlaceAligned bars
// so that all bars in a group start at the same column.
type groupBarAligned struct {
	hasLeft  bool
	hasRight bool
	maxBar   int
	maxLeft  int
	maxParts int
	maxRight int
}

type groupBarLayout struct {
	aligned  groupBarAligned
	leftPad  groupBarColumns
	rightPad groupBarColumns
}

type groupFieldLayout struct {
	alignment fx.FieldAlignment
	maxStart  int
}

type groupRenderLayout struct {
	bar    groupBarLayout
	fields groupFieldLayout
}

const (
	barColumnCount    = 3
	monotonicBarScale = 1000
)

func shouldRenderTask(gt *groupTask, isDone bool, now time.Time) bool {
	if gt.visible {
		return true
	}

	delay := gt.Builder.DelayDur
	if delay <= 0 {
		gt.visible = true
		return true
	}
	if !gt.Started() {
		return false
	}
	if isDone {
		if finishedAt := gt.FinishTime(); !finishedAt.IsZero() {
			if gt.Duration(finishedAt) < delay {
				return false
			}
			gt.visible = true
			return true
		}
	}
	if gt.Duration(now) < delay {
		return false
	}

	gt.visible = true
	return true
}

func visibleTaskIndexes(gts []*groupTask, done []bool, hideDone bool, now time.Time) []int {
	indexes := make([]int, 0, len(gts))
	for i, gt := range gts {
		if hideDone && done[i] {
			continue
		}
		if shouldRenderTask(gt, done[i], now) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

// prioritiseActive selects up to maxLines indexes, preferring active
// (started and not done) tasks over completed or pending ones.
// Within each group the original registration order is preserved.
func prioritiseActive(visible []int, gts []*groupTask, done []bool, maxLines int) []int {
	active := make([]int, 0, len(visible))
	other := make([]int, 0, len(visible))
	for _, idx := range visible {
		if gts[idx].Started() && !done[idx] {
			active = append(active, idx)
		} else {
			other = append(other, idx)
		}
	}
	out := make([]int, 0, maxLines)
	out = append(out, active...)
	if len(out) < maxLines {
		out = append(out, other...)
	}
	if len(out) > maxLines {
		out = out[:maxLines]
	}
	return out
}

func renderBarProgress(progress float64, s bar.Style, termWidth int) string {
	progress = max(0, min(1, progress))
	current := int(math.Round(progress * monotonicBarScale))
	return bar.Render(current, monotonicBarScale, s, termWidth)
}

func (l *groupFieldLayout) enabled() bool {
	return l != nil && l.alignment == fx.FieldAlignmentMessage && l.maxStart > 0
}

func (l *groupBarLayout) observe(
	parts, leftText, barStr, rightText string,
	placement bar.Placement,
) {
	partsW := lipgloss.Width(parts)
	leftW := lipgloss.Width(leftText)
	barW := lipgloss.Width(barStr)
	rightW := lipgloss.Width(rightText)

	switch placement {
	case bar.PlaceLeftPad:
		l.leftPad.hasLeft = l.leftPad.hasLeft || leftText != ""
		l.leftPad.hasRight = l.leftPad.hasRight || rightText != ""
		l.leftPad.maxLeft = max(l.leftPad.maxLeft, leftW)
		l.leftPad.maxParts = max(l.leftPad.maxParts, partsW)
		l.leftPad.maxBar = max(l.leftPad.maxBar, barW)
		l.leftPad.maxRight = max(l.leftPad.maxRight, rightW)
	case bar.PlaceRightPad:
		l.rightPad.hasLeft = l.rightPad.hasLeft || leftText != ""
		l.rightPad.hasRight = l.rightPad.hasRight || rightText != ""
		l.rightPad.maxLeft = max(l.rightPad.maxLeft, leftW)
		l.rightPad.maxParts = max(l.rightPad.maxParts, partsW)
		l.rightPad.maxBar = max(l.rightPad.maxBar, barW)
		l.rightPad.maxRight = max(l.rightPad.maxRight, rightW)
	case bar.PlaceAligned:
		l.aligned.hasLeft = l.aligned.hasLeft || leftText != ""
		l.aligned.hasRight = l.aligned.hasRight || rightText != ""
		l.aligned.maxLeft = max(l.aligned.maxLeft, leftW)
		l.aligned.maxParts = max(l.aligned.maxParts, partsW)
		l.aligned.maxBar = max(l.aligned.maxBar, barW)
		l.aligned.maxRight = max(l.aligned.maxRight, rightW)
	case bar.PlaceInline, bar.PlaceLeft, bar.PlaceRight:
		return
	}
}

func (l *groupBarLayout) format(
	parts, leftText, barStr, rightText, sep string,
	placement bar.Placement,
	termWidth int,
) string {
	switch placement {
	case bar.PlaceLeftPad:
		return l.formatLeftPad(parts, leftText, barStr, rightText, sep, termWidth)
	case bar.PlaceRightPad:
		return l.formatRightPad(parts, leftText, barStr, rightText, sep, termWidth)
	case bar.PlaceAligned:
		return l.formatAligned(parts, leftText, barStr, rightText, sep)
	case bar.PlaceInline, bar.PlaceLeft, bar.PlaceRight:
		barFull := assembleBarColumns(groupBarColumns{}, leftText, barStr, rightText, sep)
		return bar.FormatLine(parts, barFull, sep, placement, termWidth)
	}

	return ""
}

func (l *groupBarLayout) formatLeftPad(
	parts, leftText, barStr, rightText, sep string,
	termWidth int,
) string {
	shared := l.leftPad
	barFull := assembleBarColumns(shared, leftText, barStr, rightText, sep)
	if shared.maxParts == 0 || shared.maxBar == 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceLeftPad, termWidth)
	}

	gap := termWidth - shared.maxParts - shared.barWidth(sep)
	if gap < 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceLeftPad, termWidth)
	}

	return barFull +
		strings.Repeat(" ", gap+shared.maxParts-lipgloss.Width(parts)) +
		parts
}

func (l *groupBarLayout) formatRightPad(
	parts, leftText, barStr, rightText, sep string,
	termWidth int,
) string {
	shared := l.rightPad
	barFull := assembleBarColumns(shared, leftText, barStr, rightText, sep)
	if shared.maxParts == 0 || shared.maxBar == 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceRightPad, termWidth)
	}

	gap := termWidth - shared.maxParts - shared.barWidth(sep)
	if gap < 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceRightPad, termWidth)
	}

	return parts + strings.Repeat(" ", shared.maxParts-lipgloss.Width(parts)+gap) + barFull
}

func (l *groupBarLayout) formatAligned(
	parts, leftText, barStr, rightText, sep string,
) string {
	a := l.aligned
	barCols := groupBarColumns{
		hasLeft:  a.hasLeft,
		hasRight: a.hasRight,
		maxBar:   a.maxBar,
		maxLeft:  a.maxLeft,
		maxRight: a.maxRight,
	}
	barFull := assembleBarColumns(barCols, leftText, barStr, rightText, sep)
	if a.maxParts == 0 {
		return parts + sep + barFull
	}
	padding := a.maxParts - lipgloss.Width(parts)
	if padding <= 0 {
		return parts + sep + barFull
	}
	return parts + strings.Repeat(" ", padding) + sep + barFull
}

func (c groupBarColumns) barWidth(sep string) int {
	width := c.maxBar
	sepW := lipgloss.Width(sep)
	if c.hasLeft {
		width += c.maxLeft + sepW
	}
	if c.hasRight {
		width += sepW + c.maxRight
	}
	return width
}

func assembleBarColumns(cols groupBarColumns, leftText, barStr, rightText, sep string) string {
	parts := make([]string, 0, barColumnCount)
	if cols.hasLeft {
		parts = append(parts, padBarColumnLeft(leftText, cols.maxLeft))
	}
	parts = append(parts, padBarColumnRight(barStr, cols.maxBar))
	if cols.hasRight {
		parts = append(parts, padBarColumnRight(rightText, cols.maxRight))
	}
	return strings.Join(parts, sep)
}

func padBarColumnLeft(text string, width int) string {
	padding := width - lipgloss.Width(text)
	if padding <= 0 {
		return text
	}
	return strings.Repeat(" ", padding) + text
}

func padBarColumnRight(text string, width int) string {
	padding := width - lipgloss.Width(text)
	if padding <= 0 {
		return text
	}
	return text + strings.Repeat(" ", padding)
}

func loadBarWidgetState(gt *groupTask, now time.Time) barWidgetState {
	current := int(gt.Builder.BarProgressPtr.Load())
	elapsed := gt.Duration(now)
	rate := 0.0
	if secs := elapsed.Seconds(); secs > 0 && current > 0 {
		rate = float64(current) / secs
	}

	state := barWidgetState{
		elapsed:  elapsed,
		rate:     rate,
		renderAt: now,
	}

	updateInterval := gt.Builder.BarStyle.UpdateInterval
	if updateInterval <= 0 {
		gt.barWidgetState = state
		gt.barWidgetValid = true
		return state
	}

	if !gt.barWidgetValid || now.Sub(gt.barWidgetState.renderAt) >= updateInterval {
		gt.barWidgetState = state
		gt.barWidgetValid = true
	}

	return gt.barWidgetState
}

func resetBarWidgetState(gt *groupTask) {
	gt.barWidgetValid = false
}

// captureTaskConfig locks the builder's logger, snapshots all fields into
// s.cfg, and pre-computes s.tickRate, s.symbol, s.fieldOpts, s.cfg.levelSymbol,
// and shimmer LUTs.
func captureTaskConfig(gt *groupTask) {
	b := gt.Builder
	l := b.Log.(fxLogger).l //nolint:errcheck,forcetypeassert // fxLogger is the only Logger impl
	l.mu.Lock()
	animInterval := l.animationInterval
	order := l.parts
	if b.PartOverrides != nil {
		order = *b.PartOverrides
	}
	combinedTree := l.tree
	if len(b.TreePos) > 0 {
		combinedTree = append(append([]TreePos{}, l.tree...), b.TreePos...)
	}
	gt.cfg = taskConfig{
		indentation: computeIndent(
			l.indent+b.IndentDepth,
			l.indentWidth,
			l.indentPrefixes,
			l.indentPrefixSep,
		) + computeTreeIndent(combinedTree, l.treeChars),
		isTTY:        l.output.IsTTY(),
		label:        l.formatLabel(b.Level),
		labels:       l.allPaddedLabels(),
		noColor:      l.output.ColorsDisabled(),
		nonTTYSilent: b.SuppressNonTTY || (l.nonTTYLevel != UnsetLevel && b.Level < l.nonTTYLevel),
		order:        order,
		out:          l.output.Writer(),
		output:       l.output,
		reportTS:     l.reportTimestamp,
		styles:       l.styles,
		timeFmt:      l.timeFormat,
		timeLoc:      l.timeLocation,
	}
	gt.fieldOpts = formatFieldsOpts{
		fieldSort:       l.fieldSort,
		fieldStyleLevel: l.fieldStyleLevel,
		level:           b.Level,
		noColor:         l.output.ColorsDisabled(),
		quoteOpen:       l.quoteOpen,
		quoteClose:      l.quoteClose,
		quoteMode:       l.quoteMode,
		separatorText:   l.separatorText,
		sliceClose:      l.sliceClose,
		sliceOpen:       l.sliceOpen,
		sliceSep:        l.sliceSep,
		styles:          l.styles,
		timeFormat:      l.fieldTimeFormat,
	}
	l.mu.Unlock()

	// Styled level symbol.
	if style := gt.cfg.styles.Levels[b.Level]; style != nil && !gt.cfg.noColor {
		gt.cfg.levelSymbol = style.Render(gt.cfg.label)
	} else {
		gt.cfg.levelSymbol = gt.cfg.label
	}

	// Determine tick rate and pre-compute mode-specific resources.
	switch b.Mode {
	case fx.AnimationNone:
		if b.AnimatedSymbol {
			gt.tickRate = b.SpinnerStyle.Interval
		}
	case fx.AnimationPulse:
		gt.tickRate = pulse.TickRate
	case fx.AnimationShimmer:
		gt.tickRate = shimmer.TickRate
		gt.hexLUT = shimmer.BuildLUT(b.ShimmerStops)
		gt.styleLUT = shimmer.BuildStyleLUT(gt.hexLUT)
	case fx.AnimationBar:
		gt.tickRate = bar.TickRate
	}
	// When animated symbol is enabled on a non-spinner mode, ensure the
	// tick rate is fast enough for smooth spinner frame changes.
	if b.AnimatedSymbol && b.SpinnerStyle.Interval > 0 && gt.tickRate > 0 {
		gt.tickRate = min(gt.tickRate, b.SpinnerStyle.Interval)
	}

	// Guard against missing spinner frames when animated symbol is enabled.
	if b.AnimatedSymbol && len(b.SpinnerStyle.Frames) == 0 {
		b.SpinnerStyle.Frames = spinner.DefaultStyle().Frames
	}
	if b.AnimatedSymbol && b.SpinnerStyle.Boomerang {
		b.SpinnerStyle.Frames = spinner.BoomerangFrames(b.SpinnerStyle.Frames)
	}
	if gt.tickRate <= 0 {
		gt.tickRate = spinner.DefaultStyle().Interval
	}
	if animInterval > 0 && gt.tickRate < animInterval {
		gt.tickRate = animInterval
	}
}

// buildLine assembles a log line from the configured parts order.
func buildLine(order []Part, reportTS bool, tsStr, levelStr, symbol, msg, fieldsStr string) string {
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
		case PartSymbol:
			part = symbol
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

func supportsFieldAlignment(order []Part, alignment fx.FieldAlignment) bool {
	if alignment != fx.FieldAlignmentMessage {
		return false
	}

	messageIndex := -1
	fieldsIndex := -1
	for i, part := range order {
		switch part {
		case PartMessage:
			if messageIndex == -1 {
				messageIndex = i
			}
		case PartFields:
			if fieldsIndex == -1 {
				fieldsIndex = i
			}
		}
	}

	return messageIndex >= 0 && fieldsIndex == messageIndex+1
}

func alignMessageForFields(
	order []Part,
	reportTS bool,
	tsStr, levelStr, symbol, msg, fieldsStr string,
	layout *groupRenderLayout,
) string {
	if fieldsStr == "" || layout == nil || !layout.fields.enabled() {
		return msg
	}
	if !supportsFieldAlignment(order, layout.fields.alignment) {
		return msg
	}

	currentStart := lipgloss.Width(buildLine(order, reportTS, tsStr, levelStr, symbol, msg, ""))
	padding := layout.fields.maxStart - currentStart
	if padding <= 0 {
		return msg
	}

	return msg + strings.Repeat(" ", padding)
}

// styledMsg applies the message style for the given level, if any.
func styledMsg(msg string, level Level, styles *style.Config, noColor bool) string {
	if noColor {
		return msg
	}
	if s := styles.Messages[level]; s != nil {
		return s.Render(msg)
	}
	if styles.Message != nil {
		return styles.Message.Render(msg)
	}
	return msg
}

func styledSymbol(symbol string, level Level, styles *style.Config, noColor bool) string {
	if !noColor {
		if s := styles.Symbols[level]; s != nil {
			return s.Render(symbol)
		}
	}
	return symbol
}

// renderTaskFields formats the fields for a task, caching the result when
// the atomic pointer has not changed.
func renderTaskFields(
	gt *groupTask,
	fieldsPtr *[]core.Field,
	dur time.Duration,
	current, total int,
) string {
	b := gt.Builder
	if b.ElapsedKey != "" || b.BarPercentKey != "" {
		resolved := resolveDynamicFields(*fieldsPtr, b, dur, current, total)
		gt.cachedFieldsStr = strings.TrimLeft(formatFields(resolved, gt.fieldOpts), " ")
	} else if fieldsPtr != gt.cachedFieldsPtr {
		gt.cachedFieldsStr = strings.TrimLeft(formatFields(*fieldsPtr, gt.fieldOpts), " ")
	}
	gt.cachedFieldsPtr = fieldsPtr
	return gt.cachedFieldsStr
}

func resolveDynamicFields(
	fields []core.Field,
	b *fx.Builder,
	dur time.Duration,
	current, total int,
) []core.Field {
	out := make([]core.Field, len(fields))
	copy(out, fields)

	if total <= 0 {
		total = 1
	}
	current = max(current, 0)

	pctMax := percent.Maximum()
	pct := float64(current) / float64(total) * pctMax
	if pct > pctMax {
		pct = pctMax
	}

	for i := range out {
		switch out[i].Key {
		case b.ElapsedKey:
			out[i].Value = core.ElapsedField(dur)
		case b.BarPercentKey:
			out[i].Value = core.Percent{Value: pct}
		}
	}

	return out
}

// renderTaskTimestamp returns the styled timestamp string for a task.
func renderTaskTimestamp(gt *groupTask, now time.Time) string {
	if !gt.cfg.reportTS {
		return ""
	}
	ts := now.In(gt.cfg.timeLoc).Format(gt.cfg.timeFmt)
	if gt.cfg.styles.Timestamp != nil && !gt.cfg.noColor {
		return gt.cfg.styles.Timestamp.Render(ts)
	}
	return ts
}

func measureTaskFieldStart(
	gt *groupTask,
	isDone bool,
	now time.Time,
	alignment fx.FieldAlignment,
) int {
	if !supportsFieldAlignment(gt.cfg.order, alignment) {
		return 0
	}

	tsStr := renderTaskTimestamp(gt, now)

	if isDone {
		renderLevel, levelSymbol := gt.resolveLevel()
		msg := gt.cfg.indentation + styledMsg(
			*gt.MsgPtr.Load(),
			renderLevel,
			gt.cfg.styles,
			gt.cfg.noColor,
		)
		return lipgloss.Width(buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			levelSymbol,
			styledSymbol(*gt.SymbolPtr.Load(), renderLevel, gt.cfg.styles, gt.cfg.noColor),
			msg,
			"",
		))
	}

	msg, char := renderTaskMessageSymbol(gt, now)

	return lipgloss.Width(buildLine(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelSymbol,
		char,
		msg,
		"",
	))
}

// renderTaskLine renders a single animation frame line for a task.
// For done tasks, it renders the frozen final state with the level's default symbol.
// For active tasks, it renders the current animation frame.
// It does not perform any I/O.
func renderTaskLine(gt *groupTask, isDone bool, now time.Time, layout *groupRenderLayout) string {
	b := gt.Builder
	fieldsPtr := gt.FieldsPtr.Load()
	current, total := 0, 0
	if b.Mode == fx.AnimationBar {
		current = int(b.BarProgressPtr.Load())
		total = int(b.BarTotalPtr.Load())
	}
	dur := gt.Duration(now)
	fieldsStr := renderTaskFields(gt, fieldsPtr, dur, current, total)
	tsStr := renderTaskTimestamp(gt, now)

	if isDone {
		// Show the frozen final line with the level's default symbol.
		// If SetLevel was called, use the overridden level for styling.
		renderLevel, levelSymbol := gt.resolveLevel()
		msg := gt.cfg.indentation + styledMsg(
			*gt.MsgPtr.Load(),
			renderLevel,
			gt.cfg.styles,
			gt.cfg.noColor,
		)
		doneSymbol := styledSymbol(*gt.SymbolPtr.Load(), renderLevel, gt.cfg.styles, gt.cfg.noColor)
		msg = alignMessageForFields(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			levelSymbol,
			doneSymbol,
			msg,
			fieldsStr,
			layout,
		)
		return buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			levelSymbol,
			doneSymbol,
			msg,
			fieldsStr,
		)
	}

	// Bar mode has its own rendering path.
	if b.Mode == fx.AnimationBar {
		state := loadBarWidgetState(gt, now)
		current := int(gt.Builder.BarProgressPtr.Load())
		total := int(gt.Builder.BarTotalPtr.Load())
		if gt.monotonic {
			progress := float64(current) / float64(max(total, 1))
			clamped := max(gt.maxBarProgress, progress)
			current = int(math.Round(clamped * float64(max(total, 1))))
			total = max(total, 1)
		}
		fieldsStr = renderTaskFields(
			gt,
			gt.FieldsPtr.Load(),
			state.elapsed,
			current,
			total,
		)
		return renderTaskBarLine(gt, fieldsStr, tsStr, state, layout, now)
	}

	msg, char := renderTaskMessageSymbol(gt, now)
	msg = alignMessageForFields(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelSymbol,
		char,
		msg,
		fieldsStr,
		layout,
	)

	return buildLine(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelSymbol,
		char,
		msg,
		fieldsStr,
	)
}

func renderAnimatedTaskMessage(gt *groupTask, now time.Time) (string, string) {
	b := gt.Builder
	msg := *gt.MsgPtr.Load()
	dur := gt.animDuration(now)

	// Message animation.
	switch b.Mode { //nolint:exhaustive // animationBar handled by caller
	case fx.AnimationPulse:
		t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*b.Speed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
		msg = pulse.TextCached(msg, t, b.PulseStops, &gt.pCache)
	case fx.AnimationShimmer:
		phase := math.Mod(dur.Seconds()*b.Speed, 1.0)
		msg = shimmer.Text(msg, phase, b.ShimmerDir, gt.hexLUT, gt.styleLUT)
	default:
		msg = styledMsg(msg, b.Level, gt.cfg.styles, gt.cfg.noColor)
	}

	// Symbol: animated spinner frames or static icon.
	char := resolveSymbol(gt, now)

	return msg, char
}

// animDuration returns the duration used for animation phase calculations.
// When a sync epoch is set (via [Group.SyncAnimations]), all tasks in the
// group share the same epoch so their animations stay in lockstep.
// Otherwise it falls back to the per-task elapsed duration.
func (gt *groupTask) animDuration(now time.Time) time.Duration {
	if !gt.syncEpoch.IsZero() {
		return now.Sub(gt.syncEpoch)
	}
	return gt.Duration(now)
}

// resolveSymbol returns the styled symbol for the current animation frame.
// When [Builder.AnimatedSymbol] is true, it cycles through [SpinnerStyle]
// frames based on wall-clock time. When the symbol has been explicitly
// overridden via [Update.SetSymbol], the static symbol is returned instead
// so the caller can replace the spinner with a checkmark or other icon.
func resolveSymbol(gt *groupTask, now time.Time) string {
	b := gt.Builder
	if b.AnimatedSymbol && !gt.SymbolOverride.Load() &&
		gt.Started() && len(b.SpinnerStyle.Frames) > 0 &&
		b.SpinnerStyle.Interval > 0 {
		n := len(b.SpinnerStyle.Frames)
		i := int(gt.animDuration(now)/b.SpinnerStyle.Interval) % n
		if b.SpinnerStyle.Reverse {
			i = n - 1 - i
		}
		return styledSymbol(b.SpinnerStyle.Frames[i], b.Level, gt.cfg.styles, gt.cfg.noColor)
	}
	return styledSymbol(*gt.SymbolPtr.Load(), b.Level, gt.cfg.styles, gt.cfg.noColor)
}

func renderTaskMessageSymbol(gt *groupTask, now time.Time) (string, string) {
	if !gt.Started() {
		return gt.cfg.indentation + styledMsg(
			*gt.MsgPtr.Load(),
			gt.Builder.Level,
			gt.cfg.styles,
			gt.cfg.noColor,
		), styledSymbol(*gt.SymbolPtr.Load(), gt.Builder.Level, gt.cfg.styles, gt.cfg.noColor)
	}

	if gt.Builder.Mode == fx.AnimationBar {
		return gt.cfg.indentation + styledMsg(
			*gt.MsgPtr.Load(),
			gt.Builder.Level,
			gt.cfg.styles,
			gt.cfg.noColor,
		), resolveSymbol(gt, now)
	}

	msg, char := renderAnimatedTaskMessage(gt, now)
	return gt.cfg.indentation + msg, char
}

// renderTaskBarLine renders a bar-animation frame for a task. Factored out to
// keep renderTaskLine focused.
func renderTaskBarLine(
	gt *groupTask,
	fieldsStr, tsStr string,
	state barWidgetState,
	layout *groupRenderLayout,
	now time.Time,
) string {
	parts, leftText, barStr, rightText, sep, showBar := buildTaskBarParts(
		gt,
		fieldsStr,
		tsStr,
		state,
		layout,
		now,
	)
	if !showBar || gt.Builder.BarStyle.Placement == bar.PlaceInline {
		return parts
	}
	if layout != nil {
		return layout.bar.format(
			parts,
			leftText,
			barStr,
			rightText,
			sep,
			gt.Builder.BarStyle.Placement,
			gt.cfg.output.Width(),
		)
	}
	barFull := assembleBarColumns(groupBarColumns{}, leftText, barStr, rightText, sep)
	return bar.FormatLine(parts, barFull, sep, gt.Builder.BarStyle.Placement, gt.cfg.output.Width())
}

func buildTaskBarParts(
	gt *groupTask,
	fieldsStr, tsStr string,
	state barWidgetState,
	layout *groupRenderLayout,
	now time.Time,
) (string, string, string, string, string, bool) {
	b := gt.Builder
	symbol := resolveSymbol(gt, now)
	msg := gt.cfg.indentation + styledMsg(*gt.MsgPtr.Load(), b.Level, gt.cfg.styles, gt.cfg.noColor)
	msg = alignMessageForFields(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelSymbol,
		symbol,
		msg,
		fieldsStr,
		layout,
	)

	current := int(b.BarProgressPtr.Load())
	total := int(b.BarTotalPtr.Load())
	progress := float64(current) / float64(max(total, 1))
	renderProgress := progress
	if gt.monotonic {
		if total != gt.maxBarTotal {
			gt.maxBarProgress = progress
			gt.maxBarTotal = total
		} else {
			gt.maxBarProgress = max(gt.maxBarProgress, progress)
		}
		renderProgress = gt.maxBarProgress
		// Derive synthetic current/total so widgets and percent text
		// also reflect the clamped progress.
		current = int(math.Round(renderProgress * float64(max(total, 1))))
		total = max(total, 1)
	}
	if b.BarStyle.Smoothing == bar.SmoothEase {
		now := state.renderAt
		if !gt.smoothedInit {
			gt.smoothedProgress = renderProgress
			gt.smoothedTime = now
			gt.smoothedInit = true
		} else {
			tau := b.BarStyle.SmoothingTau
			if tau <= 0 {
				tau = bar.DefaultSmoothingTau
			}
			dt := now.Sub(gt.smoothedTime).Seconds()
			gt.smoothedTime = now
			alpha := 1.0 - math.Exp(-dt/tau.Seconds())
			gt.smoothedProgress += (renderProgress - gt.smoothedProgress) * alpha
		}
		renderProgress = gt.smoothedProgress
		// Re-derive current so widgets (percent, ETA, etc.) reflect the
		// smoothed position, not the raw target.
		current = int(math.Round(renderProgress * float64(max(total, 1))))
	}
	sep := b.BarStyle.Separator
	if sep == "" {
		sep = " "
	}

	parts := buildLine(
		gt.cfg.order,
		gt.cfg.reportTS,
		tsStr,
		gt.cfg.levelSymbol,
		symbol,
		msg,
		fieldsStr,
	)
	if !bar.ShowPending(b.BarStyle, current) {
		return parts, "", "", "", sep, false
	}

	// Cache the gradient style to avoid lipgloss.NewStyle() per frame.
	barStyle := b.BarStyle
	if len(barStyle.ProgressGradient) > 0 {
		if !gt.gradientValid || gt.gradientProgress != renderProgress {
			c := style.InterpolateGradient(renderProgress, barStyle.ProgressGradient)
			gt.gradientStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(c.Clamped().Hex()))
			gt.gradientProgress = renderProgress
			gt.gradientValid = true
		}
		barStyle.StyleFill = &gt.gradientStyle
		barStyle.ProgressGradient = nil // prevent renderBar from recomputing
	}
	var barStr string
	if gt.monotonic || b.BarStyle.Smoothing == bar.SmoothEase {
		barStr = renderBarProgress(renderProgress, barStyle, gt.cfg.output.Width())
	} else {
		barStr = bar.Render(current, total, barStyle, gt.cfg.output.Width())
	}

	widgetState := bar.State{
		Current: current,
		Total:   total,
		Elapsed: state.elapsed,
		Rate:    state.rate,
	}

	var leftText, rightText string
	if b.BarStyle.WidgetLeft != nil {
		leftText = b.BarStyle.WidgetLeft(widgetState)
	}
	if b.BarStyle.WidgetRight != nil && b.BarPercentKey == "" {
		rightText = b.BarStyle.WidgetRight(widgetState)
	} else if b.BarStyle.WidgetLeft == nil && b.BarStyle.WidgetRight == nil && b.BarPercentKey == "" {
		// Default: padded percent on the right when no widgets are configured
		// and no BarPercent field is set.
		rightText = bar.FormatPercent(current, total, 0, true)
	}

	// writeFrame equivalent: build the complete line string.
	if b.BarStyle.Placement == bar.PlaceInline {
		barFull := assembleBarColumns(groupBarColumns{
			hasLeft:  leftText != "",
			hasRight: rightText != "",
			maxBar:   lipgloss.Width(barStr),
			maxLeft:  lipgloss.Width(leftText),
			maxRight: lipgloss.Width(rightText),
		}, leftText, barStr, rightText, sep)
		return buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			gt.cfg.levelSymbol,
			symbol,
			msg+sep+barFull,
			fieldsStr,
		), "", "", "", sep, true
	}
	return parts, leftText, barStr, rightText, sep, true
}

func measureGroupRenderLayout(
	g *fx.Group,
	gts []*groupTask,
	done []bool,
	now time.Time,
) *groupRenderLayout {
	hideDone := g.HideDone
	layout := &groupRenderLayout{
		fields: groupFieldLayout{alignment: g.FieldAlignment},
	}
	if layout.fields.alignment != fx.FieldAlignmentNone {
		for i, gt := range gts {
			if (hideDone && done[i]) || !shouldRenderTask(gt, done[i], now) {
				continue
			}
			layout.fields.maxStart = max(
				layout.fields.maxStart,
				measureTaskFieldStart(gt, done[i], now, layout.fields.alignment),
			)
		}
	}

	for i, gt := range gts {
		if (hideDone && done[i]) || !shouldRenderTask(gt, done[i], now) || done[i] ||
			gt.Builder.Mode != fx.AnimationBar {
			continue
		}

		state := loadBarWidgetState(gt, now)
		current := int(gt.Builder.BarProgressPtr.Load())
		total := int(gt.Builder.BarTotalPtr.Load())
		fieldsStr := renderTaskFields(
			gt,
			gt.FieldsPtr.Load(),
			state.elapsed,
			current,
			total,
		)
		tsStr := renderTaskTimestamp(gt, now)
		parts, leftText, barStr, rightText, _, showBar := buildTaskBarParts(
			gt,
			fieldsStr,
			tsStr,
			state,
			layout,
			now,
		)
		if !showBar {
			continue
		}
		layout.bar.observe(parts, leftText, barStr, rightText, gt.Builder.BarStyle.Placement)
	}

	// For PlaceAligned, also measure done tasks so completed messages
	// (which may be longer) are included in the max parts width.
	// Skip when HideDone is set since done tasks are not rendered.
	for i, gt := range gts {
		if hideDone || !shouldRenderTask(gt, done[i], now) || !done[i] {
			continue
		}
		if gt.Builder.Mode != fx.AnimationBar || gt.Builder.BarStyle.Placement != bar.PlaceAligned {
			continue
		}
		tsStr := renderTaskTimestamp(gt, now)
		msg := gt.cfg.indentation + styledMsg(
			*gt.MsgPtr.Load(), gt.Builder.Level, gt.cfg.styles, gt.cfg.noColor,
		)
		fieldsStr := renderTaskFields(gt, gt.FieldsPtr.Load(), gt.Duration(now), 0, 0)
		parts := buildLine(
			gt.cfg.order,
			gt.cfg.reportTS,
			tsStr,
			gt.cfg.levelSymbol,
			styledSymbol(*gt.SymbolPtr.Load(), gt.Builder.Level, gt.cfg.styles, gt.cfg.noColor),
			msg,
			fieldsStr,
		)
		layout.bar.aligned.maxParts = max(layout.bar.aligned.maxParts, lipgloss.Width(parts))
	}

	return layout
}

// runGroupLoop runs the group render loop, blocking until all tasks complete
// or the context is cancelled. Called by fxLogger.RunGroup.
func runGroupLoop(ctx context.Context, g *fx.Group) error {
	g.Mu.Lock()
	fxTasks := g.Tasks
	g.Mu.Unlock()

	if len(fxTasks) == 0 {
		return nil
	}

	// Wrap each fx.GroupTask with rendering state.
	var syncEpoch time.Time
	if g.SyncAnimations {
		syncEpoch = time.Now()
	}
	gts := make([]*groupTask, len(fxTasks))
	for i, ft := range fxTasks {
		gt := &groupTask{
			GroupTask: ft,
			monotonic: g.Monotonic,
			syncEpoch: syncEpoch,
		}
		captureTaskConfig(gt)
		gts[i] = gt
	}

	// Build groupTasks for header/footer status lines.
	initStatus := func(s *fx.GroupStatus) (*groupTask, *fx.Update) {
		if s == nil {
			return nil, nil
		}
		b := s.Builder
		if b.Log == nil {
			b.Log = g.Log
		}
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]core.Field]{}
		symbolPtr := &atomic.Pointer[string]{}
		msgPtr.Store(&b.Message)
		fieldsPtr.Store(&b.Fields)
		sym := b.SymbolIcon
		if sym == "" {
			sym = fx.DefaultSymbol
		}
		symbolPtr.Store(&sym)

		gt := &groupTask{
			GroupTask: &fx.GroupTask{
				Builder:   b,
				FieldsPtr: fieldsPtr,
				MsgPtr:    msgPtr,
				SymbolPtr: symbolPtr,
			},
			syncEpoch: syncEpoch,
		}
		gt.StartedAt.Store(time.Now().UnixNano())
		captureTaskConfig(gt)

		u := &fx.Update{
			MsgText:   b.Message,
			MsgPtr:    msgPtr,
			FieldsPtr: fieldsPtr,
			Base:      b.Fields,
			SymbolPtr: symbolPtr,
		}
		u.InitSelf(u)
		return gt, u
	}

	headerGT, headerUpdate := initStatus(g.Header)
	footerGT, footerUpdate := initStatus(g.Footer)

	// Non-TTY: print each task's initial line, then block on all results.
	// Dynamic fields (elapsed, bar percent) are stripped because their
	// initial zero values are meaningless without live updates.
	if !gts[0].cfg.isTTY {
		for _, gt := range gts {
			b := gt.Builder
			fieldsStr := strings.TrimLeft(
				formatFields(b.StripDynamicFields(*gt.FieldsPtr.Load()), gt.fieldOpts), " ",
			)
			line := buildLine(
				gt.cfg.order,
				gt.cfg.reportTS,
				time.Now().In(gt.cfg.timeLoc).Format(gt.cfg.timeFmt),
				gt.cfg.label,
				styledSymbol(*gt.SymbolPtr.Load(), gt.Builder.Level, gt.cfg.styles, gt.cfg.noColor),
				gt.cfg.indentation+*gt.MsgPtr.Load(),
				fieldsStr,
			)
			writeString(gt.cfg.out, line+"\n")
		}
		for _, ft := range fxTasks {
			select {
			case ft.Err = <-ft.DoneErr:
			case <-ctx.Done():
				for _, ft2 := range fxTasks {
					if ft2.Err == nil {
						select {
						case ft2.Err = <-ft2.DoneErr:
						default:
							ft2.Err = ctx.Err()
						}
					}
				}
				return ctx.Err()
			}
		}
		return nil
	}

	// Tick rate = fastest task's rate.
	tickRate := gts[0].tickRate
	for _, gt := range gts[1:] {
		tickRate = min(tickRate, gt.tickRate)
	}

	out := gts[0].cfg.out
	output := gts[0].cfg.output
	stopResize := output.ListenResize()
	defer stopResize()

	writeString(out, hideCursor)
	defer writeString(out, showCursor)
	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	numLines := 0
	done := make([]bool, len(gts))
	remaining := len(gts)
	var frameBuf strings.Builder

	for remaining > 0 {
		select {
		case <-ctx.Done():
			clearBlock(out, numLines)
			for i, ft := range fxTasks {
				if !done[i] {
					ft.Err = ctx.Err()
				}
			}
			return ctx.Err()
		case <-ticker.C:
			now := time.Now()
			// Drain completed tasks.
			for i, ft := range fxTasks {
				if done[i] {
					continue
				}
				select {
				case err := <-ft.DoneErr:
					ft.Err = err
					done[i] = true
					remaining--
				default:
				}
			}
			maxLines := output.Height() - 1
			if maxLines <= 0 {
				maxLines = len(gts) // no height info; no cap
			}
			// Cap visible tasks to terminal height so cursor-up
			// escapes never need to reach scrolled-off lines.
			// Prioritise active (in-progress) tasks over done or
			// pending ones when space is limited.
			visible := visibleTaskIndexes(gts, done, g.HideDone, now)

			// Update header/footer via callbacks and render as log lines.
			doneCount := len(gts) - remaining
			totalCount := len(gts)
			statusLines := 0
			showHeader := false
			showFooter := false
			if headerGT != nil {
				g.Header.Callback(doneCount, totalCount, headerUpdate)
				if msg := *headerGT.MsgPtr.Load(); msg != "" {
					showHeader = true
					statusLines++
				}
			}
			if footerGT != nil {
				g.Footer.Callback(doneCount, totalCount, footerUpdate)
				if msg := *footerGT.MsgPtr.Load(); msg != "" {
					showFooter = true
					statusLines++
				}
			}

			if maxTasks := maxLines - statusLines; len(visible) > maxTasks {
				visible = prioritiseActive(visible, gts, done, maxTasks)
			}
			frameBuf.Reset()
			totalLines := len(visible) + statusLines
			if numLines > 1 {
				fmt.Fprintf(&frameBuf, cursorUpFmt, numLines-1)
			}
			layout := measureGroupRenderLayout(g, gts, done, now)
			renderLines := max(numLines, totalLines)
			lineIdx := 0

			writeLine := func(line string) {
				if lineIdx < renderLines-1 {
					fmt.Fprintf(&frameBuf, "%s%s\n", clearLine, line)
				} else {
					fmt.Fprintf(&frameBuf, "%s%s", clearLine, line)
				}
				lineIdx++
			}

			// Header.
			if showHeader {
				writeLine(renderTaskLine(headerGT, false, now, layout))
			}
			// Task lines.
			for _, taskIndex := range visible {
				writeLine(renderTaskLine(gts[taskIndex], done[taskIndex], now, layout))
			}
			// Footer.
			if showFooter {
				writeLine(renderTaskLine(footerGT, false, now, layout))
			}
			// Clear any leftover lines from the previous frame.
			for lineIdx < renderLines {
				writeLine("")
			}
			switch {
			case numLines > totalLines && totalLines > 0:
				fmt.Fprintf(&frameBuf, cursorUpFmt, numLines-totalLines)
			case numLines > 1 && totalLines == 0:
				fmt.Fprintf(&frameBuf, cursorUpFmt, numLines-1)
			}
			writeString(out, frameBuf.String())
			numLines = totalLines
			// If all done, break out after one final render.
			if remaining == 0 {
				break
			}
		}
	}

	clearBlock(out, numLines)
	return nil
}

// clearBlock erases n lines starting from the current cursor line and
// repositions the cursor back to the first cleared line. The cursor is
// expected to be on the last line of the block (no trailing newline).
func clearBlock(out io.Writer, n int) {
	if n == 0 {
		return
	}
	var buf strings.Builder
	if n > 1 {
		fmt.Fprintf(&buf, cursorUpFmt, n-1)
	}
	for range n {
		buf.WriteString(clearLine + "\n")
	}
	fmt.Fprintf(&buf, cursorUpFmt, n)
	writeString(out, buf.String())
}
