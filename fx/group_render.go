package fx

import (
	"context"
	"io"
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
	return int(gt.builder.barProgressPtr.Load()), int(gt.builder.barTotalPtr.Load())
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
		if gt.builder.barProgressPtr != nil {
			gt.frameBarCurrent = gt.builder.barProgressPtr.Load()
		}
		if gt.builder.barTotalPtr != nil {
			gt.frameBarTotal = gt.builder.barTotalPtr.Load()
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
	return gt.builder.lvl
}

// resolveLevel returns the effective level and styled level symbol for a
// completed task. If SetLevel was called on the Update, the overridden
// level is used; otherwise the builder's original level applies.
func (gt *renderTask) resolveLevel() (core.Level, string) {
	lvl := gt.effectiveLevel()
	if lvl == gt.builder.lvl {
		return lvl, gt.cfg.LevelSymbol
	}
	return lvl, gt.cfg.StyleLevel(lvl)
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

func (l *groupBarLayout) formatWithTruncationMarker(
	parts, leftText, barStr, rightText, sep string,
	placement bar.Placement,
	termWidth int,
	truncationMarker string,
) string {
	switch placement {
	case bar.PlaceLeftPad:
		return l.formatLeftPad(parts, leftText, barStr, rightText, sep, termWidth)
	case bar.PlaceRightPad:
		return l.formatRightPad(parts, leftText, barStr, rightText, sep, termWidth)
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

func (l *groupBarLayout) formatLeftPad(
	parts, leftText, barStr, rightText, sep string,
	termWidth int,
) string {
	shared := l.leftPad
	barFull := assembleBarColumns(shared, leftText, barStr, rightText, sep)
	if shared.maxParts == 0 || shared.maxBar == 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceLeftPad, termWidth)
	}

	gap := termWidth - rightEdgeSlack - shared.maxParts - shared.barWidth(sep)
	if gap < 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceLeftPad, termWidth)
	}

	// The message is not part of the per-tick frame snapshot, so parts can
	// drift wider than the measured max between layout and render; clamp so
	// the pad count never goes negative (strings.Repeat panics on negative).
	return barFull +
		strings.Repeat(" ", max(0, gap+shared.maxParts-lipgloss.Width(parts))) +
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

	gap := termWidth - rightEdgeSlack - shared.maxParts - shared.barWidth(sep)
	if gap < 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceRightPad, termWidth)
	}

	// Clamp for the same measure/render message drift as formatLeftPad.
	return parts + strings.Repeat(" ", max(0, shared.maxParts-lipgloss.Width(parts)+gap)) + barFull
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
	current := int(gt.builder.barProgressPtr.Load())
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

	updateInterval := gt.builder.barConfig.UpdateInterval
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
func captureTaskConfig(gt *renderTask) {
	b := gt.builder
	gt.cfg = b.log.TaskConfig(b)

	// Determine tick rate and pre-compute mode-specific resources.
	switch b.mode {
	case AnimationNone:
		if b.animatedSymbol {
			gt.tickRate = b.spinnerConfig.Interval
		}
	case AnimationPulse:
		gt.tickRate = pulse.TickRate
	case AnimationShimmer:
		gt.tickRate = shimmer.TickRate
		gt.hexLUT = shimmer.BuildLUT(b.shimmerStops)
		gt.styleLUT = shimmer.BuildStyleLUT(gt.hexLUT)
	case AnimationBar:
		gt.tickRate = bar.TickRate
	}
	// When animated symbol is enabled on a non-spinner mode, ensure the
	// tick rate is fast enough for smooth spinner frame changes.
	if b.animatedSymbol && b.spinnerConfig.Interval > 0 && gt.tickRate > 0 {
		gt.tickRate = min(gt.tickRate, b.spinnerConfig.Interval)
	}

	// Guard against missing spinner frames when animated symbol is enabled.
	if b.animatedSymbol && len(b.spinnerConfig.Frames) == 0 {
		b.spinnerConfig.Frames = spinner.DefaultConfig().Frames
	}
	if b.animatedSymbol && b.spinnerConfig.Boomerang {
		b.spinnerConfig.Frames = spinner.BoomerangFrames(b.spinnerConfig.Frames)
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
	if b.elapsedKey != "" || b.barPercentKey != "" {
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
	out := make([]core.Field, len(fields))
	copy(out, fields)

	if total <= 0 {
		total = 1
	}
	current = max(current, 0)

	pctMax := b.percentMaximum()
	pct := float64(current) / float64(total) * pctMax
	if pct > pctMax {
		pct = pctMax
	}

	for i := range out {
		switch out[i].Key {
		case b.elapsedKey:
			f := b.elapsedOverride
			f.Value = dur
			out[i].Value = f
		case b.barPercentKey:
			out[i].Value = core.Percent{Value: pct}
		}
	}

	return out
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
		renderLevel, levelSymbol := gt.resolveLevel()
		msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), renderLevel)
		return lipgloss.Width(buildLine(
			gt.cfg.Order,
			gt.cfg.ReportTimestamp,
			tsStr,
			levelSymbol,
			gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), renderLevel),
			msg,
			"",
		))
	}

	msg, char := renderTaskMessageSymbol(gt, now)

	return lipgloss.Width(buildLine(
		gt.cfg.Order,
		gt.cfg.ReportTimestamp,
		tsStr,
		gt.cfg.LevelSymbol,
		char,
		msg,
		"",
	))
}

