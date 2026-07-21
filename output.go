package clog

import (
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/colorprofile"
	"github.com/gechr/clog/field/hyperlink"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/theme"
	xansi "github.com/gechr/x/ansi"
	"github.com/gechr/x/terminal"
	"golang.org/x/term"
)

// Output bundles an [io.Writer] with its detected terminal capabilities
// (TTY, width, color profile). Each [Logger] holds an *Output so that
// capability detection is per-writer instead of per-process.
type Output struct {
	w       io.Writer
	fd      int // -1 for non-fd writers
	isTTY   bool
	profile colorprofile.Profile

	sizeMu   sync.Mutex
	sizeDone bool
	width    int
	height   int

	bgMu   sync.Mutex
	bgDone bool
	bg     theme.Background
	bgOK   bool

	cursorMu sync.Mutex

	// hyperlinks holds the hyperlink rendering configuration pushed down
	// from the owning logger's FieldFormats. nil means the default
	// (enabled, plain file:// URLs).
	hyperlinks atomic.Pointer[hyperlink.Config]

	// region coordinates live animation repaints with regular log writes.
	// Lazily created on first use so outputs that never animate pay nothing;
	// nil means no animation has ever run on this output.
	region atomic.Pointer[core.LiveRegion]

	// Tests may override cursor probing to avoid real terminal I/O.
	queryCursorPosition func(io.Writer) (cursorPosition, bool)
}

type cursorPosition struct {
	row    int
	column int
}

const (
	cursorPositionFieldCount    = 2
	cursorPositionResponseLimit = 32
	cursorPositionTimeout       = 50 * time.Millisecond
)

// NewOutput creates a new Output that wraps w. TTY detection is automatic
// for writers that expose an Fd() uintptr method (e.g. [*os.File]). The
// [ColorMode] determines how colors are handled:
//   - [ColorAuto] respects TTY detection and NO_COLOR.
//   - [ColorAlways] forces colors even on non-TTY writers.
//   - [ColorNever] disables all colors.
func NewOutput(w io.Writer, mode ColorMode) *Output {
	o := &Output{w: w, fd: -1}

	if f, ok := w.(interface{ Fd() uintptr }); ok {
		o.fd = int(f.Fd())
		o.isTTY = term.IsTerminal(o.fd)
	}

	o.profile = detectProfile(w, o.isTTY, mode)

	return o
}

// Stdout returns a new Output for [os.Stdout].
func Stdout(mode ColorMode) *Output {
	return NewOutput(os.Stdout, mode)
}

// Stderr returns a new Output for [os.Stderr].
func Stderr(mode ColorMode) *Output {
	return NewOutput(os.Stderr, mode)
}

// TestOutput returns a non-TTY Output with colors disabled, suitable for tests.
func TestOutput(w io.Writer) *Output {
	return NewOutput(w, ColorNever)
}

// Writer returns the underlying [io.Writer].
func (o *Output) Writer() io.Writer { return o.w }

// LiveRegion returns the live-animation region for this output, creating it
// on first use. There is exactly one region per Output so every animation and
// log line on the same writer coordinates through it. The fx render loops
// discover the region via an optional-capability type assertion, which keeps
// the fx.Output interface (and external implementations) unchanged.
func (o *Output) LiveRegion() *core.LiveRegion {
	if r := o.region.Load(); r != nil {
		return r
	}
	r := core.NewLiveRegion(o.w, o.Width)
	if o.region.CompareAndSwap(nil, r) {
		return r
	}
	return o.region.Load()
}

// SetSuppressEchoDuringAnimations controls whether the terminal's local echo
// is disabled while animations are live on this output. See
// [Logger.SetSuppressEchoDuringAnimations]. No-op when the writer is not a
// terminal.
func (o *Output) SetSuppressEchoDuringAnimations(suppress bool) {
	if !o.isTTY || o.fd < 0 {
		return
	}
	if !suppress {
		// Nothing to remove if no animation ever ran on this output.
		if r := o.region.Load(); r != nil {
			r.SetEchoController(nil)
		}
		return
	}
	o.LiveRegion().SetEchoController(newEchoController(o.fd))
}

