// Package shimmer provides shimmer animation rendering for clog.
package shimmer

import (
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog/internal/gradient"
	"github.com/lucasb-eyer/go-colorful"
)

// Direction controls which way an animation travels.
type Direction int

const (
	// Right moves the shimmer wave from left to right (default).
	Right Direction = iota
	// Left moves the shimmer wave from right to left.
	Left
	// MiddleIn sends the shimmer wave inward from both edges.
	MiddleIn
	// MiddleOut sends the shimmer wave outward from the center.
	MiddleOut
	// BounceIn sends the shimmer wave inward from both edges, then
	// bounces it back outward, creating a ping-pong effect.
	BounceIn
	// BounceOut sends the shimmer wave outward from the center, then
	// bounces it back inward, creating a ping-pong effect.
	BounceOut
)

const (
	// Speed is the number of full gradient cycles per second.
	Speed = 0.5

	// TickRate is the repaint interval when shimmer is active (~30fps).
	TickRate = 33 * time.Millisecond

	// lutSize is the number of entries in the pre-computed gradient lookup table.
	lutSize = 64
)

// LUT is a pre-computed gradient lookup table of hex color strings.
type LUT [lutSize]string

// StyleLUT is a pre-computed lookup table of lipgloss styles, built
// from a [LUT]. Reusing styles across frames eliminates per-frame
// style allocations entirely.
type StyleLUT [lutSize]lipgloss.Style

// Style holds resolved shimmer configuration.
type Style struct {
	Direction Direction
	Gradient  []gradient.ColorStop
	Speed     float64
}

// DefaultStyle returns the default shimmer configuration.
func DefaultStyle() Style {
	return Style{
		Direction: Right,
		Gradient:  DefaultGradient(),
		Speed:     Speed,
	}
}

// DefaultGradient returns a wave-shaped gradient for shimmer effects:
// a subtle red-to-green-to-blue cycle that wraps seamlessly.
func DefaultGradient() []gradient.ColorStop {
	rR, rG, rB := 1.0, 0.7, 0.7
	gR, gG, gB := 0.7, 1.0, 0.75
	bR, bG, bB := 0.7, 0.8, 1.0
	third, twoThirds := 0.33, 0.67
	red := colorful.Color{R: rR, G: rG, B: rB}
	green := colorful.Color{R: gR, G: gG, B: gB}
	blue := colorful.Color{R: bR, G: bG, B: bB}
	return []gradient.ColorStop{
		{Position: 0, Color: red},
		{Position: third, Color: green},
		{Position: twoThirds, Color: blue},
		{Position: 1.0, Color: red},
	}
}

// BuildLUT pre-computes a gradient lookup table of hex color strings
// from the given color stops.
func BuildLUT(stops []gradient.ColorStop) *LUT {
	var lut LUT
	for i := range lut {
		t := float64(i) / float64(lutSize-1)
		//nolint:gosec // i is bounded by range lut
		lut[i] = gradient.Interpolate(t, stops).Clamped().Hex()
	}
	return &lut
}

// BuildStyleLUT pre-computes a lipgloss style for every entry in the
// hex LUT. Call once after [BuildLUT] and pass the result to
// [Text] to avoid style allocations in the render loop. When r is
// non-nil, styles are bound to that renderer instead of the global default;
// pass nil to use [lipgloss.NewStyle].
func BuildStyleLUT(lut *LUT, r *lipgloss.Renderer) *StyleLUT {
	var s StyleLUT
	for i, hex := range lut {
		//nolint:gosec // i is bounded by range lut
		if r != nil {
			s[i] = r.NewStyle().Foreground(lipgloss.Color(hex))
		} else {
			s[i] = lipgloss.NewStyle().Foreground(lipgloss.Color(hex))
		}
	}
	return &s
}

// Text renders each character of text with a gradient-interpolated
// foreground color, creating an animated shimmer when called with advancing
// phase values. Spaces are passed through unstyled. The caller must supply a
// pre-built LUT from [BuildLUT]. Passing a pre-built [StyleLUT]
// from [BuildStyleLUT] eliminates per-frame style allocations; pass
// nil to create styles on the fly.
func Text(
	text string,
	phase float64,
	dir Direction,
	lut *LUT,
	styleLUT *StyleLUT,
) string {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return text
	}

	// Map each character position to a LUT index, then batch adjacent
	// characters that share the same index into a single style.Render call.
	var buf strings.Builder
	var (
		runByteStart int
		runIdx       int
		runIsSpace   bool
		charPos      int
	)

	flushRun := func(byteEnd int) {
		run := text[runByteStart:byteEnd]
		if runIsSpace {
			buf.WriteString(run)
		} else {
			var s lipgloss.Style
			if styleLUT != nil {
				s = styleLUT[runIdx]
			} else {
				s = lipgloss.NewStyle().Foreground(lipgloss.Color(lut[runIdx]))
			}
			buf.WriteString(s.Render(run))
		}
	}

	for byteIdx, r := range text {
		curIsSpace := unicode.IsSpace(r)
		var curIdx int
		if !curIsSpace {
			curIdx = charIdx(charPos, n, phase, dir)
		}

		if charPos == 0 {
			runIsSpace = curIsSpace
			runIdx = curIdx
		} else if curIsSpace != runIsSpace || (!curIsSpace && curIdx != runIdx) {
			flushRun(byteIdx)
			runByteStart = byteIdx
			runIsSpace = curIsSpace
			runIdx = curIdx
		}
		charPos++
	}
	// Flush final run.
	flushRun(len(text))

	return buf.String()
}

// charIdx returns the LUT index for character at position i of n,
// given the animation phase and direction.
func charIdx(i, n int, phase float64, dir Direction) int {
	pos := float64(i) / float64(n)

	var t float64
	switch dir {
	case Left:
		t = math.Mod(pos+phase, 1.0)
	case MiddleIn:
		fold := math.Abs(2*pos - 1.0)
		t = math.Mod(fold+phase, 1.0)
	case MiddleOut:
		fold := 1.0 - math.Abs(2*pos-1.0)
		t = math.Mod(fold+phase, 1.0)
	case Right:
		t = math.Mod(pos-phase+1.0, 1.0)
	case BounceIn:
		//nolint:mnd // triangle wave amplitude for ping-pong phase
		bounce := 0.75 * (1.0 - math.Abs(2*phase-1.0))
		fold := math.Abs(2*pos - 1.0)
		t = math.Mod(fold+bounce, 1.0)
	case BounceOut:
		//nolint:mnd // triangle wave amplitude for ping-pong phase
		bounce := 0.75 * (1.0 - math.Abs(2*phase-1.0))
		fold := 1.0 - math.Abs(2*pos-1.0)
		t = math.Mod(fold+bounce, 1.0)
	}

	idx := int(t * float64(lutSize-1))
	if idx >= lutSize {
		idx = lutSize - 1
	}
	if idx < 0 {
		idx = 0
	}
	return idx
}
