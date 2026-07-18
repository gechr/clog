package fx

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
	xansi "github.com/gechr/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lockedWriter is a goroutine-safe buffer: the group loop goroutine, the
// region's render loop, and the test goroutine write and poll concurrently.
type lockedWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *lockedWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

// regionStubOutput is a stubOutput that exposes a shared [core.LiveRegion],
// driving the render loops down the live-region path the way the root clog
// Output does.
type regionStubOutput struct {
	stubOutput

	region *core.LiveRegion
}

func (o *regionStubOutput) LiveRegion() *core.LiveRegion { return o.region }

// regionStubLogger wraps stubLogger so TaskConfig hands the render loops the
// region-capable output instead of the bare stub.
type regionStubLogger struct {
	*stubLogger

	regionOut *regionStubOutput
}

func (s *regionStubLogger) Output() Output { return s.regionOut }

func (s *regionStubLogger) TaskConfig(b *Builder) TaskConfig {
	cfg := s.stubLogger.TaskConfig(b)
	cfg.Output = s.regionOut
	return cfg
}

// newRegionStubLogger returns a logger on a simulated 80x24 TTY whose output
// exposes a live region writing to w.
func newRegionStubLogger(w io.Writer) (*regionStubLogger, *regionStubOutput) {
	o := &regionStubOutput{
		stubOutput: stubOutput{
			cursorOK:  true,
			cursorRow: 1,
			height:    24,
			tty:       true,
			w:         w,
			width:     80,
		},
	}
	o.region = core.NewLiveRegion(w, func() int { return o.width })
	return &regionStubLogger{
		stubLogger: &stubLogger{out: &o.stubOutput},
		regionOut:  o,
	}, o
}

// testStaticTask returns a builder whose rendered line never changes between
// ticks (no animated symbol, no dynamic fields), so live-region byte streams
// stay deterministic regardless of how many render ticks elapse.
func testStaticTask(log Logger, msg string) *Builder {
	return NewBuilder(BuilderConfig{
		Logger:        log,
		Level:         level.Info,
		Message:       msg,
		Mode:          AnimationNone,
		SpinnerConfig: spinner.DefaultConfig(),
	})
}

// gateTask returns a task that completes when release is closed.
func gateTask(release <-chan struct{}) TaskFunc {
	return func(context.Context) error {
		<-release
		return nil
	}
}

func TestRunGroupLoopRegionCoexistsWithStandaloneSlot(t *testing.T) {
	var buf lockedWriter
	log, out := newRegionStubLogger(&buf)

	// A standalone animation is already live on the region.
	standalone := out.region.Register(func(time.Time) string { return "standalone" }, time.Hour)

	release := make(chan struct{})
	g := NewGroup(context.Background(), log)
	g.Add(testStaticTask(log, "alpha")).Run(gateTask(release))
	g.Add(testStaticTask(log, "beta")).Run(gateTask(release))

	loopDone := make(chan error, 1)
	go func() { loopDone <- runGroupLoop(context.Background(), g) }()

	// The group's multi-row block joins the standalone slot in one stacked
	// repaint, in registration order.
	stacked := xansi.ClearLine + "standalone\n" +
		xansi.ClearLine + "INF ⏳ alpha\n" +
		xansi.ClearLine + "INF ⏳ beta"
	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), stacked)
	}, 2*time.Second, time.Millisecond)

	close(release)
	require.NoError(t, <-loopDone)

	// Group done: its rows leave the block and the standalone slot repaints
	// alone over the three former rows.
	// The full stream: hide the cursor, paint the standalone slot, join it with
	// the group's stacked block, then repaint the standalone alone once the
	// group's rows leave.
	full := xansi.HideCursor +
		xansi.EnableSyncOutput + xansi.ClearLine + "standalone\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput +
		xansi.EnableSyncOutput + xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) +
		stacked + "\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput +
		xansi.EnableSyncOutput + xansi.CursorUp(3) + xansi.CursorHorizontalAbsolute(1) +
		xansi.ClearLine + "standalone\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput
	assert.Equal(t, full, buf.String())
	assert.True(t, out.region.Active())

	out.region.Unregister(standalone)
	assert.False(t, out.region.Active())
}

func TestRunGroupLoopRegionLogLinesDisplaceBlock(t *testing.T) {
	var buf lockedWriter
	log, out := newRegionStubLogger(&buf)

	release := make(chan struct{})
	g := NewGroup(context.Background(), log)
	g.Add(testStaticTask(log, "alpha")).Run(gateTask(release))

	loopDone := make(chan error, 1)
	go func() { loopDone <- runGroupLoop(context.Background(), g) }()

	// Wait until the group's slot is live, then write a log line the way
	// Output.WriteLine routes it while animations are on screen.
	require.Eventually(t, func() bool {
		return out.region.Active()
	}, 2*time.Second, time.Millisecond)
	out.region.WriteLines("log line\n")

	// One synchronized frame, byte-ordered: erase the block, the log line
	// lands where the block's top row was, the block repaints below it.
	displaced := xansi.EnableSyncOutput +
		xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
		"log line\n" +
		xansi.EnableSyncOutput + xansi.ClearLine + "INF ⏳ alpha"
	// The full stream: hide the cursor, paint the block's initial frame, then
	// the synchronized log-line displacement and repaint.
	full := xansi.HideCursor +
		xansi.EnableSyncOutput + xansi.ClearLine + "INF ⏳ alpha\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput +
		displaced + "\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
		xansi.DisableSyncOutput + xansi.DisableSyncOutput
	assert.Equal(t, full, buf.String())

	close(release)
	require.NoError(t, <-loopDone)
	assert.False(t, out.region.Active())
}

