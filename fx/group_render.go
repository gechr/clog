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
	return int(gt.builder.BarProgressPtr.Load()), int(gt.builder.BarTotalPtr.Load())
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
		if gt.builder.BarProgressPtr != nil {
			gt.frameBarCurrent = gt.builder.BarProgressPtr.Load()
		}
		if gt.builder.BarTotalPtr != nil {
			gt.frameBarTotal = gt.builder.BarTotalPtr.Load()
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
	return gt.builder.Level
}

// resolveLevel returns the effective level and styled level symbol for a
// completed task. If SetLevel was called on the Update, the overridden
// level is used; otherwise the builder's original level applies.
func (gt *renderTask) resolveLevel() (core.Level, string) {
	lvl := gt.effectiveLevel()
	if lvl == gt.builder.Level {
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

	delay := gt.builder.DelayDur
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
	progress = max(0, min(1, progress))
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
		return l.formatAligned(parts, leftText, barStr, rightText, sep, termWidth)
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
const rightEdgeSlack = 1

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

	gap := termWidth - rightEdgeSlack - shared.maxParts - shared.barWidth(sep)
	if gap < 0 {
		return bar.FormatLine(parts, barFull, sep, bar.PlaceRightPad, termWidth)
	}

	return parts + strings.Repeat(" ", shared.maxParts-lipgloss.Width(parts)+gap) + barFull
}

func (l *groupBarLayout) formatAligned(
	parts, leftText, barStr, rightText, sep string,
	termWidth int,
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
	effectiveMax := a.maxParts
	if termWidth > 0 && termWidth <= rightEdgeSlack {
		return ""
	}
	if termWidth > 0 {
		var ok bool
		parts, effectiveMax, ok = capAlignedParts(parts, barFull, sep, effectiveMax, termWidth)
		if !ok {
			return xansi.Truncate(barFull, termWidth-rightEdgeSlack, "")
		}
	}
	if a.maxParts == 0 {
		return parts + sep + barFull
	}
	padding := effectiveMax - lipgloss.Width(parts)
	if padding <= 0 {
		return parts + sep + barFull
	}
	return parts + strings.Repeat(" ", padding) + sep + barFull
}

func capAlignedParts(parts, barFull, sep string, maxParts, termWidth int) (string, int, bool) {
	target := termWidth - rightEdgeSlack
	budget := target - lipgloss.Width(sep) - lipgloss.Width(barFull)
	if budget < 0 {
		return parts, maxParts, false
	}
	maxParts = min(maxParts, budget)
	if lipgloss.Width(parts) > budget {
		parts = xansi.Truncate(parts, budget, "")
	}
	return parts, maxParts, true
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
	current := int(gt.builder.BarProgressPtr.Load())
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

	updateInterval := gt.builder.BarConfig.UpdateInterval
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
	gt.cfg = b.Log.TaskConfig(b)

	// Determine tick rate and pre-compute mode-specific resources.
	switch b.Mode {
	case AnimationNone:
		if b.AnimatedSymbol {
			gt.tickRate = b.SpinnerConfig.Interval
		}
	case AnimationPulse:
		gt.tickRate = pulse.TickRate
	case AnimationShimmer:
		gt.tickRate = shimmer.TickRate
		gt.hexLUT = shimmer.BuildLUT(b.ShimmerStops)
		gt.styleLUT = shimmer.BuildStyleLUT(gt.hexLUT)
	case AnimationBar:
		gt.tickRate = bar.TickRate
	}
	// When animated symbol is enabled on a non-spinner mode, ensure the
	// tick rate is fast enough for smooth spinner frame changes.
	if b.AnimatedSymbol && b.SpinnerConfig.Interval > 0 && gt.tickRate > 0 {
		gt.tickRate = min(gt.tickRate, b.SpinnerConfig.Interval)
	}

	// Guard against missing spinner frames when animated symbol is enabled.
	if b.AnimatedSymbol && len(b.SpinnerConfig.Frames) == 0 {
		b.SpinnerConfig.Frames = spinner.DefaultConfig().Frames
	}
	if b.AnimatedSymbol && b.SpinnerConfig.Boomerang {
		b.SpinnerConfig.Frames = spinner.BoomerangFrames(b.SpinnerConfig.Frames)
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
	if b.ElapsedKey != "" || b.BarPercentKey != "" {
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
		case b.ElapsedKey:
			out[i].Value = core.ElapsedField(dur)
		case b.BarPercentKey:
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
	if b.Mode == AnimationBar {
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
	if b.Mode == AnimationBar {
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
	switch b.Mode { //nolint:exhaustive // animationBar handled by caller
	case AnimationPulse:
		t := (1.0 + math.Sin(2*math.Pi*dur.Seconds()*b.Speed-math.Pi/2)) / 2 //nolint:mnd // half-wave normalisation
		msg = pulse.TextCached(msg, t, b.PulseStops, &gt.pCache)
	case AnimationShimmer:
		phase := math.Mod(dur.Seconds()*b.Speed, 1.0)
		msg = shimmer.Text(msg, phase, b.ShimmerDir, gt.hexLUT, gt.styleLUT)
	default:
		msg = gt.cfg.StyleMessage(msg, b.Level)
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
// When [Builder.AnimatedSymbol] is true, it cycles through [SpinnerConfig]
// frames based on wall-clock time. When the symbol has been explicitly
// overridden via [Update.SetSymbol], the static symbol is returned instead
// so the caller can replace the spinner with a checkmark or other icon.
// If [Update.SetLevel] was also called, the overridden level is used for
// styling so the symbol color matches the intended level.
func resolveSymbol(gt *renderTask, now time.Time) string {
	b := gt.builder
	if b.AnimatedSymbol && !gt.symbolOverride.Load() &&
		gt.started() && len(b.SpinnerConfig.Frames) > 0 &&
		b.SpinnerConfig.Interval > 0 {
		n := len(b.SpinnerConfig.Frames)
		i := int(gt.animDuration(now)/b.SpinnerConfig.Interval) % n
		if b.SpinnerConfig.Reverse {
			i = n - 1 - i
		}
		return gt.cfg.StyleSymbol(b.SpinnerConfig.Frames[i], b.Level)
	}
	return gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.effectiveLevel())
}

func renderTaskMessageSymbol(gt *renderTask, now time.Time) (string, string) {
	if !gt.started() {
		return gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.builder.Level),
			gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.builder.Level)
	}

	if gt.builder.Mode == AnimationBar {
		return gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.builder.Level),
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
	if !showBar || gt.builder.BarConfig.Placement == bar.PlaceInline {
		return parts
	}
	if layout != nil {
		return layout.bar.format(
			parts,
			leftText,
			barStr,
			rightText,
			sep,
			gt.builder.BarConfig.Placement,
			gt.cfg.Output.Width(),
		)
	}
	barFull := assembleBarColumns(groupBarColumns{}, leftText, barStr, rightText, sep)
	return bar.FormatLine(
		parts,
		barFull,
		sep,
		gt.builder.BarConfig.Placement,
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
	msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), b.Level)
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
	if b.BarConfig.Smoothing == bar.SmoothEase {
		now := state.renderAt
		if !gt.smoothedInit {
			gt.smoothedProgress = renderProgress
			gt.smoothedTime = now
			gt.smoothedInit = true
		} else {
			tau := b.BarConfig.SmoothingTau
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
	sep := b.BarConfig.Separator
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
	if !bar.ShowPending(b.BarConfig, current) {
		return parts, "", "", "", sep, false
	}

	// Cache the gradient style to avoid lipgloss.NewStyle() per frame.
	barStyle := b.BarConfig
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
	if gt.monotonic || b.BarConfig.Smoothing == bar.SmoothEase {
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
	if b.BarConfig.WidgetLeft != nil {
		leftText = b.BarConfig.WidgetLeft(widgetState)
	}
	if b.BarConfig.WidgetRight != nil && b.BarPercentKey == "" {
		rightText = b.BarConfig.WidgetRight(widgetState)
	} else if b.BarConfig.WidgetLeft == nil && b.BarConfig.WidgetRight == nil && b.BarPercentKey == "" {
		// Default: padded percent on the right when no widgets are configured
		// and no BarPercent field is set.
		rightText = bar.FormatPercent(current, total, 0, true)
	}

	// writeFrame equivalent: build the complete line string.
	if b.BarConfig.Placement == bar.PlaceInline {
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
			gt.builder.Mode != AnimationBar {
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
		layout.bar.observe(parts, leftText, barStr, rightText, gt.builder.BarConfig.Placement)
	}

	// For PlaceAligned, also measure done tasks so completed messages
	// (which may be longer) are included in the max parts width.
	// Skip when HideDone is set since done tasks are not rendered.
	for _, i := range indexes {
		gt := gts[i]
		if hideDone || !shouldRenderTask(gt, done[i], now) || !done[i] {
			continue
		}
		if gt.builder.Mode != AnimationBar || gt.builder.BarConfig.Placement != bar.PlaceAligned {
			continue
		}
		tsStr := renderTaskTimestamp(gt, now)
		msg := gt.cfg.Indentation + gt.cfg.StyleMessage(*gt.msgPtr.Load(), gt.builder.Level)
		fieldsStr := renderTaskFields(gt, gt.fieldsSnapshot(), gt.duration(now), 0, 0)
		parts := buildLine(
			gt.cfg.Order,
			gt.cfg.ReportTimestamp,
			tsStr,
			gt.cfg.LevelSymbol,
			gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.builder.Level),
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
	rows := 0
	for _, line := range lines {
		rows += frameRows(line, termWidth)
	}
	return rows
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
			if b := gts[i].builder; b.Mode == AnimationBar && b.BarProgressPtr != nil {
				b.BarProgressPtr.Store(b.BarTotalPtr.Load())
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
		if b.Log == nil {
			b.Log = g.log
		}
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]core.Field]{}
		symbolPtr := &atomic.Pointer[string]{}
		msgPtr.Store(&b.Message)
		fieldsPtr.Store(&b.Fields)
		sym := b.SymbolIcon
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
			MsgText:   b.Message,
			MsgPtr:    msgPtr,
			FieldsPtr: fieldsPtr,
			Base:      b.Fields,
			SymbolPtr: symbolPtr,
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
			b := gt.builder
			fieldsStr := strings.TrimLeft(
				gt.cfg.FormatFields(b.StripDynamicFields(*gt.fieldsPtr.Load())), " ",
			)
			line := buildLine(
				gt.cfg.Order,
				gt.cfg.ReportTimestamp,
				time.Now().In(gt.cfg.TimeLocation).Format(gt.cfg.TimeFormat),
				gt.cfg.Label,
				gt.cfg.StyleSymbol(*gt.symbolPtr.Load(), gt.builder.Level),
				gt.cfg.Indentation+*gt.msgPtr.Load(),
				fieldsStr,
			)
			writeString(gt.cfg.Out, line+nl)
		}
		for _, ft := range fxTasks {
			select {
			case ft.err = <-ft.doneErr:
			case <-ctx.Done():
				for _, ft2 := range fxTasks {
					if ft2.err == nil {
						select {
						case ft2.err = <-ft2.doneErr:
						default:
							ft2.err = ctx.Err()
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

	out := gts[0].cfg.Out
	output := gts[0].cfg.Output
	stopResize := output.ListenResize()
	defer stopResize()

	blockTopRow := 0
	if row, ok := output.CursorPosition(); ok {
		blockTopRow = row
	}

	ticker := time.NewTicker(tickRate)
	defer ticker.Stop()

	renderStart := time.Now()
	renderedRows := 0
	hasRendered := false
	var lastLines []string
	lastWidth := 0
	done := make([]bool, len(gts))
	justCompleted := make([]bool, len(gts))
	remaining := len(gts)
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

	for remaining > 0 {
		select {
		case <-ctx.Done():
			if g.clearOnCancel {
				eraseBlockSync(out, renderedRows)
			} else {
				preserveBlock(out, renderedRows)
			}
			for i, ft := range fxTasks {
				if !done[i] {
					select {
					case ft.err = <-ft.doneErr:
					default:
						ft.err = ctx.Err()
					}
				}
			}
			return ctx.Err()
		case <-ticker.C:
			now := time.Now()
			// Refresh terminal dimensions every tick so the layout is
			// computed against current reality, not a stale SIGWINCH-cached
			// value. SIGWINCH delivery can be coalesced or one-frame-lagged
			// under script(1), tmux pane resize, etc.; an extra ioctl per
			// tick is cheaper than emitting a line wider than the viewport.
			output.RefreshWidth()
			output.RefreshHeight()
			remaining = drainGroupCompletions(fxTasks, gts, done, justCompleted, remaining)
			effectiveDone := effectiveGroupDone(done, justCompleted)
			// Snapshot per-task atomics once per tick so measure and render
			// observe the same values. Must run after drainGroupCompletions,
			// which forces just-completed bars to 100%, and before
			// measureGroupRenderLayout consumes them.
			snapshotFrameValues(gts)
			if g.renderDelay > 0 && now.Sub(renderStart) < g.renderDelay {
				continue
			}
			doneCount := len(gts) - remaining
			totalCount := len(gts)
			headerCandidate := false
			footerCandidate := false
			if headerGT != nil {
				g.header.callback(doneCount, totalCount, headerUpdate)
				if msg := *headerGT.msgPtr.Load(); msg != "" {
					headerCandidate = true
				}
			}
			if footerGT != nil {
				g.footer.callback(doneCount, totalCount, footerUpdate)
				if msg := *footerGT.msgPtr.Load(); msg != "" {
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
			maxLines := groupHeightCap(output.Height(), blockTopRow, len(gts)+candidateStatusLines)
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
				visible = prioritiseActive(visible, gts, done, maxTasks)
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
				visible = prioritiseActive(visible, gts, done, maxTasks)
			}
			if len(visible) == 0 {
				if g.transientHeader {
					showHeader = false
				}
				if g.transientFooter {
					showFooter = false
				}
			}
			frameBuf.Reset()
			layout := measureGroupRenderLayoutForIndexes(g, gts, effectiveDone, visible, now)
			lines, headerLineCount, taskLineCount, footerLineCount := buildGroupFrameLines(
				headerGT,
				footerGT,
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
					headerGT,
					footerGT,
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

// syncFrame brackets s with DEC 2026 synchronized-output markers so
// supporting terminals apply the whole sequence atomically (no tearing);
// terminals without support ignore the markers.
func syncFrame(s string) string {
	return xansi.EnableSyncOutput + s + xansi.DisableSyncOutput
}

// appendRepaint appends an overwrite-in-place repaint of lines over the
// previously rendered block of prevRows physical rows, bracketed by DEC 2026
// synchronized-output markers. The block is not erased up front - that would
// flash blank on terminals without synchronized output - instead each line
// clears only the rows it touches, and rows the new frame no longer covers
// are erased at the end. Returns the physical row count of the new frame.
func appendRepaint(buf *strings.Builder, lines []string, prevRows, width int) int {
	rows := groupFrameRows(lines, width)
	buf.WriteString(xansi.EnableSyncOutput)
	if prevRows > 0 {
		buf.WriteString(xansi.CursorUp(prevRows))
		buf.WriteString(xansi.CursorHorizontalAbsolute(1))
		if width <= 0 {
			// Wrap math is unreliable without a known width; fall back to
			// erasing the whole block before repainting.
			buf.WriteString(xansi.EraseScreenBelow)
		}
	}
	for i, line := range lines {
		if i > 0 {
			// Literal newline so the terminal advances (and scrolls when
			// the block reaches the viewport bottom). CursorNextLine would
			// clamp at the bottom row and silently fail to advance, leaving
			// the row arithmetic out of sync with reality. xansi.ClearLine
			// ends with "\r" so the column is reset before the next line is
			// written.
			buf.WriteString(nl)
		}
		buf.WriteString(xansi.ClearLine)
		buf.WriteString(line)
		// ClearLine only cleared the first physical row of a wrapped line;
		// intermediate rows are fully overwritten by the content itself,
		// but a partial final row keeps its stale tail. Trim it unless the
		// row is exactly full: the cursor then sits in the deferred-wrap
		// state, where EL0 would erase the last glyph instead.
		if w := xansi.StringWidth(line); width > 0 && w > width && w%width != 0 {
			buf.WriteString(xansi.EraseLineRight)
		}
	}
	// Park the cursor one line below the block only while a block is still
	// rendered, so zero-line frames don't leave a blank gap. Literal newline
	// for the same scroll-at-bottom reason as above.
	if len(lines) > 0 {
		buf.WriteString(nl)
	}
	// Rows the new frame no longer covers (a shrinking block, or a zero-line
	// frame replacing a previous block) are erased below the park position.
	// Steady-state frames skip the erase entirely.
	if width > 0 && rows != prevRows {
		buf.WriteString(xansi.CursorHorizontalAbsolute(1))
		buf.WriteString(xansi.EraseScreenBelow)
	}
	buf.WriteString(xansi.DisableSyncOutput)
	return rows
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
