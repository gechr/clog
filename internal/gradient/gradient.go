// Package gradient provides color gradient interpolation shared by the style
// and fx packages. It lives under internal/ so that fx subpackages can use
// gradient math without importing the style package.
package gradient

import "github.com/lucasb-eyer/go-colorful"

// ColorStop defines a color at a specific position along a gradient.
// Position is in the range 0.0-1.0.
type ColorStop struct {
	Position float64        // 0.0-1.0
	Color    colorful.Color // from github.com/lucasb-eyer/go-colorful
}

// Interpolate computes the color at position t (0.0-1.0) along the
// given gradient stops using CIE-LCh blending for perceptually uniform
// transitions. Edge cases: empty -> white, single stop -> that color,
// t outside range -> clamp to nearest stop.
func Interpolate(t float64, stops []ColorStop) colorful.Color {
	if len(stops) == 0 {
		return colorful.Color{R: 1, G: 1, B: 1} // white fallback
	}

	if len(stops) == 1 {
		return stops[0].Color
	}

	// Clamp t to the range of the gradient.
	if t <= stops[0].Position {
		return stops[0].Color
	}

	if t >= stops[len(stops)-1].Position {
		return stops[len(stops)-1].Color
	}

	// Find the two bracketing stops.
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].Position {
			segLen := stops[i].Position - stops[i-1].Position
			if segLen <= 0 {
				return stops[i].Color
			}

			localT := (t - stops[i-1].Position) / segLen
			return stops[i-1].Color.BlendLuvLCh(stops[i].Color, localT)
		}
	}
	return stops[len(stops)-1].Color
}
