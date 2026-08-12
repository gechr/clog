package fx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
	xansi "github.com/gechr/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubOutput is a minimal Output implementation for render tests.
type stubOutput struct {
	colorsOn  bool
	cursorOK  bool
	cursorRow int
	height    int
	tty       bool
	w         io.Writer
	width     int
}

func (o *stubOutput) IsTTY() bool { return o.tty }

func (o *stubOutput) Writer() io.Writer {
	if o.w != nil {
		return o.w
	}
	return io.Discard
}

func (o *stubOutput) Width() int                            { return o.width }
func (o *stubOutput) Height() int                           { return o.height }
func (o *stubOutput) ColorsDisabled() bool                  { return !o.colorsOn }
func (o *stubOutput) PathLink(path string, _, _ int) string { return path }
func (o *stubOutput) Hyperlink(_, text string) string       { return text }
func (o *stubOutput) CursorPosition() (int, bool)           { return o.cursorRow, o.cursorOK }
func (o *stubOutput) ListenResize() func()                  { return func() {} }
func (o *stubOutput) RefreshSize()                          {}

// stubLogger is a minimal Logger implementation whose TaskConfig mirrors a
// colorless root logger with default parts and an "INF" level label.
type stubLogger struct {
	animationInterval time.Duration
	out               *stubOutput
}

func (s *stubLogger) Done(DoneEvent) {}

func (s *stubLogger) WithIndent(int, []TreePos) Logger { return s }

func (s *stubLogger) Output() Output { return s.out }

func (s *stubLogger) TaskConfig(b *Builder) TaskConfig {
	out := s.out
	if out == nil {
		out = &stubOutput{}
	}
	return TaskConfig{
		AnimationInterval: s.animationInterval,
		IsTTY:             out.tty,
		NonTTYSilent:      b.SuppressesNonTTY(),
		Label:             "INF",
		LevelSymbol:       "INF",
		Order:             testParts(),
		Out:               out.Writer(),
		Output:            out,
		TimeFormat:        time.Kitchen,
		TimeLocation:      time.UTC,
		FormatFields:      stubFormatFields,
		StyleLevel:        func(level.Level) string { return "INF" },
		StyleMessage:      func(msg string, _ level.Level) string { return msg },
		StyleSymbol:       func(symbol string, _ level.Level) string { return symbol },
		StyleTimestamp:    func(ts string) string { return ts },
	}
}

func testParts() []Part {
	return []Part{
		PartTimestamp,
		PartLevel,
		PartSymbol,
		PartMessage,
		PartFields,
	}
}

// percentScale converts a 0–1 percent fraction to its 0–100 display value.
const percentScale = 100

// stubFormatFields formats fields as "k=v" pairs with a leading space,
// mirroring root clog's colorless output for the value types these tests use.
func stubFormatFields(fields []Field) string {
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(" ")
		b.WriteString(f.Key)
		b.WriteByte('=')
		switch v := f.Value.(type) {
		case Percent:
			fmt.Fprintf(&b, "%.0f%%", v.Value*percentScale)
		case core.ElapsedField:
			b.WriteString(v.Value.Truncate(time.Second).String())
		default:
			fmt.Fprint(&b, v)
		}
	}
	return b.String()
}

func newStubLogger() *stubLogger {
	return &stubLogger{out: &stubOutput{}}
}

// testSpinner mirrors root clog's Logger.Spinner builder construction.
func testSpinner(log Logger, msg string, opts ...spinner.Option) *Builder {
	style := spinner.DefaultConfig()
	for _, o := range opts {
		o(&style)
	}
	return NewBuilder(BuilderConfig{
		AnimatedSymbol: true,
		Logger:         log,
		Level:          level.Info,
		Message:        msg,
		Mode:           AnimationNone,
		SpinnerConfig:  style,
	})
}

// testBar mirrors root clog's Logger.Bar builder construction.
func testBar(log Logger, msg string, total int, opts ...bar.Option) *Builder {
	if total <= 0 {
		total = 1
	}
	progressPtr := new(atomic.Int64)
	totalPtr := new(atomic.Int64)
	totalPtr.Store(int64(total))
	return NewBuilder(BuilderConfig{
		Logger:        log,
		Mode:          AnimationBar,
		Level:         level.Info,
		Message:       msg,
		BarConfig:     bar.Apply(opts...),
		BarProgress:   progressPtr,
		BarTotal:      totalPtr,
		SpinnerConfig: spinner.DefaultConfig(),
	})
}

func TestGroupFieldAlignmentMessageAlignsFields(t *testing.T) {
	log := newStubLogger()
	g := NewGroup(context.Background(), log, WithFieldAlignment(FieldAlignmentMessage))
	g.Add(testSpinner(log, "short").Str("stage", "queued"))
	g.Add(testSpinner(log, "much longer repo").Str("stage", "queued"))

	gts := make([]*renderTask, len(g.tasks))
	for i, task := range g.tasks {
		gt := &renderTask{groupTask: task}
		captureTaskConfig(gt)
		gts[i] = gt
	}

	now := time.Unix(1, 0)
	layout := measureGroupRenderLayout(g, gts, []bool{true, true}, now)
	line1 := renderTaskLine(gts[0], true, now, layout)
	line2 := renderTaskLine(gts[1], true, now, layout)

	assert.Equal(t, strings.Index(line1, "stage="), strings.Index(line2, "stage="))
}

func TestShouldRenderTaskAfterDelay(t *testing.T) {
	log := newStubLogger()
	b := testSpinner(log, "delayed").After(time.Second)

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "delayed"
	fields := []Field{{Key: "stage", Value: "running"}}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			start:     time.Unix(0, 0),
			symbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	assert.False(t, shouldRenderTask(gt, false, time.Unix(0, int64(500*time.Millisecond))))
	assert.True(t, shouldRenderTask(gt, false, time.Unix(1, 0)))
	assert.True(t, gt.visible)
}

