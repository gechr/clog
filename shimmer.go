package clog

import (
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// Direction controls which way an animation travels.
type Direction int

const (
	// DirectionRight moves the shimmer wave from left to right (default).
	DirectionRight Direction = iota
	// DirectionLeft moves the shimmer wave from right to left.
	DirectionLeft
	// DirectionMiddleIn sends the shimmer wave inward from both edges.
	DirectionMiddleIn
	// DirectionMiddleOut sends the shimmer wave outward from the center.
	DirectionMiddleOut
)

const (
	// shimmerSpeed is the number of full gradient cycles per second.
	shimmerSpeed = 0.5

	// shimmerTickRate is the repaint interval when shimmer is active (~30fps).
	shimmerTickRate = 33 * time.Millisecond
)

// DefaultShimmerGradient returns a wave-shaped gradient for shimmer effects:
// a muted blue-gray base with a narrow bright blue-white highlight band.
// The gradient is symmetric so it wraps seamlessly.
func DefaultShimmerGradient() []ColorStop {
	baseR, baseG, baseB := 0.8, 0.2, 0.0
	peakR, peakG, peakB := 1.0, 0.9, 0.2
	rampStart, peakPos, rampEnd := 0.35, 0.5, 0.65

	base := colorful.Color{R: baseR, G: baseG, B: baseB}
	peak := colorful.Color{R: peakR, G: peakG, B: peakB}
	return []ColorStop{
		{Position: 0, Color: base},
		{Position: rampStart, Color: base},
		{Position: peakPos, Color: peak},
		{Position: rampEnd, Color: base},
		{Position: 1.0, Color: base},
	}
}

const shimmerLUTSize = 64

// shimmerLUT is a pre-computed gradient lookup table of hex color strings.
type shimmerLUT [shimmerLUTSize]string

// buildShimmerLUT pre-computes a gradient lookup table of hex color strings
// from the given color stops. The LUT is phase-independent and can be reused
// across frames.
func buildShimmerLUT(stops []ColorStop) *shimmerLUT {
	var lut shimmerLUT
	for i := range lut {
		t := float64(i) / float64(shimmerLUTSize-1)
		//nolint:gosec // i is bounded by range lut
		lut[i] = interpolateGradient(
			t,
			stops,
		).Clamped().
			Hex()
	}
	return &lut
}

// shimmerText renders each character of text with a gradient-interpolated
// foreground color, creating an animated shimmer when called with advancing
// phase values. Spaces are passed through unstyled. The caller must supply a
// pre-built LUT from buildShimmerLUT.
func shimmerText(text string, phase float64, dir Direction, lut *shimmerLUT) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return text
	}

	// Map each character position to a LUT index, then batch adjacent
	// characters that share the same hex color into a single style.Render
	// call. This reduces style creations from ~N/frame to ~5-10/frame.
	var buf strings.Builder
	runHex := ""
	runStart := 0
	runIsSpace := runes[0] == ' '

	flushRun := func(end int) {
		run := string(runes[runStart:end])
		if runIsSpace {
			buf.WriteString(run)
		} else {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color(runHex))
			buf.WriteString(style.Render(run))
		}
	}

	if !runIsSpace {
		runHex = shimmerCharHex(0, n, phase, dir, lut)
	}

	for i := 1; i <= n; i++ {
		atEnd := i == n
		curIsSpace := !atEnd && runes[i] == ' '

		if atEnd {
			flushRun(i)
			break
		}

		var curHex string
		if !curIsSpace {
			curHex = shimmerCharHex(i, n, phase, dir, lut)
		}

		// Start a new run when transitioning between space/non-space or
		// when the quantized color changes.
		if curIsSpace != runIsSpace || (!curIsSpace && curHex != runHex) {
			flushRun(i)
			runStart = i
			runIsSpace = curIsSpace
			runHex = curHex
		}
	}

	return buf.String()
}

// shimmerCharHex returns the hex color string for character at index i of n
// characters, given the animation phase and direction, using the pre-computed LUT.
func shimmerCharHex(i, n int, phase float64, dir Direction, lut *shimmerLUT) string {
	pos := float64(i) / float64(n)

	var t float64
	switch dir {
	case DirectionLeft:
		t = math.Mod(pos+phase, 1.0)
	case DirectionMiddleIn:
		fold := math.Abs(2*pos - 1.0)
		t = math.Mod(fold+phase, 1.0)
	case DirectionMiddleOut:
		fold := 1.0 - math.Abs(2*pos-1.0)
		t = math.Mod(fold+phase, 1.0)
	case DirectionRight:
		t = math.Mod(pos-phase+1.0, 1.0)
	}

	idx := int(t * float64(shimmerLUTSize-1))
	if idx >= shimmerLUTSize {
		idx = shimmerLUTSize - 1
	}
	if idx < 0 {
		idx = 0
	}

	return lut[idx]
}
