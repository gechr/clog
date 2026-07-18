package fx

import (
	"strconv"
	"sync/atomic"
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
)

// overflowSymbol is the symbol-column marker for the overflow indicator line.
const overflowSymbol = "…"

// overflowMinTaskRows is the smallest task-row allotment that can host the
// indicator: one row for the indicator itself plus one task row to anchor it.
const overflowMinTaskRows = 2

// overflowMessage returns the overflow indicator's message for n hidden tasks.
func overflowMessage(n int) string {
	return strconv.Itoa(n) + " more"
}

// resolveOverflowText returns the group's configured indicator formatter,
// falling back to the default "N more" format.
func (g *Group) resolveOverflowText() OverflowIndicatorFunc {
	if g.overflowFunc != nil {
		return g.overflowFunc
	}
	return overflowMessage
}

// groupFramePlanInput carries the per-tick facts the frame planner needs: the
// group's configured caps, the candidate task rows, and which status lines
// want to render this tick.
type groupFramePlanInput struct {
	g   *Group
	gts []*renderTask
	// candidates holds the task indexes that pass the visibility filters
	// (Silent, hideDone, render delay), in registration order.
	candidates []int
	// done is the completion state used to prioritise active tasks when the
	// budget forces a selection.
	done            []bool
	headerCandidate bool // header has a non-empty message this tick
	footerCandidate bool // footer has a non-empty message this tick
	termHeight      int
	blockTopRow     int
}

// groupFramePlan is the planner's decision for one frame: which tasks render,
// whether the header/footer/overflow-indicator lines show, and the
// physical-row budget the rendered block must fit within.
type groupFramePlan struct {
	visible        []int
	candidateCount int
	hidden         int // candidates dropped by the row budget
	budget         int // physical-row budget for the whole block
	showHeader     bool
	showFooter     bool
	showOverflow   bool
}

// resolveRowBudget merges the automatic terminal-height cap with the group's
// explicit [WithMaxLines] / [WithMaxHeightPercent] caps; the smallest wins.
func resolveRowBudget(g *Group, termHeight, blockTopRow, fallback int) int {
	budget := groupHeightCap(termHeight, blockTopRow, fallback)
	if g.maxHeightPercent > 0 {
		pctLines := int(float64(termHeight) * g.maxHeightPercent)
		if pctLines > 0 && pctLines < budget {
			budget = pctLines
		}
	}
	if g.maxLines > 0 && g.maxLines < budget {
		budget = g.maxLines
	}
	return budget
}

// planGroupFrame allocates the frame's row budget in priority order:
// persistent status lines first (they render even without task rows), then
// task rows, then transient status lines (header before footer, and only
// alongside at least one task row), and finally the overflow indicator when
// candidates were dropped and a task row remains to anchor it.
func planGroupFrame(in groupFramePlanInput) groupFramePlan {
	g := in.g
	candidateStatus := 0
	if in.headerCandidate {
		candidateStatus++
	}
	if in.footerCandidate {
		candidateStatus++
	}
	budget := resolveRowBudget(g, in.termHeight, in.blockTopRow, len(in.gts)+candidateStatus)

	showHeader := in.headerCandidate && !g.transientHeader
	showFooter := in.footerCandidate && !g.transientFooter
	statusRows := 0
	if showHeader {
		statusRows++
	}
	if showFooter {
		statusRows++
	}
	// A transient status line consumes a row only when one is spare after
	// reserving at least one task row: tasks outrank transient status lines
	// when the budget is tight.
	if len(in.candidates) > 0 && budget-statusRows >= 1 {
		if in.headerCandidate && g.transientHeader && budget-statusRows >= 2 {
			showHeader = true
			statusRows++
		}
		if in.footerCandidate && g.transientFooter && budget-statusRows >= 2 {
			showFooter = true
			statusRows++
		}
	}

	maxTasks := max(0, budget-statusRows)
	// The indicator consumes a task row, so it is only worth reserving when
	// at least one task row remains to anchor it.
	showOverflow := g.overflowIndicator &&
		len(in.candidates) > maxTasks &&
		maxTasks >= overflowMinTaskRows
	if showOverflow {
		maxTasks--
	}
	visible := in.candidates
	if len(visible) > maxTasks {
		visible = prioritiseActive(visible, in.gts, in.done, maxTasks)
	}
	return groupFramePlan{
		visible:        visible,
		candidateCount: len(in.candidates),
		hidden:         len(in.candidates) - len(visible),
		budget:         budget,
		showHeader:     showHeader,
		showFooter:     showFooter,
		showOverflow:   showOverflow,
	}
}