func TestShouldRenderTaskAfterDelaySkipsTaskDoneBeforeDelay(t *testing.T) {
	log := newStubLogger()
	b := testSpinner(log, "delayed").After(time.Second)

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "delayed"
	fields := []Field{{Key: "stage", Value: "running"}}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			start:     time.Unix(0, 0),
			symbolPtr: symbolPtr,
		},
	}
	gt.markFinished(time.Unix(0, int64(500*time.Millisecond)))
	captureTaskConfig(gt)

	assert.False(t, shouldRenderTask(gt, true, time.Unix(2, 0)))
	assert.False(t, gt.visible)
}

func TestBuildLine(t *testing.T) {
	order := testParts()

	t.Run("all parts", func(t *testing.T) {
		line := buildLine(order, true, "12:00:00", "INF", "ℹ️", "hello", "k=v")
		assert.Equal(t, "12:00:00 INF ℹ️ hello k=v", line)
	})

	t.Run("no timestamp", func(t *testing.T) {
		line := buildLine(order, false, "", "INF", "ℹ️", "hello", "k=v")
		assert.Equal(t, "INF ℹ️ hello k=v", line)
	})

	t.Run("empty fields", func(t *testing.T) {
		line := buildLine(order, false, "", "INF", "ℹ️", "hello", "")
		assert.Equal(t, "INF ℹ️ hello", line)
	})

	t.Run("empty message", func(t *testing.T) {
		line := buildLine(order, false, "", "INF", "ℹ️", "", "k=v")
		assert.Equal(t, "INF ℹ️ k=v", line)
	})

	t.Run("custom part", func(t *testing.T) {
		// Custom parts render on regular log lines, not on task rows.
		const custom core.Part = 100
		line := buildLine(
			append([]core.Part{custom}, order...),
			false,
			"",
			"INF",
			"ℹ️",
			"hello",
			"k=v",
		)
		assert.Equal(t, "INF ℹ️ hello k=v", line)
	})
}

func TestAnimationIntervalClampsTickRate(t *testing.T) {
	t.Run("bar clamped to 200ms", func(t *testing.T) {
		log := &stubLogger{out: &stubOutput{}, animationInterval: 200 * time.Millisecond}

		b := testBar(log, "downloading", 100)
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]Field]{}
		msg := "downloading"
		fields := []Field{}
		msgPtr.Store(&msg)
		fieldsPtr.Store(&fields)
		s := &renderTask{
			groupTask: &groupTask{builder: b, fieldsPtr: fieldsPtr, msgPtr: msgPtr},
		}
		captureTaskConfig(s)

		assert.Equal(t, 200*time.Millisecond, s.tickRate)
	})

	t.Run("spinner clamped to 200ms", func(t *testing.T) {
		log := &stubLogger{out: &stubOutput{}, animationInterval: 200 * time.Millisecond}

		b := testSpinner(log, "loading", spinner.WithConfig(spinner.Config{
			Frames:   []string{".", "..", "..."},
			Interval: 17 * time.Millisecond,
		}))
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]Field]{}
		msg := "loading"
		fields := []Field{}
		msgPtr.Store(&msg)
		fieldsPtr.Store(&fields)
		s := &renderTask{
			groupTask: &groupTask{builder: b, fieldsPtr: fieldsPtr, msgPtr: msgPtr},
		}
		captureTaskConfig(s)

		assert.Equal(t, 200*time.Millisecond, s.tickRate)
	})

	t.Run("zero interval leaves tick rate unchanged", func(t *testing.T) {
		log := newStubLogger() // no clamping

		b := testBar(log, "downloading", 100)
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]Field]{}
		msg := "downloading"
		fields := []Field{}
		msgPtr.Store(&msg)
		fieldsPtr.Store(&fields)
		s := &renderTask{
			groupTask: &groupTask{builder: b, fieldsPtr: fieldsPtr, msgPtr: msgPtr},
		}
		captureTaskConfig(s)

		assert.Equal(t, bar.TickRate, s.tickRate)
	})

	// The gradient tick rate is a target, not an override: the logger's
	// interval floor is applied last and still wins.
	t.Run("time-based gradient still clamped to 200ms", func(t *testing.T) {
		log := &stubLogger{
			out:               &stubOutput{colorsOn: true},
			animationInterval: 200 * time.Millisecond,
		}

		b := testSpinner(log, "loading",
			spinner.WithConfig(spinner.Config{
				Frames:   []string{".", "..", "..."},
				Interval: 17 * time.Millisecond,
			}),
			spinner.WithGradient(),
			spinner.WithGradientTiming(spinner.GradientTimeBased),
		)
		s := newSpinnerRenderTask(b)

		assert.Equal(t, 200*time.Millisecond, s.tickRate)
	})
}

