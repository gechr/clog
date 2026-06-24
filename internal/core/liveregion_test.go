package core

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	xansi "github.com/gechr/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// neverTick is a tick rate large enough that the shared render loop never
// fires during a test, so every write is driven deterministically by the
// test goroutine (Register/Unregister/RenderFrame/WriteLines all paint
// synchronously under the region lock).
const neverTick = time.Hour

const testWidth = 80

func newTestRegion(buf *bytes.Buffer) *LiveRegion {
	return NewLiveRegion(buf, func() int { return testWidth })
}

// staticSlot returns a render closure that always renders line.
func staticSlot(line string) func(time.Time) string {
	return func(time.Time) string { return line }
}

// repaint returns the exact byte sequence AppendRepaint emits for a
// non-wrapping block of lines painted over prevRows physical rows at
// testWidth. Mirrors the documented escape contract rather than calling
// AppendRepaint, so a regression in the production sequence fails the test.
func repaint(prevRows int, lines ...string) string {
	var out strings.Builder
	out.WriteString(xansi.EnableSyncOutput)
	if prevRows > 0 {
		out.WriteString(xansi.CursorUp(prevRows) + xansi.CursorHorizontalAbsolute(1))
	}
	for i, line := range lines {
		if i > 0 {
			out.WriteString("\n")
		}
		out.WriteString(xansi.ClearLine + line)
	}
	if len(lines) > 0 {
		out.WriteString("\n")
	}
	if len(lines) != prevRows {
		out.WriteString(xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow)
	}
	return out.String() + xansi.DisableSyncOutput
}

func TestLiveRegionRegisterUnregisterLifecycle(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	assert.False(t, r.Active())

	id := r.Register(staticSlot("spinning"), neverTick)

	// First registration hides the cursor and paints the slot immediately.
	assert.True(t, r.Active())
	assert.Equal(t, xansi.HideCursor+repaint(0, "spinning"), buf.String())

	buf.Reset()
	r.Unregister(id)

	// Last unregistration erases the block and shows the cursor again.
	assert.False(t, r.Active())
	assert.Equal(t, repaint(1)+xansi.ShowCursor, buf.String())
}

func TestLiveRegionWriteLinesPlainWhenInactive(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	r.WriteLines("plain log line\n")

	// No block on screen: bytes pass through untouched, with no escapes.
	assert.Equal(t, "plain log line\n", buf.String())
}

func TestLiveRegionWriteLinesDisplacesBlock(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	id := r.Register(staticSlot("spinning"), neverTick)
	buf.Reset()

	r.WriteLines("log line\n")

	// One synchronized frame: erase the block, write the log line where the
	// block's top row was, repaint the block below.
	want := xansi.EnableSyncOutput +
		xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
		"log line\n" +
		repaint(0, "spinning") +
		xansi.DisableSyncOutput
	assert.Equal(t, want, buf.String())

	// The log line must appear before the repainted animation block in the
	// byte stream so it ends up above the block on screen.
	out := buf.String()
	assert.Less(t, strings.Index(out, "log line"), strings.Index(out, "spinning"))

	r.Unregister(id)
}

func TestLiveRegionWriteLinesAddsMissingNewlineWhileLive(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	id := r.Register(staticSlot("spinning"), neverTick)
	buf.Reset()

	r.WriteLines("no newline")

	// The repaint must start on a fresh row, so a newline is appended before
	// the block is painted below the log line.
	want := xansi.EnableSyncOutput +
		xansi.CursorUp(1) + xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
		"no newline\n" +
		repaint(0, "spinning") +
		xansi.DisableSyncOutput
	assert.Equal(t, want, buf.String())

	r.Unregister(id)
}

func TestLiveRegionStacksSlotsInRegistrationOrder(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	first := r.Register(staticSlot("first"), neverTick)
	second := r.Register(staticSlot("second"), neverTick)
	third := r.Register(staticSlot("third"), neverTick)

	buf.Reset()
	r.RenderFrame(time.Now().Add(time.Second))

	// The dedup check skips the write because the block has not changed
	// since the paint triggered by the last Register.
	assert.Empty(t, buf.String())

	// Force a repaint by removing the middle slot: the remaining slots keep
	// registration order in a single three-rows-to-two-rows repaint.
	r.Unregister(second)
	assert.Equal(t, repaint(3, "first", "third"), buf.String())

	r.Unregister(first)
	r.Unregister(third)
	assert.False(t, r.Active())
}

func TestLiveRegionSkipsUnchangedFrames(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	frame := "frame-0"
	id := r.Register(func(time.Time) string { return frame }, neverTick)
	buf.Reset()

	// Identical render output: no bytes written.
	r.RenderFrame(time.Now())
	r.RenderFrame(time.Now())
	assert.Empty(t, buf.String())

	// Changed render output: exactly one in-place repaint.
	frame = "frame-1"
	r.RenderFrame(time.Now())
	assert.Equal(t, repaint(1, "frame-1"), buf.String())

	r.Unregister(id)
}

