package clog

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/internal/core"
	xansi "github.com/gechr/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAfterSetsDelay(t *testing.T) {
	b := Spinner("test").After(500 * time.Millisecond)
	assert.Equal(t, 500*time.Millisecond, b.Delay())
}

func TestAfterTaskFinishesBeforeDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	start := time.Now()
	result := Spinner("loading").
		After(1*time.Second).
		Wait(context.Background(), func(_ context.Context) error {
			// Task completes immediately, well before the 1s delay.
			return nil
		})

	require.NoError(t, result.TaskErr)
	// Should return almost immediately since the task finishes before the delay.
	assert.Less(t, time.Since(start), 500*time.Millisecond)
}

func TestAfterTaskFinishesAfterDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	result := Spinner("loading").
		After(10*time.Millisecond).
		Wait(context.Background(), func(_ context.Context) error {
			// Task takes longer than the delay, so animation would appear.
			time.Sleep(50 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.TaskErr)
}

func TestAfterTaskErrorBeforeDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	testErr := assert.AnError
	result := Spinner("loading").
		After(1*time.Second).
		Wait(context.Background(), func(_ context.Context) error {
			return testErr
		})

	require.ErrorIs(t, result.TaskErr, testErr)
}

func TestAfterContextCancelledDuringDelay(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	result := Spinner("loading").
		After(1*time.Second).
		Wait(ctx, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

	require.ErrorIs(t, result.TaskErr, context.Canceled)
}

func TestNonTTYSilentSetsField(t *testing.T) {
	b := Spinner("test").NonTTYSilent(true)
	assert.True(t, b.SuppressesNonTTY())

	b2 := Spinner("test").NonTTYSilent(false)
	assert.False(t, b2.SuppressesNonTTY())
}

func TestNonTTYSilentSuppressesOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf) // &bytes.Buffer has no Fd(), so isTTY = false

	result := l.Spinner("loading").
		NonTTYSilent(true).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond) // long enough to pass the delay gate
			return nil
		})

	require.NoError(t, result.Silent())
	assert.Empty(t, buf.String(), "NonTTYSilent(true) should produce no output on a non-TTY writer")
}

func TestNonTTYSilentFalseStillPrints(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)

	result := l.Spinner("loading").
		NonTTYSilent(false).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.Silent())
	assert.NotEmpty(
		t,
		buf.String(),
		"NonTTYSilent(false) should still print the static line on a non-TTY writer",
	)
}

func TestNonTTYLevelViaLogger(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetNonTTYLevel(LevelError)

	result := l.Spinner("loading").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(20 * time.Millisecond)
			return nil
		})

	require.NoError(t, result.Silent())
	assert.Empty(
		t,
		buf.String(),
		"SetNonTTYLevel(LevelError) should suppress info-level animation output",
	)
}

func TestNonTTYLevelSuppressesLogEvents(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetNonTTYLevel(LevelError)

	l.Info().Msg("should be suppressed")
	l.Warn().Msg("should be suppressed")
	assert.Empty(t, buf.String(), "Info and Warn should be suppressed below LevelError on non-TTY")

	l.Error().Msg("should appear")
	assert.NotEmpty(t, buf.String(), "Error should pass through at LevelError threshold")
}

func TestNonTTYLevelResetWithUnsetLevel(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.SetNonTTYLevel(LevelError)
	l.SetNonTTYLevel(UnsetLevel)

	l.Info().Msg("should appear after reset")
	assert.NotEmpty(t, buf.String(), "UnsetLevel restores normal non-TTY output")
}

func TestElapsedFieldOrdering(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *fx.Builder
		wantKeys []string
	}{
		{
			name: "elapsed between other fields",
			build: func() *fx.Builder {
				return Spinner("test").
					Str("a", "1").
					Elapsed("elapsed").
					Int("b", 2)
			},
			wantKeys: []string{"a", "elapsed", "b"},
		},
		{
			name: "elapsed first",
			build: func() *fx.Builder {
				return Spinner("test").
					Elapsed("timer").
					Str("x", "y")
			},
			wantKeys: []string{"timer", "x"},
		},
		{
			name: "elapsed last",
			build: func() *fx.Builder {
				return Spinner("test").
					Str("x", "y").
					Int("n", 1).
					Elapsed("dur")
			},
			wantKeys: []string{"x", "n", "dur"},
		},
		{
			name: "elapsed only",
			build: func() *fx.Builder {
				return Spinner("test").
					Elapsed("t")
			},
			wantKeys: []string{"t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := tt.build()
			require.Len(t, b.Fields, len(tt.wantKeys))
			for i, key := range tt.wantKeys {
				assert.Equal(t, key, b.Fields[i].Key)
			}
			// The elapsed placeholder must have the elapsed type.
			for _, f := range b.Fields {
				if f.Key == b.ElapsedFieldKey() {
					_, ok := f.Value.(core.ElapsedField)
					assert.True(t, ok, "elapsed field should have elapsed type, got %T", f.Value)
				}
			}
		})
	}
}

