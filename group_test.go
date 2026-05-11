package clog

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	xansi "github.com/gechr/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupConcurrentRun(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	r1 := g.Add(logger.Spinner("task one")).
		Run(func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})
	r2 := g.Add(logger.Spinner("task two")).
		Run(func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})
	g.Wait()

	require.NoError(t, r1.Msg("one done"))
	require.NoError(t, r2.Msg("two done"))

	out := buf.String()
	assert.Contains(t, out, "one done")
	assert.Contains(t, out, "two done")
}

func TestGroupProgress(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	var capturedProgress int64
	r := g.Add(logger.Bar("downloading", 100)).
		Progress(func(_ context.Context, p *fx.Update) error {
			p.SetProgress(75)
			capturedProgress = p.ProgressPtr.Load()
			return nil
		})
	g.Wait()

	assert.Equal(t, int64(75), capturedProgress)
	require.NoError(t, r.Symbol("✅").Msg("download complete"))

	out := buf.String()
	assert.Equal(t, "INF ⏳ downloading\nINF ✅ download complete\n", out)
}

func TestGroupMixedAnimations(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	r1 := g.Add(logger.Spinner("spinning")).
		Run(func(_ context.Context) error {
			return nil
		})
	r2 := g.Add(logger.Bar("barring", 50)).
		Progress(func(_ context.Context, p *fx.Update) error {
			p.SetProgress(50).Send()
			return nil
		})
	r3 := g.Add(logger.Pulse("pulsing")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r1.Msg("spin done"))
	require.NoError(t, r2.Msg("bar done"))
	require.NoError(t, r3.Msg("pulse done"))

	out := buf.String()
	assert.Contains(t, out, "spin done")
	assert.Contains(t, out, "bar done")
	assert.Contains(t, out, "pulse done")
}

func TestGroupFieldAlignmentOption(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithFieldAlignment(FieldAlignmentMessage))

	assert.Equal(t, FieldAlignmentMessage, g.FieldAlignment)
}

func TestGroupParallelismOption(t *testing.T) {
	logger := NewWriter(io.Discard)

	g := logger.Group(context.Background(), WithParallelism(3))
	assert.Equal(t, 3, g.Parallelism)

	g = logger.Group(context.Background(), WithParallelism(0))
	assert.Zero(t, g.Parallelism)

	g = logger.Group(context.Background(), WithParallelism(-1))
	assert.Equal(t, -1, g.Parallelism)
}

func TestGroupHeaderOption(t *testing.T) {
	logger := NewWriter(io.Discard)
	cb := func(_, _ int, _ *Update) {}
	g := logger.Group(context.Background(), WithHeader(logger.Spinner("header"), cb))

	assert.NotNil(t, g.Header)
	assert.Equal(t, "header", g.Header.Builder.Message)
}

func TestGroupFooterOption(t *testing.T) {
	logger := NewWriter(io.Discard)
	cb := func(_, _ int, _ *Update) {}
	g := logger.Group(context.Background(), WithFooter(logger.Spinner("footer"), cb))

	assert.NotNil(t, g.Footer)
	assert.Equal(t, "footer", g.Footer.Builder.Message)
}

func TestGroupTransientStatusOptions(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithTransientHeader(), WithTransientFooter())

	assert.True(t, g.TransientHeader)
	assert.True(t, g.TransientFooter)
}

func TestGroupRenderDelayOption(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithRenderDelay(250*time.Millisecond))

	assert.Equal(t, 250*time.Millisecond, g.RenderDelay)
}

func TestGroupRenderDelaySkipsShortLivedTTYGroup(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 80
	out.heightDone = true
	out.height = 24
	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 1, column: 1}, true
	}
	logger := New(out)
	logger.SetAnimationInterval(time.Millisecond)

	g := logger.Group(context.Background(), WithRenderDelay(time.Second))
	g.Add(logger.Spinner("quick")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	assert.Empty(t, buf.String())
}

