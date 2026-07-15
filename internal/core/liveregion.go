package core

import (
	"io"
	"slices"
	"strings"
	"sync"
	"time"

	xansi "github.com/gechr/x/ansi"
)

// defaultSlotTick is the fallback repaint cadence for slots registered
// without a positive tick rate. It only applies to defensive callers; the
// animation builders always resolve a concrete per-mode tick rate first.
const defaultSlotTick = 100 * time.Millisecond

// liveSlot is one registered animation within a [LiveRegion]. A slot usually
// renders a single line, but may render a multi-row block (newline-separated
// lines) or nothing at all for a given tick.
type liveSlot struct {
	id     uint64
	render func(now time.Time) string
	tick   time.Duration
}

// EchoController toggles the terminal's local echo while a live block is on
// screen. Characters typed by the user are otherwise echoed by the terminal
// into the block, which corrupts the region's row accounting and leaves stale
// frames behind in the scrollback. Implementations must be idempotent: a
// Suppress while already suppressed and a Restore while already restored are
// no-ops. Both methods are only ever invoked under the region lock.
type EchoController interface {
	Suppress()
	Restore()
}

// LiveRegion coordinates a single repaintable block of animation lines with
// regular log writes on the same terminal. All concurrent animations on one
// output - standalone spinners and bars as single-line slots, whole groups as
// multi-line slots - register here and are painted as one stacked block (in
// registration order) by a single shared render loop, so they never clobber
// each other. Log lines written through [LiveRegion.WriteLines]
// while the block is on screen displace it: the block is erased, the log
// line lands above, and the block is repainted below - all inside one
// synchronized-output frame so nothing tears or gets lost.
//
// Single-writer discipline: while slots exist, every write to the terminal
// must go through this mutex - the render loop, slot registration, and log
// displacement all serialize here.
type LiveRegion struct {
	mu    sync.Mutex
	out   io.Writer
	width func() int

	slots  []liveSlot
	nextID uint64

	rows      int      // physical rows the block currently occupies on screen
	lastLines []string // last painted slot lines, for skip-identical dedup
	lastWidth int      // terminal width at last paint
	suspended bool

	// echo, when non-nil, has the terminal's local echo suppressed while the
	// block is live and restored whenever the terminal is released (last
	// unregistration and Suspend), mirroring cursor hide/show exactly.
	echo EchoController

	// wake nudges the render loop to recompute its tick rate after the slot
	// set changes; buffered so Register/Unregister never block on it.
	wake chan struct{}
	// stop terminates the current render-loop generation. A new channel (and
	// goroutine) is created when the first slot registers; the channel is
	// closed when the last slot unregisters.
	stop chan struct{}
}

// NewLiveRegion creates a LiveRegion writing to out. width is queried at
// every paint so wrapped-line row accounting tracks terminal resizes.
func NewLiveRegion(out io.Writer, width func() int) *LiveRegion {
	return &LiveRegion{
		out:   out,
		width: width,
		wake:  make(chan struct{}, 1),
	}
}

// SetEchoController installs (or, with nil, removes) the controller that
// suppresses terminal echo while the block is live. A change takes effect
// immediately: the previous controller is restored and, when slots are
// already live, the new one suppresses straight away - so toggling
// mid-animation never strands the terminal with echo disabled.
func (r *LiveRegion) SetEchoController(c EchoController) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.echo != nil {
		r.echo.Restore()
	}
	r.echo = c
	if c != nil && len(r.slots) > 0 && !r.suspended {
		c.Suppress()
	}
}

// suppressEchoLocked and restoreEchoLocked forward to the installed
// [EchoController], if any. Callers must hold r.mu.
func (r *LiveRegion) suppressEchoLocked() {
	if r.echo != nil {
		r.echo.Suppress()
	}
}

func (r *LiveRegion) restoreEchoLocked() {
	if r.echo != nil {
		r.echo.Restore()
	}
}

// Register adds an animation slot rendered by render at the given tick rate
// and returns its id for [LiveRegion.Unregister]. The first registration
// hides the cursor and starts the shared render loop; the new slot is
// painted immediately rather than waiting for the next tick. render is only
// ever invoked under the region lock, so it needs no synchronization of its
// own beyond not calling back into the region. render may return a
// newline-separated multi-row block (e.g. a group of animations) or an empty
// string to contribute no rows this tick.
func (r *LiveRegion) Register(render func(now time.Time) string, tick time.Duration) uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tick <= 0 {
		tick = defaultSlotTick
	}
	r.nextID++
	id := r.nextID
	r.slots = append(r.slots, liveSlot{id: id, render: render, tick: tick})

	if len(r.slots) == 1 && !r.suspended {
		// First slot: take over the region. The cursor stays hidden (and
		// terminal echo stays suppressed) until the last slot unregisters so
		// per-frame repaints don't flicker, and typed characters can't push
		// the block's rows out of alignment.
		writeString(r.out, xansi.HideCursor)
		r.suppressEchoLocked()
		stop := make(chan struct{})
		r.stop = stop
		go r.loop(stop)
	}

	if !r.suspended {
		r.paintLocked(time.Now())
		r.wakeLocked()
	}
	return id
}

