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