func TestGroupTransientHeaderHidesWhenNoTaskRowsVisible(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 80
	out.heightDone = true
	out.height = 24
	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 1, column: 1}, true
	}
	logger := New(out)
	logger.SetAnimationInterval(time.Millisecond)

	g := logger.Group(
		context.Background(),
		WithTransientHeader(),
		WithHeader(logger.Spinner("header", spinner.WithInterval(time.Millisecond)),
			func(done, total int, update *Update) {
				update.Msg("header").Int("done", done).Int("total", total).Send()
			}),
	)
	g.Add(logger.Spinner("quick", spinner.WithInterval(time.Millisecond)).After(time.Second)).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	assert.NotContains(t, buf.String(), "header")
}

func TestGroupMaxHeightPercentOption(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithMaxHeightPercent(0.5))
	assert.InDelta(t, 0.5, g.MaxHeightPercent, 0.001)
}

func TestGroupMaxHeightPercentClamped(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithMaxHeightPercent(1.5))
	assert.InDelta(t, 1.0, g.MaxHeightPercent, 0.001)

	g = logger.Group(context.Background(), WithMaxHeightPercent(-0.5))
	assert.InDelta(t, 0.0, g.MaxHeightPercent, 0.001)
}

func TestGroupMaxLinesOption(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithMaxLines(10))

	assert.Equal(t, 10, g.MaxLines)
}

func TestGroupMonotonicOption(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithMonotonic())

	assert.True(t, g.Monotonic)
}

func TestGroupParallelismLimitsConcurrentTasks(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithParallelism(2))

	var active atomic.Int64
	var maxActive atomic.Int64
	started := make(chan struct{}, 3)
	release := make(chan struct{})
	results := make([]*fx.TaskResult, 0, 3)

	for range 3 {
		result := g.Add(logger.Spinner("task")).
			Run(func(_ context.Context) error {
				current := active.Add(1)
				for {
					maxSeen := maxActive.Load()
					if current <= maxSeen || maxActive.CompareAndSwap(maxSeen, current) {
						break
					}
				}
				started <- struct{}{}
				<-release
				active.Add(-1)
				return nil
			})
		results = append(results, result)
	}

	<-started
	<-started

	select {
	case <-started:
		t.Fatal("task started before a parallelism slot was released")
	case <-time.After(20 * time.Millisecond):
	}

	close(release)
	g.Wait()

	for _, result := range results {
		require.NoError(t, result.Silent())
	}
	assert.Equal(t, int64(2), maxActive.Load())
}

func TestGroupFieldAlignmentMessageAlignsFields(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), WithFieldAlignment(FieldAlignmentMessage))
	g.Add(logger.Spinner("short").Str("stage", "queued"))
	g.Add(logger.Spinner("much longer repo").Str("stage", "queued"))

	gts := make([]*groupTask, len(g.Tasks))
	for i, task := range g.Tasks {
		gt := &groupTask{GroupTask: task}
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
	logger := NewWriter(io.Discard)
	b := logger.Spinner("delayed").After(time.Second)

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "delayed"
	fields := []Field{{Key: "stage", Value: "running"}}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fieldsPtr,
			MsgPtr:    msgPtr,
			StartTime: time.Unix(0, 0),
			SymbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	assert.False(t, shouldRenderTask(gt, false, time.Unix(0, int64(500*time.Millisecond))))
	assert.True(t, shouldRenderTask(gt, false, time.Unix(1, 0)))
	assert.True(t, gt.visible)
}

func TestShouldRenderTaskAfterDelaySkipsTaskDoneBeforeDelay(t *testing.T) {
	logger := NewWriter(io.Discard)
	b := logger.Spinner("delayed").After(time.Second)

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "delayed"
	fields := []Field{{Key: "stage", Value: "running"}}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fieldsPtr,
			MsgPtr:    msgPtr,
			StartTime: time.Unix(0, 0),
			SymbolPtr: symbolPtr,
		},
	}
	gt.MarkFinished(time.Unix(0, int64(500*time.Millisecond)))
	captureTaskConfig(gt)

	assert.False(t, shouldRenderTask(gt, true, time.Unix(2, 0)))
	assert.False(t, gt.visible)
}

