package style_test

import (
	"testing"

	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
)

// colorEqual checks that two colorful.Color values are within tolerance.
func colorEqual(t *testing.T, want, got colorful.Color, tol float64, msg string) {
	t.Helper()
	assert.InDelta(t, want.R, got.R, tol, "%s: R channel", msg)
	assert.InDelta(t, want.G, got.G, tol, "%s: G channel", msg)
	assert.InDelta(t, want.B, got.B, tol, "%s: B channel", msg)
}

func TestInterpolateGradientEmptyStops(t *testing.T) {
	// Empty stops -> white fallback.
	got := style.InterpolateGradient(0.5, nil)
	colorEqual(t, colorful.Color{R: 1, G: 1, B: 1}, got, 0.001, "empty stops should return white")
}

func TestInterpolateGradientEmptySlice(t *testing.T) {
	got := style.InterpolateGradient(0.5, []style.ColorStop{})
	colorEqual(t, colorful.Color{R: 1, G: 1, B: 1}, got, 0.001, "empty slice should return white")
}

func TestInterpolateGradientSingleStop(t *testing.T) {
	blue := colorful.Color{R: 0, G: 0, B: 1}
	stops := []style.ColorStop{
		{Position: 0.5, Color: blue},
	}

	tests := []struct {
		name string
		t    float64
	}{
		{"at_stop", 0.5},
		{"below_stop", 0.0},
		{"above_stop", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := style.InterpolateGradient(tt.t, stops)
			colorEqual(t, blue, got, 0.001, "single stop should always return that color")
		})
	}
}

func TestInterpolateGradientClampLeft(t *testing.T) {
	// t <= stops[0].Position -> first color.
	red := colorful.Color{R: 1, G: 0, B: 0}
	green := colorful.Color{R: 0, G: 1, B: 0}
	stops := []style.ColorStop{
		{Position: 0.2, Color: red},
		{Position: 0.8, Color: green},
	}

	tests := []struct {
		name string
		t    float64
	}{
		{"at_first_stop", 0.2},
		{"below_first_stop", 0.0},
		{"well_below_first_stop", -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := style.InterpolateGradient(tt.t, stops)
			colorEqual(t, red, got, 0.001, "t <= first stop should clamp to first color")
		})
	}
}

func TestInterpolateGradientClampRight(t *testing.T) {
	// t >= stops[last].Position -> last color.
	red := colorful.Color{R: 1, G: 0, B: 0}
	green := colorful.Color{R: 0, G: 1, B: 0}
	stops := []style.ColorStop{
		{Position: 0.2, Color: red},
		{Position: 0.8, Color: green},
	}

	tests := []struct {
		name string
		t    float64
	}{
		{"at_last_stop", 0.8},
		{"above_last_stop", 1.0},
		{"well_above_last_stop", 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := style.InterpolateGradient(tt.t, stops)
			colorEqual(t, green, got, 0.001, "t >= last stop should clamp to last color")
		})
	}
}

func TestInterpolateGradientAtExactStopPosition(t *testing.T) {
	red := colorful.Color{R: 1, G: 0, B: 0}
	yellow := colorful.Color{R: 1, G: 1, B: 0}
	green := colorful.Color{R: 0, G: 1, B: 0}
	stops := style.DefaultPercentGradient() // red(0), yellow(0.5), green(1.0)

	t.Run("at_zero", func(t *testing.T) {
		got := style.InterpolateGradient(0.0, stops)
		colorEqual(t, red, got, 0.001, "t=0 should give red")
	})

	t.Run("at_half", func(t *testing.T) {
		got := style.InterpolateGradient(0.5, stops)
		// BlendLuvLCh at localT=1.0 gives exactly the second stop.
		colorEqual(t, yellow, got, 0.05, "t=0.5 should give approximately yellow")
	})

	t.Run("at_one", func(t *testing.T) {
		got := style.InterpolateGradient(1.0, stops)
		colorEqual(t, green, got, 0.001, "t=1.0 should give green")
	})
}

func TestInterpolateGradientBetweenStops(t *testing.T) {
	// Two stops: black at 0.0, white at 1.0.
	// At t=0.5 the result should be roughly mid-grey (R, G, B all ~0.5 ± tolerance).
	black := colorful.Color{R: 0, G: 0, B: 0}
	white := colorful.Color{R: 1, G: 1, B: 1}
	stops := []style.ColorStop{
		{Position: 0.0, Color: black},
		{Position: 1.0, Color: white},
	}

	got := style.InterpolateGradient(0.5, stops)

	// BlendLuvLCh is perceptually uniform; assert that all channels are
	// between 0 and 1 (i.e. a valid intermediate color).
	assert.GreaterOrEqual(t, got.R, 0.0, "R should be >= 0")
	assert.LessOrEqual(t, got.R, 1.0, "R should be <= 1")
	assert.GreaterOrEqual(t, got.G, 0.0, "G should be >= 0")
	assert.LessOrEqual(t, got.G, 1.0, "G should be <= 1")
	assert.GreaterOrEqual(t, got.B, 0.0, "B should be >= 0")
	assert.LessOrEqual(t, got.B, 1.0, "B should be <= 1")

	// Ensure it's neither pure black nor pure white.
	isBlack := got.R < 0.01 && got.G < 0.01 && got.B < 0.01
	isWhite := got.R > 0.99 && got.G > 0.99 && got.B > 0.99
	assert.False(t, isBlack, "midpoint should not be black")
	assert.False(t, isWhite, "midpoint should not be white")
}