// renderTaskLine renders a single animation frame line for a task.
// For done tasks, it renders the frozen final state with the level's default symbol.
// For active tasks, it renders the current animation frame.
// It does not perform any I/O.
func renderTaskLine(gt *renderTask, isDone bool, now time.Time, layout *groupRenderLayout) string {
	b := gt.builder
	fieldsPtr := gt.fieldsSnapshot()
	current, total := 0, 0
	if b.mode == AnimationBar {
		current, total = gt.barProgress()
	}
	dur := gt.duration(now)
	fieldsStr := renderTaskFields(gt, fieldsPtr, dur, current, total)
	tsStr := renderTaskTimestamp(gt, now)

	if isDone {
		// Show the frozen final line with the level's default symbol.
		// If SetLevel was called, use the overridden level for styling.
		renderLevel, levelSymbol := gt.resolveLevel()
		msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), renderLevel)
		doneSymbol := gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), renderLevel)
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
		return buildLine(
			gt.cfg.Order,
			gt.cfg.ReportTimestamp,
			tsStr,
			levelSymbol,
			doneSymbol,
			msg,
			fieldsStr,
		)
	}

	// Bar mode has its own rendering path.
	if b.mode == AnimationBar {
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
	msg = alignMessageForFields(
		gt.cfg.Order,
		gt.cfg.ReportTimestamp,
		tsStr,
		gt.cfg.LevelSymbol,
		char,
		msg,
		fieldsStr,
		layout,
	)

	return buildLine(
		gt.cfg.Order,
		gt.cfg.ReportTimestamp,
		tsStr,
		gt.cfg.LevelSymbol,
		char,
		msg,
		fieldsStr,
	)
}

func renderAnimatedTaskMessage(gt *renderTask, now time.Time) (string, string) {
	b := gt.builder
	msg := *gt.msgPtr.Load()
	dur := gt.animDuration(now)

	// Message animation.
	switch b.mode { //nolint:exhaustive // animationBar handled by caller
	case AnimationPulse:
		t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*b.speed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
		msg = pulse.TextCached(msg, t, b.pulseStops, &gt.pCache)
	case AnimationShimmer:
		phase := math.Mod(dur.Seconds()*b.speed, 1.0)
		msg = shimmer.Text(msg, phase, b.shimmerDir, gt.hexLUT, gt.styleLUT)
	default:
		msg = gt.cfg.StyleMessage(msg, b.lvl)
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
	if b.animatedSymbol && !gt.symbolOverride.Load() &&
		gt.started() && len(b.spinnerConfig.Frames) > 0 &&
		b.spinnerConfig.Interval > 0 {
		n := len(b.spinnerConfig.Frames)
		i := int(gt.animDuration(now)/b.spinnerConfig.Interval) % n
		if b.spinnerConfig.Reverse {
			i = n - 1 - i
		}
		return gt.cfg.StyleSymbol(b.spinnerConfig.Frames[i], b.lvl)
	}
	return gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.effectiveLevel())
}