func TestGroupErrorCollection(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	testErr := errors.New("task failed")

	g := logger.Group(context.Background())
	r1 := g.Add(logger.Spinner("succeeder")).
		Run(func(_ context.Context) error {
			return nil
		})
	r2 := g.Add(logger.Spinner("failer")).
		Run(func(_ context.Context) error {
			return testErr
		})
	g.Wait()

	require.NoError(t, r1.Send())
	require.ErrorIs(t, r2.Send(), testErr)

	out := buf.String()
	// The error case should log at error level.
	assert.Contains(t, out, "task failed")
}

func TestGroupContextCancel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	ctx, cancel := context.WithCancel(context.Background())

	g := logger.Group(ctx)
	r1 := g.Add(logger.Spinner("blocker")).
		Run(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})
	r2 := g.Add(logger.Spinner("blocker2")).
		Run(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

	// Cancel after a short delay.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	g.Wait()

	require.ErrorIs(t, r1.Silent(), context.Canceled)
	require.ErrorIs(t, r2.Silent(), context.Canceled)
}

func TestGroupNonTTY(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("non-tty task")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	// Non-TTY: should have printed the initial line.
	out := buf.String()
	assert.Equal(t, "INF ⏳ non-tty task\n", out)

	require.NoError(t, r.Msg("done"))
}

func TestGroupNonTTYStripsDynamicFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	elapsed.SetMinimum(0)
	t.Cleanup(func() { elapsed.SetMinimum(time.Second) })

	g := logger.Group(context.Background())
	r := g.Add(logger.Bar("downloading", 100).
		Str("file", "release.tar.gz").
		BarPercent("progress").
		Elapsed("elapsed")).
		Progress(func(_ context.Context, p *fx.Update) error {
			p.SetProgress(50).Send()
			return nil
		})
	g.Wait()

	out := buf.String()
	assert.Equal(t, "INF ⏳ downloading file=release.tar.gz\n", out)
	// Dynamic fields should be stripped in non-TTY output.
	assert.NotContains(t, out, "progress=")
	assert.NotContains(t, out, "elapsed=")

	require.NoError(t, r.Msg("done"))
}

func TestGroupEmptyWait(_ *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background())
	// Should return immediately without panicking.
	g.Wait()
}

func TestGroupTaskResultFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("fielded").Str("base", "val")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	// Add extra fields on the TaskResult.
	require.NoError(t, r.Str("extra", "field").Msg("done"))

	out := buf.String()
	assert.Equal(t, "INF ⏳ fielded base=val\nINF ℹ️ done base=val extra=field\n", out)
}

func TestGroupTaskResultOnError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	testErr := errors.New("boom")

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("will fail")).
		Run(func(_ context.Context) error {
			return testErr
		})
	g.Wait()

	err := r.OnErrorLevel(LevelWarn).OnErrorMessage("custom error msg").Send()
	require.ErrorIs(t, err, testErr)

	out := buf.String()
	assert.Equal(t, "INF ⏳ will fail\nWRN ⚠️ custom error msg error=boom\n", out)
}

func TestGroupTaskResultElapsed(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	elapsed.SetMinimum(0) // show all elapsed values
	t.Cleanup(func() { elapsed.SetMinimum(time.Second) })

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("timed").Elapsed("elapsed")).
		Run(func(_ context.Context) error {
			time.Sleep(10 * time.Millisecond)
			return nil
		})
	g.Wait()

	require.NoError(t, r.Msg("done"))

	out := buf.String()
	assert.Contains(t, out, "elapsed=")
}

func TestGroupUpdate(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	var lastMsg atomic.Value
	r := g.Add(logger.Spinner("updating")).
		Progress(func(_ context.Context, p *fx.Update) error {
			p.Msg("step 1").Send()
			lastMsg.Store(*p.MsgPtr.Load())
			p.Msg("step 2").Str("key", "val").Send()
			return nil
		})
	g.Wait()

	loaded, ok := lastMsg.Load().(string)
	require.True(t, ok)
	assert.Equal(t, "step 1", loaded)
	require.NoError(t, r.Msg("updated"))
}

func TestGroupDefaultLogger(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()
	Default = NewWriter(io.Discard)

	g := Group(context.Background())
	r := g.Add(Spinner("default")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.Msg("done"))
}

func TestGroupTaskResultOnSuccessLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	logger.SetLevel(LevelDebug)

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("test")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.OnSuccessLevel(LevelDebug).OnSuccessMessage("debug msg").Send())

	out := buf.String()
	assert.Equal(t, "INF ⏳ test\nDBG 🐞 debug msg\n", out)
}

func TestGroupTaskResultSilent(t *testing.T) {
	logger := NewWriter(io.Discard)

	g := logger.Group(context.Background())
	testErr := errors.New("silent error")
	r := g.Add(logger.Spinner("test")).
		Run(func(_ context.Context) error {
			return testErr
		})
	g.Wait()

	// Silent should return the error without logging.
	assert.ErrorIs(t, r.Silent(), testErr)
}

func TestGroupResultMsg(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("task one")).
		Run(func(_ context.Context) error { return nil })
	g.Add(logger.Spinner("task two")).
		Run(func(_ context.Context) error { return nil })

	require.NoError(t, g.Wait().Symbol("✅").Msg("All done"))

	out := buf.String()
	assert.Equal(t, "INF ⏳ task one\nINF ⏳ task two\nINF ✅ All done\n", out)
}

func TestGroupResultError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	testErr := errors.New("boom")

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("ok")).
		Run(func(_ context.Context) error { return nil })
	g.Add(logger.Spinner("fail")).
		Run(func(_ context.Context) error { return testErr })

	err := g.Wait().Msg("Summary")
	require.ErrorIs(t, err, testErr)

	out := buf.String()
	assert.Contains(t, out, "boom")
}

func TestGroupResultSilent(t *testing.T) {
	logger := NewWriter(io.Discard)

	testErr := errors.New("silent boom")

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("fail")).
		Run(func(_ context.Context) error { return testErr })

	err := g.Wait().Silent()
	require.ErrorIs(t, err, testErr)
}

func TestGroupResultAllSucceed(t *testing.T) {
	logger := NewWriter(io.Discard)

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("a")).
		Run(func(_ context.Context) error { return nil })
	g.Add(logger.Spinner("b")).
		Run(func(_ context.Context) error { return nil })

	require.NoError(t, g.Wait().Silent())
}

func TestGroupResultOnError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	testErr := errors.New("oops")

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("fail")).
		Run(func(_ context.Context) error { return testErr })

	err := g.Wait().OnErrorLevel(LevelWarn).OnErrorMessage("custom").Send()
	require.ErrorIs(t, err, testErr)

	out := buf.String()
	assert.Equal(t, "INF ⏳ fail\nWRN ⚠️ custom error=oops\n", out)
}

func TestGroupResultFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("task")).
		Run(func(_ context.Context) error { return nil })

	require.NoError(t, g.Wait().Str("total", "1").Symbol("✅").Msg("Done"))

	out := buf.String()
	assert.Equal(t, "INF ⏳ task\nINF ✅ Done total=1\n", out)
}

func TestBuildLine(t *testing.T) {
	order := DefaultParts()

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
		logger := NewWriter(io.Discard)
		logger.SetAnimationInterval(200 * time.Millisecond)

		b := logger.Bar("downloading", 100)
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]Field]{}
		msg := "downloading"
		fields := []Field{}
		msgPtr.Store(&msg)
		fieldsPtr.Store(&fields)
		s := &groupTask{
			GroupTask: &fx.GroupTask{Builder: b, FieldsPtr: fieldsPtr, MsgPtr: msgPtr},
		}
		captureTaskConfig(s)

		assert.Equal(t, 200*time.Millisecond, s.tickRate)
	})

	t.Run("spinner clamped to 200ms", func(t *testing.T) {
		logger := NewWriter(io.Discard)
		logger.SetAnimationInterval(200 * time.Millisecond)

		b := logger.Spinner("loading", spinner.WithStyle(spinner.Style{
			Frames:   []string{".", "..", "..."},
			Interval: 17 * time.Millisecond,
		}))
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]Field]{}
		msg := "loading"
		fields := []Field{}
		msgPtr.Store(&msg)
		fieldsPtr.Store(&fields)
		s := &groupTask{
			GroupTask: &fx.GroupTask{Builder: b, FieldsPtr: fieldsPtr, MsgPtr: msgPtr},
		}
		captureTaskConfig(s)

		assert.Equal(t, 200*time.Millisecond, s.tickRate)
	})

	t.Run("zero interval leaves tick rate unchanged", func(t *testing.T) {
		logger := NewWriter(io.Discard)
		logger.SetAnimationInterval(0) // disable clamping

		b := logger.Bar("downloading", 100)
		msgPtr := &atomic.Pointer[string]{}
		fieldsPtr := &atomic.Pointer[[]Field]{}
		msg := "downloading"
		fields := []Field{}
		msgPtr.Store(&msg)
		fieldsPtr.Store(&fields)
		s := &groupTask{
			GroupTask: &fx.GroupTask{Builder: b, FieldsPtr: fieldsPtr, MsgPtr: msgPtr},
		}
		captureTaskConfig(s)

		assert.Equal(t, bar.TickRate, s.tickRate)
	})
}

