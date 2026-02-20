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

// shimmerText renders each character of text with a gradient-interpolated
// foreground color, creating an animated shimmer when called with advancing
// phase values. Spaces are passed through unstyled.
func shimmerText(text string, phase float64, stops []ColorStop, dir Direction) string {
	runes := []rune(text)
	n := len(runes)
	if n == 0 {
		return text
	}

	var buf strings.Builder
	for i, r := range runes {
		if r == ' ' {
			buf.WriteRune(r)
			continue
		}

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

		c := interpolateGradient(t, stops)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Clamped().Hex()))
		buf.WriteString(style.Render(string(r)))
	}
	return buf.String()
}