func TestAppendRepaint(t *testing.T) {
	const width = 10

	repaint := func(lines []string, prevRows, w int) (string, int) {
		var buf strings.Builder
		rows := core.AppendRepaint(&buf, lines, prevRows, w)
		return buf.String(), rows
	}

	t.Run("first frame", func(t *testing.T) {
		got, rows := repaint([]string{"task"}, 0, width)
		// No block yet: clear the current line, paint, park, then erase
		// below (row count changed from 0 to 1).
		want := xansi.EnableSyncOutput +
			xansi.ClearLine + "task" + nl +
			xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
			xansi.DisableSyncOutput
		assert.Equal(t, want, got)
		assert.Equal(t, 1, rows)
	})

	t.Run("steady state has no erase", func(t *testing.T) {
		got, rows := repaint([]string{"one", "two"}, 2, width)
		want := xansi.EnableSyncOutput +
			xansi.CursorUp(2) + xansi.CursorHorizontalAbsolute(1) +
			xansi.ClearLine + "one" + nl +
			xansi.ClearLine + "two" + nl +
			xansi.DisableSyncOutput
		assert.Equal(t, want, got)
		assert.Equal(t, 2, rows)
	})

	t.Run("shrinking block erases below the park row", func(t *testing.T) {
		got, rows := repaint([]string{"one"}, 3, width)
		want := xansi.EnableSyncOutput +
			xansi.CursorUp(3) + xansi.CursorHorizontalAbsolute(1) +
			xansi.ClearLine + "one" + nl +
			xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
			xansi.DisableSyncOutput
		assert.Equal(t, want, got)
		assert.Equal(t, 1, rows)
	})

	t.Run("zero-line frame erases the previous block", func(t *testing.T) {
		got, rows := repaint(nil, 2, width)
		// No lines and no park newline: the cursor stays at the block top
		// and everything below it is erased.
		want := xansi.EnableSyncOutput +
			xansi.CursorUp(2) + xansi.CursorHorizontalAbsolute(1) +
			xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
			xansi.DisableSyncOutput
		assert.Equal(t, want, got)
		assert.Equal(t, 0, rows)
	})

	t.Run("partial final wrap row is trimmed with EL0", func(t *testing.T) {
		line := strings.Repeat("x", 2*width+5) // wraps to 3 rows, last one partial
		got, rows := repaint([]string{line}, 3, width)
		want := xansi.EnableSyncOutput +
			xansi.CursorUp(3) + xansi.CursorHorizontalAbsolute(1) +
			xansi.ClearLine + line + xansi.EraseLineRight + nl +
			xansi.DisableSyncOutput
		assert.Equal(t, want, got)
		assert.Equal(t, 3, rows)
	})

	t.Run("exactly full final row skips EL0", func(t *testing.T) {
		// The cursor sits in the deferred-wrap state after an exactly full
		// row; EL0 there would erase the last glyph. The row is fully
		// overwritten anyway, so nothing needs trimming.
		line := strings.Repeat("x", 2*width)
		got, rows := repaint([]string{line}, 2, width)
		want := xansi.EnableSyncOutput +
			xansi.CursorUp(2) + xansi.CursorHorizontalAbsolute(1) +
			xansi.ClearLine + line + nl +
			xansi.DisableSyncOutput
		assert.Equal(t, want, got)
		assert.Equal(t, 2, rows)
	})

	t.Run("unknown width falls back to upfront erase", func(t *testing.T) {
		got, rows := repaint([]string{"task"}, 2, 0)
		want := xansi.EnableSyncOutput +
			xansi.CursorUp(2) + xansi.CursorHorizontalAbsolute(1) +
			xansi.EraseScreenBelow +
			xansi.ClearLine + "task" + nl +
			xansi.DisableSyncOutput
		assert.Equal(t, want, got)
		assert.Equal(t, 1, rows)
	})
}

func TestPrioritiseActiveZeroLimit(t *testing.T) {
	log := newStubLogger()
	visible := []int{0, 1}
	done := []bool{false, false}
	gts := []*renderTask{
		{groupTask: &groupTask{builder: testSpinner(log, "one")}},
		{groupTask: &groupTask{builder: testSpinner(log, "two")}},
	}

	got := prioritiseActive(visible, gts, done, 0)

	assert.Empty(t, got)
}

func TestGroupHeightCap(t *testing.T) {
	assert.Equal(t, 14, groupHeightCap(24, 10, 99))
	assert.Equal(t, 23, groupHeightCap(24, 0, 99))
	assert.Equal(t, 4, groupHeightCap(5, 8, 99))
	assert.Equal(t, 99, groupHeightCap(0, 0, 99))
	assert.Equal(t, 1, groupHeightCap(24, 24, 99))
}

func TestGroupFrameRowsCountsWrappedLines(t *testing.T) {
	lines := []string{
		"short",
		"this line wraps across more than one terminal row",
	}

	assert.Equal(t, 4, groupFrameRows(lines, 20))
}

func TestGroupFrameFitsViewportRequiresParkingRow(t *testing.T) {
	out := &stubOutput{tty: true, height: 24, cursorOK: true}

	out.cursorRow = 24
	assert.False(t, groupFrameFitsViewport(out, 0, 1))

	out.cursorRow = 23
	assert.True(t, groupFrameFitsViewport(out, 0, 1))
	assert.False(t, groupFrameFitsViewport(out, 0, 2))

	out.cursorRow = 24
	assert.True(t, groupFrameFitsViewport(out, 1, 1))
	assert.False(t, groupFrameFitsViewport(out, 1, 2))
}

func TestDrainGroupCompletionsOnlyFinalFlashesSuccessfulBars(t *testing.T) {
	log := newStubLogger()
	successBuilder := testBar(log, "success", 10)
	failedBuilder := testBar(log, "failed", 10)
	successBuilder.cfg.BarProgress.Store(9)
	failedBuilder.cfg.BarProgress.Store(9)

	successTask := &groupTask{
		builder: successBuilder,
		doneErr: make(chan error, 1),
	}
	failedTask := &groupTask{
		builder: failedBuilder,
		doneErr: make(chan error, 1),
	}
	successTask.doneErr <- nil
	failedTask.doneErr <- errors.New("boom")

	done := []bool{false, false}
	justCompleted := []bool{false, false}
	remaining := drainGroupCompletions(
		[]*groupTask{successTask, failedTask},
		[]*renderTask{{groupTask: successTask}, {groupTask: failedTask}},
		done,
		justCompleted,
		2,
	)

	assert.Equal(t, 0, remaining)
	assert.Equal(t, []bool{true, true}, done)
	assert.Equal(t, []bool{true, false}, justCompleted)
	assert.Equal(t, int64(10), successBuilder.cfg.BarProgress.Load())
	assert.Equal(t, int64(9), failedBuilder.cfg.BarProgress.Load())
}