func TestTaskResultParts(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("task").Parts(PartMessage)).
		Run(func(_ context.Context) error { return nil })
	g.Wait()

	// Non-TTY group prints initial lines; reset to isolate completion.
	buf.Reset()
	require.NoError(t, r.Msg("done"))

	assert.Equal(t, "done\n", buf.String())
}

func TestTaskResultPartsOverride(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("task").Parts(PartMessage, PartLevel)).
		Run(func(_ context.Context) error { return nil })
	g.Wait()

	buf.Reset()
	require.NoError(t, r.Parts(PartMessage).Msg("done"))

	assert.Equal(t, "done\n", buf.String())
}

func TestGroupResultParts(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("task")).
		Run(func(_ context.Context) error { return nil })

	// Non-TTY group prints initial lines; reset to isolate the summary.
	result := g.Wait()
	buf.Reset()
	require.NoError(t, result.Parts(PartMessage).Msg("All done"))

	assert.Equal(t, "All done\n", buf.String())
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
	logger := NewWriter(io.Discard)
	visible := []int{0, 1}
	done := []bool{false, false}
	gts := []*groupTask{
		{GroupTask: &fx.GroupTask{Builder: logger.Spinner("one")}},
		{GroupTask: &fx.GroupTask{Builder: logger.Spinner("two")}},
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
	out := TestOutput(io.Discard)
	out.isTTY = true
	out.heightDone = true
	out.height = 24

	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 24, column: 1}, true
	}
	assert.False(t, groupFrameFitsViewport(out, 0, 1))

	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 23, column: 1}, true
	}
	assert.True(t, groupFrameFitsViewport(out, 0, 1))
	assert.False(t, groupFrameFitsViewport(out, 0, 2))

	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 24, column: 1}, true
	}
	assert.True(t, groupFrameFitsViewport(out, 1, 1))
	assert.False(t, groupFrameFitsViewport(out, 1, 2))
}

func TestGroupSuppressesLiveFrameAtViewportBottom(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 80
	out.heightDone = true
	out.height = 24
	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 24, column: 1}, true
	}
	logger := New(out)
	logger.SetAnimationInterval(time.Millisecond)

	release := make(chan struct{})
	g := logger.Group(context.Background())
	g.Add(logger.Spinner("processing", spinner.WithInterval(time.Millisecond))).
		Progress(func(ctx context.Context, _ *Update) error {
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

	result := make(chan error, 1)
	go func() { result <- g.Wait().Silent() }()
	time.Sleep(20 * time.Millisecond)
	close(release)

	require.NoError(t, <-result)
	assert.Empty(t, buf.String())
}

func TestDrainGroupCompletionsOnlyFinalFlashesSuccessfulBars(t *testing.T) {
	logger := NewWriter(io.Discard)
	successBuilder := logger.Bar("success", 10)
	failedBuilder := logger.Bar("failed", 10)
	successBuilder.BarProgressPtr.Store(9)
	failedBuilder.BarProgressPtr.Store(9)

	successTask := &fx.GroupTask{
		Builder: successBuilder,
		DoneErr: make(chan error, 1),
	}
	failedTask := &fx.GroupTask{
		Builder: failedBuilder,
		DoneErr: make(chan error, 1),
	}
	successTask.DoneErr <- nil
	failedTask.DoneErr <- errors.New("boom")

	done := []bool{false, false}
	justCompleted := []bool{false, false}
	remaining := drainGroupCompletions(
		[]*fx.GroupTask{successTask, failedTask},
		[]*groupTask{{GroupTask: successTask}, {GroupTask: failedTask}},
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
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 80
	out.heightDone = true
	out.height = 24
	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 1, column: 1}, true
	}
	logger := New(out)
	logger.SetAnimationInterval(time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	g := logger.Group(ctx)
	realErr := errors.New("real failure")
	g.Add(logger.Spinner("failed", spinner.WithInterval(time.Hour)))
	g.Add(logger.Spinner("gated", spinner.WithInterval(time.Hour)))
	g.Tasks[0].DoneErr <- realErr
	cancel()

	err := runGroupLoop(ctx, g)

	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, g.Tasks[0].Err, realErr)
	require.ErrorIs(t, g.Tasks[1].Err, context.Canceled)
}