// groupFrameSpec bundles the inputs [buildGroupFrameLines] assembles into one
// frame's rendered lines.
type groupFrameSpec struct {
	headerGT      *renderTask
	footerGT      *renderTask
	overflowGT    *renderTask
	overflowText  OverflowIndicatorFunc
	gts           []*renderTask
	visible       []int
	effectiveDone []bool
	layout        *groupRenderLayout
	hidden        int
	showHeader    bool
	showFooter    bool
	showOverflow  bool
}

// groupFrameLines is one assembled frame: the rendered lines plus per-section
// line counts for the wrap-trim loop and transient-status stripping.
type groupFrameLines struct {
	lines    []string
	header   int
	tasks    int
	overflow int
	footer   int
}

// buildGroupFrameLines renders the frame's lines in display order: header,
// task rows, overflow indicator, footer.
func buildGroupFrameLines(spec groupFrameSpec, now time.Time) groupFrameLines {
	lineCap := len(spec.visible) + 3 //nolint:mnd // header + overflow + footer
	built := groupFrameLines{lines: make([]string, 0, lineCap)}

	if spec.showHeader {
		built.lines = append(built.lines, renderTaskLine(spec.headerGT, false, now, spec.layout))
		built.header = 1
	}
	for _, taskIndex := range spec.visible {
		built.lines = append(built.lines, renderTaskLine(
			spec.gts[taskIndex], spec.effectiveDone[taskIndex], now, spec.layout,
		))
		built.tasks++
	}
	if spec.showOverflow && spec.hidden > 0 && spec.overflowGT != nil && built.tasks > 0 {
		msg := spec.overflowText(spec.hidden)
		spec.overflowGT.msgPtr.Store(&msg)
		built.lines = append(built.lines, renderTaskLine(spec.overflowGT, false, now, spec.layout))
		built.overflow = 1
	}
	if spec.showFooter {
		built.lines = append(built.lines, renderTaskLine(spec.footerGT, false, now, spec.layout))
		built.footer = 1
	}
	return built
}

// renderPlannedFrame lays out and renders the planned frame, then trims task
// rows from the tail until the block's physical rows (after wrapping at
// width) fit the plan's budget. Trimmed rows join the hidden count so the
// overflow indicator stays accurate.
func (st *groupLoopState) renderPlannedFrame(
	plan groupFramePlan, effectiveDone []bool, now time.Time, width int,
) []string {
	g := st.g
	spec := groupFrameSpec{
		headerGT:      st.headerGT,
		footerGT:      st.footerGT,
		overflowGT:    st.overflowGT,
		overflowText:  g.resolveOverflowText(),
		gts:           st.gts,
		visible:       plan.visible,
		effectiveDone: effectiveDone,
		hidden:        plan.hidden,
		showHeader:    plan.showHeader,
		showFooter:    plan.showFooter,
		showOverflow:  plan.showOverflow,
	}
	spec.layout = measureGroupRenderLayoutForIndexes(g, st.gts, effectiveDone, spec.visible, now)
	built := buildGroupFrameLines(spec, now)
	for groupFrameRows(built.lines, width) > plan.budget && built.tasks > 0 {
		spec.visible = spec.visible[:len(spec.visible)-1]
		spec.hidden = plan.candidateCount - len(spec.visible)
		spec.layout = measureGroupRenderLayoutForIndexes(
			g,
			st.gts,
			effectiveDone,
			spec.visible,
			now,
		)
		built = buildGroupFrameLines(spec, now)
	}
	return stripEmptyTaskStatus(g, built)
}

// stripEmptyTaskStatus removes transient header/footer lines from a frame
// whose task rows were all trimmed away, so the block never renders a
// transient-status-only frame.
func stripEmptyTaskStatus(g *Group, built groupFrameLines) []string {
	if built.tasks > 0 {
		return built.lines
	}
	lines := built.lines
	if built.footer == 1 && g.transientFooter {
		lines = lines[:len(lines)-1]
	}
	if built.header == 1 && g.transientHeader {
		lines = lines[1:]
	}
	return lines
}

// newOverflowTask builds the synthetic render task backing the overflow
// indicator line: a static "…" symbol with a per-frame message, styled
// through the group logger like a header/footer status line. The message
// style honors [WithOverflowStyle] via the builder's message override.
func newOverflowTask(g *Group, gts []*renderTask, syncEpoch time.Time) *renderTask {
	log := g.log
	if log == nil {
		log = gts[0].builder.log
	}
	b := NewBuilder(BuilderConfig{Logger: log, Level: level.Info})
	if g.overflowStyle != nil {
		b.MessageStyle(g.overflowStyle)
	}
	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	empty := ""
	msgPtr.Store(&empty)
	fieldsPtr.Store(&b.Fields)
	sym := overflowSymbol
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
	captureTaskConfig(gt)
	return gt
}
