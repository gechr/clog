package style

import (
	"github.com/gechr/clog/internal/gradient"
	"github.com/lucasb-eyer/go-colorful"
)

// InterpolateGradient computes the color at position t (0.0-1.0) along the
// given gradient stops using CIE-LCh blending for perceptually uniform
// transitions. Edge cases: empty -> white, single stop -> that color,
// t outside range -> clamp to nearest stop.
func InterpolateGradient(t float64, stops []ColorStop) colorful.Color {
	return gradient.Interpolate(t, stops)
}

// StepGradient returns the color of the last stop whose position is <= t.
// Edge cases: empty -> white, single stop -> that color, t before first stop ->
// first stop's color.
func StepGradient(t float64, stops []ColorStop) colorful.Color {
	if len(stops) == 0 {
		return colorful.Color{R: 1, G: 1, B: 1}
	}

	// Walk stops; pick the last one with Position <= t.
	best := stops[0]
	for _, s := range stops[1:] {
		if s.Position <= t {
			best = s
		}
	}
	return best.Color
}
