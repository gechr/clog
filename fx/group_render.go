package fx

import (
	"context"
	"math"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/pulse"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
	"github.com/gechr/clog/style"
	xansi "github.com/gechr/x/ansi"
	xmath "github.com/gechr/x/math"
	xstrings "github.com/gechr/x/strings"
)

// renderTask holds per-animation mutable state for both the single-animation
// (runAnimation) and multi-animation (Group) paths. It embeds *groupTask
// for shared state and adds rendering-specific fields.
type renderTask struct {
	*groupTask

	cfg            TaskConfig
	maxBarProgress float64
	maxBarTotal    int
	monotonic      bool
	syncEpoch      time.Time
	tickRate       time.Duration
	visible        bool

	// per-tick mutable state
	cachedFieldsPtr *[]core.Field // dedup: last-formatted fields pointer
	cachedFieldsStr string        // dedup: last-formatted fields string
	hexLUT          *shimmer.LUT  // shimmer only, immutable after init
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

	// per-frame atomic snapshot. Populated by snapshotFrameValues at the
	// start of each render tick so that measureGroupRenderLayout and the
	// subsequent renderTaskLine calls observe the same current/total/fields
	// values. Without this, an in-flight progress update between layout
	// measurement and task rendering produces lines wider than the layout
	// allowed for, which wraps visibly at the right edge.
	frameBarCurrent    int64
	frameBarTotal      int64
	frameFieldsPtr     *[]core.Field
	frameSnapshotValid bool
}

// barProgress returns the per-frame snapshot of (current, total). Falls back
// to a direct atomic load when no snapshot is in effect (e.g. utility paths
// outside the render loop).
func (gt *renderTask) barProgress() (int, int) {
	if gt.frameSnapshotValid {
		return int(gt.frameBarCurrent), int(gt.frameBarTotal)
	}
	return int(gt.builder.cfg.BarProgress.Load()), int(gt.builder.cfg.BarTotal.Load())
}

// fieldsSnapshot returns the per-frame snapshot of the fields pointer. Falls
// back to a direct atomic load when no snapshot is in effect.
func (gt *renderTask) fieldsSnapshot() *[]core.Field {
	if gt.frameSnapshotValid {
		return gt.frameFieldsPtr
	}
	return gt.fieldsPtr.Load()
}

// snapshotFrameValues captures BarProgressPtr / BarTotalPtr / fieldsPtr for
// every task in the group at the start of a render tick. Call once per tick
// before measureGroupRenderLayout so that all width calculations and
// per-task rendering observe identical atomic values.
func snapshotFrameValues(gts []*renderTask) {
	for _, gt := range gts {
		if gt.builder.cfg.BarProgress != nil {
			gt.frameBarCurrent = gt.builder.cfg.BarProgress.Load()
		}
		if gt.builder.cfg.BarTotal != nil {
			gt.frameBarTotal = gt.builder.cfg.BarTotal.Load()
		}
		gt.frameFieldsPtr = gt.fieldsPtr.Load()
		gt.frameSnapshotValid = true
	}
}

// effectiveLevel returns the level set via [Update.SetLevel] if present,
// otherwise the builder's original level.
func (gt *renderTask) effectiveLevel() core.Level {
	if gt.levelPtr != nil {
		if override := core.Level(gt.levelPtr.Load()); override != level.Unset {
			return override
		}
	}
	return gt.builder.cfg.Level
}

// resolveLevel returns the effective level and styled level symbol for a
// task's rendered line, live or completed. If SetLevel was called on the
// Update, the overridden level is used; otherwise the builder's original
// level applies.
func (gt *renderTask) resolveLevel() (core.Level, string) {
	lvl := gt.effectiveLevel()
	if lvl == gt.builder.cfg.Level {
		return lvl, gt.cfg.LevelSymbol
	}
	return lvl, gt.cfg.StyleLevel(lvl)
}

// doneLineParts returns the styled level symbol, done symbol, and indented
// message for a completed task's frozen line, honoring any level override
// set via [Update.SetLevel].
func (gt *renderTask) doneLineParts() (string, string, string) {
	renderLevel, levelSymbol := gt.resolveLevel()
	doneSymbol := gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), renderLevel)
	msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), renderLevel)
	return levelSymbol, doneSymbol, msg
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

type groupBarLayout struct {
	// aligned tracks the maximum message-parts width for PlaceAligned bars
	// so that all bars in a group start at the same column.
	aligned  groupBarColumns
	leftPad  groupBarColumns
	rightPad groupBarColumns
}