func TestRunGroupLoopTTYCancelPreservesReadyTaskError(t *testing.T) {
	out := &stubOutput{tty: true, width: 80, height: 24, cursorRow: 1, cursorOK: true}
	log := &stubLogger{out: out, animationInterval: time.Hour}

	ctx, cancel := context.WithCancel(context.Background())
	g := NewGroup(ctx, log)
	realErr := errors.New("real failure")
	g.Add(testSpinner(log, "failed", spinner.WithInterval(time.Hour)))
	g.Add(testSpinner(log, "gated", spinner.WithInterval(time.Hour)))
	g.tasks[0].doneErr <- realErr
	cancel()

	err := runGroupLoop(ctx, g)

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, g.tasks[0].err, realErr)
	require.ErrorIs(t, g.tasks[1].err, context.Canceled)
}

func TestRunGroupLoopNonTTYCancelPreservesSuccess(t *testing.T) {
	var buf strings.Builder
	log := &stubLogger{out: &stubOutput{w: &buf}} // tty: false

	ctx, cancel := context.WithCancel(context.Background())
	g := NewGroup(ctx, log)
	g.Add(testSpinner(log, "done"))
	g.Add(testSpinner(log, "pending"))
	g.tasks[0].doneErr <- nil // succeeded before cancellation
	cancel()

	err := runGroupLoop(ctx, g)

	require.ErrorIs(t, err, context.Canceled)
	require.NoError(t, g.tasks[0].err, "successful task must keep its nil error")
	require.ErrorIs(t, g.tasks[1].err, context.Canceled)
}

func TestGroupTaskDurationFrozenAtFinish(t *testing.T) {
	finished := &groupTask{}
	finished.markStarted(time.Unix(10, 0))
	finished.markFinished(time.Unix(15, 0))
	assert.Equal(t, 5*time.Second, finished.duration(time.Unix(100, 0)))

	running := &groupTask{}
	running.markStarted(time.Unix(10, 0))
	assert.Equal(t, 90*time.Second, running.duration(time.Unix(100, 0)))
}

func TestGroupProgressPanicMarksFinished(t *testing.T) {
	log := newStubLogger()
	g := NewGroup(context.Background(), log)
	entry := g.Add(testSpinner(log, "boom"))

	_ = entry.Run(func(context.Context) error { panic("kaboom") })

	err := <-entry.task.doneErr
	require.EqualError(t, err, "panic: kaboom")
	require.NotZero(t, entry.task.finishedAt.Load())
}

func TestRunGroupNonTTYSkipsNonTTYSilentTask(t *testing.T) {
	var buf strings.Builder
	log := &stubLogger{out: &stubOutput{w: &buf}} // tty: false

	ctx := context.Background()
	g := NewGroup(ctx, log)
	g.Add(testSpinner(log, "loud"))
	g.Add(testSpinner(log, "quiet").NonTTYSilent(true))
	g.tasks[0].doneErr <- nil
	g.tasks[1].doneErr <- nil

	require.NoError(t, runGroupLoop(ctx, g))

	// Only the normal task prints its static non-TTY line; the NonTTYSilent
	// task is fully suppressed, leaving no second line.
	require.Equal(t, "INF ⏳ loud\n", buf.String())
}

func formatGroupBar(
	layout *groupBarLayout,
	parts, leftText, rightText string,
	placement bar.Placement,
	termWidth int,
) string {
	return layout.formatWithTruncationMarker(
		parts,
		leftText,
		"BAR",
		rightText,
		" ",
		placement,
		termWidth,
		bar.DefaultTruncationMarker,
	)
}

func TestGroupBarLayoutRightPad(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("short", " 29%", "BAR", "ETA 10s", bar.PlaceRightPad)
	layout.observe("much longer message", "", "BAR", "", bar.PlaceRightPad)

	line1 := formatGroupBar(layout, "short", " 29%", "ETA 10s", bar.PlaceRightPad, 40)
	line2 := formatGroupBar(
		layout,
		"much longer message",
		"",
		"",
		bar.PlaceRightPad,
		40,
	)

	assert.Equal(t, strings.Index(line1, "BAR"), strings.Index(line2, "BAR"))
	// 1-column right-edge slack means rendered lines are at most tw-1.
	assert.Len(t, line1, 39)
	assert.Len(t, line2, 39)
}

func TestGroupBarLayoutPadClampsDriftedParts(t *testing.T) {
	// The message is not frame-snapshotted, so parts rendered can be wider
	// than the parts measured. The pad count must clamp to zero, not panic.
	drifted := strings.Repeat("x", 50)

	leftPad := &groupBarLayout{}
	leftPad.observe("short", "", "BAR", "", bar.PlaceLeftPad)
	assert.NotPanics(t, func() {
		formatGroupBar(leftPad, drifted, "", "", bar.PlaceLeftPad, 20)
	})

	rightPad := &groupBarLayout{}
	rightPad.observe("short", "", "BAR", "", bar.PlaceRightPad)
	assert.NotPanics(t, func() {
		formatGroupBar(rightPad, drifted, "", "", bar.PlaceRightPad, 20)
	})
}

