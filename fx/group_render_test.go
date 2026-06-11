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

	"github.com/gechr/clog/field/percent"
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
func (o *stubOutput) ColorsDisabled() bool                  { return true }
func (o *stubOutput) PathLink(path string, _, _ int) string { return path }
func (o *stubOutput) Hyperlink(_, text string) string       { return text }
func (o *stubOutput) CursorPosition() (int, bool)           { return o.cursorRow, o.cursorOK }
func (o *stubOutput) ListenResize() func()                  { return func() {} }
func (o *stubOutput) RefreshWidth()                         {}
func (o *stubOutput) RefreshHeight()                        {}

// stubLogger is a minimal Logger implementation whose TaskConfig mirrors a
// colorless root logger with default parts and an "INF" level label.
type stubLogger struct {
	animationInterval time.Duration
	out               *stubOutput
}

func (s *stubLogger) Done(DoneEvent) {}

func (s *stubLogger) WithIndent(int, []core.TreePos) Logger { return s }

func (s *stubLogger) Output() Output { return s.out }

func (s *stubLogger) TaskConfig(*Builder) TaskConfig {
	out := s.out
	if out == nil {
		out = &stubOutput{}
	}
	return TaskConfig{
		AnimationInterval: s.animationInterval,
		IsTTY:             out.tty,
		Label:             "INF",
		LevelSymbol:       "INF",
		Order:             testParts(),
		Out:               out.Writer(),
		Output:            out,
		TimeFormat:        time.Kitchen,
		TimeLocation:      time.UTC,
		FormatFields:      stubFormatFields,
		StyleLevel:        func(core.Level) string { return "INF" },
		StyleMessage:      func(msg string, _ core.Level) string { return msg },
		StyleSymbol:       func(symbol string, _ core.Level) string { return symbol },
		StyleTimestamp:    func(ts string) string { return ts },
	}
}

func testParts() []core.Part {
	return []core.Part{
		core.PartTimestamp,
		core.PartLevel,
		core.PartSymbol,
		core.PartMessage,
		core.PartFields,
	}
}

// stubFormatFields formats fields as "k=v" pairs with a leading space,
// mirroring root clog's colorless output for the value types these tests use.
func stubFormatFields(fields []core.Field) string {
	var b strings.Builder
	for _, f := range fields {
		b.WriteString(" ")
		b.WriteString(f.Key)
		b.WriteByte('=')
		switch v := f.Value.(type) {
		case core.Percent:
			fmt.Fprintf(&b, "%.0f%%", v.Value/percent.EffectiveMaximum(v)*100)
		case core.ElapsedField:
			b.WriteString(time.Duration(v).Truncate(time.Second).String())
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
	style := spinner.DefaultStyle()
	for _, o := range opts {
		o(&style)
	}
	return NewBuilder(BuilderConfig{
		AnimatedSymbol: true,
		Logger:         log,
		Level:          level.Info,
		Message:        msg,
		Mode:           AnimationNone,
		SpinnerStyle:   style,
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
		Logger:       log,
		Mode:         AnimationBar,
		Level:        level.Info,
		Message:      msg,
		BarStyle:     bar.ApplyOptions(opts),
		BarProgress:  progressPtr,
		BarTotal:     totalPtr,
		SpinnerStyle: spinner.DefaultStyle(),
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
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "delayed"
	fields := []core.Field{{Key: "stage", Value: "running"}}
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
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "delayed"
	fields := []core.Field{{Key: "stage", Value: "running"}}
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
}

func TestAnimationIntervalClampsTickRate(t *testing.T) {
	t.Run("bar clamped to 200ms", func(t *testing.T) {
		log := &stubLogger{out: &stubOutput{}, animationInterval: 200 * time.Millisecond}

		b := testBar(log, "downloading", 100)
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]core.Field]{}
		msg := "downloading"
		fields := []core.Field{}
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

		b := testSpinner(log, "loading", spinner.WithStyle(spinner.Style{
			Frames:   []string{".", "..", "..."},
			Interval: 17 * time.Millisecond,
		}))
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]core.Field]{}
		msg := "loading"
		fields := []core.Field{}
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
		fieldsPtr := &atomic.Pointer[[]core.Field]{}
		msg := "downloading"
		fields := []core.Field{}
		msgPtr.Store(&msg)
		fieldsPtr.Store(&fields)
		s := &renderTask{
			groupTask: &groupTask{builder: b, fieldsPtr: fieldsPtr, msgPtr: msgPtr},
		}
		captureTaskConfig(s)

		assert.Equal(t, bar.TickRate, s.tickRate)
	})
}