func TestRunGroupLoopRegionTwoConcurrentGroups(t *testing.T) {
	var buf lockedWriter
	log, out := newRegionStubLogger(&buf)

	releaseA := make(chan struct{})
	releaseB := make(chan struct{})
	ga := NewGroup(context.Background(), log)
	ga.Add(testStaticTask(log, "a-one")).Run(gateTask(releaseA))
	ga.Add(testStaticTask(log, "a-two")).Run(gateTask(releaseA))
	gb := NewGroup(context.Background(), log)
	gb.Add(testStaticTask(log, "b-one")).Run(gateTask(releaseB))

	doneA := make(chan error, 1)
	go func() { doneA <- runGroupLoop(context.Background(), ga) }()

	// Start the second group only once the first is painted so the block's
	// slot order is deterministic.
	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), "a-one")
	}, 2*time.Second, time.Millisecond)
	doneB := make(chan error, 1)
	go func() { doneB <- runGroupLoop(context.Background(), gb) }()

	// Both groups render as one stacked block in registration order instead
	// of repainting over each other.
	stacked := xansi.ClearLine + "INF ⏳ a-one\n" +
		xansi.ClearLine + "INF ⏳ a-two\n" +
		xansi.ClearLine + "INF ⏳ b-one"
	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), stacked)
	}, 2*time.Second, time.Millisecond)

	// Finishing the first group removes its rows; the second keeps rendering.
	close(releaseA)
	require.NoError(t, <-doneA)
	// The full stream so far: group A's initial block, group B joining it as a
	// stacked block, then b-one repainting alone once group A's rows leave.
	throughGroupADone := xansi.HideCursor +
		xansi.EnableSyncOutput +
		xansi.ClearLine + "INF ⏳ a-one\n" +
		xansi.ClearLine + "INF ⏳ a-two\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput +
		xansi.EnableSyncOutput + xansi.CursorUp(2) + xansi.CursorHorizontalAbsolute(1) +
		stacked + "\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput +
		xansi.EnableSyncOutput + xansi.CursorUp(3) + xansi.CursorHorizontalAbsolute(1) +
		xansi.ClearLine + "INF ⏳ b-one\n" +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput
	assert.Equal(t, throughGroupADone, buf.String())
	assert.True(t, out.region.Active())

	close(releaseB)
	require.NoError(t, <-doneB)
	assert.False(t, out.region.Active())
	// Group B done: its final row leaves and the cursor is restored.
	full := throughGroupADone +
		xansi.EnableSyncOutput + xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) +
		xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput +
		xansi.ShowCursor
	assert.Equal(t, full, buf.String())
}

// runScriptedGroup runs a group of gated tasks to completion on log, waiting
// for waitFor to appear in buf before releasing the tasks, and returns the
// full byte stream the run produced.
func runScriptedGroup(
	t *testing.T,
	log Logger,
	buf *lockedWriter,
	waitFor string,
	builders ...*Builder,
) string {
	t.Helper()

	release := make(chan struct{})
	g := NewGroup(context.Background(), log)
	for _, b := range builders {
		g.Add(b).Run(gateTask(release))
	}

	loopDone := make(chan error, 1)
	go func() { loopDone <- runGroupLoop(context.Background(), g) }()

	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), waitFor)
	}, 2*time.Second, time.Millisecond)
	close(release)
	require.NoError(t, <-loopDone)
	return buf.String()
}

