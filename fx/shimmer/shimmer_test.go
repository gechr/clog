package shimmer_test

import (
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

	assert.Equal(
		t,
		"\x1b[38;2;255;179;179ma\x1b[m \x1b[38;2;144;251;211mb\x1b[m \x1b[38;2;232;185;248mc\x1b[m",
		got,
	)
}

func TestShimmerTextContainsANSI(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("hello", 0, shimmer.Right, lut, nil)

	assert.Equal(
		t,
		"\x1b[38;2;255;179;179mh\x1b[m\x1b[38;2;229;224;148me\x1b[m\x1b[38;2;144;251;211ml\x1b[m\x1b[38;2;144;222;253ml\x1b[m\x1b[38;2;232;185;248mo\x1b[m",
		got,
	)
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

	for dir, want := range map[shimmer.Direction]string{
		shimmer.Right:     "\x1b[38;2;214;192;254mh\x1b[m\x1b[38;2;242;181;241me\x1b[m\x1b[38;2;255;176;211ml\x1b[m\x1b[38;2;255;182;174ml\x1b[m\x1b[38;2;247;204;151mo\x1b[m \x1b[38;2;196;246;172mw\x1b[m\x1b[38;2;151;252;206mo\x1b[m\x1b[38;2;122;242;233mr\x1b[m\x1b[38;2;133;229;249ml\x1b[m\x1b[38;2;171;208;255md\x1b[m",
		shimmer.Left:      "\x1b[38;2;214;235;156mh\x1b[m\x1b[38;2;177;255;192me\x1b[m\x1b[38;2;132;248;220ml\x1b[m\x1b[38;2;122;237;240ml\x1b[m\x1b[38;2;150;219;254mo\x1b[m \x1b[38;2;232;185;248mw\x1b[m\x1b[38;2;252;177;227mo\x1b[m\x1b[38;2;255;177;192mr\x1b[m\x1b[38;2;253;192;161ml\x1b[m\x1b[38;2;238;216;147md\x1b[m",
		shimmer.MiddleIn:  "\x1b[38;2;214;235;156mh\x1b[m\x1b[38;2;253;192;161me\x1b[m\x1b[38;2;252;177;227ml\x1b[m\x1b[38;2;192;199;255ml\x1b[m\x1b[38;2;122;237;240mo\x1b[m \x1b[38;2;177;255;192mw\x1b[m\x1b[38;2;122;237;240mo\x1b[m\x1b[38;2;192;199;255mr\x1b[m\x1b[38;2;252;177;227ml\x1b[m\x1b[38;2;253;192;161md\x1b[m",
		shimmer.MiddleOut: "\x1b[38;2;214;235;156mh\x1b[m\x1b[38;2;132;248;220me\x1b[m\x1b[38;2;150;219;254ml\x1b[m\x1b[38;2;232;185;248ml\x1b[m\x1b[38;2;255;177;192mo\x1b[m \x1b[38;2;238;216;147mw\x1b[m\x1b[38;2;255;177;192mo\x1b[m\x1b[38;2;232;185;248mr\x1b[m\x1b[38;2;150;219;254ml\x1b[m\x1b[38;2;132;248;220md\x1b[m",
		shimmer.BounceIn:  "\x1b[38;2;159;253;201mh\x1b[m\x1b[38;2;229;224;148me\x1b[m\x1b[38;2;255;179;179ml\x1b[m\x1b[38;2;242;181;241ml\x1b[m\x1b[38;2;164;212;255mo\x1b[m \x1b[38;2;124;244;229mw\x1b[m\x1b[38;2;164;212;255mo\x1b[m\x1b[38;2;242;181;241mr\x1b[m\x1b[38;2;255;179;179ml\x1b[m\x1b[38;2;229;224;148md\x1b[m",
		shimmer.BounceOut: "\x1b[38;2;159;253;201mh\x1b[m\x1b[38;2;133;229;249me\x1b[m\x1b[38;2;207;194;255ml\x1b[m\x1b[38;2;255;176;217ml\x1b[m\x1b[38;2;250;200;154mo\x1b[m \x1b[38;2;202;242;166mw\x1b[m\x1b[38;2;250;200;154mo\x1b[m\x1b[38;2;255;176;217mr\x1b[m\x1b[38;2;207;194;255ml\x1b[m\x1b[38;2;133;229;249md\x1b[m",
	} {
		got := shimmer.Text(text, 0.25, dir, lut, nil)
		assert.Equal(t, want, got, "direction %d", dir)
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

	assert.Equal(
		t,
		"\x1b[38;2;255;0;0ma\x1b[m\x1b[38;2;239;0;206mb\x1b[m\x1b[38;2;51;0;255mc\x1b[m\x1b[38;2;242;0;199md\x1b[m\x1b[38;2;255;0;0me\x1b[m\x1b[38;2;242;0;199mf\x1b[m\x1b[38;2;51;0;255mg\x1b[m\x1b[38;2;239;0;206mh\x1b[m",
		got,
	)

	// MiddleIn should produce different output than shimmer.Right.
	gotRight := shimmer.Text(text, 0, shimmer.Right, lut, nil)
	assert.NotEqual(t, got, gotRight,
		"MiddleIn should produce different output than shimmer.Right")
}

func TestShimmerTextSingleChar(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("x", 0, shimmer.Right, lut, nil)

	assert.Equal(t, "\x1b[38;2;255;179;179mx\x1b[m", got)
}

func TestShimmerTextUnicode(t *testing.T) {
	lut := shimmer.BuildLUT(shimmer.DefaultGradient())

	got := shimmer.Text("héllo wörld", 0, shimmer.Right, lut, nil)

	assert.Equal(
		t,
		"\x1b[38;2;255;179;179mh\x1b[m\x1b[38;2;252;196;157mé\x1b[m\x1b[38;2;234;220;147ml\x1b[m\x1b[38;2;202;242;166ml\x1b[m\x1b[38;2;168;254;197mo\x1b[m \x1b[38;2;128;232;246mw\x1b[m\x1b[38;2;164;212;255mö\x1b[m\x1b[38;2;200;197;255mr\x1b[m\x1b[38;2;237;183;245ml\x1b[m\x1b[38;2;255;176;217md\x1b[m",
		got,
	)
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