func TestInterpolateGradientDefaultPercentStops(t *testing.T) {
	// Validate the default percent gradient: red(0) -> yellow(0.5) -> green(1.0).
	stops := style.DefaultPercentGradient()
	assert.Len(t, stops, 3, "DefaultPercentGradient should have 3 stops")

	// At t=0.0: should be close to red.
	at0 := style.InterpolateGradient(0.0, stops)
	assert.InDelta(t, 1.0, at0.R, 0.01, "t=0 R should be ~1 (red)")
	assert.InDelta(t, 0.0, at0.G, 0.01, "t=0 G should be ~0")
	assert.InDelta(t, 0.0, at0.B, 0.01, "t=0 B should be ~0")

	// At t=1.0: should be close to green.
	at1 := style.InterpolateGradient(1.0, stops)
	assert.InDelta(t, 0.0, at1.R, 0.01, "t=1 R should be ~0")
	assert.InDelta(t, 1.0, at1.G, 0.01, "t=1 G should be ~1 (green)")
	assert.InDelta(t, 0.0, at1.B, 0.01, "t=1 B should be ~0")

	// At t=0.25 (between red and yellow): G should be > 0, R close to 1.
	at025 := style.InterpolateGradient(0.25, stops)
	assert.Greater(t, at025.R, 0.0, "t=0.25 R should be > 0 (toward red)")
	assert.Greater(t, at025.G, 0.0, "t=0.25 G should be > 0 (toward yellow)")

	// At t=0.75 (between yellow and green): G should be > 0, R decreasing.
	at075 := style.InterpolateGradient(0.75, stops)
	assert.Greater(t, at075.G, 0.0, "t=0.75 G should be > 0 (toward green)")
}

func TestStepGradientEmptyStops(t *testing.T) {
	got := style.StepGradient(0.5, nil)
	colorEqual(t, colorful.Color{R: 1, G: 1, B: 1}, got, 0.001, "empty stops should return white")
}

func TestStepGradientSingleStop(t *testing.T) {
	blue := colorful.Color{R: 0, G: 0, B: 1}
	stops := []style.ColorStop{
		{Position: 0.5, Color: blue},
	}

	for _, tt := range []float64{0.0, 0.5, 1.0} {
		got := style.StepGradient(tt, stops)
		colorEqual(t, blue, got, 0.001, "single stop should always return that color")
	}
}

func TestStepGradientDiscreteJumps(t *testing.T) {
	green := colorful.Color{R: 0, G: 1, B: 0}
	yellow := colorful.Color{R: 1, G: 1, B: 0}
	red := colorful.Color{R: 1, G: 0, B: 0}
	stops := []style.ColorStop{
		{Position: 0.0, Color: green},
		{Position: 0.5, Color: yellow},
		{Position: 1.0, Color: red},
	}

	// Before second stop → green.
	colorEqual(t, green, style.StepGradient(0.0, stops), 0.001, "t=0.0 should be green")
	colorEqual(t, green, style.StepGradient(0.3, stops), 0.001, "t=0.3 should be green")
	colorEqual(t, green, style.StepGradient(0.49, stops), 0.001, "t=0.49 should be green")

	// At and after second stop → yellow.
	colorEqual(t, yellow, style.StepGradient(0.5, stops), 0.001, "t=0.5 should be yellow")
	colorEqual(t, yellow, style.StepGradient(0.7, stops), 0.001, "t=0.7 should be yellow")
	colorEqual(t, yellow, style.StepGradient(0.99, stops), 0.001, "t=0.99 should be yellow")

	// At last stop → red.
	colorEqual(t, red, style.StepGradient(1.0, stops), 0.001, "t=1.0 should be red")
}

func TestInterpolateGradientTwoStopsFullRange(t *testing.T) {
	red := colorful.Color{R: 1, G: 0, B: 0}
	green := colorful.Color{R: 0, G: 1, B: 0}
	stops := []style.ColorStop{
		{Position: 0.0, Color: red},
		{Position: 1.0, Color: green},
	}

	// Spot-check a few positions to confirm monotonic behaviour.
	prev := style.InterpolateGradient(0.0, stops)
	for _, pos := range []float64{0.1, 0.25, 0.5, 0.75, 0.9, 1.0} {
		got := style.InterpolateGradient(pos, stops)
		// R should be decreasing (or at least not jumping up drastically).
		assert.LessOrEqual(t, got.R, prev.R+0.05, "R should not increase sharply at t=%.2f", pos)
		prev = got
	}
}
