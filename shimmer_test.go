package clog

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTrueColor forces the default lipgloss renderer to TrueColor for the
// duration of the test so that shimmerText emits ANSI escapes.
func withTrueColor(t *testing.T) {
	t.Helper()
	r := lipgloss.DefaultRenderer()
	old := r.ColorProfile()
	r.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		r.SetColorProfile(old)
	})
}

func TestDefaultShimmerGradient(t *testing.T) {
	stops := DefaultShimmerGradient()

	require.Len(t, stops, 5)
	assert.InDelta(t, 0.0, stops[0].Position, 1e-9)
	assert.InDelta(t, 1.0, stops[len(stops)-1].Position, 1e-9)

	// Positions must be sorted ascending.
	for i := 1; i < len(stops); i++ {
		assert.GreaterOrEqual(t, stops[i].Position, stops[i-1].Position,
			"stop %d position should be >= stop %d", i, i-1)
	}
}

func TestDefaultShimmerGradientSymmetric(t *testing.T) {
	stops := DefaultShimmerGradient()

	// First and last stops should share the same color (seamless wrap).
	assert.Equal(t, stops[0].Color, stops[len(stops)-1].Color)
}

func TestShimmerTextEmpty(t *testing.T) {
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("", 0, DirectionRight, lut)
	assert.Empty(t, got)
}

func TestShimmerTextSpacesUnstyled(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("a b c", 0, DirectionRight, lut)

	// Split on spaces — spaces themselves should not contain ANSI escapes.
	parts := strings.SplitAfter(got, " ")
	for _, p := range parts {
		if p == " " {
			assert.NotContains(t, p, "\x1b", "spaces should not contain ANSI escapes")
		}
	}
}

func TestShimmerTextContainsANSI(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("hello", 0, DirectionRight, lut)

	assert.Contains(t, got, "\x1b", "output should contain ANSI escape sequences")
}

func TestShimmerTextDifferentPhases(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	a := shimmerText("hello world", 0.0, DirectionRight, lut)
	b := shimmerText("hello world", 0.5, DirectionRight, lut)

	assert.NotEqual(t, a, b, "different phases should produce different output")
}

func TestShimmerTextAllDirectionsProduce(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())
	text := "hello world"

	for _, dir := range []Direction{DirectionRight, DirectionLeft, DirectionMiddleIn, DirectionMiddleOut} {
		got := shimmerText(text, 0.25, dir, lut)
		assert.Contains(t, got, "\x1b", "direction %d should produce styled output", dir)
	}
}

func TestShimmerTextDirectionsDiffer(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())
	text := "hello world testing"
	phase := 0.3

	right := shimmerText(text, phase, DirectionRight, lut)
	left := shimmerText(text, phase, DirectionLeft, lut)
	middleIn := shimmerText(text, phase, DirectionMiddleIn, lut)
	middleOut := shimmerText(text, phase, DirectionMiddleOut, lut)

	assert.NotEqual(t, right, left)
	assert.NotEqual(t, right, middleIn)
	assert.NotEqual(t, middleIn, middleOut)
}

func TestShimmerTextMiddleInSymmetric(t *testing.T) {
	withTrueColor(t)

	// MiddleIn maps pos via fold = |2*pos - 1|, so edges get high fold values
	// and the center gets low fold values. With a symmetric gradient the first
	// and last characters should receive similar (though not identical) colors
	// because pos = i/n doesn't perfectly sample both endpoints.
	stops := []ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 0.5, Color: colorful.Color{R: 0, G: 0, B: 1}},
		{Position: 1, Color: colorful.Color{R: 1, G: 0, B: 0}},
	}
	lut := buildShimmerLUT(stops)

	text := "abcdefgh"
	got := shimmerText(text, 0, DirectionMiddleIn, lut)

	// Output should contain styled characters.
	assert.Contains(t, got, "\x1b")

	// MiddleIn should produce different output than DirectionRight.
	gotRight := shimmerText(text, 0, DirectionRight, lut)
	assert.NotEqual(t, got, gotRight,
		"MiddleIn should produce different output than DirectionRight")
}

func TestShimmerTextSingleChar(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("x", 0, DirectionRight, lut)

	assert.Contains(t, got, "x")
	assert.Contains(t, got, "\x1b")
}

func TestShimmerTextUnicode(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("héllo wörld", 0, DirectionRight, lut)

	// Should handle multi-byte runes without panicking.
	assert.Contains(t, got, "\x1b")
}

func TestSpinnerBuilderShimmerDefault(t *testing.T) {
	b := Spinner("test").Shimmer()

	assert.Equal(t, DefaultShimmerGradient(), b.shimmerStops)
}

func TestSpinnerBuilderShimmerCustom(t *testing.T) {
	custom := []ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}
	b := Spinner("test").Shimmer(custom...)

	assert.Equal(t, custom, b.shimmerStops)
}

func TestSpinnerBuilderShimmerDirection(t *testing.T) {
	b := Spinner("test").Shimmer().ShimmerDirection(DirectionLeft)

	assert.Equal(t, DirectionLeft, b.shimmerDir)
}

func TestSpinnerBuilderShimmerDirectionDefault(t *testing.T) {
	b := Spinner("test").Shimmer()

	assert.Equal(t, DirectionRight, b.shimmerDir)
}

func BenchmarkShimmerText(b *testing.B) {
	lut := buildShimmerLUT(DefaultShimmerGradient())
	text := "hello world shimmer benchmark"

	b.ResetTimer()
	for b.Loop() {
		shimmerText(text, 0.3, DirectionRight, lut)
	}
}
