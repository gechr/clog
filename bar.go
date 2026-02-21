package clog

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// barTickRate is the repaint interval when a bar animation is active (~20fps).
	barTickRate = 50 * time.Millisecond

	barDefaultBarMinWidth = 10 // default minimum auto-sized inner width
	barDefaultMaxWidth    = 40 // default maximum auto-sized inner width
	barWidthDivisor       = 4  // terminal width fraction used for auto-sizing
)

// BarStyle configures the visual appearance of a determinate progress bar.
type BarStyle struct {
	FilledChar   rune   // full filled cell; default '━'
	EmptyChar    rune   // full empty cell; default '─'
	HeadChar     rune   // decorative head at leading edge (1x resolution); 0 = disabled; ignored when HalfFilled is set
	HalfFilled   rune   // half-cell at leading edge of filled (enables 2x resolution); 0 = disabled
	HalfEmpty    rune   // half-cell at start of empty when HalfFilled is not shown; 0 = disabled
	FillGradient []rune // sub-cell fill chars, least→most filled; enables Nx resolution (N = len+1); overrides HalfFilled/HeadChar
	LeftCap      string // left bracket; default "["
	RightCap     string // right bracket; default "]"
	Separator    string // separator between message, bar, and percentage; default " "
	Width        int    // fixed inner width; 0 = auto-size
	MinWidth     int    // minimum auto-sized width; default 10
	MaxWidth     int    // maximum auto-sized width; default 40
	FilledStyle  Style  // lipgloss style for filled cells; nil = plain text
	EmptyStyle   Style  // lipgloss style for empty cells; nil = plain text
}

// DefaultBarStyle returns the default [BarStyle].
// It uses box-drawing characters with half-cell resolution for smooth progress.
func DefaultBarStyle() BarStyle { return BarThin }

// Bar creates a new [AnimationBuilder] using the [Default] logger with a
// determinate progress bar animation.
// total is the maximum progress value. Use [ProgressUpdate.SetProgress] to update progress.
func Bar(msg string, total int) *AnimationBuilder { return Default.Bar(msg, total) }

// Bar creates a new [AnimationBuilder] with a determinate progress bar animation.
// total is the maximum progress value. Use [ProgressUpdate.SetProgress] to update progress.
func (l *Logger) Bar(msg string, total int) *AnimationBuilder {
	if total <= 0 {
		total = 1
	}

	progressPtr := new(atomic.Int64)
	totalPtr := new(atomic.Int64)
	totalPtr.Store(int64(total))

	b := &AnimationBuilder{
		level:          InfoLevel,
		logger:         l,
		mode:           animationBar,
		msg:            msg,
		barStyle:       DefaultBarStyle(),
		barProgressPtr: progressPtr,
		barTotalPtr:    totalPtr,
		spinner:        DefaultSpinner,
	}
	b.initSelf(b)
	return b
}

// BarStyle sets the visual style for the progress bar.
// Only meaningful when the builder was created with [Bar].
func (b *AnimationBuilder) BarStyle(style BarStyle) *AnimationBuilder {
	b.barStyle = style
	return b
}

// renderBar renders the visual bar string for the given progress values.
// termWidth is the terminal column count (0 = fall back to auto-sizing from style).
func renderBar(current, total int, style BarStyle, termWidth int) string {
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}

	filledChar := style.FilledChar
	if filledChar == 0 {
		filledChar = '━'
	}
	emptyChar := style.EmptyChar
	if emptyChar == 0 {
		emptyChar = '─'
	}

	innerWidth := resolveBarWidth(style, termWidth)

	// Compute filled/empty counts and boundary characters.
	var filledCount, emptyCount int
	var headStr, trailStr string

	switch {
	case len(style.FillGradient) > 0:
		// Nx sub-cell resolution.
		subUnits := len(style.FillGradient) + 1
		completeParts := min(
			innerWidth*subUnits,
			int(float64(innerWidth)*float64(subUnits)*float64(current)/float64(total)),
		)
		filledCount = completeParts / subUnits
		remainder := completeParts % subUnits
		emptyCount = innerWidth - filledCount
		if remainder > 0 {
			headStr = string(style.FillGradient[remainder-1])
			emptyCount--
		}
	case style.HalfFilled != 0:
		// Half-cell (2x) resolution.
		completeHalves := min(
			innerWidth*2, //nolint:mnd // 2x resolution
			int(float64(innerWidth)*2*float64(current)/float64(total)),
		)
		filledCount = completeHalves / 2 //nolint:mnd // halves to cells
		emptyCount = innerWidth - filledCount
		if completeHalves%2 == 1 {
			headStr = string(style.HalfFilled)
			emptyCount--
		} else if filledCount > 0 && emptyCount > 0 && style.HalfEmpty != 0 {
			trailStr = string(style.HalfEmpty)
			emptyCount--
		}
	default:
		// Full-cell (1x) resolution.
		filledCount = min(innerWidth, int(float64(current)/float64(total)*float64(innerWidth)))
		emptyCount = innerWidth - filledCount
		if style.HeadChar != 0 && filledCount > 0 && filledCount < innerWidth {
			headStr = string(style.HeadChar)
			filledCount--
		}
	}

	filledStr := strings.Repeat(string(filledChar), filledCount)
	emptyStr := strings.Repeat(string(emptyChar), emptyCount)

	var buf strings.Builder
	buf.WriteString(style.LeftCap)
	barWriteStyled(&buf, filledStr, style.FilledStyle)
	barWriteStyled(&buf, headStr, style.FilledStyle)
	barWriteStyled(&buf, trailStr, style.EmptyStyle)
	barWriteStyled(&buf, emptyStr, style.EmptyStyle)
	buf.WriteString(style.RightCap)
	return buf.String()
}

// barWriteStyled writes s to buf with an optional lipgloss style.
func barWriteStyled(buf *strings.Builder, s string, style Style) {
	if s == "" {
		return
	}
	if style != nil {
		buf.WriteString(style.Render(s))
	} else {
		buf.WriteString(s)
	}
}

// barPercent formats the percentage string for display alongside the bar.
func barPercent(current, total int) string {
	if total <= 0 {
		return "0%"
	}
	pct := float64(current) / float64(total) * percentMax
	if pct > percentMax {
		pct = percentMax
	}
	return fmt.Sprintf("%.0f%%", pct)
}

// resolveBarWidth computes the inner cell count for the bar from the style
// and the terminal width. A fixed Width takes priority; otherwise the width
// is derived from termWidth and clamped to [MinWidth, MaxWidth].
func resolveBarWidth(style BarStyle, termWidth int) int {
	if style.Width > 0 {
		return style.Width
	}

	minW := style.MinWidth
	if minW <= 0 {
		minW = barDefaultBarMinWidth
	}
	maxW := style.MaxWidth
	if maxW <= 0 {
		maxW = barDefaultMaxWidth
	}

	w := minW
	if termWidth > 0 {
		w = termWidth / barWidthDivisor
	}

	return max(minW, min(maxW, w))
}
