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
	"github.com/gechr/clog/fx/spinner"
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
	assert.Equal(t, "\x1b[2K\r\n\x1b[1A", out)

	buf.Reset()
	clearBlock(&buf, 2)
	out = buf.String()
	// Two lines: move up 1 (not 2), clear both, then move up 2.
	assert.Equal(t, "\x1b[1A\x1b[2K\r\n\x1b[2K\r\n\x1b[2A", out)
}