func TestClearBlock(t *testing.T) {
	var buf strings.Builder
	clearBlock(&buf, 0)
	assert.Empty(t, buf.String())

	buf.Reset()
	clearBlock(&buf, 1)
	out := buf.String()
	// Single line: no initial move-up, clear one line, then move up 1.
	assert.Equal(t, "\x1b[2K\r\n\x1b[A", out)

	buf.Reset()
	clearBlock(&buf, 2)
	out = buf.String()
	// Two lines: move up 1 (not 2), clear both, then move up 2.
	assert.Equal(t, "\x1b[A\x1b[2K\r\n\x1b[2K\r\n\x1b[2A", out)
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
	successBuilder.BarProgressPtr.Store(9)
	failedBuilder.BarProgressPtr.Store(9)

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
	assert.Equal(t, int64(10), successBuilder.BarProgressPtr.Load())
	assert.Equal(t, int64(9), failedBuilder.BarProgressPtr.Load())
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

func TestGroupBarLayoutRightPad(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("short", " 29%", "BAR", "ETA 10s", bar.PlaceRightPad)
	layout.observe("much longer message", "", "BAR", "", bar.PlaceRightPad)

	line1 := layout.format("short", " 29%", "BAR", "ETA 10s", " ", bar.PlaceRightPad, 40)
	line2 := layout.format("much longer message", "", "BAR", "", " ", bar.PlaceRightPad, 40)

	assert.Equal(t, strings.Index(line1, "BAR"), strings.Index(line2, "BAR"))
	// 1-column right-edge slack means rendered lines are at most tw-1.
	assert.Len(t, line1, 39)
	assert.Len(t, line2, 39)
}

func TestGroupBarLayoutRightPadFallsBackWhenTooNarrow(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("very long message", " 29%", "BAR", "ETA 10s", bar.PlaceRightPad)

	got := layout.format("very long message", " 29%", "BAR", "ETA 10s", " ", bar.PlaceRightPad, 10)
	want := bar.FormatLine("very long message", " 29% BAR ETA 10s", " ", bar.PlaceRightPad, 10)

	assert.Equal(t, want, got)
}

func TestGroupBarLayoutRightPadAlignsRightWidget(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("one", "  0%", "BAR", "ETA 1h13m", bar.PlaceRightPad)
	layout.observe("two", "  7%", "BAR", "ETA 34s", bar.PlaceRightPad)

	line1 := layout.format("one", "  0%", "BAR", "ETA 1h13m", " ", bar.PlaceRightPad, 40)
	line2 := layout.format("two", "  7%", "BAR", "ETA 34s", " ", bar.PlaceRightPad, 40)

	assert.Equal(t, strings.Index(line1, "BAR"), strings.Index(line2, "BAR"))
	assert.Equal(t, strings.Index(line1, "ETA"), strings.Index(line2, "ETA"))
}

func TestGroupBarLayoutAligned(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe("short", " 29%", "BAR", "", bar.PlaceAligned)
	layout.observe("much longer message", " 50%", "BAR", "", bar.PlaceAligned)

	line1 := layout.format("short", " 29%", "BAR", "", " ", bar.PlaceAligned, 80)
	line2 := layout.format("much longer message", " 50%", "BAR", "", " ", bar.PlaceAligned, 80)

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
	line := layout.format("longest msg", "", "BAR", "", " ", bar.PlaceAligned, 80)
	assert.Equal(t, "longest msg BAR", line)
}

func TestGroupBarLayoutAlignedCapsToTerminalWidth(t *testing.T) {
	layout := &groupBarLayout{}
	layout.observe(strings.Repeat("long ", 20), "", "BAR", "", bar.PlaceAligned)

	line := layout.format("short", "", "BAR", "", " ", bar.PlaceAligned, 20)

	assert.LessOrEqual(t, xansi.StringWidth(line), 19)
	assert.Contains(t, line, "BAR")
}

func TestGroupBarLayoutAlignedTruncatesOutlierParts(t *testing.T) {
	layout := &groupBarLayout{}
	parts := strings.Repeat("long ", 20)
	layout.observe(parts, "", "BAR", "", bar.PlaceAligned)

	line := layout.format(parts, "", "BAR", "", " ", bar.PlaceAligned, 20)

	assert.LessOrEqual(t, xansi.StringWidth(line), 19)
	assert.Contains(t, line, "BAR")
	assert.NotEqual(t, parts+" BAR", line)
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

	visibleMsg := visibleBuilder.Message
	hiddenMsg := hiddenBuilder.Message
	emptyFields := []core.Field{}
	symbol := "·"
	visibleMsgPtr := &atomic.Pointer[string]{}
	hiddenMsgPtr := &atomic.Pointer[string]{}
	visibleFieldsPtr := &atomic.Pointer[[]core.Field]{}
	hiddenFieldsPtr := &atomic.Pointer[[]core.Field]{}
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
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "queued"
	fields := []core.Field{{Key: "stage", Value: "queued"}}
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
	assert.Contains(t, line, "queued")
	assert.NotContains(t, line, "│")
}

func TestRenderTaskLineCoalescesTimingButKeepsProgressLive(t *testing.T) {
	log := newStubLogger()
	style := bar.Style{
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
		bar.WithStyle(style),
		bar.WithUpdateInterval(time.Second),
	)

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "repo"
	fields := []core.Field{{Key: "stage", Value: "receiving"}}
	symbol := "📡"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)
	b.BarProgressPtr.Store(10)

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
	updatedFields := []core.Field{{Key: "stage", Value: "resolving"}}
	updatedSymbol := "🧩"
	msgPtr.Store(&updatedMsg)
	fieldsPtr.Store(&updatedFields)
	symbolPtr.Store(&updatedSymbol)
	b.BarProgressPtr.Store(90)

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
	style := bar.Style{
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
		bar.WithStyle(style),
		bar.WithUpdateInterval(time.Second),
	).
		BarPercent("progress").
		Elapsed("elapsed")

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "repo"
	fields := []core.Field{
		{Key: "stage", Value: "receiving"},
		{Key: "progress", Value: core.Percent{}},
		{Key: "elapsed", Value: core.ElapsedField(0)},
	}
	symbol := "📡"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)
	b.BarProgressPtr.Store(10)

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

	b.BarProgressPtr.Store(90)
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