func TestRunGroupLoopRegionMatchesLegacyByteStreamWhenAlone(t *testing.T) {
	// The legacy renderer ends by erasing the block bottom-up (clearing the
	// park row first); the region erases it as a zero-line repaint from the
	// top. Both leave the screen blank with the cursor on the block's former
	// top row, so parity is asserted on everything before the erase.
	tails := func(rows int) (string, string) {
		var lt strings.Builder
		eraseBlockSync(&lt, rows)
		var rt strings.Builder
		appendRepaint(&rt, nil, rows, 80)
		return lt.String() + xansi.ShowCursor, rt.String() + xansi.ShowCursor
	}

	run := func(t *testing.T, region bool, builders func(log Logger) []*Builder) string {
		t.Helper()
		var buf lockedWriter
		var log Logger
		if region {
			log, _ = newRegionStubLogger(&buf)
		} else {
			log = &stubLogger{out: &stubOutput{
				cursorOK:  true,
				cursorRow: 1,
				height:    24,
				tty:       true,
				w:         &buf,
				width:     80,
			}}
		}
		return runScriptedGroup(t, log, &buf, "INF", builders(log)...)
	}

	t.Run("static tasks", func(t *testing.T) {
		builders := func(log Logger) []*Builder {
			return []*Builder{testStaticTask(log, "alpha"), testStaticTask(log, "beta")}
		}
		legacy := run(t, false, builders)
		region := run(t, true, builders)

		legacyTail, regionTail := tails(2)
		require.True(t, strings.HasSuffix(legacy, legacyTail))
		require.True(t, strings.HasSuffix(region, regionTail))
		assert.Equal(t,
			strings.TrimSuffix(legacy, legacyTail),
			strings.TrimSuffix(region, regionTail),
		)
	})

	t.Run("bar final frame", func(t *testing.T) {
		style := bar.Config{
			CapLeft:   "[",
			CapRight:  "]",
			CharEmpty: '-',
			CharFill:  '=',
			Separator: " ",
			Smoothing: bar.SmoothNone,
			Width:     10,
		}
		builders := func(log Logger) []*Builder {
			b := testBar(log, "fetch", 10, bar.WithConfig(style))
			b.barProgressPtr.Store(4)
			return []*Builder{b}
		}
		legacy := run(t, false, builders)
		region := run(t, true, builders)

		legacyTail, regionTail := tails(1)
		require.True(t, strings.HasSuffix(legacy, legacyTail))
		require.True(t, strings.HasSuffix(region, regionTail))
		assert.Equal(t,
			strings.TrimSuffix(legacy, legacyTail),
			strings.TrimSuffix(region, regionTail),
		)
		// The full region stream: the 40% frame, the forced 100% final flash,
		// then the block erase and cursor restore.
		pad := strings.Repeat(" ", 50)
		line40 := "INF ⏳ fetch" + pad + "[====------]  40%"
		line100 := "INF ⏳ fetch" + pad + "[==========] 100%"
		wantRegion := xansi.HideCursor +
			xansi.EnableSyncOutput + xansi.ClearLine + line40 + "\n" +
			xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow + xansi.DisableSyncOutput +
			xansi.EnableSyncOutput + xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) +
			xansi.ClearLine + line100 + "\n" + xansi.DisableSyncOutput +
			regionTail
		assert.Equal(t, wantRegion, region)
	})
}

func TestRunGroupLoopRegionCancel(t *testing.T) {
	t.Run("preserves block as scrollback by default", func(t *testing.T) {
		var buf lockedWriter
		log, out := newRegionStubLogger(&buf)

		ctx, cancel := context.WithCancel(context.Background())
		g := NewGroup(ctx, log)
		g.Add(testStaticTask(log, "alpha")).Run(gateTask(make(chan struct{})))

		loopDone := make(chan error, 1)
		go func() { loopDone <- runGroupLoop(ctx, g) }()
		require.Eventually(t, func() bool {
			return out.region.Active()
		}, 2*time.Second, time.Millisecond)

		cancel()
		require.ErrorIs(t, <-loopDone, context.Canceled)

		// The slot is gone but the last frame was re-written as permanent
		// lines after the erase, so it survives in scrollback.
		assert.False(t, out.region.Active())
		assert.True(t, strings.HasSuffix(buf.String(), xansi.ShowCursor+"INF ⏳ alpha\n"))
	})

	t.Run("clear on cancel leaves nothing behind", func(t *testing.T) {
		var buf lockedWriter
		log, out := newRegionStubLogger(&buf)

		ctx, cancel := context.WithCancel(context.Background())
		g := NewGroup(ctx, log, WithClearOnCancel())
		g.Add(testStaticTask(log, "alpha")).Run(gateTask(make(chan struct{})))

		loopDone := make(chan error, 1)
		go func() { loopDone <- runGroupLoop(ctx, g) }()
		require.Eventually(t, func() bool {
			return out.region.Active()
		}, 2*time.Second, time.Millisecond)

		cancel()
		require.ErrorIs(t, <-loopDone, context.Canceled)

		// The erase is the last thing written: no frozen frame follows it.
		assert.False(t, out.region.Active())
		assert.True(t, strings.HasSuffix(buf.String(), xansi.ShowCursor))
	})
}

func TestRunGroupLoopRegionNonTTYPassThrough(t *testing.T) {
	var buf lockedWriter
	log, out := newRegionStubLogger(&buf)
	out.tty = false

	g := NewGroup(context.Background(), log)
	g.Add(testStaticTask(log, "alpha")).Run(func(context.Context) error { return nil })
	require.NoError(t, runGroupLoop(context.Background(), g))

	// Non-TTY output is the plain initial line: no escape sequences and the
	// region is never touched.
	assert.False(t, out.region.Active())
	assert.Equal(t, "INF ⏳ alpha\n", buf.String())
}
