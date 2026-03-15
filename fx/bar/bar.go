// Package bar provides bar animation styles and presets for clog.
// Widget functions (Percent, ETA, Bytes, Rate, etc.) live in the
// [github.com/gechr/clog/fx/bar/widget] subpackage.
package bar

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/gradient"
	"github.com/gechr/clog/internal/numfmt"
	"github.com/gechr/clog/style"
)

const (
	// TickRate is the repaint interval when a bar animation is active (~20fps).
	TickRate = 50 * time.Millisecond

	DefaultWidthMin = 10 // default minimum auto-sized inner width
	DefaultWidthMax = 40 // default maximum auto-sized inner width
	WidthDivisor    = 4  // terminal width fraction used for auto-sizing
)

// PercentDisplayMax is the maximum display percentage (always 100).
const PercentDisplayMax = 100.0

// Placement controls the horizontal placement of the progress bar within the terminal line.
type Placement int

const (
	// PlaceRightPad pushes the bar to the right edge of the terminal (default).
	PlaceRightPad Placement = iota
	// PlaceLeftPad places the bar before the message, padding the message to the right edge.
	PlaceLeftPad
	// PlaceInline places the bar immediately after the message with no gap.
	PlaceInline
	// PlaceLeft places the bar before the message with no padding.
	PlaceLeft
	// PlaceRight places the bar after the message with no padding.
	PlaceRight
)

// PendingMode controls how a bar is rendered before any progress is reported.
type PendingMode int

const (
	// PendingShow renders the bar and its widgets immediately, even at 0 progress.
	PendingShow PendingMode = iota
	// PendingHide suppresses the entire bar block until progress becomes positive.
	PendingHide
)

// Style configures the visual appearance of a determinate progress bar.
type Style struct {
	CapLeft          string               // left bracket; default "["
	CapRight         string               // right bracket; default "]"
	CapStyle         *lipgloss.Style      // lipgloss style for left/right caps; nil = plain text
	CharEmpty        rune                 // full empty cell; default '─'
	CharFill         rune                 // full filled cell; default '━'
	CharHead         rune                 // decorative head at leading edge (1x resolution); 0 = disabled; ignored when HalfFilled is set
	GradientFill     []rune               // sub-cell fill chars, least→most filled; enables Nx resolution (N = len+1); overrides HalfFilled/CharHead
	HalfEmpty        rune                 // half-cell at start of empty when HalfFilled is not shown; 0 = disabled
	HalfFilled       rune                 // half-cell at leading edge of filled (enables 2x resolution); 0 = disabled
	Placement        Placement            // horizontal bar placement; default PlaceRightPad
	PendingMode      PendingMode          // whether to show the bar before progress starts; default PendingShow
	ProgressGradient []gradient.ColorStop // when set, colors filled cells based on progress; overrides StyleFill foreground
	Separator        string               // separator between message, bar, and widget text; default " "
	StyleEmpty       *lipgloss.Style      // lipgloss style for empty cells; nil = plain text
	StyleFill        *lipgloss.Style      // lipgloss style for filled cells; nil = plain text
	Width            int                  // fixed inner width; 0 = auto-size
	WidgetLeft       Widget               // widget to the left of the bar; nil = nothing
	WidgetRight      Widget               // widget to the right of the bar; nil = default padded percent
	WidthMax         int                  // maximum auto-sized width; default 40
	WidthMin         int                  // minimum auto-sized width; default 10
}

// DefaultStyle returns the default bar [Style].
// It uses box-drawing characters with half-cell resolution for smooth progress.
func DefaultStyle() Style { return Thin }

// DefaultGradient returns the default red → yellow → green gradient
// used for [Style.ProgressGradient].
func DefaultGradient() []gradient.ColorStop { return style.DefaultPercentGradient() }

// State holds progress information passed to [Widget] functions.
type State struct {
	Current int
	Elapsed time.Duration
	Rate    float64 // items per second (0 when elapsed or current is 0)
	Total   int
}

// Widget renders a text label from the current bar state.
// Return "" to display nothing for this tick.
type Widget func(State) string

// ShowPending reports whether the bar block should be shown at the current progress.
func ShowPending(s Style, current int) bool {
	return s.PendingMode != PendingHide || current > 0
}