// WriteLine writes a fully formatted log line (or multi-line payload ending
// in a newline) to the underlying writer. While animations are live on this
// output, the write is routed through the [core.LiveRegion] so the animation
// block is displaced: erased, the line written above, and the block repainted
// below in one synchronized frame. Without an active region this is a plain
// write.
func (o *Output) WriteLine(s string) {
	if r := o.region.Load(); r != nil {
		// WriteLines falls back to a plain write when no slots are live.
		r.WriteLines(s)
		return
	}
	writeString(o.w, s)
}

// IsTTY returns true if the writer is connected to a terminal.
func (o *Output) IsTTY() bool { return o.isTTY }

// ColorsDisabled returns true if this output should suppress colors.
func (o *Output) ColorsDisabled() bool {
	return o.profile == colorprofile.NoTTY
}

// background lazily detects and caches the terminal background for this
// output, querying the terminal at most once. The boolean reports whether
// detection succeeded; it is false for non-terminal or non-file writers.
func (o *Output) background() (theme.Background, bool) {
	o.bgMu.Lock()
	defer o.bgMu.Unlock()

	if !o.bgDone {
		o.bgDone = true
		if terminal.Is(o.file()) {
			o.bg, o.bgOK = theme.DetectBackground()
		}
	}

	return o.bg, o.bgOK
}

// file returns the underlying [*os.File], or nil for non-file writers.
func (o *Output) file() *os.File {
	if f, ok := o.w.(*os.File); ok {
		return f
	}
	return nil
}

// Width returns the terminal width, or 0 for non-TTY writers.
// The value is lazily detected and cached; call [Output.RefreshWidth]
// to re-detect.
func (o *Output) Width() int {
	o.sizeMu.Lock()
	defer o.sizeMu.Unlock()
	o.detectSizeLocked()
	return o.width
}

// Height returns the terminal height, or 0 for non-TTY writers.
// The value is lazily detected and cached; call [Output.RefreshHeight]
// to re-detect.
func (o *Output) Height() int {
	o.sizeMu.Lock()
	defer o.sizeMu.Unlock()
	o.detectSizeLocked()
	return o.height
}

// detectSizeLocked lazily queries both terminal dimensions with a single
// size query. o.sizeMu must be held.
func (o *Output) detectSizeLocked() {
	if o.sizeDone {
		return
	}
	o.sizeDone = true

	if o.isTTY && o.fd >= 0 {
		if w, h, err := term.GetSize(o.fd); err == nil {
			o.width, o.height = w, h
		}
	}
}

// refreshSize re-queries the terminal size and updates both cached
// dimensions. If the query fails (or the writer is not a TTY) the cache is
// left untouched, so manually-seeded test sizes survive a refresh.
func (o *Output) refreshSize() {
	o.sizeMu.Lock()
	defer o.sizeMu.Unlock()
	if !o.isTTY || o.fd < 0 {
		return
	}
	if w, h, err := term.GetSize(o.fd); err == nil {
		o.width, o.height = w, h
		o.sizeDone = true
	}
}

// RefreshWidth re-queries the terminal size and updates the cached
// dimensions. See [Output.refreshSize].
func (o *Output) RefreshWidth() { o.refreshSize() }

// RefreshHeight re-queries the terminal size and updates the cached
// dimensions. See [Output.refreshSize].
func (o *Output) RefreshHeight() { o.refreshSize() }

// ListenResize starts a background goroutine that refreshes the cached
// terminal width and height on SIGWINCH. Call the returned stop function
// to unregister the signal handler and release the goroutine.
// No-op for non-TTY outputs.
func (o *Output) ListenResize() func() {
	if !o.isTTY {
		return func() {}
	}
	ch := make(chan os.Signal, 1)
	notifyResize(ch)
	go func() {
		for range ch {
			o.refreshSize()
		}
	}()
	return func() {
		signal.Stop(ch)
		close(ch)
	}
}