func TestRenderTaskLineMonotonic(t *testing.T) {
	log := newStubLogger()
	style := bar.Style{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Smoothing:   bar.SmoothNone,
		WidgetRight: widget.None(),
		Width:       10,
	}
	b := testBar(log, "repo", 100, bar.WithStyle(style))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "repo"
	fields := []core.Field{{Key: "stage", Value: "receiving"}}
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

	b.BarProgressPtr.Store(90)
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)

	b.BarProgressPtr.Store(80)
	secondAt := time.Unix(3, 0)
	secondLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, secondAt)
	second := renderTaskLine(gt, false, secondAt, secondLayout)

	assert.Equal(t, "INF 📡 repo stage=receiving [=========-]", first)
	assert.Equal(t, first, second)
}

func TestRenderTaskLineSmoothEase(t *testing.T) {
	log := newStubLogger()
	style := bar.Style{
		CapLeft:     "[",
		CapRight:    "]",
		CharEmpty:   '-',
		CharFill:    '=',
		Separator:   " ",
		Smoothing:   bar.SmoothEase,
		WidgetRight: widget.None(),
		Width:       10,
	}
	b := testBar(log, "task", 100, bar.WithStyle(style))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]core.Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "task"
	fields := []core.Field{}
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
	b.BarProgressPtr.Store(10)
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&Group{}, []*renderTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)
	assert.Equal(t, "INF ⏳ task [=---------]", first)

	// Jump to 90% - shortly after, smoothing should lag behind the target.
	b.BarProgressPtr.Store(90)
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