func TestGroupRepaintClearsWrappedRows(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 32
	out.heightDone = true
	out.height = 24
	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 1, column: 1}, true
	}
	logger := New(out)
	logger.SetAnimationInterval(time.Millisecond)

	release := make(chan struct{})
	g := logger.Group(context.Background(), WithHeader(
		logger.Spinner("overall", spinner.WithInterval(time.Millisecond)),
		func(_, _ int, update *Update) {
			update.Msg("overall progress").Str("count", "0/1").Send()
		},
	))
	g.Add(logger.Spinner("processing", spinner.WithInterval(time.Millisecond)).
		Str("detail", strings.Repeat("x", 64))).
		Progress(func(ctx context.Context, _ *Update) error {
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})

	result := make(chan error, 1)
	go func() { result <- g.Wait().Silent() }()
	time.Sleep(20 * time.Millisecond)
	close(release)

	require.NoError(t, <-result)
	got := buf.String()
	assert.Contains(t, got, xansi.CursorUp(5)+xansi.CursorHorizontalAbsolute(1))
	assert.NotContains(t, got, xansi.CursorUp(2)+xansi.CursorHorizontalAbsolute(1))
}

// TestGroupAdvancementUsesLineFeed guards against reintroducing
// xansi.CursorNextLine for inter-line or parking advancement. CSI E clamps
// at the viewport bottom and silently fails to scroll, leaving renderedRows
// out of sync with the cursor's true position. Group rendering must always
// use a literal newline so the terminal scrolls when the block reaches the
// last viewport row.
func TestGroupAdvancementUsesLineFeed(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 80
	out.heightDone = true
	out.height = 24
	out.queryCursorPosition = func(io.Writer) (cursorPosition, bool) {
		return cursorPosition{row: 1, column: 1}, true
	}
	logger := New(out)
	logger.SetAnimationInterval(time.Millisecond)

	release := make(chan struct{})
	g := logger.Group(context.Background())
	for range 3 {
		g.Add(logger.Spinner("task", spinner.WithInterval(time.Millisecond))).
			Progress(func(ctx context.Context, _ *Update) error {
				select {
				case <-release:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			})
	}

	result := make(chan error, 1)
	go func() { result <- g.Wait().Silent() }()
	time.Sleep(20 * time.Millisecond)
	close(release)

	require.NoError(t, <-result)
	got := buf.String()
	assert.NotContains(
		t,
		got,
		xansi.CursorNextLine(1),
		"group rendering must use \\n for advancement, never CSI E (CursorNextLine), "+
			"because CSI E clamps at the viewport bottom and breaks the cursor-up arithmetic",
	)
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
	logger := NewWriter(io.Discard)
	styleOpts := []bar.Option{
		bar.WithPlacement(bar.PlaceAligned),
		bar.WithWidth(10),
		bar.WithWidgetLeft(widget.None()),
		bar.WithWidgetRight(widget.None()),
	}
	visibleBuilder := logger.Bar("short", 1, styleOpts...)
	hiddenBuilder := logger.Bar(strings.Repeat("long ", 40), 1, styleOpts...)

	visibleMsg := visibleBuilder.Message
	hiddenMsg := hiddenBuilder.Message
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

	visibleGT := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   visibleBuilder,
			FieldsPtr: visibleFieldsPtr,
			MsgPtr:    visibleMsgPtr,
			SymbolPtr: visibleSymbolPtr,
		},
	}
	hiddenGT := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   hiddenBuilder,
			FieldsPtr: hiddenFieldsPtr,
			MsgPtr:    hiddenMsgPtr,
			SymbolPtr: hiddenSymbolPtr,
		},
	}
	captureTaskConfig(visibleGT)
	captureTaskConfig(hiddenGT)

	gts := []*groupTask{visibleGT, hiddenGT}
	done := []bool{false, false}
	now := time.Unix(1, 0)
	allLayout := measureGroupRenderLayoutForIndexes(&fx.Group{}, gts, done, []int{0, 1}, now)
	visibleLayout := measureGroupRenderLayoutForIndexes(&fx.Group{}, gts, done, []int{0}, now)

	allLine := renderTaskLine(visibleGT, false, now, allLayout)
	visibleLine := renderTaskLine(visibleGT, false, now, visibleLayout)

	assert.Greater(t, xansi.StringWidth(allLine), 100)
	assert.Less(t, xansi.StringWidth(visibleLine), 40)
}