func renderTaskMessageSymbol(gt *renderTask, now time.Time) (string, string) {
	if !gt.started() {
		return gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.builder.lvl),
			gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.builder.lvl)
	}

	if gt.builder.mode == AnimationBar {
		return gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.builder.lvl),
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
	if !showBar || gt.builder.barConfig.Placement == bar.PlaceInline {
		return parts
	}
	if layout != nil {
		return layout.bar.formatWithTruncationMarker(
			parts,
			leftText,
			barStr,
			rightText,
			sep,
			gt.builder.barConfig.Placement,
			gt.cfg.Output.Width(),
			bar.ResolveTruncationMarker(gt.builder.barConfig),
		)
	}
	barFull := assembleBarColumns(groupBarColumns{}, leftText, barStr, rightText, sep)
	return bar.FormatLine(
		parts,
		barFull,
		sep,
		gt.builder.barConfig.Placement,
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
	msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), b.lvl)
	msg = alignMessageForFields(
		gt.cfg.Order,
		gt.cfg.ReportTimestamp,
		tsStr,
		gt.cfg.LevelSymbol,
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
	if b.barConfig.Smoothing == bar.SmoothEase {
		now := state.renderAt
		if !gt.smoothedInit {
			gt.smoothedProgress = renderProgress
			gt.smoothedTime = now
			gt.smoothedInit = true
		} else {
			tau := b.barConfig.SmoothingTau
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
	sep := b.barConfig.Separator
	if sep == "" {
		sep = " "
	}

	parts := buildLine(
		gt.cfg.Order,
		gt.cfg.ReportTimestamp,
		tsStr,
		gt.cfg.LevelSymbol,
		symbol,
		msg,
		fieldsStr,
	)
	if !bar.ShowPending(b.barConfig, current) {
		return parts, "", "", "", sep, false
	}

	// Cache the gradient style to avoid lipgloss.NewStyle() per frame.
	barStyle := b.barConfig
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
	if gt.monotonic || b.barConfig.Smoothing == bar.SmoothEase {
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
	if b.barConfig.WidgetLeft != nil {
		leftText = b.barConfig.WidgetLeft(widgetState)
	}
	if b.barConfig.WidgetRight != nil && b.barPercentKey == "" {
		rightText = b.barConfig.WidgetRight(widgetState)
	} else if b.barConfig.WidgetLeft == nil && b.barConfig.WidgetRight == nil && b.barPercentKey == "" {
		// Default: padded percent on the right when no widgets are configured
		// and no BarPercent field is set.
		rightText = bar.FormatPercent(current, total, 0, true)
	}

	// writeFrame equivalent: build the complete line string.
	if b.barConfig.Placement == bar.PlaceInline {
		barFull := assembleBarColumns(groupBarColumns{
			hasLeft:  leftText != "",
			hasRight: rightText != "",
			maxBar:   lipgloss.Width(barStr),
			maxLeft:  lipgloss.Width(leftText),
			maxRight: lipgloss.Width(rightText),
		}, leftText, barStr, rightText, sep)
		return buildLine(
			gt.cfg.Order,
			gt.cfg.ReportTimestamp,
			tsStr,
			gt.cfg.LevelSymbol,
			symbol,
			msg+sep+barFull,
			fieldsStr,
		), "", "", "", sep, true
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
			gt.builder.mode != AnimationBar {
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
		layout.bar.observe(parts, leftText, barStr, rightText, gt.builder.barConfig.Placement)
	}

	// For PlaceAligned, also measure done tasks so completed messages
	// (which may be longer) are included in the max parts width.
	// Skip when HideDone is set since done tasks are not rendered.
	for _, i := range indexes {
		gt := gts[i]
		if hideDone || !shouldRenderTask(gt, done[i], now) || !done[i] {
			continue
		}
		if gt.builder.mode != AnimationBar || gt.builder.barConfig.Placement != bar.PlaceAligned {
			continue
		}
		tsStr := renderTaskTimestamp(gt, now)
		msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.builder.lvl)
		fieldsStr := renderTaskFields(gt, gt.fieldsSnapshot(), gt.duration(now), 0, 0)
		parts := buildLine(
			gt.cfg.Order,
			gt.cfg.ReportTimestamp,
			tsStr,
			gt.cfg.LevelSymbol,
			gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.builder.lvl),
			msg,
			fieldsStr,
		)
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

func buildGroupFrameLines(
	headerGT, footerGT *renderTask,
	gts []*renderTask,
	visible []int,
	effectiveDone []bool,
	now time.Time,
	layout *groupRenderLayout,
	showHeader, showFooter bool,
) ([]string, int, int, int) {
	lineCap := len(visible)
	if showHeader {
		lineCap++
	}
	if showFooter {
		lineCap++
	}
	lines := make([]string, 0, lineCap)

	headerLineCount := 0
	if showHeader {
		lines = append(lines, renderTaskLine(headerGT, false, now, layout))
		headerLineCount = 1
	}

	taskLineCount := 0
	for _, taskIndex := range visible {
		lines = append(lines, renderTaskLine(gts[taskIndex], effectiveDone[taskIndex], now, layout))
		taskLineCount++
	}

	footerLineCount := 0
	if showFooter {
		lines = append(lines, renderTaskLine(footerGT, false, now, layout))
		footerLineCount = 1
	}

	return lines, headerLineCount, taskLineCount, footerLineCount
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
			if b := gts[i].builder; b.mode == AnimationBar && b.barProgressPtr != nil {
				b.barProgressPtr.Store(b.barTotalPtr.Load())
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
		if b.log == nil {
			b.log = g.log
		}
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]core.Field]{}
		symbolPtr := &atomic.Pointer[string]{}
		msgPtr.Store(&b.message)
		fieldsPtr.Store(&b.Fields)
		sym := b.symbolIcon
		if sym == "" {
			sym = DefaultSymbol
		}
		symbolPtr.Store(&sym)

		gt := &renderTask{
			groupTask: &groupTask{
				builder:   b,
				fieldsPtr: fieldsPtr,
				msgPtr:    msgPtr,
				symbolPtr: symbolPtr,
			},
			syncEpoch: syncEpoch,
		}
		gt.startedAt.Store(time.Now().UnixNano())
		captureTaskConfig(gt)

		u := &Update{
			msgText:   b.message,
			msgPtr:    msgPtr,
			fieldsPtr: fieldsPtr,
			base:      b.Fields,
			symbolPtr: symbolPtr,
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
			// standalone animation path; the rest still print and block.
			if gt.cfg.NonTTYSilent {
				continue
			}
			b := gt.builder
			fieldsStr := strings.TrimLeft(
				gt.cfg.FormatFields(b.StripDynamicFields(*gt.fieldsPtr.Load())), " ",
			)
			line := buildLine(
				gt.cfg.Order,
				gt.cfg.ReportTimestamp,
				time.Now().In(gt.cfg.TimeLocation).Format(gt.cfg.TimeFormat),
				gt.cfg.Label,
				gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.builder.lvl),
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

	out := gts[0].cfg.Out
	output := gts[0].cfg.Output
	stopResize := output.ListenResize()
	defer stopResize()

	blockTopRow := 0
	if row, ok := output.CursorPosition(); ok {
		blockTopRow = row
	}

	st := &groupLoopState{
		g:             g,
		gts:           gts,
		fxTasks:       fxTasks,
		headerGT:      headerGT,
		footerGT:      footerGT,
		headerUpdate:  headerUpdate,
		footerUpdate:  footerUpdate,
		output:        output,
		blockTopRow:   blockTopRow,
		renderStart:   time.Now(),
		done:          make([]bool, len(gts)),
		justCompleted: make([]bool, len(gts)),
		remaining:     len(gts),
	}

	// Live-region path: when the output exposes a shared LiveRegion, the
	// group's block becomes one multi-line slot of the region's stacked
	// block. The region owns cursor visibility and the writer discipline, so
	// the group coordinates with standalone animations, other groups, and
	// log lines on the same output instead of repainting over them. Outputs
	// without the capability (external Output implementations and test
	// stubs) keep the legacy direct renderer below.
	if p, ok := output.(liveRegionProvider); ok {
		if region := p.LiveRegion(); region != nil {
			return runGroupLoopRegion(ctx, st, region, tickRate)
		}
	}

	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	renderedRows := 0
	hasRendered := false
	var lastLines []string
	lastWidth := 0
	var frameBuf strings.Builder
	cursorHidden := false
	defer func() {
		if cursorHidden {
			writeString(out, xansi.ShowCursor)
		}
	}()
	hideCursor := func() {
		if cursorHidden {
			return
		}
		writeString(out, xansi.HideCursor)
		cursorHidden = true
	}

	for st.remaining > 0 {
		select {
		case <-ctx.Done():
			if g.clearOnCancel {
				eraseBlockSync(out, renderedRows)
			} else {
				preserveBlock(out, renderedRows)
			}
			st.failPending(ctx)
			return ctx.Err()
		case <-ticker.C:
			lines, width, ok := st.composeFrame(time.Now())
			if !ok {
				continue
			}
			frameRows := groupFrameRows(lines, width)
			if !groupFrameFitsViewport(output, renderedRows, frameRows) {
				continue
			}
			// Skip identical frames: nothing on screen would change, so a
			// write would be wasted bandwidth. The cursor is already parked
			// one line below the block from the previous frame.
			if hasRendered && width == lastWidth && slices.Equal(lines, lastLines) {
				continue
			}
			frameBuf.Reset()
			renderedRows = appendRepaint(&frameBuf, lines, renderedRows, width)
			hideCursor()
			writeString(out, frameBuf.String())
			hasRendered = true
			lastLines = lines
			lastWidth = width
		}
	}

	eraseBlockSync(out, renderedRows)
	return nil
}

// groupLoopState bundles the per-tick state shared by the legacy and
// live-region group render paths so both drive the exact same frame
// composition.
type groupLoopState struct {
	g             *Group
	gts           []*renderTask
	fxTasks       []*groupTask
	headerGT      *renderTask
	footerGT      *renderTask
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
// invoking the header/footer callbacks, selecting visible tasks, and capping
// the block to its physical-row budget - and returns the frame's rendered
// lines together with the terminal width they were laid out against. The
// boolean is false while the configured render delay is still pending,
// meaning the caller should keep whatever frame is currently on screen.
func (st *groupLoopState) composeFrame(now time.Time) ([]string, int, bool) {
	g := st.g
	gts := st.gts
	output := st.output

	// Refresh terminal dimensions every tick so the layout is
	// computed against current reality, not a stale SIGWINCH-cached
	// value. SIGWINCH delivery can be coalesced or one-frame-lagged
	// under script(1), tmux pane resize, etc.; an extra ioctl per
	// tick is cheaper than emitting a line wider than the viewport.
	output.RefreshWidth()
	output.RefreshHeight()
	st.remaining = drainGroupCompletions(st.fxTasks, gts, st.done, st.justCompleted, st.remaining)
	effectiveDone := effectiveGroupDone(st.done, st.justCompleted)
	// Snapshot per-task atomics once per tick so measure and render
	// observe the same values. Must run after drainGroupCompletions,
	// which forces just-completed bars to 100%, and before
	// measureGroupRenderLayout consumes them.
	snapshotFrameValues(gts)
	if g.renderDelay > 0 && now.Sub(st.renderStart) < g.renderDelay {
		return nil, 0, false
	}
	doneCount := len(gts) - st.remaining
	totalCount := len(gts)
	headerCandidate := false
	footerCandidate := false
	if st.headerGT != nil {
		g.header.callback(doneCount, totalCount, st.headerUpdate)
		if msg := *st.headerGT.msgPtr.Load(); msg != "" {
			headerCandidate = true
		}
	}
	if st.footerGT != nil {
		g.footer.callback(doneCount, totalCount, st.footerUpdate)
		if msg := *st.footerGT.msgPtr.Load(); msg != "" {
			footerCandidate = true
		}
	}

	candidateStatusLines := 0
	if headerCandidate {
		candidateStatusLines++
	}
	if footerCandidate {
		candidateStatusLines++
	}
	maxLines := groupHeightCap(output.Height(), st.blockTopRow, len(gts)+candidateStatusLines)
	if g.maxHeightPercent > 0 {
		if pctLines := int(
			float64(output.Height()) * g.maxHeightPercent,
		); pctLines > 0 &&
			pctLines < maxLines {
			maxLines = pctLines
		}
	}
	if g.maxLines > 0 && g.maxLines < maxLines {
		maxLines = g.maxLines
	}
	// Cap visible tasks to terminal height so cursor-up
	// escapes never need to reach scrolled-off lines.
	// Prioritise active (in-progress) tasks over done or
	// pending ones when space is limited.
	visible := visibleTaskIndexes(gts, effectiveDone, g.hideDone, now)
	persistentStatusLines := 0
	if headerCandidate && !g.transientHeader {
		persistentStatusLines++
	}
	if footerCandidate && !g.transientFooter {
		persistentStatusLines++
	}
	maxTasks := max(0, maxLines-persistentStatusLines)
	if len(visible) > maxTasks {
		visible = prioritiseActive(visible, gts, st.done, maxTasks)
	}
	showHeader := headerCandidate && (!g.transientHeader || len(visible) > 0)
	showFooter := footerCandidate && (!g.transientFooter || len(visible) > 0)
	statusLines := 0
	if showHeader {
		statusLines++
	}
	if showFooter {
		statusLines++
	}
	if len(visible) > 0 && maxLines-statusLines <= 0 {
		if showFooter && g.transientFooter {
			showFooter = false
			statusLines--
		}
		if showHeader && g.transientHeader && maxLines-statusLines <= 0 {
			showHeader = false
			statusLines--
		}
	}
	maxTasks = max(0, maxLines-statusLines)
	if len(visible) > maxTasks {
		visible = prioritiseActive(visible, gts, st.done, maxTasks)
	}
	if len(visible) == 0 {
		if g.transientHeader {
			showHeader = false
		}
		if g.transientFooter {
			showFooter = false
		}
	}
	layout := measureGroupRenderLayoutForIndexes(g, gts, effectiveDone, visible, now)
	lines, headerLineCount, taskLineCount, footerLineCount := buildGroupFrameLines(
		st.headerGT,
		st.footerGT,
		gts,
		visible,
		effectiveDone,
		now,
		layout,
		showHeader,
		showFooter,
	)
	width := output.Width()
	// Physical-row cap: maxLines is a physical-row budget (terminal
	// rows after wrap). Drop the lowest-priority task lines from the
	// tail until the rendered block fits. Header and footer are
	// preserved when possible because they were already accounted for
	// when computing maxTasks above.
	physRows := groupFrameRows(lines, width)
	for physRows > maxLines && taskLineCount > 0 {
		visible = visible[:taskLineCount-1]
		layout = measureGroupRenderLayoutForIndexes(g, gts, effectiveDone, visible, now)
		lines, headerLineCount, taskLineCount, footerLineCount = buildGroupFrameLines(
			st.headerGT,
			st.footerGT,
			gts,
			visible,
			effectiveDone,
			now,
			layout,
			showHeader,
			showFooter,
		)
		physRows = groupFrameRows(lines, width)
	}
	// If we couldn't fit any tasks, transient header/footer should
	// drop too so we don't render a status-only block.
	if taskLineCount == 0 {
		if footerLineCount == 1 && g.transientFooter {
			lines = lines[:len(lines)-1]
		}
		if headerLineCount == 1 && g.transientHeader {
			lines = lines[1:]
		}
	}
	return lines, width, true
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
// stays on this goroutine - the same ticker cadence as the legacy renderer -
// and each accepted frame is published for the closure to hand back as a
// cheap pointer load. Frames the region repaints between group ticks are
// therefore identical and deduped away, keeping the byte stream equal to the
// legacy renderer's when the group runs alone.
func runGroupLoopRegion(
	ctx context.Context,
	st *groupLoopState,
	region *core.LiveRegion,
	tickRate time.Duration,
) error {
	var block atomic.Pointer[string]
	empty := ""
	block.Store(&empty)

	registered := false
	var slotID uint64
	// Physical rows of the last accepted frame, mirroring the legacy
	// renderer's renderedRows for the viewport-fit check. With other slots
	// live this undercounts the full block, which only makes the check more
	// conservative about painting near the viewport bottom.
	rows := 0
	unregister := func() {
		if registered {
			region.Unregister(slotID)
			registered = false
		}
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
				// Preserve the in-progress block as scrollback, mirroring the
				// legacy renderer which leaves the last frame on screen. The
				// frozen frame is written as regular displaced lines so it
				// lands above any animations still live in the region.
				region.WriteLines(last + nl)
			}
			st.failPending(ctx)
			return ctx.Err()
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
			if !registered {
				if len(lines) == 0 {
					continue
				}
				// Register paints the new slot immediately, so this is the
				// group's first frame. Registration is deferred until a
				// non-empty frame exists so an idle group (render delay, all
				// tasks delayed) doesn't hide the cursor or hold a slot.
				slotID = region.Register(func(time.Time) string {
					return *block.Load()
				}, tickRate)
				registered = true
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

// syncFrame brackets s with DEC 2026 synchronized-output markers so
// supporting terminals apply the whole sequence atomically (no tearing);
// terminals without support ignore the markers. Delegates to [core.SyncFrame]
// so the live-region renderer shares the exact same framing.
func syncFrame(s string) string {
	return core.SyncFrame(s)
}

// appendRepaint appends an overwrite-in-place repaint of lines over the
// previously rendered block of prevRows physical rows, bracketed by DEC 2026
// synchronized-output markers. Delegates to [core.AppendRepaint] so the
// live-region renderer shares the exact same repaint sequence; see there for
// the line-by-line erase rationale. Returns the physical row count of the
// new frame.
func appendRepaint(buf *strings.Builder, lines []string, prevRows, width int) int {
	return core.AppendRepaint(buf, lines, prevRows, width)
}

// eraseBlockSync erases a rendered block of n physical rows as a single
// synchronized-output frame. The cursor is expected to be one line below the
// block (after the park newline written each frame); the park line is
// cleared first because it may hold terminal-echoed characters (e.g. ^C
// from SIGINT). The cursor ends up back on the block's first row.
func eraseBlockSync(out io.Writer, n int) {
	if n == 0 {
		return
	}
	var buf strings.Builder
	buf.WriteString(xansi.ClearLine + xansi.CursorUp(1))
	appendClearBlock(&buf, n)
	writeString(out, syncFrame(buf.String()))
}

// preserveBlock moves the cursor past the rendered block without erasing it.
// The cursor is expected to be one line below the block (after the trailing
// newline written each tick). A single newline is written to move past any
// terminal-echoed characters (e.g. ^C from SIGINT).
func preserveBlock(out io.Writer, n int) {
	if n > 0 {
		writeString(out, nl)
	}
}

// clearBlock erases n lines starting from the current cursor line and
// repositions the cursor back to the first cleared line. The cursor is
// expected to be on the last line of the block (no trailing newline).
func clearBlock(out io.Writer, n int) {
	if n == 0 {
		return
	}
	var buf strings.Builder
	appendClearBlock(&buf, n)
	writeString(out, buf.String())
}

// appendClearBlock appends the escape sequence that erases n lines starting
// from the current cursor line and repositions the cursor back to the first
// cleared line.
func appendClearBlock(buf *strings.Builder, n int) {
	if n > 1 {
		buf.WriteString(xansi.CursorUp(n - 1))
	}
	for range n {
		buf.WriteString(xansi.ClearLine + nl)
	}
	buf.WriteString(xansi.CursorUp(n))
}