func TestGroupBarLayoutRightPadFallsBackWhenTooNarrow(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("very long message", " 29%", "BAR", "ETA 10s", bar.PlaceRightPad)

	got := formatGroupBar(
		layout,
		"very long message",
		" 29%",
		"ETA 10s",
		bar.PlaceRightPad,
		10,
	)
	want := bar.FormatLine("very long message", " 29% BAR ETA 10s", " ", bar.PlaceRightPad, 10)

	assert.Equal(t, want, got)
}

func TestGroupBarLayoutRightPadAlignsRightWidget(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("one", "  0%", "BAR", "ETA 1h13m", bar.PlaceRightPad)
	layout.observe("two", "  7%", "BAR", "ETA 34s", bar.PlaceRightPad)

	line1 := formatGroupBar(layout, "one", "  0%", "ETA 1h13m", bar.PlaceRightPad, 40)
	line2 := formatGroupBar(layout, "two", "  7%", "ETA 34s", bar.PlaceRightPad, 40)

	assert.Equal(t, strings.Index(line1, "BAR"), strings.Index(line2, "BAR"))
	assert.Equal(t, strings.Index(line1, "ETA"), strings.Index(line2, "ETA"))
}

func TestGroupBarLayoutAligned(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("short", " 29%", "BAR", "", bar.PlaceAligned)
	layout.observe("much longer message", " 50%", "BAR", "", bar.PlaceAligned)

	line1 := formatGroupBar(layout, "short", " 29%", "", bar.PlaceAligned, 80)
	line2 := formatGroupBar(
		layout,
		"much longer message",
		" 50%",
		"",
		bar.PlaceAligned,
		80,
	)

	// Both bars should start at the same column.
	assert.Equal(t, strings.Index(line1, "BAR"), strings.Index(line2, "BAR"))
	// The shorter message should be padded.
	assert.Greater(t, len(line1), len("short BAR"))
}

func TestGroupBarLayoutAlignedNoPadForLongest(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("longest msg", "", "BAR", "", bar.PlaceAligned)
	layout.observe("short", "", "BAR", "", bar.PlaceAligned)

	// The longest message should not get extra padding.
	line := formatGroupBar(layout, "longest msg", "", "", bar.PlaceAligned, 80)
	assert.Equal(t, "longest msg BAR", line)
}

func TestGroupBarLayoutAlignedCapsToTerminalWidth(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe(strings.Repeat("long ", 20), "", "BAR", "", bar.PlaceAligned)

	line := formatGroupBar(layout, "short", "", "", bar.PlaceAligned, 20)

	assert.LessOrEqual(t, xansi.StringWidth(line), 19)
	assert.Equal(t, "short           BAR", line)
}

func TestGroupBarLayoutAlignedTruncatesOutlierParts(t *testing.T) {
	layout := &groupBarLayout{}
	parts := strings.Repeat("long ", 20)
	layout.observe(parts, "", "BAR", "", bar.PlaceAligned)

	line := formatGroupBar(layout, parts, "", "", bar.PlaceAligned, 20)

	assert.LessOrEqual(t, xansi.StringWidth(line), 19)
	//nolint:dupword // repeated "long" is truncated test data, not a typo
	assert.Equal(t, "long long long… BAR", line)
	assert.NotEqual(t, parts+" BAR", line)
}

func TestGroupBarLayoutAlignedHidesTinyTruncatedParts(t *testing.T) {
	layout := &groupBarLayout{}
	parts := strings.Repeat("long ", 20)
	layout.observe(parts, "", "BAR", "", bar.PlaceAligned)

	line := formatGroupBar(layout, parts, "", "", bar.PlaceAligned, 9)

	assert.Equal(t, "BAR", line)
}

func TestGroupBarLayoutAlignedTruncatesProgressWithoutMarker(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("message", "94%", "BARBARBAR", "", bar.PlaceAligned)

	line := layout.formatWithTruncationMarker(
		"message",
		"94%",
		"BARBARBAR",
		"",
		" ",
		bar.PlaceAligned,
		8,
		"~~~",
	)

	assert.LessOrEqual(t, xansi.StringWidth(line), 7)
	assert.Equal(t, "BARBARB", line)
	assert.NotEmpty(t, line)
}

func TestGroupBarLayoutAlignedKeepsWholeProgressWidgetWhenItFits(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("message", "94%", "BAR", "", bar.PlaceAligned)

	line := formatGroupBar(layout, "message", "94%", "", bar.PlaceAligned, 9)

	assert.Equal(t, "94% BAR", line)
}

func TestGroupBarLayoutAlignedDropsWholeProgressWidgetWhenItDoesNotFit(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("message", "94%", "BAR", "", bar.PlaceAligned)

	line := formatGroupBar(layout, "message", "94%", "", bar.PlaceAligned, 7)

	assert.Equal(t, "BAR", line)
}

func TestGroupBarLayoutAlignedDropsWholeRightProgressWidgetWhenItDoesNotFit(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("message", "", "BAR", "94%", bar.PlaceAligned)

	line := formatGroupBar(layout, "message", "", "94%", bar.PlaceAligned, 7)

	assert.Equal(t, "BAR", line)
}

