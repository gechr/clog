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
		Progress(func(_ context.Context, p *ProgressUpdate) error {
			p.SetProgress(75)
			capturedProgress = p.progressPtr.Load()
			return nil
		})
	g.Wait()

	assert.Equal(t, int64(75), capturedProgress)
	require.NoError(t, r.Prefix("✅").Msg("download complete"))

	out := buf.String()
	assert.Contains(t, out, "download complete")
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
		Progress(func(_ context.Context, p *ProgressUpdate) error {
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
	assert.Contains(t, out, "non-tty task")

	require.NoError(t, r.Msg("done"))
}

func TestGroupEmptyWait(_ *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background())
	// Should return immediately without panicking.
	g.Wait()
}

func TestGroupSlotResultFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("fielded").Str("base", "val")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	// Add extra fields on the SlotResult.
	require.NoError(t, r.Str("extra", "field").Msg("done"))

	out := buf.String()
	assert.Contains(t, out, "base=val")
	assert.Contains(t, out, "extra=field")
	assert.Contains(t, out, "done")
}

func TestGroupSlotResultOnError(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	testErr := errors.New("boom")

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("will fail")).
		Run(func(_ context.Context) error {
			return testErr
		})
	g.Wait()

	err := r.OnErrorLevel(WarnLevel).OnErrorMessage("custom error msg").Send()
	require.ErrorIs(t, err, testErr)

	out := buf.String()
	assert.Contains(t, out, "custom error msg")
	assert.Contains(t, out, "boom")
}

func TestGroupSlotResultElapsed(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	logger.SetElapsedMinimum(0) // show all elapsed values

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

func TestGroupProgressUpdate(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	var lastMsg atomic.Value
	r := g.Add(logger.Spinner("updating")).
		Progress(func(_ context.Context, p *ProgressUpdate) error {
			p.Msg("step 1").Send()
			lastMsg.Store(*p.msgPtr.Load())
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

	g := NewGroup(context.Background())
	r := g.Add(Spinner("default")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.Msg("done"))
}

func TestGroupSlotResultOnSuccessLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))
	logger.SetLevel(DebugLevel)

	g := logger.Group(context.Background())
	r := g.Add(logger.Spinner("test")).
		Run(func(_ context.Context) error {
			return nil
		})
	g.Wait()

	require.NoError(t, r.OnSuccessLevel(DebugLevel).OnSuccessMessage("debug msg").Send())

	out := buf.String()
	assert.Contains(t, out, "debug msg")
}

func TestGroupSlotResultSilent(t *testing.T) {
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

	require.NoError(t, g.Wait().Prefix("✅").Msg("All done"))

	out := buf.String()
	assert.Contains(t, out, "All done")
	// Should be a single log line, not per-slot.
	assert.Equal(t, 1, strings.Count(out, "All done"))
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

	err := g.Wait().OnErrorLevel(WarnLevel).OnErrorMessage("custom").Send()
	require.ErrorIs(t, err, testErr)

	out := buf.String()
	assert.Contains(t, out, "custom")
	assert.Contains(t, out, "oops")
}

func TestGroupResultFields(t *testing.T) {
	var buf bytes.Buffer
	logger := New(TestOutput(&buf))

	g := logger.Group(context.Background())
	g.Add(logger.Spinner("task")).
		Run(func(_ context.Context) error { return nil })

	require.NoError(t, g.Wait().Str("total", "1").Prefix("✅").Msg("Done"))

	out := buf.String()
	assert.Contains(t, out, "total=1")
	assert.Contains(t, out, "Done")
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

func TestClearBlock(t *testing.T) {
	var buf strings.Builder
	clearBlock(&buf, 0)
	assert.Empty(t, buf.String())

	buf.Reset()
	clearBlock(&buf, 2)
	out := buf.String()
	// Should move up, clear lines, then move up again.
	assert.Contains(t, out, "\x1b[2A")
	assert.Contains(t, out, "\x1b[2K\r\n")
}