type groupFieldLayout struct {
	alignment FieldAlignment
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

func shouldRenderTask(gt *renderTask, isDone bool, now time.Time) bool {
	if gt.visible {
		return true
	}

	delay := gt.builder.delayDur
	if delay <= 0 {
		gt.visible = true
		return true
	}
	if !gt.started() {
		return false
	}
	if isDone {
		if finishedAt := gt.finishTime(); !finishedAt.IsZero() {
			if gt.duration(finishedAt) < delay {
				return false
			}
			gt.visible = true
			return true
		}
	}
	if gt.duration(now) < delay {
		return false
	}

	gt.visible = true
	return true
}

func visibleTaskIndexes(gts []*renderTask, done []bool, hideDone bool, now time.Time) []int {
	indexes := make([]int, 0, len(gts))
	for i, gt := range gts {
		if gt.cfg.Silent {
			continue
		}
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
func prioritiseActive(visible []int, gts []*renderTask, done []bool, maxLines int) []int {
	if maxLines <= 0 {
		return nil
	}

	active := make([]int, 0, len(visible))
	other := make([]int, 0, len(visible))
	for _, idx := range visible {
		if gts[idx].started() && !done[idx] {
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

func renderBarProgress(progress float64, s bar.Config, termWidth int) string {
	progress = xmath.Clamp01(progress)
	current := int(math.Round(progress * monotonicBarScale))
	return bar.Render(current, monotonicBarScale, s, termWidth)
}

func (l *groupFieldLayout) enabled() bool {
	return l != nil && l.alignment == FieldAlignmentMessage && l.maxStart > 0
}

// columnsFor returns the column tracker for placement, or nil for placements
// that don't participate in group-wide alignment.
func (l *groupBarLayout) columnsFor(placement bar.Placement) *groupBarColumns {
	switch placement {
	case bar.PlaceLeftPad:
		return &l.leftPad
	case bar.PlaceRightPad:
		return &l.rightPad
	case bar.PlaceAligned:
		return &l.aligned
	case bar.PlaceInline, bar.PlaceLeft, bar.PlaceRight:
		return nil
	}
	return nil
}

func (l *groupBarLayout) observe(
	parts, leftText, barStr, rightText string,
	placement bar.Placement,
) {
	c := l.columnsFor(placement)
	if c == nil {
		return
	}
	c.hasLeft = c.hasLeft || leftText != ""
	c.hasRight = c.hasRight || rightText != ""
	c.maxLeft = max(c.maxLeft, lipgloss.Width(leftText))
	c.maxParts = max(c.maxParts, lipgloss.Width(parts))
	c.maxBar = max(c.maxBar, lipgloss.Width(barStr))
	c.maxRight = max(c.maxRight, lipgloss.Width(rightText))
}

func (l *groupBarLayout) formatWithTruncationMarker(
	parts, leftText, barStr, rightText, sep string,
	placement bar.Placement,
	termWidth int,
	truncationMarker string,
) string {
	switch placement {
	case bar.PlaceLeftPad, bar.PlaceRightPad:
		return l.formatPadded(parts, leftText, barStr, rightText, sep, placement, termWidth)
	case bar.PlaceAligned:
		return l.formatAligned(parts, leftText, barStr, rightText, sep, termWidth, truncationMarker)
	case bar.PlaceInline, bar.PlaceLeft, bar.PlaceRight:
		barFull := assembleBarColumns(groupBarColumns{}, leftText, barStr, rightText, sep)
		return bar.FormatLine(parts, barFull, sep, placement, termWidth)
	}

	return ""
}

// rightEdgeSlack is the column count reserved at the right edge of every
// padded bar layout. Lines that would otherwise sit at exactly termWidth
// trigger the terminal's auto-margin boundary, which can cause visible wraps
// when a per-task width snapshot is one column wider than the layout's
// observed max (e.g. "99%" -> "100%" between layout measurement and task
// rendering). Reserving a column makes the math forgiving of that drift.
const (
	rightEdgeSlack              = 1
	minUsefulBarWidth           = 1
	minUsefulTruncatedTextWidth = 5
)

// formatPadded renders a left- or right-padded bar layout: the bar columns
// and the message parts separated by a pad computed from the group-wide
// maxima, with the bar side chosen by placement.
func (l *groupBarLayout) formatPadded(
	parts, leftText, barStr, rightText, sep string,
	placement bar.Placement,
	termWidth int,
) string {
	shared := l.leftPad
	if placement == bar.PlaceRightPad {
		shared = l.rightPad
	}
	barFull := assembleBarColumns(shared, leftText, barStr, rightText, sep)
	if shared.maxParts == 0 || shared.maxBar == 0 {
		return bar.FormatLine(parts, barFull, sep, placement, termWidth)
	}

	gap := termWidth - rightEdgeSlack - shared.maxParts - shared.barWidth(sep)
	if gap < 0 {
		return bar.FormatLine(parts, barFull, sep, placement, termWidth)
	}

	// The message is not part of the per-tick frame snapshot, so parts can
	// drift wider than the measured max between layout and render; clamp so
	// the pad count never goes negative (strings.Repeat panics on negative).
	pad := strings.Repeat(" ", max(0, gap+shared.maxParts-lipgloss.Width(parts)))
	if placement == bar.PlaceRightPad {
		return parts + pad + barFull
	}
	return barFull + pad + parts
}

func (l *groupBarLayout) formatAligned(
	parts, leftText, barStr, rightText, sep string,
	termWidth int,
	truncationMarker string,
) string {
	a := l.aligned
	barFull := assembleBarColumns(a, leftText, barStr, rightText, sep)
	effectiveMax := a.maxParts
	if termWidth > 0 && termWidth <= rightEdgeSlack {
		return ""
	}
	if termWidth > 0 {
		var ok bool
		parts, effectiveMax, ok = capAlignedParts(
			parts,
			barFull,
			sep,
			effectiveMax,
			termWidth,
			truncationMarker,
		)
		if !ok {
			return truncateProgressColumns(
				a,
				leftText,
				barStr,
				rightText,
				sep,
				termWidth-rightEdgeSlack,
			)
		}
	}
	if parts == "" && effectiveMax == 0 {
		return barFull
	}
	if a.maxParts == 0 {
		if parts == "" {
			return barFull
		}
		return parts + sep + barFull
	}
	padding := effectiveMax - lipgloss.Width(parts)
	if padding <= 0 {
		if parts == "" {
			return barFull
		}
		return parts + sep + barFull
	}
	return parts + strings.Repeat(" ", padding) + sep + barFull
}

func capAlignedParts(
	parts, barFull, sep string,
	maxParts, termWidth int,
	truncationMarker string,
) (string, int, bool) {
	target := termWidth - rightEdgeSlack
	budget := target - lipgloss.Width(sep) - lipgloss.Width(barFull)
	if budget < 0 {
		return parts, maxParts, false
	}
	maxParts = min(maxParts, budget)
	if lipgloss.Width(parts) > budget {
		if budget < minUsefulTruncatedTextWidth {
			return "", 0, true
		}
		parts = xansi.Truncate(parts, budget, truncationMarker)
	}
	return parts, maxParts, true
}

func truncateProgressColumns(
	cols groupBarColumns,
	leftText, barStr, rightText, sep string,
	width int,
) string {
	if width < minUsefulBarWidth {
		return ""
	}

	barFull := assembleBarColumns(cols, leftText, barStr, rightText, sep)
	if lipgloss.Width(barFull) <= width {
		return barFull
	}

	if rightText != "" {
		next := cols
		next.hasRight = false
		next.maxRight = 0
		barFull = assembleBarColumns(next, leftText, barStr, "", sep)
		if lipgloss.Width(barFull) <= width {
			return barFull
		}
		cols = next
		rightText = ""
	}

	if leftText != "" {
		next := cols
		next.hasLeft = false
		next.maxLeft = 0
		barFull = assembleBarColumns(next, "", barStr, rightText, sep)
		if lipgloss.Width(barFull) <= width {
			return barFull
		}
		cols = next
		leftText = ""
	}

	barFull = assembleBarColumns(cols, leftText, barStr, rightText, sep)
	return xansi.Truncate(barFull, width, "")
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

func loadBarWidgetState(gt *renderTask, now time.Time) barWidgetState {
	current := int(gt.builder.cfg.BarProgress.Load())
	elapsed := gt.duration(now)
	rate := 0.0
	if secs := elapsed.Seconds(); secs > 0 && current > 0 {
		rate = float64(current) / secs
	}

	state := barWidgetState{
		elapsed:  elapsed,
		rate:     rate,
		renderAt: now,
	}

	updateInterval := gt.builder.cfg.BarConfig.UpdateInterval
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

func resetBarWidgetState(gt *renderTask) {
	gt.barWidgetValid = false
}

// captureTaskConfig snapshots the builder's logger settings via
// [Logger.TaskConfig] and pre-computes gt.tickRate and mode-specific
// resources (shimmer LUTs, spinner frame guards).
// newSyntheticTask builds a renderTask for a line rendered outside the
// group's task list (status lines, the overflow indicator), seeding its
// pointers from b and capturing its render config.
func newSyntheticTask(b *Builder, syncEpoch time.Time) *renderTask {
	msgPtr, fieldsPtr, symbolPtr := newTaskPointers(b)
	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			symbolPtr: symbolPtr,
		},
		syncEpoch: syncEpoch,
	}
	captureTaskConfig(gt)
	return gt
}

func captureTaskConfig(gt *renderTask) {
	b := gt.builder
	gt.cfg = b.cfg.Logger.TaskConfig(b)

	// Determine tick rate and pre-compute mode-specific resources.
	switch b.cfg.Mode {
	case AnimationNone:
		if b.cfg.AnimatedSymbol {
			gt.tickRate = b.cfg.SpinnerConfig.Interval
		}
	case AnimationPulse:
		gt.tickRate = pulse.TickRate
	case AnimationShimmer:
		gt.tickRate = shimmer.TickRate
		gt.hexLUT = shimmer.BuildLUT(b.cfg.ShimmerStops)
		gt.styleLUT = shimmer.BuildStyleLUT(gt.hexLUT)
	case AnimationBar:
		gt.tickRate = bar.TickRate
	}
	// When animated symbol is enabled on a non-spinner mode, ensure the
	// tick rate is fast enough for smooth spinner frame changes.
	if b.cfg.AnimatedSymbol && b.cfg.SpinnerConfig.Interval > 0 && gt.tickRate > 0 {
		gt.tickRate = min(gt.tickRate, b.cfg.SpinnerConfig.Interval)
	}

	// Guard against missing spinner frames when animated symbol is enabled.
	if b.cfg.AnimatedSymbol && len(b.cfg.SpinnerConfig.Frames) == 0 {
		b.cfg.SpinnerConfig.Frames = spinner.DefaultConfig().Frames
	}
	if b.cfg.AnimatedSymbol && b.cfg.SpinnerConfig.Boomerang {
		b.cfg.SpinnerConfig.Frames = spinner.BoomerangFrames(b.cfg.SpinnerConfig.Frames)
	}
	if gt.tickRate <= 0 {
		gt.tickRate = spinner.DefaultConfig().Interval
	}
	if interval := gt.cfg.AnimationInterval; interval > 0 && gt.tickRate < interval {
		gt.tickRate = interval
	}
}

// buildLine assembles a log line from the configured parts order.
func buildLine(
	order []core.Part,
	reportTS bool,
	tsStr, levelStr, symbol, msg, fieldsStr string,
) string {
	parts := make([]string, 0, len(order))
	for _, p := range order {
		var part string
		switch p {
		case core.PartTimestamp:
			if !reportTS {
				continue
			}
			part = tsStr
		case core.PartLevel:
			part = levelStr
		case core.PartSymbol:
			part = symbol
		case core.PartMessage:
			part = msg
		case core.PartFields:
			part = fieldsStr
		}
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " ")
}

// line assembles a log line from the task's configured parts order.
func (gt *renderTask) line(tsStr, levelStr, symbol, msg, fieldsStr string) string {
	return buildLine(gt.cfg.Order, gt.cfg.ReportTimestamp, tsStr, levelStr, symbol, msg, fieldsStr)
}

func supportsFieldAlignment(order []core.Part, alignment FieldAlignment) bool {
	if alignment != FieldAlignmentMessage {
		return false
	}

	messageIndex := -1
	fieldsIndex := -1
	for i, part := range order {
		switch part { //nolint:exhaustive // only the message/fields positions matter
		case core.PartMessage:
			if messageIndex == -1 {
				messageIndex = i
			}
		case core.PartFields:
			if fieldsIndex == -1 {
				fieldsIndex = i
			}
		}
	}

	return messageIndex >= 0 && fieldsIndex == messageIndex+1
}

func alignMessageForFields(
	order []core.Part,
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

// renderTaskFields formats the fields for a task, caching the result when
// the atomic pointer has not changed.
func renderTaskFields(
	gt *renderTask,
	fieldsPtr *[]core.Field,
	dur time.Duration,
	current, total int,
) string {
	b := gt.builder
	if xstrings.AnyNonEmpty(b.deadlineKey, b.elapsedKey, b.barPercentKey) ||
		hasDynamicField(*fieldsPtr) {
		resolved := resolveDynamicFields(*fieldsPtr, b, dur, current, total)
		gt.cachedFieldsStr = strings.TrimLeft(gt.cfg.FormatFields(resolved), " ")
	} else if fieldsPtr != gt.cachedFieldsPtr {
		gt.cachedFieldsStr = strings.TrimLeft(gt.cfg.FormatFields(*fieldsPtr), " ")
	}
	gt.cachedFieldsPtr = fieldsPtr
	return gt.cachedFieldsStr
}

func resolveDynamicFields(
	fields []core.Field,
	b *Builder,
	dur time.Duration,
	current, total int,
) []core.Field {
	if total <= 0 {
		total = 1
	}
	current = max(current, 0)

	pctMax := b.percentMaximum()
	pct := min(float64(current)/float64(total)*pctMax, pctMax)
	return b.resolveFieldsWith(fields, dur, func() core.Percent {
		return core.Percent{Value: pct}
	})
}

// renderTaskTimestamp returns the styled timestamp string for a task.
func renderTaskTimestamp(gt *renderTask, now time.Time) string {
	if !gt.cfg.ReportTimestamp {
		return ""
	}
	return gt.cfg.StyleTimestamp(now.In(gt.cfg.TimeLocation).Format(gt.cfg.TimeFormat))
}

func measureTaskFieldStart(
	gt *renderTask,
	isDone bool,
	now time.Time,
	alignment FieldAlignment,
) int {
	if !supportsFieldAlignment(gt.cfg.Order, alignment) {
		return 0
	}

	tsStr := renderTaskTimestamp(gt, now)

	if isDone {
		levelSymbol, doneSymbol, msg := gt.doneLineParts()
		return lipgloss.Width(gt.line(tsStr, levelSymbol, doneSymbol, msg, ""))
	}

	msg, char := renderTaskMessageSymbol(gt, now)

	return lipgloss.Width(gt.line(tsStr, gt.cfg.LevelSymbol, char, msg, ""))
}

// renderTaskLine renders a single animation frame line for a task.
// For done tasks, it renders the frozen final state with the level's default symbol.
// For active tasks, it renders the current animation frame.
// It does not perform any I/O.
func renderTaskLine(gt *renderTask, isDone bool, now time.Time, layout *groupRenderLayout) string {
	b := gt.builder
	fieldsPtr := gt.fieldsSnapshot()
	current, total := 0, 0
	if b.cfg.Mode == AnimationBar {
		current, total = gt.barProgress()
	}
	dur := gt.duration(now)
	fieldsStr := renderTaskFields(gt, fieldsPtr, dur, current, total)
	tsStr := renderTaskTimestamp(gt, now)

	if isDone {
		// Show the frozen final line with the level's default symbol.
		levelSymbol, doneSymbol, msg := gt.doneLineParts()
		msg = alignMessageForFields(
			gt.cfg.Order,
			gt.cfg.ReportTimestamp,
			tsStr,
			levelSymbol,
			doneSymbol,
			msg,
			fieldsStr,
			layout,
		)
		return gt.line(tsStr, levelSymbol, doneSymbol, msg, fieldsStr)
	}

	// Bar mode has its own rendering path.
	if b.cfg.Mode == AnimationBar {
		state := loadBarWidgetState(gt, now)
		current, total := gt.barProgress()
		if gt.monotonic {
			progress := float64(current) / float64(max(total, 1))
			clamped := max(gt.maxBarProgress, progress)
			current = int(math.Round(clamped * float64(max(total, 1))))
			total = max(total, 1)
		}
		fieldsStr = renderTaskFields(
			gt,
			gt.fieldsSnapshot(),
			state.elapsed,
			current,
			total,
		)
		return renderTaskBarLine(gt, fieldsStr, tsStr, state, layout, now)
	}

	msg, char := renderTaskMessageSymbol(gt, now)
	_, levelSymbol := gt.resolveLevel()
	msg = alignMessageForFields(
		gt.cfg.Order,
		gt.cfg.ReportTimestamp,
		tsStr,
		levelSymbol,
		char,
		msg,
		fieldsStr,
		layout,
	)

	return gt.line(tsStr, levelSymbol, char, msg, fieldsStr)
}

func renderAnimatedTaskMessage(gt *renderTask, now time.Time) (string, string) {
	b := gt.builder
	msg := *gt.msgPtr.Load()
	dur := gt.animDuration(now)

	// Message animation.
	switch b.cfg.Mode { //nolint:exhaustive // animationBar handled by caller
	case AnimationPulse:
		t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*b.cfg.Speed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
		msg = pulse.TextCached(msg, t, b.cfg.PulseStops, &gt.pCache)
	case AnimationShimmer:
		phase := math.Mod(dur.Seconds()*b.cfg.Speed, 1.0)
		msg = shimmer.Text(msg, phase, b.cfg.ShimmerDir, gt.hexLUT, gt.styleLUT)
	default:
		msg = gt.cfg.StyleMessage(msg, gt.effectiveLevel())
	}

	// Symbol: animated spinner frames or static icon.
	char := resolveSymbol(gt, now)

	return msg, char
}

// animDuration returns the duration used for animation phase calculations.
// When a sync epoch is set (the group default; see [WithoutSyncAnimations]),
// all tasks in the group share the same epoch so their animations stay in
// lockstep. Otherwise it falls back to the per-task elapsed duration.
func (gt *renderTask) animDuration(now time.Time) time.Duration {
	if !gt.syncEpoch.IsZero() {
		return now.Sub(gt.syncEpoch)
	}
	return gt.duration(now)
}

// resolveSymbol returns the styled symbol for the current animation frame.
// When [Builder.UsesAnimatedSymbol] is true, it cycles through spinner
// frames based on wall-clock time. When the symbol has been explicitly
// overridden via [Update.SetSymbol], the static symbol is returned instead
// so the caller can replace the spinner with a checkmark or other icon.
// If [Update.SetLevel] was also called, the overridden level is used for
// styling so the symbol color matches the intended level.
func resolveSymbol(gt *renderTask, now time.Time) string {
	b := gt.builder
	if b.cfg.AnimatedSymbol && !gt.symbolOverride.Load() &&
		gt.started() && len(b.cfg.SpinnerConfig.Frames) > 0 &&
		b.cfg.SpinnerConfig.Interval > 0 {
		n := len(b.cfg.SpinnerConfig.Frames)
		i := int(gt.animDuration(now)/b.cfg.SpinnerConfig.Interval) % n
		if b.cfg.SpinnerConfig.Reverse {
			i = n - 1 - i
		}
		return gt.cfg.StyleSymbol(b.cfg.SpinnerConfig.Frames[i], gt.effectiveLevel())
	}
	return gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.effectiveLevel())
}

func renderTaskMessageSymbol(gt *renderTask, now time.Time) (string, string) {
	if !gt.started() {
		return gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.effectiveLevel()),
			gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.effectiveLevel())
	}

	if gt.builder.cfg.Mode == AnimationBar {
		return gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.effectiveLevel()),
			resolveSymbol(gt, now)
	}

	msg, char := renderAnimatedTaskMessage(gt, now)
	return gt.cfg.Indentation + msg, char
}

// renderTaskBarLine renders a bar-animation frame for a task. Factored out to
// keep renderTaskLine focused.
func renderTaskBarLine(
	gt *renderTask,
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
	if !showBar || gt.builder.cfg.BarConfig.Placement == bar.PlaceInline {
		return parts
	}
	if layout != nil {
		return layout.bar.formatWithTruncationMarker(
			parts,
			leftText,
			barStr,
			rightText,
			sep,
			gt.builder.cfg.BarConfig.Placement,
			gt.cfg.Output.Width(),
			bar.ResolveTruncationMarker(gt.builder.cfg.BarConfig),
		)
	}
	barFull := assembleBarColumns(groupBarColumns{}, leftText, barStr, rightText, sep)
	return bar.FormatLine(
		parts,
		barFull,
		sep,
		gt.builder.cfg.BarConfig.Placement,
		gt.cfg.Output.Width(),
	)
}

func buildTaskBarParts(
	gt *renderTask,
	fieldsStr, tsStr string,
	state barWidgetState,
	layout *groupRenderLayout,
	now time.Time,
) (string, string, string, string, string, bool) {
	b := gt.builder
	symbol := resolveSymbol(gt, now)
	_, levelSymbol := gt.resolveLevel()
	msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.effectiveLevel())
	msg = alignMessageForFields(
		gt.cfg.Order,
		gt.cfg.ReportTimestamp,
		tsStr,
		levelSymbol,
		symbol,
		msg,
		fieldsStr,
		layout,
	)

	current, total := gt.barProgress()
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
	if b.cfg.BarConfig.Smoothing == bar.SmoothEase {
		now := state.renderAt
		if !gt.smoothedInit {
			gt.smoothedProgress = renderProgress
			gt.smoothedTime = now
			gt.smoothedInit = true
		} else {
			tau := b.cfg.BarConfig.SmoothingTau
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
	sep := b.cfg.BarConfig.Separator
	if sep == "" {
		sep = " "
	}

	parts := gt.line(tsStr, levelSymbol, symbol, msg, fieldsStr)
	if !bar.ShowPending(b.cfg.BarConfig, current) {
		return parts, "", "", "", sep, false
	}

	// Cache the gradient style to avoid lipgloss.NewStyle() per frame.
	barStyle := b.cfg.BarConfig
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
	if gt.monotonic || b.cfg.BarConfig.Smoothing == bar.SmoothEase {
		barStr = renderBarProgress(renderProgress, barStyle, gt.cfg.Output.Width())
	} else {
		barStr = bar.Render(current, total, barStyle, gt.cfg.Output.Width())
	}

	widgetState := bar.State{
		Current: current,
		Total:   total,
		Elapsed: state.elapsed,
		Rate:    state.rate,
	}

	var leftText, rightText string
	if b.cfg.BarConfig.WidgetLeft != nil {
		leftText = b.cfg.BarConfig.WidgetLeft(widgetState)
	}
	if b.cfg.BarConfig.WidgetRight != nil && b.barPercentKey == "" {
		rightText = b.cfg.BarConfig.WidgetRight(widgetState)
	} else if b.cfg.BarConfig.WidgetLeft == nil && b.cfg.BarConfig.WidgetRight == nil && b.barPercentKey == "" {
		// Default: padded percent on the right when no widgets are configured
		// and no BarPercent field is set.
		rightText = bar.FormatPercent(current, total, 0, true)
	}

	// writeFrame equivalent: build the complete line string.
	if b.cfg.BarConfig.Placement == bar.PlaceInline {
		barFull := assembleBarColumns(groupBarColumns{
			hasLeft:  leftText != "",
			hasRight: rightText != "",
			maxBar:   lipgloss.Width(barStr),
			maxLeft:  lipgloss.Width(leftText),
			maxRight: lipgloss.Width(rightText),
		}, leftText, barStr, rightText, sep)
		return gt.line(tsStr, levelSymbol, symbol, msg+sep+barFull, fieldsStr),
			"", "", "", sep, true
	}
	return parts, leftText, barStr, rightText, sep, true
}

func measureGroupRenderLayout(
	g *Group,
	gts []*renderTask,
	done []bool,
	now time.Time,
) *groupRenderLayout {
	return measureGroupRenderLayoutForIndexes(g, gts, done, allTaskIndexes(len(gts)), now)
}

func measureGroupRenderLayoutForIndexes(
	g *Group,
	gts []*renderTask,
	done []bool,
	indexes []int,
	now time.Time,
) *groupRenderLayout {
	hideDone := g.hideDone
	layout := &groupRenderLayout{
		fields: groupFieldLayout{alignment: g.fieldAlignment},
	}
	if layout.fields.alignment != FieldAlignmentNone {
		for _, i := range indexes {
			gt := gts[i]
			if (hideDone && done[i]) || !shouldRenderTask(gt, done[i], now) {
				continue
			}
			layout.fields.maxStart = max(
				layout.fields.maxStart,
				measureTaskFieldStart(gt, done[i], now, layout.fields.alignment),
			)
		}
	}

	for _, i := range indexes {
		gt := gts[i]
		if (hideDone && done[i]) || !shouldRenderTask(gt, done[i], now) || done[i] ||
			gt.builder.cfg.Mode != AnimationBar {
			continue
		}

		state := loadBarWidgetState(gt, now)
		current, total := gt.barProgress()
		fieldsStr := renderTaskFields(
			gt,
			gt.fieldsSnapshot(),
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
		layout.bar.observe(parts, leftText, barStr, rightText, gt.builder.cfg.BarConfig.Placement)
	}

	// For PlaceAligned, also measure done tasks so completed messages
	// (which may be longer) are included in the max parts width.
	// Skip when HideDone is set since done tasks are not rendered.
	for _, i := range indexes {
		gt := gts[i]
		if hideDone || !shouldRenderTask(gt, done[i], now) || !done[i] {
			continue
		}
		if gt.builder.cfg.Mode != AnimationBar ||
			gt.builder.cfg.BarConfig.Placement != bar.PlaceAligned {
			continue
		}
		tsStr := renderTaskTimestamp(gt, now)
		levelSymbol, doneSymbol, msg := gt.doneLineParts()
		fieldsStr := renderTaskFields(gt, gt.fieldsSnapshot(), gt.duration(now), 0, 0)
		parts := gt.line(tsStr, levelSymbol, doneSymbol, msg, fieldsStr)
		layout.bar.aligned.maxParts = max(layout.bar.aligned.maxParts, lipgloss.Width(parts))
	}

	return layout
}

func allTaskIndexes(n int) []int {
	indexes := make([]int, n)
	for i := range indexes {
		indexes[i] = i
	}
	return indexes
}

// groupHeightCap returns the maximum number of physical terminal rows the
// rendered block may occupy. The caller is responsible for measuring rendered
// lines via frameRows / groupFrameRows and trimming content that would push
// the physical row count past this cap.
func groupHeightCap(termHeight, blockTopRow, fallback int) int {
	if termHeight <= 0 {
		if fallback > 0 {
			return fallback
		}
		return 1
	}

	if blockTopRow >= 1 && blockTopRow <= termHeight {
		// Leave one spare row below the block when possible. An exact-fit
		// render can still push the prompt just off-screen in some terminals.
		maxLines := termHeight - blockTopRow
		if maxLines > 0 {
			return maxLines
		}
		return 1
	}

	maxLines := termHeight - 1
	if maxLines > 0 {
		return maxLines
	}
	return 1
}

func groupFrameRows(lines []string, termWidth int) int {
	return core.BlockRows(lines, termWidth)
}

func groupFrameFitsViewport(output RenderOutput, renderedRows, frameRows int) bool {
	if frameRows == 0 {
		return true
	}
	termHeight := output.Height()
	if termHeight <= 0 {
		return true
	}
	row, ok := output.CursorPosition()
	if !ok || row <= 0 {
		return true
	}

	topRow := row
	if renderedRows > 0 {
		topRow -= renderedRows
		if topRow < 1 {
			topRow = 1
		}
	}

	// The renderer parks the cursor one row below the block after each frame.
	// Treat that parking row as required space so a one-line frame on the
	// bottom terminal row is suppressed instead of scrolling the viewport.
	return topRow+frameRows <= termHeight
}

func drainGroupCompletions(
	fxTasks []*groupTask,
	gts []*renderTask,
	done []bool,
	justCompleted []bool,
	remaining int,
) int {
	for i := range justCompleted {
		justCompleted[i] = false
	}
	for i, ft := range fxTasks {
		if done[i] {
			continue
		}
		select {
		case err := <-ft.doneErr:
			ft.err = err
			done[i] = true
			remaining--
			if err != nil {
				continue
			}
			justCompleted[i] = true
			// Force successful bars to 100% so the final flash frame shows a
			// full bar. Failed tasks should become done immediately; callers
			// often attach long error fields that must not participate in
			// aligned bar layout for one extra frame.
			if b := gts[i].builder; b.cfg.Mode == AnimationBar && b.cfg.BarProgress != nil {
				b.cfg.BarProgress.Store(b.cfg.BarTotal.Load())
			}
		default:
		}
	}
	return remaining
}

func effectiveGroupDone(done []bool, justCompleted []bool) []bool {
	for _, jc := range justCompleted {
		if jc {
			effectiveDone := slices.Clone(done)
			for i, jc2 := range justCompleted {
				if jc2 {
					effectiveDone[i] = false
				}
			}
			return effectiveDone
		}
	}
	return done
}

// runGroupLoop runs the group render loop, blocking until all tasks complete
// or the context is cancelled. Called by [Group.Wait].
func runGroupLoop(ctx context.Context, g *Group) error {
	g.mu.Lock()
	fxTasks := g.tasks
	g.mu.Unlock()

	if len(fxTasks) == 0 {
		return nil
	}

	// Wrap each groupTask with rendering state.
	var syncEpoch time.Time
	if g.syncAnimations {
		syncEpoch = time.Now()
	}
	gts := make([]*renderTask, len(fxTasks))
	for i, ft := range fxTasks {
		gt := &renderTask{
			groupTask: ft,
			monotonic: g.monotonic,
			syncEpoch: syncEpoch,
		}
		captureTaskConfig(gt)
		gts[i] = gt
	}

	// Build groupTasks for header/footer status lines.
	initStatus := func(s *groupStatus) (*renderTask, *Update) {
		if s == nil {
			return nil, nil
		}
		b := s.builder
		if b.cfg.Logger == nil {
			b.cfg.Logger = g.log
		}
		gt := newSyntheticTask(b, syncEpoch)
		gt.startedAt.Store(time.Now().UnixNano())

		u := &Update{
			msgText:   b.cfg.Message,
			msgPtr:    gt.msgPtr,
			fieldsPtr: gt.fieldsPtr,
			base:      b.Fields,
			symbolPtr: gt.symbolPtr,
			elapsed:   func() time.Duration { return gt.duration(time.Now()) },
		}
		u.InitSelf(u)
		return gt, u
	}

	headerGT, headerUpdate := initStatus(g.header)
	footerGT, footerUpdate := initStatus(g.footer)

	// Non-TTY: print each task's initial line, then block on all results.
	// Dynamic fields (elapsed, bar percent) are stripped because their
	// initial zero values are meaningless without live updates.
	if !gts[0].cfg.IsTTY {
		for _, gt := range gts {
			// A task may opt out of the non-TTY static line, matching the
			// standalone animation path; the rest still print and block. A
			// level-disabled task prints nothing on any writer.
			if gt.cfg.NonTTYSilent || gt.cfg.Silent {
				continue
			}
			b := gt.builder
			fieldsStr := strings.TrimLeft(
				gt.cfg.FormatFields(b.StripDynamicFields(*gt.fieldsPtr.Load())), " ",
			)
			line := gt.line(
				time.Now().In(gt.cfg.TimeLocation).Format(gt.cfg.TimeFormat),
				gt.cfg.Label,
				gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.builder.cfg.Level),
				gt.cfg.Indentation+*gt.msgPtr.Load(),
				fieldsStr,
			)
			writeString(gt.cfg.Out, line+nl)
		}
		for i, ft := range fxTasks {
			select {
			case ft.err = <-ft.doneErr:
			case <-ctx.Done():
				// Tasks before i already received their real results
				// (including nil for success); only the rest are pending.
				for _, ft2 := range fxTasks[i:] {
					select {
					case ft2.err = <-ft2.doneErr:
					default:
						ft2.err = ctx.Err()
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

	output := gts[0].cfg.Output
	stopResize := output.ListenResize()
	defer stopResize()

	blockTopRow := 0
	if row, ok := output.CursorPosition(); ok {
		blockTopRow = row
	}

	var overflowGT *renderTask
	if g.overflowIndicator {
		overflowGT = newOverflowTask(g, gts, syncEpoch)
	}

	st := &groupLoopState{
		g:             g,
		gts:           gts,
		fxTasks:       fxTasks,
		headerGT:      headerGT,
		footerGT:      footerGT,
		overflowGT:    overflowGT,
		headerUpdate:  headerUpdate,
		footerUpdate:  footerUpdate,
		output:        output,
		blockTopRow:   blockTopRow,
		renderStart:   time.Now(),
		done:          make([]bool, len(gts)),
		justCompleted: make([]bool, len(gts)),
		remaining:     len(gts),
	}

	// The group's block becomes one multi-line slot of the region's stacked
	// block. The region owns cursor visibility and the writer discipline, so
	// the group coordinates with standalone animations, other groups, and
	// (on a shared region) log lines on the same output instead of repainting
	// over them.
	return runGroupLoopRegion(ctx, st, liveRegionFor(output), tickRate)
}

// groupLoopState bundles the per-tick state of the group render loop.
type groupLoopState struct {
	g             *Group
	gts           []*renderTask
	fxTasks       []*groupTask
	headerGT      *renderTask
	footerGT      *renderTask
	overflowGT    *renderTask
	headerUpdate  *Update
	footerUpdate  *Update
	output        RenderOutput
	blockTopRow   int
	renderStart   time.Time
	done          []bool
	justCompleted []bool
	remaining     int
}

// failPending records ctx's error on every task that has not already
// completed, draining any result that raced the cancellation.
func (st *groupLoopState) failPending(ctx context.Context) {
	for i, ft := range st.fxTasks {
		if !st.done[i] {
			select {
			case ft.err = <-ft.doneErr:
			default:
				ft.err = ctx.Err()
			}
		}
	}
}

// composeFrame advances the group by one render tick - draining completions,
// invoking the header/footer callbacks, planning the frame against its
// physical-row budget, and rendering the planned lines - and returns the
// frame's rendered lines together with the terminal width they were laid out
// against. The boolean is false while the configured render delay is still
// pending, meaning the caller should keep whatever frame is currently on
// screen.
func (st *groupLoopState) composeFrame(now time.Time) ([]string, int, bool) {
	g := st.g
	output := st.output

	// Refresh terminal dimensions every tick so the layout is
	// computed against current reality, not a stale SIGWINCH-cached
	// value. SIGWINCH delivery can be coalesced or one-frame-lagged
	// under script(1), tmux pane resize, etc.; an extra ioctl per
	// tick is cheaper than emitting a line wider than the viewport.
	output.RefreshSize()
	st.remaining = drainGroupCompletions(
		st.fxTasks,
		st.gts,
		st.done,
		st.justCompleted,
		st.remaining,
	)
	effectiveDone := effectiveGroupDone(st.done, st.justCompleted)
	// Snapshot per-task atomics once per tick so measure and render
	// observe the same values. Must run after drainGroupCompletions,
	// which forces just-completed bars to 100%, and before
	// measureGroupRenderLayout consumes them.
	snapshotFrameValues(st.gts)
	if g.renderDelay > 0 && now.Sub(st.renderStart) < g.renderDelay {
		return nil, 0, false
	}

	headerCandidate, footerCandidate := st.updateStatusLines()
	plan := planGroupFrame(groupFramePlanInput{
		g:               g,
		gts:             st.gts,
		candidates:      visibleTaskIndexes(st.gts, effectiveDone, g.hideDone, now),
		done:            st.done,
		headerCandidate: headerCandidate,
		footerCandidate: footerCandidate,
		termHeight:      output.Height(),
		blockTopRow:     st.blockTopRow,
	})

	width := output.Width()
	lines := st.renderPlannedFrame(plan, effectiveDone, now, width)
	// Nothing to draw this tick (e.g. every task is level-disabled): skip the
	// repaint entirely rather than hiding the cursor over an empty block.
	if len(lines) == 0 {
		return nil, width, false
	}
	return lines, width, true
}

// updateStatusLines invokes the header/footer callbacks with the group's
// progress and reports which of the two has a non-empty message this tick.
func (st *groupLoopState) updateStatusLines() (bool, bool) {
	g := st.g
	doneCount := len(st.gts) - st.remaining
	totalCount := len(st.gts)
	headerCandidate := false
	footerCandidate := false
	if st.headerGT != nil {
		g.header.callback(doneCount, totalCount, st.headerUpdate)
		headerCandidate = *st.headerGT.msgPtr.Load() != ""
	}
	if st.footerGT != nil {
		g.footer.callback(doneCount, totalCount, st.footerUpdate)
		footerCandidate = *st.footerGT.msgPtr.Load() != ""
	}
	return headerCandidate, footerCandidate
}

// runGroupLoopRegion runs the group render loop with the group's block as one
// multi-line slot of the output's shared [core.LiveRegion], blocking until
// all tasks complete or the context is cancelled.
//
// The slot's render closure returns the most recently accepted frame rather
// than composing one on demand: per-tick composition mutates shared group
// state (completion draining, header/footer callbacks) and can query the
// terminal cursor position, none of which may run under the region mutex
// without stalling log displacement from other goroutines. So the heavy work
// stays on this goroutine's own ticker cadence, and each accepted frame is
// published for the closure to hand back as a cheap pointer load. Frames the
// region repaints between group ticks are therefore identical and deduped
// away.
func runGroupLoopRegion(
	ctx context.Context,
	st *groupLoopState,
	region *core.LiveRegion,
	tickRate time.Duration,
) error {
	var block atomic.Pointer[string]
	empty := ""
	block.Store(&empty)

	render := func(time.Time) string {
		return *block.Load()
	}
	// Physical rows of the last accepted frame, used for the viewport-fit
	// check. With other slots live this undercounts the full block, which
	// only makes the check more conservative about painting near the
	// viewport bottom.
	rows := 0
	unregister := func() {
		st.g.unregisterLiveSlot(region)
	}

	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	for st.remaining > 0 {
		select {
		case <-ctx.Done():
			last := *block.Load()
			// Unregister first: the region stops calling the render closure
			// and erases the group's rows from the block.
			unregister()
			if !st.g.clearOnCancel && last != "" {
				// Preserve the in-progress block as scrollback - the last
				// frame stays on screen. The frozen frame is written as
				// regular displaced lines so it lands above any animations
				// still live in the region.
				region.WriteLines(last + nl)
			}
			st.failPending(ctx)
			return ctx.Err()
		case <-st.g.liveWake:
			if *block.Load() != "" {
				st.g.registerLiveSlot(region, render, tickRate)
			}
		case <-ticker.C:
			now := time.Now()
			lines, width, ok := st.composeFrame(now)
			if !ok {
				continue
			}
			frameRows := groupFrameRows(lines, width)
			if !groupFrameFitsViewport(st.output, rows, frameRows) {
				continue
			}
			joined := strings.Join(lines, nl)
			block.Store(&joined)
			rows = frameRows
			if !st.g.hasLiveSlot() {
				if len(lines) == 0 {
					continue
				}
				// Register paints the new slot immediately, so this is the
				// group's first frame. Registration is deferred until a
				// non-empty frame exists so an idle group (render delay, all
				// tasks delayed) doesn't hide the cursor or hold a slot.
				st.g.registerLiveSlot(region, render, tickRate)
				continue
			}
			// Identical frames are deduped inside the region, so this only
			// writes when the group (or a sibling slot) actually changed.
			region.RenderFrame(now)
		}
	}

	unregister()
	return nil
}
