package fx

import (
	"context"
	"fmt"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStatic builds a task with a static symbol and message so rendered
// frames are byte-for-byte deterministic.
func testStatic(log Logger, msg string) *Builder {
	return NewBuilder(BuilderConfig{
		Logger:     log,
		Level:      level.Info,
		Message:    msg,
		Mode:       AnimationNone,
		SymbolIcon: "•",
	})
}

// newComposeState mirrors runGroupLoop's render-state construction so tests
// can drive composeFrame directly.
func newComposeState(g *Group, out *stubOutput) *groupLoopState {
	gts := make([]*renderTask, len(g.tasks))
	for i, ft := range g.tasks {
		gt := &renderTask{groupTask: ft, monotonic: g.monotonic}
		captureTaskConfig(gt)
		gts[i] = gt
	}
	var overflowGT *renderTask
	if g.overflowIndicator {
		overflowGT = newOverflowTask(g, gts, time.Time{})
	}
	return &groupLoopState{
		g:             g,
		gts:           gts,
		fxTasks:       g.tasks,
		overflowGT:    overflowGT,
		output:        out,
		blockTopRow:   1,
		renderStart:   time.Now(),
		done:          make([]bool, len(gts)),
		justCompleted: make([]bool, len(gts)),
		remaining:     len(gts),
	}
}

func TestResolveRowBudget(t *testing.T) {
	g := NewGroup(context.Background(), nil)
	assert.Equal(t, 19, resolveRowBudget(g, 20, 1, 99))
	assert.Equal(t, 99, resolveRowBudget(g, 0, 0, 99))

	g = NewGroup(context.Background(), nil, WithMaxHeightPercent(0.5))
	assert.Equal(t, 10, resolveRowBudget(g, 20, 1, 99))

	g = NewGroup(context.Background(), nil, WithMaxHeightPercent(0.5), WithMaxLines(5))
	assert.Equal(t, 5, resolveRowBudget(g, 20, 1, 99))
}

func TestPlanGroupFramePersistentStatusWithoutTasks(t *testing.T) {
	g := NewGroup(context.Background(), nil)

	plan := planGroupFrame(groupFramePlanInput{
		g:               g,
		gts:             nil,
		candidates:      nil,
		done:            nil,
		headerCandidate: true,
		termHeight:      24,
		blockTopRow:     1,
	})

	assert.True(t, plan.showHeader)
	assert.False(t, plan.showFooter)
	assert.False(t, plan.showOverflow)
	assert.Empty(t, plan.visible)
	assert.Zero(t, plan.hidden)
}

func TestPlanGroupFrameTransientStatusDemotedForTasks(t *testing.T) {
	log := newStubLogger()
	gts := make([]*renderTask, 5)
	done := make([]bool, 5)
	candidates := make([]int, 5)
	for i := range gts {
		gts[i] = &renderTask{groupTask: &groupTask{builder: testStatic(log, "t")}}
		candidates[i] = i
	}
	g := NewGroup(
		context.Background(), nil,
		WithMaxLines(2), WithTransientHeader(), WithTransientFooter(),
	)

	plan := planGroupFrame(groupFramePlanInput{
		g:               g,
		gts:             gts,
		candidates:      candidates,
		done:            done,
		headerCandidate: true,
		footerCandidate: true,
		termHeight:      24,
		blockTopRow:     1,
	})

	assert.True(t, plan.showHeader, "transient header outranks transient footer")
	assert.False(t, plan.showFooter, "no row left for the transient footer")
	assert.False(t, plan.showOverflow, "a single task row cannot anchor the indicator")
	assert.Equal(t, []int{0}, plan.visible)
	assert.Equal(t, 4, plan.hidden)
	assert.Equal(t, 2, plan.budget)
}

func TestPlanGroupFrameReservesOverflowRow(t *testing.T) {
	log := newStubLogger()
	gts := make([]*renderTask, 6)
	done := make([]bool, 6)
	candidates := make([]int, 6)
	for i := range gts {
		gts[i] = &renderTask{groupTask: &groupTask{builder: testStatic(log, "t")}}
		candidates[i] = i
	}
	g := NewGroup(context.Background(), nil, WithMaxLines(4), WithOverflowIndicator())

	plan := planGroupFrame(groupFramePlanInput{
		g:           g,
		gts:         gts,
		candidates:  candidates,
		done:        done,
		termHeight:  24,
		blockTopRow: 1,
	})

	assert.True(t, plan.showOverflow)
	assert.Equal(t, []int{0, 1, 2}, plan.visible)
	assert.Equal(t, 3, plan.hidden)
}

func TestPlanGroupFrameOverflowDisabledByDefault(t *testing.T) {
	log := newStubLogger()
	gts := make([]*renderTask, 6)
	done := make([]bool, 6)
	candidates := make([]int, 6)
	for i := range gts {
		gts[i] = &renderTask{groupTask: &groupTask{builder: testStatic(log, "t")}}
		candidates[i] = i
	}
	g := NewGroup(context.Background(), nil, WithMaxLines(4))

	plan := planGroupFrame(groupFramePlanInput{
		g:           g,
		gts:         gts,
		candidates:  candidates,
		done:        done,
		termHeight:  24,
		blockTopRow: 1,
	})

	assert.False(t, plan.showOverflow)
	assert.Equal(t, []int{0, 1, 2, 3}, plan.visible)
	assert.Equal(t, 2, plan.hidden)
}

func TestComposeFrameOverflowIndicator(t *testing.T) {
	out := &stubOutput{tty: true, width: 80, height: 6}
	log := &stubLogger{out: out}

	g := NewGroup(context.Background(), log, WithOverflowIndicator())
	for _, msg := range []string{"task-1", "task-2", "task-3", "task-4", "task-5", "task-6", "task-7", "task-8"} {
		g.Add(testStatic(log, msg))
	}
	st := newComposeState(g, out)

	lines, width, ok := st.composeFrame(time.Now())

	require.True(t, ok)
	assert.Equal(t, 80, width)
	assert.Equal(t, []string{
		"INF • task-1",
		"INF • task-2",
		"INF • task-3",
		"INF • task-4",
		"INF … 4 more",
	}, lines)
}

func TestComposeFrameOverflowDisabledByDefault(t *testing.T) {
	out := &stubOutput{tty: true, width: 80, height: 6}
	log := &stubLogger{out: out}

	g := NewGroup(context.Background(), log)
	for _, msg := range []string{"task-1", "task-2", "task-3", "task-4", "task-5", "task-6", "task-7", "task-8"} {
		g.Add(testStatic(log, msg))
	}
	st := newComposeState(g, out)

	lines, width, ok := st.composeFrame(time.Now())

	require.True(t, ok)
	assert.Equal(t, 80, width)
	assert.Equal(t, []string{
		"INF • task-1",
		"INF • task-2",
		"INF • task-3",
		"INF • task-4",
		"INF • task-5",
	}, lines)
}

func TestComposeFrameOverflowIndicatorNeedsAnchorRow(t *testing.T) {
	out := &stubOutput{tty: true, width: 80, height: 24}
	log := &stubLogger{out: out}

	g := NewGroup(context.Background(), log, WithMaxLines(1), WithOverflowIndicator())
	for _, msg := range []string{"task-1", "task-2", "task-3"} {
		g.Add(testStatic(log, msg))
	}
	st := newComposeState(g, out)

	lines, _, ok := st.composeFrame(time.Now())

	require.True(t, ok)
	assert.Equal(t, []string{"INF • task-1"}, lines)
}

func TestComposeFrameOverflowIndicatorCountsTrimmedRows(t *testing.T) {
	// Terminal narrow enough that every task line wraps onto two physical
	// rows: the wrap-trim loop must drop further tasks and fold them into
	// the indicator count.
	out := &stubOutput{tty: true, width: 6, height: 6}
	log := &stubLogger{out: out}

	g := NewGroup(context.Background(), log, WithOverflowIndicator())
	for _, msg := range []string{"task-1", "task-2", "task-3", "task-4", "task-5", "task-6", "task-7", "task-8"} {
		g.Add(testStatic(log, msg))
	}
	st := newComposeState(g, out)

	lines, _, ok := st.composeFrame(time.Now())

	require.True(t, ok)
	assert.Equal(t, []string{
		"INF • task-1",
		"INF … 7 more",
	}, lines)
}

func TestComposeFrameOverflowCustomText(t *testing.T) {
	out := &stubOutput{tty: true, width: 80, height: 6}
	log := &stubLogger{out: out}

	g := NewGroup(context.Background(), log, WithOverflowIndicator(
		WithOverflowText(func(hidden int) string {
			return fmt.Sprintf("+%d queued", hidden)
		}),
	))
	for _, msg := range []string{"task-1", "task-2", "task-3", "task-4", "task-5", "task-6", "task-7", "task-8"} {
		g.Add(testStatic(log, msg))
	}
	st := newComposeState(g, out)

	lines, _, ok := st.composeFrame(time.Now())

	require.True(t, ok)
	assert.Equal(t, []string{
		"INF • task-1",
		"INF • task-2",
		"INF • task-3",
		"INF • task-4",
		"INF … +4 queued",
	}, lines)
}

func TestNewOverflowTaskAppliesStyle(t *testing.T) {
	log := newStubLogger()
	style := new(lipgloss.NewStyle().Faint(true))
	g := NewGroup(context.Background(), log, WithOverflowIndicator(WithOverflowStyle(style)))
	g.Add(testStatic(log, "task"))
	gts := []*renderTask{{groupTask: g.tasks[0]}}
	captureTaskConfig(gts[0])

	gt := newOverflowTask(g, gts, time.Time{})

	assert.Same(t, style, gt.builder.MessageStyleOverride())
}