// CursorPosition reports the cursor's current 1-based row, if known.
// It returns false for non-TTY writers or when the terminal does not
// answer the cursor-position query in time.
func (o *Output) CursorPosition() (int, bool) {
	pos, ok := o.cursorPosition()
	return pos.row, ok
}

func (o *Output) cursorPosition() (cursorPosition, bool) {
	if !o.isTTY {
		return cursorPosition{}, false
	}

	o.cursorMu.Lock()
	defer o.cursorMu.Unlock()

	if o.queryCursorPosition != nil {
		return o.queryCursorPosition(o.w)
	}
	return queryCursorPosition(o.w)
}

// writeString writes s to w, discarding the return values.
func writeString(w io.Writer, s string) {
	_, _ = io.WriteString(w, s)
}

func queryCursorPosition(out io.Writer) (cursorPosition, bool) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return cursorPosition{}, false
	}

	state, err := term.MakeRaw(fd)
	if err != nil {
		return cursorPosition{}, false
	}
	defer func() { _ = term.Restore(fd, state) }()

	if err := os.Stdin.SetReadDeadline(time.Now().Add(cursorPositionTimeout)); err != nil {
		return cursorPosition{}, false
	}
	defer func() { _ = os.Stdin.SetReadDeadline(time.Time{}) }()

	writeString(out, xansi.RequestCursorPosition)

	var buf strings.Builder
	buf.Grow(cursorPositionResponseLimit)
	tmp := make([]byte, 1)

	for buf.Len() < cursorPositionResponseLimit {
		n, err := os.Stdin.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
			if pos, ok := parseCursorPositionReport(buf.String()); ok {
				return pos, true
			}
		}
		if err != nil {
			return cursorPosition{}, false
		}
	}

	return cursorPosition{}, false
}

func parseCursorPositionReport(report string) (cursorPosition, bool) {
	start := strings.LastIndex(report, xansi.CSI)
	if start < 0 {
		return cursorPosition{}, false
	}

	end := strings.IndexByte(report[start:], 'R')
	if end < 0 {
		return cursorPosition{}, false
	}

	parts := strings.Split(report[start+2:start+end], ";")
	if len(parts) != cursorPositionFieldCount {
		return cursorPosition{}, false
	}

	row, err := strconv.Atoi(parts[0])
	if err != nil || row < 1 {
		return cursorPosition{}, false
	}

	column, err := strconv.Atoi(parts[1])
	if err != nil || column < 1 {
		return cursorPosition{}, false
	}

	return cursorPosition{row: row, column: column}, true
}

// detectProfile determines the [colorprofile.Profile] for the given writer,
// TTY state, and color mode.
func detectProfile(w io.Writer, isTTY bool, mode ColorMode) colorprofile.Profile {
	switch mode {
	case ColorAlways:
		return colorprofile.TrueColor
	case ColorNever:
		return colorprofile.NoTTY
	case ColorAuto:
		if noColorEnvSet.Load() {
			return colorprofile.NoTTY
		}
		// A non-terminal writer disables color, unless CLICOLOR_FORCE explicitly
		// asks for it (https://bixense.com/clicolors/) - detection below then
		// honors the force, so `cmd | cat` still colors when requested.
		if !isTTY && !cliColorForced() {
			return colorprofile.NoTTY
		}
	}
	return colorprofile.Detect(w, os.Environ())
}

// cliColorForced reports whether CLICOLOR_FORCE is set to a truthy value, which
// forces color even when the writer is not a terminal. NO_COLOR takes
// precedence, so callers check it first.
func cliColorForced() bool {
	force, _ := strconv.ParseBool(os.Getenv("CLICOLOR_FORCE"))
	return force
}

// IsTerminal returns true if the [Default] logger's output is connected to a terminal.
func IsTerminal() bool {
	logger := Default()
	logger.mu.Lock()
	defer logger.mu.Unlock()
	return logger.output.IsTTY()
}
