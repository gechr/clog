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

	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/fx/spinner"
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
	var capturedProgress int
	r := g.Add(logger.Bar("downloading", 100)).
		Progress(func(_ context.Context, p *fx.Update) error {
			p.SetProgress(75)
			capturedProgress = p.Progress()
			return nil
		})
	g.Wait()

	assert.Equal(t, 75, capturedProgress)
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

	g := logger.Group(context.Background(), fx.WithRenderDelay(time.Second))
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
		fx.WithTransientHeader(),
		fx.WithHeader(logger.Spinner("header", spinner.WithInterval(time.Millisecond)),
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

func TestGroupParallelismLimitsConcurrentTasks(t *testing.T) {
	logger := NewWriter(io.Discard)
	g := logger.Group(context.Background(), fx.WithParallelism(2))

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
	formats := DefaultFieldFormats()
	formats.ElapsedMinimum = 0
	logger.SetFieldFormats(formats)

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
	formats := DefaultFieldFormats()
	formats.ElapsedMinimum = 0 // show all elapsed values
	logger.SetFieldFormats(formats)

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
			lastMsg.Store(p.Message())
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
	g := logger.Group(context.Background(), fx.WithHeader(
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

// TestGroupFrameSynchronizationAndSkip verifies that group repaints are
// bracketed in DEC 2026 synchronized-output markers, that identical frames
// are skipped instead of rewritten, and that steady-state repaints never
// erase the block up front (the erase-then-rewrite blank flash on terminals
// without synchronized output).
func TestGroupFrameSynchronizationAndSkip(t *testing.T) {
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
	// A single-frame spinner renders the same line every tick, so every
	// frame after the first must be skipped.
	g.Add(logger.Spinner("processing", spinner.WithConfig(spinner.Config{
		Frames:   []string{"·"},
		Interval: time.Millisecond,
	}))).Progress(func(ctx context.Context, _ *Update) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	result := make(chan error, 1)
	go func() { result <- g.Wait().Silent() }()
	time.Sleep(50 * time.Millisecond)
	close(release)

	require.NoError(t, <-result)
	got := buf.String()

	syncCount := strings.Count(got, xansi.EnableSyncOutput)
	assert.Positive(t, syncCount)
	assert.Equal(t, syncCount, strings.Count(got, xansi.DisableSyncOutput))

	// ~50 ticks elapsed but the frame never changed: only a handful of
	// writes may remain (initial frame, completion flash, final erase).
	assert.LessOrEqual(t, syncCount, 5,
		"identical frames must be skipped, not rewritten every tick")

	assert.NotContains(
		t,
		got,
		xansi.CursorUp(1)+xansi.CursorHorizontalAbsolute(1)+xansi.EraseScreenBelow,
		"repaints must overwrite in place, not erase the block up front",
	)
}

// TestAnimationFrameSynchronizationAndSkip is the single-animation
// equivalent of TestGroupFrameSynchronizationAndSkip.
func TestAnimationFrameSynchronizationAndSkip(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 80
	out.heightDone = true
	out.height = 24
	logger := New(out)
	logger.SetAnimationInterval(time.Millisecond)

	release := make(chan struct{})
	result := make(chan error, 1)
	go func() {
		result <- logger.Spinner("processing", spinner.WithConfig(spinner.Config{
			Frames:   []string{"·"},
			Interval: time.Millisecond,
		})).Wait(context.Background(), func(ctx context.Context) error {
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}).Silent()
	}()
	time.Sleep(50 * time.Millisecond)
	close(release)

	require.NoError(t, <-result)
	got := buf.String()

	syncCount := strings.Count(got, xansi.EnableSyncOutput)
	assert.Positive(t, syncCount)
	assert.Equal(t, syncCount, strings.Count(got, xansi.DisableSyncOutput))
	assert.LessOrEqual(t, syncCount, 5,
		"identical frames must be skipped, not rewritten every tick")

	// The completion erase is its own synchronized frame.
	assert.Contains(
		t,
		got,
		xansi.EnableSyncOutput+xansi.CursorUp(1)+xansi.EraseScreenBelow+xansi.DisableSyncOutput,
	)
}