func TestLiveRegionRepaintsOnWidthChange(t *testing.T) {
	var buf bytes.Buffer
	width := testWidth
	r := NewLiveRegion(&buf, func() int { return width })

	id := r.Register(staticSlot("steady"), neverTick)
	buf.Reset()

	// Same content but a new width: row accounting may differ, so the block
	// is repainted even though the rendered lines are identical.
	width = testWidth / 2
	r.RenderFrame(time.Now())
	assert.Equal(t, repaint(1, "steady"), buf.String())

	r.Unregister(id)
}

func TestLiveRegionMinTickSelection(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	minTick := func() time.Duration {
		r.mu.Lock()
		defer r.mu.Unlock()
		return r.minTickLocked()
	}

	// No slots: the defensive fallback cadence.
	assert.Equal(t, defaultSlotTick, minTick())

	slow := r.Register(staticSlot("slow"), 5*neverTick)
	assert.Equal(t, 5*neverTick, minTick())

	// A faster slot drives the shared cadence.
	fast := r.Register(staticSlot("fast"), 3*neverTick)
	assert.Equal(t, 3*neverTick, minTick())

	// Non-positive tick rates fall back to the default instead of spinning.
	defensive := r.Register(staticSlot("defensive"), 0)
	assert.Equal(t, defaultSlotTick, minTick())

	// Removing the fastest slot restores the next-fastest cadence.
	r.Unregister(defensive)
	assert.Equal(t, 3*neverTick, minTick())
	r.Unregister(fast)
	assert.Equal(t, 5*neverTick, minTick())

	r.Unregister(slow)
}

func TestLiveRegionMultiLineSlot(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	// A slot may render a newline-separated block; each line occupies its
	// own physical row of the painted block.
	id := r.Register(staticSlot("top\nbottom"), neverTick)
	assert.Equal(t, xansi.HideCursor+repaint(0, "top", "bottom"), buf.String())

	// Displacement accounts for the block's full physical height: the erase
	// moves up over both rows and the whole block repaints below the line.
	buf.Reset()
	r.WriteLines("log line\n")
	want := xansi.EnableSyncOutput +
		xansi.CursorUp(2) + xansi.CursorHorizontalAbsolute(1) + xansi.EraseScreenBelow +
		"log line\n" +
		repaint(0, "top", "bottom") +
		xansi.DisableSyncOutput
	assert.Equal(t, want, buf.String())

	buf.Reset()
	r.Unregister(id)
	assert.Equal(t, repaint(2)+xansi.ShowCursor, buf.String())
}

func TestLiveRegionMixedSingleAndMultiLineSlots(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	single := r.Register(staticSlot("spinner"), neverTick)
	buf.Reset()

	// Registering the multi-line slot repaints the block with its rows
	// stacked below the existing slot, in registration order.
	multi := r.Register(staticSlot("group-1\ngroup-2"), neverTick)
	assert.Equal(t, repaint(1, "spinner", "group-1", "group-2"), buf.String())

	buf.Reset()
	r.Unregister(single)
	assert.Equal(t, repaint(3, "group-1", "group-2"), buf.String())

	r.Unregister(multi)
	assert.False(t, r.Active())
}

func TestLiveRegionEmptySlotFrames(t *testing.T) {
	var buf bytes.Buffer
	r := newTestRegion(&buf)

	frame := ""
	id := r.Register(func(time.Time) string { return frame }, neverTick)

	// An empty first frame hides the cursor (the slot owns the region) but
	// paints nothing - not even sync markers.
	assert.Equal(t, xansi.HideCursor, buf.String())

	// While the block occupies zero rows, log writes pass through untouched.
	buf.Reset()
	r.WriteLines("plain\n")
	assert.Equal(t, "plain\n", buf.String())

	// A later non-empty frame paints the block as usual.
	buf.Reset()
	frame = "visible\nrows"
	r.RenderFrame(time.Now())
	assert.Equal(t, repaint(0, "visible", "rows"), buf.String())

	// Rendering empty again erases the block but keeps the slot registered.
	buf.Reset()
	frame = ""
	r.RenderFrame(time.Now())
	assert.Equal(t, repaint(2), buf.String())
	assert.True(t, r.Active())

	// Repeated empty frames write no bytes at all.
	buf.Reset()
	r.RenderFrame(time.Now())
	assert.Empty(t, buf.String())

	r.Unregister(id)
	assert.Equal(t, xansi.ShowCursor, buf.String())
}

// syncWriter is a goroutine-safe buffer for the one test whose render loop
// runs concurrently with assertions.
type syncWriter struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}

func (w *syncWriter) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.String()
}

func TestLiveRegionRenderLoopRepaintsChangingFrames(t *testing.T) {
	var buf syncWriter
	r := NewLiveRegion(&buf, func() int { return testWidth })

	// A time-varying frame so every loop tick produces a new block.
	var frame int64
	id := r.Register(func(time.Time) string {
		frame++
		return fmt.Sprintf("frame-%d", frame)
	}, time.Millisecond)

	// The shared loop (not the test goroutine) must repaint past the
	// initial registration frame.
	require.Eventually(t, func() bool {
		return strings.Contains(buf.String(), "frame-3")
	}, 2*time.Second, time.Millisecond)

	r.Unregister(id)
}