// Unregister removes the slot with the given id, erases its line from the
// block, and repaints the remaining slots before returning. The last
// unregistration erases the whole block, stops the render loop, and shows
// the cursor again, leaving the cursor at the block's former top row so
// subsequent log lines land where the block used to start.
func (r *LiveRegion) Unregister(id uint64) {
	r.UnregisterWithCursor(id, true)
}

// UnregisterWithCursor removes the slot with the given id and controls
// whether the cursor is shown when that slot was the last visible owner of
// the terminal. It is used by temporarily hidden animation owners that need
// to preserve the cursor state across a pause. Ordinary animation completion
// should use [LiveRegion.Unregister].
func (r *LiveRegion) UnregisterWithCursor(id uint64, showCursor bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	idx := slices.IndexFunc(r.slots, func(s liveSlot) bool { return s.id == id })
	if idx < 0 {
		return
	}
	r.slots = slices.Delete(r.slots, idx, idx+1)

	if r.suspended {
		if len(r.slots) == 0 {
			r.suspended = false
			r.lastLines = nil
			r.lastWidth = 0
			r.rows = 0
			if showCursor {
				writeString(r.out, xansi.ShowCursor)
			}
			// Suspend already restored echo; a repeat is an idempotent no-op.
			r.restoreEchoLocked()
		}
		return
	}

	// Repaint the shrunken block; with no slots left this erases it entirely
	// (AppendRepaint handles the zero-line frame).
	r.paintLocked(time.Now())

	if len(r.slots) == 0 {
		if showCursor {
			writeString(r.out, xansi.ShowCursor)
		}
		r.restoreEchoLocked()
		if r.stop != nil {
			close(r.stop)
		}
		r.stop = nil
		r.lastLines = nil
		r.lastWidth = 0
		return
	}
	r.wakeLocked()
}

// Active reports whether any animation slots are currently registered.
func (r *LiveRegion) Active() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.slots) > 0
}

// Suspend temporarily releases the terminal while keeping registered slots.
// It stops the render loop and erases the current live block. When showCursor
// is true it also shows the cursor for the released period. It is idempotent
// and does nothing when no slots are active.
func (r *LiveRegion) Suspend(showCursor bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.suspended || len(r.slots) == 0 {
		return
	}

	if r.stop != nil {
		close(r.stop)
		r.stop = nil
	}
	if r.rows > 0 {
		var buf strings.Builder
		r.rows = AppendRepaint(&buf, nil, r.rows, r.lastWidth)
		writeString(r.out, buf.String())
	}
	if showCursor {
		writeString(r.out, xansi.ShowCursor)
	}
	// The terminal is released for the suspended period, so echo comes back
	// unconditionally - a suspension usually means someone else (a prompt, a
	// subprocess) needs the terminal in its normal state.
	r.restoreEchoLocked()
	r.lastLines = nil
	r.lastWidth = 0
	r.suspended = true
}

// Resume restores a previously suspended live region. It repaints the current
// slot state immediately and restarts the render loop. It is idempotent and
// does nothing when the region is not suspended or no slots are active.
func (r *LiveRegion) Resume() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.suspended {
		return
	}
	if len(r.slots) == 0 {
		r.suspended = false
		return
	}

	r.suspended = false
	writeString(r.out, xansi.HideCursor)
	r.suppressEchoLocked()
	stop := make(chan struct{})
	r.stop = stop
	go r.loop(stop)
	r.paintLocked(time.Now())
	r.wakeLocked()
}

