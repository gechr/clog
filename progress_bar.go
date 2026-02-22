package clog

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	// barTickRate is the repaint interval when a bar animation is active (~20fps).
	barTickRate = 50 * time.Millisecond

	barDefaultBarWidthMin = 10 // default minimum auto-sized inner width
	barDefaultWidthMax    = 40 // default maximum auto-sized inner width
	barWidthDivisor       = 4  // terminal width fraction used for auto-sizing
)

// BarAlign controls the horizontal placement of the progress bar within the terminal line.
type BarAlign int

const (
	// BarAlignRightPad pushes the bar to the right edge of the terminal (default).
	BarAlignRightPad BarAlign = iota
	// BarAlignLeftPad places the bar before the message, padding the message to the right edge.
	BarAlignLeftPad
	// BarAlignInline places the bar immediately after the message with no gap.
	BarAlignInline
	// BarAlignLeft places the bar before the message with no padding.
	BarAlignLeft
	// BarAlignRight places the bar after the message with no padding.
	BarAlignRight
)

// BarStyle configures the visual appearance of a determinate progress bar.
type BarStyle struct {
	Align            BarAlign    // horizontal bar placement; default BarAlignRightPad
	CapLeft          string      // left bracket; default "["
	CapRight         string      // right bracket; default "]"
	CapStyle         Style       // lipgloss style for left/right caps; nil = plain text
	CharEmpty        rune        // full empty cell; default '─'
	CharFill         rune        // full filled cell; default '━'
	CharHead         rune        // decorative head at leading edge (1x resolution); 0 = disabled; ignored when HalfFilled is set
	GradientFill     []rune      // sub-cell fill chars, least→most filled; enables Nx resolution (N = len+1); overrides HalfFilled/CharHead
	HalfEmpty        rune        // half-cell at start of empty when HalfFilled is not shown; 0 = disabled
	HalfFilled       rune        // half-cell at leading edge of filled (enables 2x resolution); 0 = disabled
	ProgressGradient []ColorStop // when set, colors filled cells based on progress; overrides StyleFill foreground
	Separator        string      // separator between message, bar, and widget text; default " "
	StyleEmpty       Style       // lipgloss style for empty cells; nil = plain text
	StyleFill        Style       // lipgloss style for filled cells; nil = plain text
	Width            int         // fixed inner width; 0 = auto-size
	WidgetLeft       BarWidget   // widget to the left of the bar; nil = nothing
	WidgetRight      BarWidget   // widget to the right of the bar; nil = default padded percent
	WidthMax         int         // maximum auto-sized width; default 40
	WidthMin         int         // minimum auto-sized width; default 10
}

func (s BarStyle) applyAnimation(b *AnimationBuilder) { b.barStyle = s }

// DefaultBarStyle returns the default [BarStyle].
// It uses box-drawing characters with half-cell resolution for smooth progress.
func DefaultBarStyle() BarStyle { return BarThin }

// DefaultBarGradient returns the default red → yellow → green gradient
// used for [BarStyle.ProgressGradient].
func DefaultBarGradient() []ColorStop { return DefaultPercentGradient() }

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
		spinner:        DefaultSpinnerStyle(),
	}
	b.initSelf(b)
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

	filledChar := style.CharFill
	if filledChar == 0 {
		filledChar = '━'
	}
	charEmpty := style.CharEmpty
	if charEmpty == 0 {
		charEmpty = '─'
	}

	innerWidth := resolveBarWidth(style, termWidth)

	// Compute filled/empty counts and boundary characters.
	var filledCount, emptyCount int
	var headStr, trailStr string

	switch {
	case len(style.GradientFill) > 0:
		// Nx sub-cell resolution.
		subUnits := len(style.GradientFill) + 1
		completeParts := min(
			innerWidth*subUnits,
			int(float64(innerWidth)*float64(subUnits)*float64(current)/float64(total)),
		)
		filledCount = completeParts / subUnits
		remainder := completeParts % subUnits
		emptyCount = innerWidth - filledCount
		if remainder > 0 {
			headStr = string(style.GradientFill[remainder-1])
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
		if style.CharHead != 0 && filledCount > 0 && filledCount < innerWidth {
			headStr = string(style.CharHead)
			filledCount--
		}
	}

	filledStr := strings.Repeat(string(filledChar), filledCount)
	emptyStr := strings.Repeat(string(charEmpty), emptyCount)

	// When ProgressGradient is set, compute a single color from the gradient
	// at the current progress position and use it for filled cells.
	styleFill := style.StyleFill
	if len(style.ProgressGradient) > 0 {
		progress := float64(current) / float64(total)
		c := interpolateGradient(progress, style.ProgressGradient)
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
		styleFill = &s
	}

	var buf strings.Builder
	barWriteStyled(&buf, style.CapLeft, style.CapStyle)
	barWriteStyled(&buf, filledStr, styleFill)
	barWriteStyled(&buf, headStr, styleFill)
	barWriteStyled(&buf, trailStr, style.StyleEmpty)
	barWriteStyled(&buf, emptyStr, style.StyleEmpty)
	barWriteStyled(&buf, style.CapRight, style.CapStyle)
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
// digits controls decimal places (0 → "50%", 1 → "50%" or "42.5%").
// Trailing decimal zeros are always stripped ("50.0%" → "50%").
// When pad is true, the result is right-aligned to the width of "100%"
// at the given digit count for stable display.
func barPercent(current, total, digits int, pad bool) string {
	var pct float64
	if total > 0 {
		pct = float64(current) / float64(total) * percentMax
		if pct > percentMax {
			pct = percentMax
		}
	}
	s := trimDecimalZeros(fmt.Sprintf("%.*f", digits, pct)) + "%"
	if pad {
		padWidth := len(fmt.Sprintf("%.*f%%", digits, percentMax))
		return fmt.Sprintf("%*s", padWidth, s)
	}
	return s
}

// resolveBarWidth computes the inner cell count for the bar from the style
// and the terminal width. A fixed Width takes priority; otherwise the width
// is derived from termWidth and clamped to [WidthMin, WidthMax].
func resolveBarWidth(style BarStyle, termWidth int) int {
	if style.Width > 0 {
		return style.Width
	}

	minW := style.WidthMin
	if minW <= 0 {
		minW = barDefaultBarWidthMin
	}
	maxW := style.WidthMax
	if maxW <= 0 {
		maxW = barDefaultWidthMax
	}

	w := minW
	if termWidth > 0 {
		w = termWidth / barWidthDivisor
	}

	return max(minW, min(maxW, w))
}