func TestGroupBarLayoutMeasuresVisibleIndexesOnly(t *testing.T) {
	log := newStubLogger()
	styleOpts := []bar.Option{
		bar.WithPlacement(bar.PlaceAligned),
		bar.WithWidth(10),
		bar.WithWidgetLeft(widget.None()),
		bar.WithWidgetRight(widget.None()),
	}
	visibleBuilder := testBar(log, "short", 1, styleOpts...)
	hiddenBuilder := testBar(log, strings.Repeat("long ", 40), 1, styleOpts...)

	visibleMsg := visibleBuilder.cfg.Message
	hiddenMsg := hiddenBuilder.cfg.Message
	emptyFields := []Field{}
	symbol := "·"
	visibleMsgPtr := &atomic.Pointer[string]{}
	hiddenMsgPtr := &atomic.Pointer[string]{}
	visibleFieldsPtr := &atomic.Pointer[[]Field]{}
	hiddenFieldsPtr := &atomic.Pointer[[]Field]{}
	visibleSymbolPtr := &atomic.Pointer[string]{}
	hiddenSymbolPtr := &atomic.Pointer[string]{}
	visibleMsgPtr.Store(&visibleMsg)
	hiddenMsgPtr.Store(&hiddenMsg)
	visibleFieldsPtr.Store(&emptyFields)
	hiddenFieldsPtr.Store(&emptyFields)
	visibleSymbolPtr.Store(&symbol)
	hiddenSymbolPtr.Store(&symbol)

	visibleGT := &renderTask{
		groupTask: &groupTask{
			builder:   visibleBuilder,
			fieldsPtr: visibleFieldsPtr,
			msgPtr:    visibleMsgPtr,
			symbolPtr: visibleSymbolPtr,
		},
	}
	hiddenGT := &renderTask{
		groupTask: &groupTask{
			builder:   hiddenBuilder,
			fieldsPtr: hiddenFieldsPtr,
			msgPtr:    hiddenMsgPtr,
			symbolPtr: hiddenSymbolPtr,
		},
	}
	captureTaskConfig(visibleGT)
	captureTaskConfig(hiddenGT)

	gts := []*renderTask{visibleGT, hiddenGT}
	done := []bool{false, false}
	now := time.Unix(1, 0)
	allLayout := measureGroupRenderLayoutForIndexes(&Group{}, gts, done, []int{0, 1}, now)
	visibleLayout := measureGroupRenderLayoutForIndexes(&Group{}, gts, done, []int{0}, now)

	allLine := renderTaskLine(visibleGT, false, now, allLayout)
	visibleLine := renderTaskLine(visibleGT, false, now, visibleLayout)

	assert.Greater(t, xansi.StringWidth(allLine), 100)
	assert.Less(t, xansi.StringWidth(visibleLine), 40)
}

func TestBuildTaskBarPartsPendingHide(t *testing.T) {
	log := newStubLogger()
	b := testBar(log, "queued", 1, bar.WithPendingMode(bar.PendingHide))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "queued"
	fields := []Field{{Key: "stage", Value: "queued"}}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			symbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	line := renderTaskLine(gt, false, time.Now(), nil)
	assert.Equal(t, "INF ⏳ queued stage=queued", line)
}

func TestRenderTaskLineCoalescesTimingButKeepsProgressLive(t *testing.T) {
	log := newStubLogger()
	style := bar.Config{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Smoothing:   bar.SmoothNone,
		WidgetLeft:  widget.ETA(),
		WidgetRight: widget.Percent(),
		Width:       10,
	}
	b := testBar(
		log,
		"repo",
		100,
		bar.WithConfig(style),
		bar.WithUpdateInterval(time.Second),
	)

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "repo"
	fields := []Field{{Key: "stage", Value: "receiving"}}
	symbol := "📡"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)
	b.cfg.BarProgress.Store(10)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			start:     time.Unix(0, 0),
			symbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	g := &Group{}
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(g, []*renderTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)

	updatedMsg := "repo (updated)"
	updatedFields := []Field{{Key: "stage", Value: "resolving"}}
	updatedSymbol := "🧩"
	msgPtr.Store(&updatedMsg)
	fieldsPtr.Store(&updatedFields)
	symbolPtr.Store(&updatedSymbol)
	b.cfg.BarProgress.Store(90)

	secondAt := time.Unix(2, int64(500*time.Millisecond))
	secondLayout := measureGroupRenderLayout(g, []*renderTask{gt}, []bool{false}, secondAt)
	second := renderTaskLine(gt, false, secondAt, secondLayout)
	thirdAt := time.Unix(3, int64(100*time.Millisecond))
	thirdLayout := measureGroupRenderLayout(g, []*renderTask{gt}, []bool{false}, thirdAt)
	third := renderTaskLine(gt, false, thirdAt, thirdLayout)

	assert.Equal(t, "INF 📡 repo stage=receiving ETA 18s [=---------]  10%", first)
	assert.Equal(t, "INF 🧩 repo (updated) stage=resolving  ETA 2s [=========-]  90%", second)
	assert.Equal(t, "INF 🧩 repo (updated) stage=resolving  ETA 1s [=========-]  90%", third)
}

func TestRenderTaskLineCoalescesElapsedFieldButKeepsBarPercentLive(t *testing.T) {
	log := newStubLogger()
	style := bar.Config{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Smoothing:   bar.SmoothNone,
		WidgetRight: widget.None(),
		Width:       10,
	}
	b := testBar(
		log,
		"repo",
		100,
		bar.WithConfig(style),
		bar.WithUpdateInterval(time.Second),
	).
		BarPercent("progress").
		Elapsed("elapsed")

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "repo"
	fields := []Field{
		{Key: "stage", Value: "receiving"},
		{Key: "progress", Value: Percent{}},
		{Key: "elapsed", Value: core.ElapsedField{Value: 0}},
	}
	symbol := "📡"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)
	b.cfg.BarProgress.Store(10)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			start:     time.Unix(0, 0),
			symbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)

	b.cfg.BarProgress.Store(90)
	secondAt := time.Unix(2, int64(500*time.Millisecond))
	secondLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, secondAt)
	second := renderTaskLine(gt, false, secondAt, secondLayout)
	thirdAt := time.Unix(3, int64(100*time.Millisecond))
	thirdLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, thirdAt)
	third := renderTaskLine(gt, false, thirdAt, thirdLayout)

	assert.Equal(t, "INF 📡 repo stage=receiving progress=10% elapsed=2s [=---------]", first)
	assert.Equal(t, "INF 📡 repo stage=receiving progress=90% elapsed=2s [=========-]", second)
	assert.Equal(t, "INF 📡 repo stage=receiving progress=90% elapsed=3s [=========-]", third)
}