func TestBuildTaskBarPartsPendingHide(t *testing.T) {
	logger := NewWriter(io.Discard)
	b := logger.Bar("queued", 1, bar.WithPendingMode(bar.PendingHide))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "queued"
	fields := []Field{{Key: "stage", Value: "queued"}}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fieldsPtr,
			MsgPtr:    msgPtr,
			SymbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	line := renderTaskLine(gt, false, time.Now(), nil)
	assert.Contains(t, line, "queued")
	assert.NotContains(t, line, "│")
}

func TestRenderTaskLineCoalescesTimingButKeepsProgressLive(t *testing.T) {
	logger := NewWriter(io.Discard)
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
	b := logger.Bar(
		"repo",
		100,
		bar.WithStyle(style),
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
	b.BarProgressPtr.Store(10)

	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fieldsPtr,
			MsgPtr:    msgPtr,
			StartTime: time.Unix(0, 0),
			SymbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	g := &fx.Group{}
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(g, []*groupTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)

	updatedMsg := "repo (updated)"
	updatedFields := []Field{{Key: "stage", Value: "resolving"}}
	updatedSymbol := "🧩"
	msgPtr.Store(&updatedMsg)
	fieldsPtr.Store(&updatedFields)
	symbolPtr.Store(&updatedSymbol)
	b.BarProgressPtr.Store(90)

	secondAt := time.Unix(2, int64(500*time.Millisecond))
	secondLayout := measureGroupRenderLayout(g, []*groupTask{gt}, []bool{false}, secondAt)
	second := renderTaskLine(gt, false, secondAt, secondLayout)
	thirdAt := time.Unix(3, int64(100*time.Millisecond))
	thirdLayout := measureGroupRenderLayout(g, []*groupTask{gt}, []bool{false}, thirdAt)
	third := renderTaskLine(gt, false, thirdAt, thirdLayout)

	assert.Equal(t, "INF 📡 repo stage=receiving ETA 18s [=---------]  10%", first)
	assert.Equal(t, "INF 🧩 repo (updated) stage=resolving  ETA 2s [=========-]  90%", second)
	assert.Equal(t, "INF 🧩 repo (updated) stage=resolving  ETA 1s [=========-]  90%", third)
}

func TestRenderTaskLineCoalescesElapsedFieldButKeepsBarPercentLive(t *testing.T) {
	logger := NewWriter(io.Discard)
	elapsed.SetMinimum(0)
	t.Cleanup(func() { elapsed.SetMinimum(time.Second) })

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
	b := logger.Bar(
		"repo",
		100,
		bar.WithStyle(style),
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
		{Key: "progress", Value: core.Percent{}},
		{Key: "elapsed", Value: core.ElapsedField(0)},
	}
	symbol := "📡"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)
	b.BarProgressPtr.Store(10)

	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fieldsPtr,
			MsgPtr:    msgPtr,
			StartTime: time.Unix(0, 0),
			SymbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&fx.Group{}, []*groupTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)

	b.BarProgressPtr.Store(90)
	secondAt := time.Unix(2, int64(500*time.Millisecond))
	secondLayout := measureGroupRenderLayout(&fx.Group{}, []*groupTask{gt}, []bool{false}, secondAt)
	second := renderTaskLine(gt, false, secondAt, secondLayout)
	thirdAt := time.Unix(3, int64(100*time.Millisecond))
	thirdLayout := measureGroupRenderLayout(&fx.Group{}, []*groupTask{gt}, []bool{false}, thirdAt)
	third := renderTaskLine(gt, false, thirdAt, thirdLayout)

	assert.Equal(t, "INF 📡 repo stage=receiving progress=10% elapsed=2s [=---------]", first)
	assert.Equal(t, "INF 📡 repo stage=receiving progress=90% elapsed=2s [=========-]", second)
	assert.Equal(t, "INF 📡 repo stage=receiving progress=90% elapsed=3s [=========-]", third)
}