// Render renders the visual bar string for the given progress values.
// termWidth is the terminal column count (0 = fall back to auto-sizing from style).
func Render(current, total int, s Style, termWidth int) string {
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}

	filledChar := s.CharFill
	if filledChar == 0 {
		filledChar = '━'
	}
	charEmpty := s.CharEmpty
	if charEmpty == 0 {
		charEmpty = '─'
	}

	innerWidth := resolveWidth(s, termWidth)

	// Compute filled/empty counts and boundary characters.
	var filledCount, emptyCount int
	var headStr, trailStr string

	switch {
	case len(s.GradientFill) > 0:
		// Nx sub-cell resolution.
		subUnits := len(s.GradientFill) + 1
		completeParts := min(
			innerWidth*subUnits,
			int(float64(innerWidth)*float64(subUnits)*float64(current)/float64(total)),
		)
		filledCount = completeParts / subUnits
		remainder := completeParts % subUnits
		emptyCount = innerWidth - filledCount
		if remainder > 0 {
			headStr = string(s.GradientFill[remainder-1])
			emptyCount--
		}
	case s.HalfFilled != 0:
		// Half-cell (2x) resolution.
		completeHalves := min(
			innerWidth*2, //nolint:mnd // 2x resolution
			int(float64(innerWidth)*2*float64(current)/float64(total)),
		)
		filledCount = completeHalves / 2 //nolint:mnd // halves to cells
		emptyCount = innerWidth - filledCount
		if completeHalves%2 == 1 {
			headStr = string(s.HalfFilled)
			emptyCount--
		} else if filledCount > 0 && emptyCount > 0 && s.HalfEmpty != 0 {
			trailStr = string(s.HalfEmpty)
			emptyCount--
		}
	default:
		// Full-cell (1x) resolution.
		filledCount = min(innerWidth, int(float64(current)/float64(total)*float64(innerWidth)))
		emptyCount = innerWidth - filledCount
		if s.CharHead != 0 && filledCount > 0 && filledCount < innerWidth {
			headStr = string(s.CharHead)
			filledCount--
		}
	}

	filledStr := strings.Repeat(string(filledChar), filledCount)
	emptyStr := strings.Repeat(string(charEmpty), emptyCount)

	// When ProgressGradient is set, compute a single color from the gradient
	// at the current progress position and use it for filled cells.
	styleFill := s.StyleFill
	if len(s.ProgressGradient) > 0 {
		progress := float64(current) / float64(total)
		c := gradient.Interpolate(progress, s.ProgressGradient)
		ls := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
		styleFill = &ls
	}

	var buf strings.Builder
	writeStyled(&buf, s.CapLeft, s.CapStyle)
	writeStyled(&buf, filledStr, styleFill)
	writeStyled(&buf, headStr, styleFill)
	writeStyled(&buf, trailStr, s.StyleEmpty)
	writeStyled(&buf, emptyStr, s.StyleEmpty)
	writeStyled(&buf, s.CapRight, s.CapStyle)
	return buf.String()
}

// writeStyled writes s to buf with an optional lipgloss style.
func writeStyled(buf *strings.Builder, s string, st *lipgloss.Style) {
	if s == "" {
		return
	}
	if st != nil {
		buf.WriteString(st.Render(s))
	} else {
		buf.WriteString(s)
	}
}

// FormatPercent formats the percentage string for display alongside the bar.
// digits controls decimal places (0 → "50%", 1 → "50%" or "42.5%").
// Trailing decimal zeros are always stripped ("50.0%" → "50%").
// When pad is true, the result is right-aligned to the width of "100%"
// at the given digit count for stable display.
func FormatPercent(current, total, digits int, pad bool) string {
	var pct float64
	if total > 0 {
		pct = float64(current) / float64(total) * PercentDisplayMax
		if pct > PercentDisplayMax {
			pct = PercentDisplayMax
		}
	}
	s := numfmt.TrimDecimalZeros(fmt.Sprintf("%.*f", digits, pct)) + "%"
	if pad {
		padWidth := len(fmt.Sprintf("%.*f%%", digits, PercentDisplayMax))
		return fmt.Sprintf("%*s", padWidth, s)
	}
	return s
}

// resolveWidth computes the inner cell count for the bar from the style
// and the terminal width. A fixed Width takes priority; otherwise the width
// is derived from termWidth and clamped to [WidthMin, WidthMax].
func resolveWidth(s Style, termWidth int) int {
	if s.Width > 0 {
		return s.Width
	}

	minW := s.WidthMin
	if minW <= 0 {
		minW = DefaultWidthMin
	}
	maxW := s.WidthMax
	if maxW <= 0 {
		maxW = DefaultWidthMax
	}

	w := minW
	if termWidth > 0 {
		w = termWidth / WidthDivisor
	}

	return max(minW, min(maxW, w))
}

// FormatLine positions barPart relative to msgParts according to the
// placement mode and terminal width. sep is the fallback separator used
// when the terminal is too narrow for padding.
func FormatLine(msgParts, barPart, sep string, placement Placement, tw int) string {
	switch placement {
	case PlaceRightPad:
		gap := tw - lipgloss.Width(msgParts) - lipgloss.Width(barPart)
		if gap > 0 {
			return msgParts + strings.Repeat(" ", gap) + barPart
		}
		return msgParts + sep + barPart
	case PlaceLeftPad:
		gap := tw - lipgloss.Width(barPart) - lipgloss.Width(msgParts)
		if gap > 0 {
			return barPart + strings.Repeat(" ", gap) + msgParts
		}
		return barPart + sep + msgParts
	case PlaceRight:
		return msgParts + sep + barPart
	case PlaceLeft:
		return barPart + sep + msgParts
	case PlaceInline:
		return msgParts
	default:
		return msgParts
	}
}

// PercentValue computes the clamped percentage as a float64.
func PercentValue(current, total int) float64 {
	if total <= 0 {
		return 0
	}
	pct := float64(current) / float64(total) * PercentDisplayMax
	return min(pct, PercentDisplayMax)
}