func TestRenderTaskLineDoneFreezesElapsed(t *testing.T) {
	log := newStubLogger()
	b := testSpinner(log, "task").Elapsed("elapsed")

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "task"
	fields := []Field{{Key: "elapsed", Value: core.ElapsedField{Value: 0}}}
	symbol := "✓"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			symbolPtr: symbolPtr,
		},
	}
	gt.markStarted(time.Unix(1, 0))
	gt.markFinished(time.Unix(3, 0))
	captureTaskConfig(gt)

	// Rendered long after the task finished (siblings still running): the
	// elapsed field stays frozen at the task's 2s runtime.
	line := renderTaskLine(gt, true, time.Unix(60, 0), nil)
	assert.Equal(t, "INF ✓ task elapsed=2s", line)
}

func TestRenderTaskLineMonotonic(t *testing.T) {
	log := newStubLogger()
	style := bar.Config{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Smoothing:   bar.SmoothNone,
		WidgetRight: widget.None(),
		Width:       10,
	}
	b := testBar(log, "repo", 100, bar.WithConfig(style))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "repo"
	fields := []Field{{Key: "stage", Value: "receiving"}}
	symbol := "📡"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			start:     time.Unix(0, 0),
			symbolPtr: symbolPtr,
		},
		monotonic: true,
	}
	captureTaskConfig(gt)

	b.cfg.BarProgress.Store(90)
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)

	b.cfg.BarProgress.Store(80)
	secondAt := time.Unix(3, 0)
	secondLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, secondAt)
	second := renderTaskLine(gt, false, secondAt, secondLayout)

	assert.Equal(t, "INF 📡 repo stage=receiving [=========-]", first)
	assert.Equal(t, first, second)
}

func TestRenderTaskLineSmoothEase(t *testing.T) {
	log := newStubLogger()
	style := bar.Config{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Smoothing:   bar.SmoothEase,
		WidgetRight: widget.None(),
		Width:       10,
	}
	b := testBar(log, "task", 100, bar.WithConfig(style))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "task"
	fields := []Field{}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			start:     time.Unix(0, 0),
			symbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	// First render at 10% - smoothing initializes to 10%.
	b.cfg.BarProgress.Store(10)
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)
	assert.Equal(t, "INF ⏳ task [=---------]", first)

	// Jump to 90% - shortly after, smoothing should lag behind the target.
	b.cfg.BarProgress.Store(90)
	shortAt := firstAt.Add(50 * time.Millisecond)
	shortLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, shortAt)
	smoothed := renderTaskLine(gt, false, shortAt, shortLayout)
	// Without smoothing this would be [=========-]; with smoothing it should be less.
	assert.NotEqual(t, "INF ⏳ task [=========-]", smoothed)

	// After enough time (~10τ = 2s), smoothing converges to the actual progress.
	convergedAt := firstAt.Add(2 * time.Second)
	convergedLayout := measureGroupRenderLayout(
		&Group{},
		[]*renderTask{gt},
		[]bool{false},
		convergedAt,
	)
	converged := renderTaskLine(gt, false, convergedAt, convergedLayout)
	assert.Equal(t, "INF ⏳ task [=========-]", converged)
}

// A status line's Update must honor SetLevel like a task row's: the synthetic
// task carries its own level override slot, initialized to unset so the
// builder's level applies until a callback overrides it.
func TestNewSyntheticTaskWiresLevelOverride(t *testing.T) {
	b := testSpinner(newStubLogger(), "status")
	gt := newSyntheticTask(b, time.Now())

	assert.NotNil(t, gt.levelPtr)
	assert.Equal(t, b.cfg.Level, gt.effectiveLevel())

	gt.levelPtr.Store(int64(level.Warn))
	assert.Equal(t, level.Warn, gt.effectiveLevel())
}

// A done task whose final message was blanked opts out of its done line - it
// must vanish from the rendered set instead of freezing a stale mid-state
// snapshot into the terminal history. A live task with an empty message still
// renders.
func TestVisibleTaskIndexesSkipsBlankedDoneTasks(t *testing.T) {
	log := newStubLogger()
	blanked := newSyntheticTask(testSpinner(log, "working"), time.Now())
	empty := ""
	blanked.msgPtr.Store(&empty)
	kept := newSyntheticTask(testSpinner(log, "finished"), time.Now())
	live := newSyntheticTask(testSpinner(log, ""), time.Now())

	got := visibleTaskIndexes(
		[]*renderTask{blanked, kept, live},
		[]bool{true, true, false},
		false,
		time.Now(),
	)

	assert.Equal(t, []int{1, 2}, got)
}

// newSpinnerRenderTask builds a started renderTask for b with stub pointers,
// then captures its task config.
func newSpinnerRenderTask(b *Builder) *renderTask {
	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "msg"
	fields := []Field{}
	symbol := "•"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)
	gt := &renderTask{
		groupTask: &groupTask{
			builder:   b,
			fieldsPtr: fieldsPtr,
			msgPtr:    msgPtr,
			start:     time.Unix(0, 0),
			symbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)
	return gt
}

// testGradientFrames is a 4-frame spinner config for gradient render tests.
func testGradientFrames() spinner.Config {
	return spinner.Config{
		Frames:   []string{"a", "b", "c", "d"},
		Interval: 100 * time.Millisecond,
	}
}