func TestRenderTaskLineMonotonic(t *testing.T) {
	logger := NewWriter(io.Discard)
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
	b := logger.Bar("repo", 100, bar.WithStyle(style))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "repo"
	fields := []Field{{Key: "stage", Value: "receiving"}}
	symbol := "📡"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fieldsPtr,
			MsgPtr:    msgPtr,
			StartTime: time.Unix(0, 0),
			SymbolPtr: symbolPtr,
		},
		monotonic: true,
	}
	captureTaskConfig(gt)

	b.BarProgressPtr.Store(90)
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&fx.Group{}, []*groupTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)

	b.BarProgressPtr.Store(80)
	secondAt := time.Unix(3, 0)
	secondLayout := measureGroupRenderLayout(&fx.Group{}, []*groupTask{gt}, []bool{false}, secondAt)
	second := renderTaskLine(gt, false, secondAt, secondLayout)

	assert.Equal(t, "INF 📡 repo stage=receiving [=========-]", first)
	assert.Equal(t, first, second)
}

func TestRenderTaskLineSmoothEase(t *testing.T) {
	logger := NewWriter(io.Discard)
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
	b := logger.Bar("task", 100, bar.WithStyle(style))

	msgPtr := &atomic.Pointer[string]{}
	fieldsPtr := &atomic.Pointer[[]Field]{}
	symbolPtr := &atomic.Pointer[string]{}
	msg := "task"
	fields := []Field{}
	symbol := "⏳"
	msgPtr.Store(&msg)
	fieldsPtr.Store(&fields)
	symbolPtr.Store(&symbol)

	gt := &groupTask{
		GroupTask: &fx.GroupTask{
			Builder:   b,
			FieldsPtr: fieldsPtr,
			MsgPtr:    msgPtr,
			StartTime: time.Unix(0, 0),
			SymbolPtr: symbolPtr,
		},
	}
	captureTaskConfig(gt)

	// First render at 10% - smoothing initializes to 10%.
	b.BarProgressPtr.Store(10)
	firstAt := time.Unix(2, 0)
	firstLayout := measureGroupRenderLayout(&fx.Group{}, []*groupTask{gt}, []bool{false}, firstAt)
	first := renderTaskLine(gt, false, firstAt, firstLayout)
	assert.Equal(t, "INF ⏳ task [=---------]", first)

	// Jump to 90% - shortly after, smoothing should lag behind the target.
	b.BarProgressPtr.Store(90)
	shortAt := firstAt.Add(50 * time.Millisecond)
	shortLayout := measureGroupRenderLayout(&fx.Group{}, []*groupTask{gt}, []bool{false}, shortAt)
	smoothed := renderTaskLine(gt, false, shortAt, shortLayout)
	// Without smoothing this would be [=========-]; with smoothing it should be less.
	assert.NotEqual(t, "INF ⏳ task [=========-]", smoothed)

	// After enough time (~10τ = 2s), smoothing converges to the actual progress.
	convergedAt := firstAt.Add(2 * time.Second)
	convergedLayout := measureGroupRenderLayout(
		&fx.Group{},
		[]*groupTask{gt},
		[]bool{false},
		convergedAt,
	)
	converged := renderTaskLine(gt, false, convergedAt, convergedLayout)
	assert.Equal(t, "INF ⏳ task [=========-]", converged)
}
