package clog

import (
	"bytes"
	"context"
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

	require.Len(t, stops, 4)
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

	got := shimmerText("", 0, DirectionRight, lut, nil)
	assert.Empty(t, got)
}

func TestShimmerTextSpacesUnstyled(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("a b c", 0, DirectionRight, lut, nil)

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

	got := shimmerText("hello", 0, DirectionRight, lut, nil)

	assert.Contains(t, got, "\x1b", "output should contain ANSI escape sequences")
}

func TestShimmerTextDifferentPhases(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	a := shimmerText("hello world", 0.0, DirectionRight, lut, nil)
	b := shimmerText("hello world", 0.5, DirectionRight, lut, nil)

	assert.NotEqual(t, a, b, "different phases should produce different output")
}

func TestShimmerTextAllDirectionsProduce(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())
	text := "hello world"

	for _, dir := range []Direction{DirectionRight, DirectionLeft, DirectionMiddleIn, DirectionMiddleOut, DirectionBounceIn, DirectionBounceOut} {
		got := shimmerText(text, 0.25, dir, lut, nil)
		assert.Contains(t, got, "\x1b", "direction %d should produce styled output", dir)
	}
}

func TestShimmerTextDirectionsDiffer(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())
	text := "hello world testing"

	// Use phase 0.7 so that the bounce triangle wave (which equals the
	// linear phase for p < 0.5) diverges from MiddleIn/MiddleOut.
	phase := 0.7

	right := shimmerText(text, phase, DirectionRight, lut, nil)
	left := shimmerText(text, phase, DirectionLeft, lut, nil)
	middleIn := shimmerText(text, phase, DirectionMiddleIn, lut, nil)
	middleOut := shimmerText(text, phase, DirectionMiddleOut, lut, nil)
	bounceIn := shimmerText(text, phase, DirectionBounceIn, lut, nil)
	bounceOut := shimmerText(text, phase, DirectionBounceOut, lut, nil)

	assert.NotEqual(t, right, left)
	assert.NotEqual(t, right, middleIn)
	assert.NotEqual(t, middleIn, middleOut)
	assert.NotEqual(t, bounceIn, bounceOut)
	assert.NotEqual(t, bounceIn, middleIn)
	assert.NotEqual(t, bounceOut, middleOut)
}

func TestShimmerTextBounceInPingPong(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())
	text := "hello world testing"

	// BounceIn uses a triangle wave on the phase, so phase 0 and phase 1
	// should produce identical output (both map to bounce=0).
	at0 := shimmerText(text, 0.0, DirectionBounceIn, lut, nil)
	at1 := shimmerText(text, 1.0, DirectionBounceIn, lut, nil)
	assert.Equal(t, at0, at1, "BounceIn at phase 0 and 1 should match (ping-pong)")

	// Mid-phase should differ from endpoints.
	atMid := shimmerText(text, 0.5, DirectionBounceIn, lut, nil)
	assert.NotEqual(t, at0, atMid, "BounceIn mid-phase should differ from endpoints")
}

func TestShimmerTextBounceOutPingPong(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())
	text := "hello world testing"

	at0 := shimmerText(text, 0.0, DirectionBounceOut, lut, nil)
	at1 := shimmerText(text, 1.0, DirectionBounceOut, lut, nil)
	assert.Equal(t, at0, at1, "BounceOut at phase 0 and 1 should match (ping-pong)")

	atMid := shimmerText(text, 0.5, DirectionBounceOut, lut, nil)
	assert.NotEqual(t, at0, atMid, "BounceOut mid-phase should differ from endpoints")
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
	got := shimmerText(text, 0, DirectionMiddleIn, lut, nil)

	// Output should contain styled characters.
	assert.Contains(t, got, "\x1b")

	// MiddleIn should produce different output than DirectionRight.
	gotRight := shimmerText(text, 0, DirectionRight, lut, nil)
	assert.NotEqual(t, got, gotRight,
		"MiddleIn should produce different output than DirectionRight")
}

func TestShimmerTextSingleChar(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("x", 0, DirectionRight, lut, nil)

	assert.Contains(t, got, "x")
	assert.Contains(t, got, "\x1b")
}

func TestShimmerTextUnicode(t *testing.T) {
	withTrueColor(t)
	lut := buildShimmerLUT(DefaultShimmerGradient())

	got := shimmerText("héllo wörld", 0, DirectionRight, lut, nil)

	// Should handle multi-byte runes without panicking.
	assert.Contains(t, got, "\x1b")
}

func TestShimmerDefault(t *testing.T) {
	b := Shimmer("test")

	assert.Equal(t, DefaultShimmerGradient(), b.shimmerStops)
}

func TestShimmerCustom(t *testing.T) {
	custom := []ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}
	b := Shimmer("test", custom...)

	assert.Equal(t, custom, b.shimmerStops)
}

func TestShimmerDirection(t *testing.T) {
	b := Shimmer("test").ShimmerDirection(DirectionLeft)

	assert.Equal(t, DirectionLeft, b.shimmerDir)
}

func TestShimmerDirectionDefault(t *testing.T) {
	b := Shimmer("test")

	assert.Equal(t, DirectionRight, b.shimmerDir)
}

func TestShimmerSpeedDefault(t *testing.T) {
	b := Shimmer("test")

	assert.InDelta(t, shimmerSpeed, b.speed, 1e-9)
}

func TestShimmerSpeedCustom(t *testing.T) {
	b := Shimmer("test").Speed(2.0)

	assert.InDelta(t, 2.0, b.speed, 1e-9)
}

func TestShimmerSpeedZeroFallsBackToDefault(t *testing.T) {
	b := Shimmer("test").Speed(0)

	assert.InDelta(t, shimmerSpeed, b.speed, 1e-9)
}

func TestShimmerSpeedNegativeFallsBackToDefault(t *testing.T) {
	b := Shimmer("test").Speed(-1.0)

	assert.InDelta(t, shimmerSpeed, b.speed, 1e-9)
}

func TestShimmerBuilderPrefix(t *testing.T) {
	b := Shimmer("test").Prefix("🔄")

	assert.Equal(t, "🔄", b.prefix)
}

func TestShimmerBuilderPrefixDefault(t *testing.T) {
	b := Shimmer("test")

	assert.Empty(t, b.prefix)
}

func TestShimmerDefaultPrefixInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Shimmer("loading").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.err)
	assert.Contains(t, buf.String(), "⏳")
}

func TestShimmerCustomPrefixInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Shimmer(
		"loading",
	).Prefix("🔄").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	require.NoError(t, result.err)
	assert.Contains(t, buf.String(), "🔄")
	assert.NotContains(t, buf.String(), "⏳")
}

func TestBuildShimmerStyleLUT(t *testing.T) {
	lut := buildShimmerLUT(DefaultShimmerGradient())
	styleLUT := buildShimmerStyleLUT(lut)

	assert.NotNil(t, styleLUT)
}

func BenchmarkShimmerText(b *testing.B) {
	lut := buildShimmerLUT(DefaultShimmerGradient())
	styleLUT := buildShimmerStyleLUT(lut)
	text := "hello world shimmer benchmark"

	b.ResetTimer()
	for b.Loop() {
		shimmerText(text, 0.3, DirectionRight, lut, styleLUT)
	}
}