func TestResolveSymbolGradientFrameSynced(t *testing.T) {
	log := &stubLogger{out: &stubOutput{colorsOn: true}}
	b := testSpinner(log, "grad",
		spinner.WithConfig(testGradientFrames()),
		spinner.WithGradient(),
	)
	gt := newSpinnerRenderTask(b)
	require.NotNil(t, gt.symbolStyles)
	assert.Equal(t, 4, gt.symbolStyles.Len(), "LUT should have one style per frame")

	epoch := time.Unix(0, 0)
	frames := []string{"a", "b", "c", "d"}
	rendered := make([]string, len(frames))
	for i, want := range frames {
		got := resolveSymbol(gt, epoch.Add(time.Duration(i)*100*time.Millisecond))
		assert.Equal(t, want, xansi.Strip(got), "tick %d glyph", i)
		rendered[i] = got
	}
	assert.NotEqual(t, rendered[0], rendered[1], "color should change on every tick")
	assert.NotEqual(t, rendered[1], rendered[2], "color should change on every tick")
	// One full revolution wraps back to the first color.
	wrap := resolveSymbol(gt, epoch.Add(4*100*time.Millisecond))
	assert.Equal(t, rendered[0], wrap)
}

func TestResolveSymbolNoGradientUnchanged(t *testing.T) {
	log := &stubLogger{out: &stubOutput{colorsOn: true}}
	b := testSpinner(log, "plain", spinner.WithConfig(testGradientFrames()))
	gt := newSpinnerRenderTask(b)
	assert.Nil(t, gt.symbolStyles)

	got := resolveSymbol(gt, time.Unix(0, 0))
	assert.Equal(t, "a", got, "no gradient should use the level symbol style unchanged")
}

func TestResolveSymbolGradientSkippedWhenColorsDisabled(t *testing.T) {
	log := newStubLogger() // stubOutput zero value: colors disabled
	b := testSpinner(log, "grad",
		spinner.WithConfig(testGradientFrames()),
		spinner.WithGradient(),
	)
	gt := newSpinnerRenderTask(b)
	assert.Nil(t, gt.symbolStyles, "LUT should not be built when colors are disabled")

	got := resolveSymbol(gt, time.Unix(0, 0))
	assert.Equal(t, "a", got)
}

func TestResolveSymbolGradientBypassedAfterSetSymbol(t *testing.T) {
	log := &stubLogger{out: &stubOutput{colorsOn: true}}
	b := testSpinner(log, "grad",
		spinner.WithConfig(testGradientFrames()),
		spinner.WithGradient(),
	)
	gt := newSpinnerRenderTask(b)
	require.NotNil(t, gt.symbolStyles)

	done := "✔"
	gt.symbolPtr.Store(&done)
	gt.symbolOverride.Store(true)

	got := resolveSymbol(gt, time.Unix(0, 0))
	assert.Equal(t, "✔", got, "symbol override should bypass the gradient")
}

func TestResolveSymbolGradientTimeBasedChangesWithinFrame(t *testing.T) {
	log := &stubLogger{out: &stubOutput{colorsOn: true}}
	cfg := testGradientFrames()
	cfg.Interval = 10 * time.Second // glyph frozen on frame 0 for this test
	b := testSpinner(log, "grad",
		spinner.WithConfig(cfg),
		spinner.WithGradient(),
		spinner.WithGradientTiming(spinner.GradientTimeBased),
		spinner.WithGradientSpeed(1.0),
	)
	gt := newSpinnerRenderTask(b)
	require.NotNil(t, gt.symbolStyles)

	epoch := time.Unix(0, 0)
	s0 := resolveSymbol(gt, epoch)
	s1 := resolveSymbol(gt, epoch.Add(250*time.Millisecond))
	assert.Equal(t, "a", xansi.Strip(s0))
	assert.Equal(t, "a", xansi.Strip(s1), "glyph should not advance within one interval")
	assert.NotEqual(t, s0, s1, "time-based gradient should recolor within one frame interval")
}

func TestGradientTickRateTimeBased(t *testing.T) {
	log := &stubLogger{out: &stubOutput{colorsOn: true}}
	b := testSpinner(log, "grad",
		spinner.WithConfig(testGradientFrames()),
		spinner.WithGradient(),
		spinner.WithGradientTiming(spinner.GradientTimeBased),
	)
	gt := newSpinnerRenderTask(b)
	assert.Equal(t, spinner.GradientTickRate, gt.tickRate,
		"time-based gradient should repaint at the gradient tick rate")
}

func TestGradientTickRateFrameSyncedUnchanged(t *testing.T) {
	log := &stubLogger{out: &stubOutput{colorsOn: true}}
	b := testSpinner(log, "grad",
		spinner.WithConfig(testGradientFrames()),
		spinner.WithGradient(),
	)
	gt := newSpinnerRenderTask(b)
	assert.Equal(t, 100*time.Millisecond, gt.tickRate,
		"frame-synced gradient should keep the spinner interval tick rate")
}

func TestGradientLUTSizedToBoomerangExpandedFrames(t *testing.T) {
	log := &stubLogger{out: &stubOutput{colorsOn: true}}
	cfg := spinner.Config{
		Boomerang: true,
		Frames:    []string{"a", "b", "c"},
		Interval:  100 * time.Millisecond,
	}
	b := testSpinner(log, "grad",
		spinner.WithConfig(cfg),
		spinner.WithGradient(),
	)
	gt := newSpinnerRenderTask(b)
	require.NotNil(t, gt.symbolStyles)
	// [a, b, c] expands to [a, b, c, b].
	assert.Equal(t, 4, gt.symbolStyles.Len())
}