// regionBuffer is a goroutine-safe buffer for live-region tests whose render
// loop and animation goroutines write concurrently with test polling.
type regionBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *regionBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *regionBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// newRegionTestLogger returns a logger on a simulated 80-column TTY writing
// to buf, with parts reduced to the message so rendered animation lines are
// byte-predictable.
func newRegionTestLogger(buf *regionBuffer) (*Logger, *Output) {
	out := TestOutput(buf)
	out.isTTY = true
	out.widthDone = true
	out.width = 80
	l := New(out)
	l.SetParts(PartMessage)
	return l, out
}

func TestStandaloneAnimationDisplacesLogLines(t *testing.T) {
	var buf regionBuffer
	l, out := newRegionTestLogger(&buf)

	result := l.Spinner("loading").
		Wait(context.Background(), func(_ context.Context) error {
			// Wait until the animation has registered its live-region slot so
			// the log line below exercises the displacement path.
			for !out.LiveRegion().Active() {
				time.Sleep(time.Millisecond)
			}
			l.Info().Msg("inside")
			return nil
		})
	require.NoError(t, result.TaskErr)
	require.NoError(t, result.Msg("spin done"))

	got := buf.String()
	// The displaced log line is one synchronized frame: erase the block,
	// write the log line where the block's top row was, repaint the block
	// below it.
	displaced := xansi.EnableSyncOutput +
		xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
		"inside\n" +
		xansi.EnableSyncOutput + xansi.ClearLine + "loading"
	assert.Contains(t, got, displaced)
	// On completion the block is erased, the cursor restored, and the
	// completion message printed as a plain line.
	assert.Contains(t, got, xansi.ShowCursor+"spin done\n")
}

func TestConcurrentStandaloneAnimationsStackInOneBlock(t *testing.T) {
	var buf regionBuffer
	l, out := newRegionTestLogger(&buf)

	release := make(chan struct{})
	second := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		l.Spinner("first").Wait(context.Background(), func(_ context.Context) error {
			<-release
			return nil
		})
	}()
	go func() {
		defer wg.Done()
		<-second
		l.Spinner("stacked").Wait(context.Background(), func(_ context.Context) error {
			<-release
			return nil
		})
	}()

	// Start the second spinner only once the first is live so the block's
	// registration order is deterministic.
	for !out.LiveRegion().Active() {
		time.Sleep(time.Millisecond)
	}
	close(second)

	// Registering the second slot repaints both animations as one stacked
	// two-line block in registration order.
	stacked := xansi.ClearLine + "first\n" + xansi.ClearLine + "stacked"
	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), stacked)
	}, 2*time.Second, time.Millisecond)

	close(release)
	wg.Wait()

	// Both slots unregistered: the block is gone and the cursor restored.
	assert.False(t, out.LiveRegion().Active())
	assert.Contains(t, buf.String(), xansi.ShowCursor)
}

func TestGroupAnimationDisplacesLogLines(t *testing.T) {
	var buf regionBuffer
	l, out := newRegionTestLogger(&buf)

	release := make(chan struct{})
	g := l.Group(context.Background())
	g.Add(l.Spinner("grouped")).Run(func(_ context.Context) error {
		<-release
		return nil
	})

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		g.Wait()
	}()

	// Wait until the group's block holds the live region, then log through
	// the logger so the line takes the displacement path.
	for !out.LiveRegion().Active() {
		time.Sleep(time.Millisecond)
	}
	l.Info().Msg("inside")
	close(release)
	<-waitDone

	got := buf.String()
	// The displaced log line is one synchronized frame: erase the group's
	// block, write the log line where the block's top row was, repaint the
	// block below it.
	displaced := xansi.EnableSyncOutput +
		xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
		"inside\n" +
		xansi.EnableSyncOutput + xansi.ClearLine + "grouped"
	assert.Contains(t, got, displaced)
	// Group finished: the block is gone and the cursor restored.
	assert.False(t, out.LiveRegion().Active())
	assert.Contains(t, got, xansi.ShowCursor)
}

func TestTaskConfigFormatFieldsOmitEmpty(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetOmitEmpty(true)

	cfg := fxLogger{l}.TaskConfig(Spinner("test"))
	got := cfg.FormatFields([]core.Field{
		{Key: "name", Value: "alice"},
		{Key: "url", Value: ""},
		{Key: "age", Value: 0},
	})
	assert.Equal(t, " name=alice age=0", got)
}

func TestTaskConfigFormatFieldsOmitZero(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetOmitZero(true)

	cfg := fxLogger{l}.TaskConfig(Spinner("test"))
	got := cfg.FormatFields([]core.Field{
		{Key: "name", Value: "alice"},
		{Key: "url", Value: ""},
		{Key: "age", Value: 0},
	})
	assert.Equal(t, " name=alice", got)
}

func TestTaskConfigFormatFieldsNoOmit(t *testing.T) {
	l := NewWriter(io.Discard)

	cfg := fxLogger{l}.TaskConfig(Spinner("test"))
	got := cfg.FormatFields([]core.Field{
		{Key: "name", Value: "alice"},
		{Key: "url", Value: ""},
	})
	assert.Equal(t, " name=alice url=", got)
}
