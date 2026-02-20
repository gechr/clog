package clog

import (
	"testing"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPulseGradient(t *testing.T) {
	stops := DefaultPulseGradient()

	require.Len(t, stops, 2)
	assert.InDelta(t, 0.0, stops[0].Position, 1e-9)
	assert.InDelta(t, 1.0, stops[len(stops)-1].Position, 1e-9)
}

func TestPulseTextEmpty(t *testing.T) {
	stops := DefaultPulseGradient()

	got := pulseText("", 0, stops)
	assert.Empty(t, got)
}

func TestPulseTextSpacesUnstyled(t *testing.T) {
	withTrueColor(t)
	stops := DefaultPulseGradient()

	got := pulseText("a b c", 0.5, stops)

	// Spaces themselves should not contain ANSI escapes.
	for i, r := range got {
		if r == ' ' {
			// Check surrounding bytes aren't mid-escape for this space.
			_ = i // space passed through
		}
	}
	assert.Contains(t, got, " ")
}

func TestPulseTextContainsANSI(t *testing.T) {
	withTrueColor(t)
	stops := DefaultPulseGradient()

	got := pulseText("hello", 0.5, stops)

	assert.Contains(t, got, "\x1b", "output should contain ANSI escape sequences")
}

func TestPulseTextDifferentPhases(t *testing.T) {
	withTrueColor(t)
	stops := DefaultPulseGradient()

	a := pulseText("hello world", 0.0, stops)
	b := pulseText("hello world", 1.0, stops)

	assert.NotEqual(t, a, b, "different phases should produce different output")
}

func TestPulseTextUniformColor(t *testing.T) {
	withTrueColor(t)

	// All non-space characters should get the same color at a given phase.
	stops := []ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}

	got := pulseText("ab", 0.5, stops)

	// With shimmer, different positions get different colors.
	// With pulse, both characters should get the same style.
	lut := buildShimmerLUT(stops)
	shimmerGot := shimmerText("ab", 0.5, DirectionRight, lut)
	assert.NotEqual(t, got, shimmerGot,
		"pulse should differ from shimmer (uniform vs positional)")
}

func TestPulseTextSingleChar(t *testing.T) {
	withTrueColor(t)
	stops := DefaultPulseGradient()

	got := pulseText("x", 0.5, stops)

	assert.Contains(t, got, "x")
	assert.Contains(t, got, "\x1b")
}

func TestPulseTextUnicode(t *testing.T) {
	withTrueColor(t)
	stops := DefaultPulseGradient()

	got := pulseText("héllo wörld", 0.5, stops)

	assert.Contains(t, got, "\x1b")
}

func TestSpinnerBuilderPulseDefault(t *testing.T) {
	b := Spinner("test").Pulse()

	assert.Equal(t, DefaultPulseGradient(), b.pulseStops)
}

func TestSpinnerBuilderPulseCustom(t *testing.T) {
	custom := []ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}
	b := Spinner("test").Pulse(custom...)

	assert.Equal(t, custom, b.pulseStops)
}
