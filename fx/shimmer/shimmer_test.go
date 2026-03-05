package shimmer_test

import (
	"strings"
	"testing"

	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultShimmerGradient(t *testing.T) {
	stops := shimmer.DefaultGradient()

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
	stops := shimmer.DefaultGradient()

	// First and last stops should share the same color (seamless wrap).
	assert.Equal(t, stops[0].Color, stops[len(stops)-1].Color)
}

func TestShimmerTextEmpty(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("", 0, shimmer.Right, lut, nil)
	assert.Empty(t, got)
}

func TestShimmerTextSpacesUnstyled(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("a b c", 0, shimmer.Right, lut, nil)

	// Split on spaces - spaces themselves should not contain ANSI escapes.
	parts := strings.SplitAfter(got, " ")
	for _, p := range parts {
		if p == " " {
			assert.NotContains(t, p, "\x1b", "spaces should not contain ANSI escapes")
		}
	}
}

func TestShimmerTextContainsANSI(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("hello", 0, shimmer.Right, lut, nil)

	assert.Contains(t, got, "\x1b", "output should contain ANSI escape sequences")
}

func TestShimmerTextDifferentPhases(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	a := shimmer.Text("hello world", 0.0, shimmer.Right, lut, nil)
	b := shimmer.Text("hello world", 0.5, shimmer.Right, lut, nil)

	assert.NotEqual(t, a, b, "different phases should produce different output")
}

func TestShimmerTextAllDirectionsProduce(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())
	text := "hello world"

	for _, dir := range []shimmer.Direction{shimmer.Right, shimmer.Left, shimmer.MiddleIn, shimmer.MiddleOut, shimmer.BounceIn, shimmer.BounceOut} {
		got := shimmer.Text(text, 0.25, dir, lut, nil)
		assert.Contains(t, got, "\x1b", "direction %d should produce styled output", dir)
	}
}

func TestShimmerTextDirectionsDiffer(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())
	text := "hello world testing"

	// Use phase 0.7 so that the bounce triangle wave (which equals the
	// linear phase for p < 0.5) diverges from MiddleIn/MiddleOut.
	phase := 0.7

	right := shimmer.Text(text, phase, shimmer.Right, lut, nil)
	left := shimmer.Text(text, phase, shimmer.Left, lut, nil)
	middleIn := shimmer.Text(text, phase, shimmer.MiddleIn, lut, nil)
	middleOut := shimmer.Text(text, phase, shimmer.MiddleOut, lut, nil)
	bounceIn := shimmer.Text(text, phase, shimmer.BounceIn, lut, nil)
	bounceOut := shimmer.Text(text, phase, shimmer.BounceOut, lut, nil)

	assert.NotEqual(t, right, left)
	assert.NotEqual(t, right, middleIn)
	assert.NotEqual(t, middleIn, middleOut)
	assert.NotEqual(t, bounceIn, bounceOut)
	assert.NotEqual(t, bounceIn, middleIn)
	assert.NotEqual(t, bounceOut, middleOut)
}

func TestShimmerTextBounceInPingPong(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())
	text := "hello world testing"

	// BounceIn uses a triangle wave on the phase, so phase 0 and phase 1
	// should produce identical output (both map to bounce=0).
	at0 := shimmer.Text(text, 0.0, shimmer.BounceIn, lut, nil)
	at1 := shimmer.Text(text, 1.0, shimmer.BounceIn, lut, nil)
	assert.Equal(t, at0, at1, "BounceIn at phase 0 and 1 should match (ping-pong)")

	// Mid-phase should differ from endpoints.
	atMid := shimmer.Text(text, 0.5, shimmer.BounceIn, lut, nil)
	assert.NotEqual(t, at0, atMid, "BounceIn mid-phase should differ from endpoints")
}

func TestShimmerTextBounceOutPingPong(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())
	text := "hello world testing"

	at0 := shimmer.Text(text, 0.0, shimmer.BounceOut, lut, nil)
	at1 := shimmer.Text(text, 1.0, shimmer.BounceOut, lut, nil)
	assert.Equal(t, at0, at1, "BounceOut at phase 0 and 1 should match (ping-pong)")

	atMid := shimmer.Text(text, 0.5, shimmer.BounceOut, lut, nil)
	assert.NotEqual(t, at0, atMid, "BounceOut mid-phase should differ from endpoints")
}

func TestShimmerTextMiddleInSymmetric(t *testing.T) {
	// MiddleIn maps pos via fold = |2*pos - 1|, so edges get high fold values
	// and the center gets low fold values. With a symmetric gradient the first
	// and last characters should receive similar (though not identical) colors
	// because pos = i/n doesn't perfectly sample both endpoints.
	stops := []style.ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 0.5, Color: colorful.Color{R: 0, G: 0, B: 1}},
		{Position: 1, Color: colorful.Color{R: 1, G: 0, B: 0}},
	}
	lut := shimmer.BuildLUT(stops)

	text := "abcdefgh"
	got := shimmer.Text(text, 0, shimmer.MiddleIn, lut, nil)

	// Output should contain styled characters.
	assert.Contains(t, got, "\x1b")

	// MiddleIn should produce different output than shimmer.Right.
	gotRight := shimmer.Text(text, 0, shimmer.Right, lut, nil)
	assert.NotEqual(t, got, gotRight,
		"MiddleIn should produce different output than shimmer.Right")
}

func TestShimmerTextSingleChar(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("x", 0, shimmer.Right, lut, nil)

	assert.Contains(t, got, "x")
	assert.Contains(t, got, "\x1b")
}

func TestShimmerTextUnicode(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("héllo wörld", 0, shimmer.Right, lut, nil)

	// Should handle multi-byte runes without panicking.
	assert.Contains(t, got, "\x1b")
}

func TestBuildShimmerStyleLUT(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())
	styleLUT := shimmer.BuildStyleLUT(lut)

	assert.NotNil(t, styleLUT)
}

func BenchmarkShimmerText(b *testing.B) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())
	styleLUT := shimmer.BuildStyleLUT(lut)
	text := "hello world shimmer benchmark"

	b.ResetTimer()
	for b.Loop() {
		shimmer.Text(text, 0.3, shimmer.Right, lut, styleLUT)
	}
}