// WriteLines writes s (one or more fully formatted log lines) to the output.
// With no block on screen it is a plain write. While the block is live, the
// write displaces it: one synchronized frame erases the block, writes s
// where the block's top row was, and repaints the block below - so log lines
// scroll up naturally above the animations instead of being overpainted by
// the next frame. s should end with a newline; one is added if missing while
// a block is live, because the repaint must start at the beginning of a
// fresh row. While suspended, writes pass through without repainting the live
// block.
func (r *LiveRegion) WriteLines(s string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.suspended {
		writeString(r.out, s)
		return
	}
	if r.rows == 0 {
		writeString(r.out, s)
		return
	}
	if !strings.HasSuffix(s, nl) {
		s += nl
	}

	var buf strings.Builder
	buf.WriteString(xansi.EnableSyncOutput)
	// The cursor is parked one row below the block after every paint, so
	// moving up by the block's physical row count lands on its top row.
	buf.WriteString(xansi.CursorUp(r.rows))
	buf.WriteString(xansi.CursorHorizontalAbsolute(1))
	buf.WriteString(xansi.EraseScreenBelow)
	buf.WriteString(s)
	// Repaint the block exactly as last rendered (no re-render: cadence stays
	// owned by the render loop, and high-rate logging must not speed up the
	// animations). AppendRepaint's nested sync markers are harmless: the
	// inner enable is a no-op and the inner disable commits the entire
	// buffered frame at once, so atomicity is preserved.
	r.rows = AppendRepaint(&buf, r.lastLines, 0, r.lastWidth)
	buf.WriteString(xansi.DisableSyncOutput)
	writeString(r.out, buf.String())
}

// RenderFrame renders every slot at now and repaints the block in place,
// skipping the write entirely when neither the rendered lines nor the
// terminal width changed. Exported so animation owners can force an
// out-of-band frame (e.g. a progress bar's final 100% frame before
// unregistering) and so tests can drive the renderer deterministically.
func (r *LiveRegion) RenderFrame(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.suspended {
		return
	}
	r.paintLocked(now)
}

// paintLocked renders all slots and repaints the block over the previous
// one. With no slots it erases any remaining block. Callers must hold r.mu.
func (r *LiveRegion) paintLocked(now time.Time) {
	if len(r.slots) == 0 && r.rows == 0 {
		// Nothing registered and nothing on screen: a stale render-loop tick
		// racing a final Unregister must not write anything.
		return
	}

	// Flatten every slot's output into one list of physical lines. A slot
	// may render a multi-row block (newline-separated) or nothing at all for
	// this tick; either way the block stacks in registration order.
	lines := make([]string, 0, len(r.slots))
	for _, s := range r.slots {
		block := s.render(now)
		if block == "" {
			continue
		}
		lines = append(lines, strings.Split(block, nl)...)
	}
	width := r.width()

	// Nothing on screen and nothing to draw: record the frame so later
	// dedup checks compare against it, but write no bytes (not even sync
	// markers).
	if len(lines) == 0 && r.rows == 0 {
		r.lastLines = lines
		r.lastWidth = width
		return
	}

	// Skip identical frames: nothing on screen would change, so a write
	// would be wasted bandwidth.
	if r.rows > 0 && width == r.lastWidth && slices.Equal(lines, r.lastLines) {
		return
	}

	var buf strings.Builder
	r.rows = AppendRepaint(&buf, lines, r.rows, width)
	writeString(r.out, buf.String())
	r.lastLines = lines
	r.lastWidth = width
}

// loop is the shared render loop: it sleeps for the minimum tick rate of the
// registered slots, repaints, and recomputes the rate whenever the slot set
// changes. It exits when stop is closed (last slot unregistered). A timer
// rebuilt per iteration is used instead of a ticker because the interval
// changes as slots come and go.
func (r *LiveRegion) loop(stop <-chan struct{}) {
	for {
		r.mu.Lock()
		tick := r.minTickLocked()
		r.mu.Unlock()

		timer := time.NewTimer(tick)
		select {
		case <-stop:
			timer.Stop()
			return
		case <-r.wake:
			// Slot set changed: recompute the tick rate. The frame itself
			// was already painted by Register/Unregister, so just loop.
			timer.Stop()
		case now := <-timer.C:
			r.RenderFrame(now)
		}
	}
}

// minTickLocked returns the fastest tick rate among registered slots, which
// is the cadence the shared render loop runs at. Callers must hold r.mu.
func (r *LiveRegion) minTickLocked() time.Duration {
	if len(r.slots) == 0 {
		return defaultSlotTick
	}
	tick := r.slots[0].tick
	for _, s := range r.slots[1:] {
		tick = min(tick, s.tick)
	}
	return tick
}

// wakeLocked nudges the render loop to recompute its tick rate. Non-blocking:
// a pending nudge already covers any number of slot-set changes.
func (r *LiveRegion) wakeLocked() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

// writeString writes s to w, discarding the return values.
func writeString(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}
