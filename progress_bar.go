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

	barDefaultBarMinWidth = 10 // default minimum auto-sized inner width
	barDefaultMaxWidth    = 40 // default maximum auto-sized inner width
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

// PercentPosition controls which side of the bar the percentage label appears on.
type PercentPosition int

const (
	// PercentRight places the percentage after the bar (default): [━━━╺──] 50%
	PercentRight PercentPosition = iota
	// PercentLeft places the percentage before the bar: 50% [━━━╺──]
	PercentLeft
)

// BarStyle configures the visual appearance of a determinate progress bar.
type BarStyle struct {
	Align            BarAlign        // horizontal bar placement; default BarAlignRightPad
	CapStyle         Style           // lipgloss style for left/right caps; nil = plain text
	EmptyChar        rune            // full empty cell; default '─'
	EmptyStyle       Style           // lipgloss style for empty cells; nil = plain text
	FilledChar       rune            // full filled cell; default '━'
	FilledStyle      Style           // lipgloss style for filled cells; nil = plain text
	FillGradient     []rune          // sub-cell fill chars, least→most filled; enables Nx resolution (N = len+1); overrides HalfFilled/HeadChar
	HalfEmpty        rune            // half-cell at start of empty when HalfFilled is not shown; 0 = disabled
	HalfFilled       rune            // half-cell at leading edge of filled (enables 2x resolution); 0 = disabled
	HeadChar         rune            // decorative head at leading edge (1x resolution); 0 = disabled; ignored when HalfFilled is set
	HidePercent      bool            // when true, the percentage label is not shown
	LeftCap          string          // left bracket; default "["
	MaxWidth         int             // maximum auto-sized width; default 40
	MinWidth         int             // minimum auto-sized width; default 10
	NoPadPercent     bool            // disable right-aligned fixed-width percentage; when false (default), the label is padded to prevent jumping
	PercentField     string          // when set, the percentage is shown as a structured field with this key instead of beside the bar; defaults to "progress" for BarAlignInline
	PercentPosition  PercentPosition // which side of the bar the percentage appears on; default PercentRight
	PercentPrecision int             // decimal places for the percentage label; default 0 (e.g. 0 → "50%", 1 → "50.0%")
	ProgressGradient []ColorStop     // when set, colors filled cells based on progress; overrides FilledStyle foreground
	RightCap         string          // right bracket; default "]"
	Separator        string          // separator between message, bar, and percentage; default " "
	Width            int             // fixed inner width; 0 = auto-size
}

// percentFieldKey returns the effective percent field key. When PercentField
// is explicitly set, that value is returned. Otherwise, [BarAlignInline]
// defaults to "progress" so the percentage is shown as a structured field.
func (s BarStyle) percentFieldKey() string {
	if s.PercentField != "" {
		return s.PercentField
	}
	if s.Align == BarAlignInline {
		return "progress"
	}
	return ""
}

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

func (s BarStyle) applyAnimation(b *AnimationBuilder) { b.barStyle = s }

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

	// When ProgressGradient is set, compute a single color from the gradient
	// at the current progress position and use it for filled cells.
	filledStyle := style.FilledStyle
	if len(style.ProgressGradient) > 0 {
		progress := float64(current) / float64(total)
		c := interpolateGradient(progress, style.ProgressGradient)
		s := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
		filledStyle = &s
	}

	var buf strings.Builder
	barWriteStyled(&buf, style.LeftCap, style.CapStyle)
	barWriteStyled(&buf, filledStr, filledStyle)
	barWriteStyled(&buf, headStr, filledStyle)
	barWriteStyled(&buf, trailStr, style.EmptyStyle)
	barWriteStyled(&buf, emptyStr, style.EmptyStyle)
	barWriteStyled(&buf, style.RightCap, style.CapStyle)
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
// precision controls decimal places (0 → "50%", 1 → "50.0%").
// When pad is true the result is right-aligned to a fixed width so the string
// width stays constant and the bar doesn't jump (e.g. "  0%", " 50%", "100%").
func barPercent(current, total, precision int, pad bool) string {
	var pct float64
	if total > 0 {
		pct = float64(current) / float64(total) * percentMax
		if pct > percentMax {
			pct = percentMax
		}
	}
	s := fmt.Sprintf("%.*f%%", precision, pct)
	if pad {
		// "100%" with the given precision is the widest possible string.
		w := len(fmt.Sprintf("%.*f%%", precision, percentMax))
		return fmt.Sprintf("%*s", w, s)
	}
	return s
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
