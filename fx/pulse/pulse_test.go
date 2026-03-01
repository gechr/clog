package pulse_test

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog/fx/pulse"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withTrueColor forces the default lipgloss renderer to TrueColor for the
// duration of the test so that pulse.Text emits ANSI escapes.
func withTrueColor(t *testing.T) {
	t.Helper()
	r := lipgloss.DefaultRenderer()
	old := r.ColorProfile()
	r.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() {
		r.SetColorProfile(old)
	})
}

func TestDefaultPulseGradient(t *testing.T) {
	stops := pulse.DefaultGradient()

	require.Len(t, stops, 3)
	assert.InDelta(t, 0.0, stops[0].Position, 1e-9)
	assert.InDelta(t, 1.0, stops[len(stops)-1].Position, 1e-9)
}

func TestPulseTextEmpty(t *testing.T) {
	stops := pulse.DefaultGradient()

	got := pulse.Text("", 0, stops)
	assert.Empty(t, got)
}

func TestPulseTextSpacesUnstyled(t *testing.T) {
	withTrueColor(t)
	stops := pulse.DefaultGradient()

	got := pulse.Text("a b c", 0.5, stops)

	// Spaces themselves should not contain ANSI escapes.
	assert.Contains(t, got, " ")
}

func TestPulseTextContainsANSI(t *testing.T) {
	withTrueColor(t)
	stops := pulse.DefaultGradient()

	got := pulse.Text("hello", 0.5, stops)

	assert.Contains(t, got, "\x1b", "output should contain ANSI escape sequences")
}

func TestPulseTextDifferentPhases(t *testing.T) {
	withTrueColor(t)
	stops := pulse.DefaultGradient()

	a := pulse.Text("hello world", 0.0, stops)
	b := pulse.Text("hello world", 1.0, stops)

	assert.NotEqual(t, a, b, "different phases should produce different output")
}

func TestPulseTextUniformColor(t *testing.T) {
	withTrueColor(t)

	// All non-space characters should get the same color at a given phase.
	stops := []style.ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}

	got := pulse.Text("ab", 0.5, stops)

	// With shimmer, different positions get different colors.
	// With pulse, both characters should get the same style.
	lut := shimmer.BuildLUT(stops)
	shimmerGot := shimmer.Text("ab", 0.5, shimmer.Right, lut, nil)
	assert.NotEqual(t, got, shimmerGot,
		"pulse should differ from shimmer (uniform vs positional)")
}

func TestPulseTextSingleChar(t *testing.T) {
	withTrueColor(t)
	stops := pulse.DefaultGradient()

	got := pulse.Text("x", 0.5, stops)

	assert.Contains(t, got, "x")
	assert.Contains(t, got, "\x1b")
}

func TestPulseTextUnicode(t *testing.T) {
	withTrueColor(t)
	stops := pulse.DefaultGradient()

	got := pulse.Text("héllo wörld", 0.5, stops)

	assert.Contains(t, got, "\x1b")
}

func TestPulseTextCached(t *testing.T) {
	withTrueColor(t)
	stops := pulse.DefaultGradient()

	t.Run("non_empty_result", func(t *testing.T) {
		cache := &pulse.Cache{}

		got := pulse.TextCached("hello", 0.5, stops, cache, nil)

		assert.NotEmpty(t, got)
		assert.NotEmpty(t, cache.Hex)
	})

	t.Run("cache_hit_same_phase", func(t *testing.T) {
		cache := &pulse.Cache{}

		first := pulse.TextCached("hello", 0.5, stops, cache, nil)
		hexAfterFirst := cache.Hex

		second := pulse.TextCached("hello", 0.5, stops, cache, nil)

		assert.Equal(t, first, second)
		assert.Equal(t, hexAfterFirst, cache.Hex, "cache hex should not change on same phase")
	})

	t.Run("empty_text", func(t *testing.T) {
		cache := &pulse.Cache{}

		got := pulse.TextCached("", 0.5, stops, cache, nil)

		assert.Empty(t, got)
	})

	t.Run("cache_miss_different_phase", func(t *testing.T) {
		cache := &pulse.Cache{}

		pulse.TextCached("hello", 0.0, stops, cache, nil)
		hexFirst := cache.Hex

		pulse.TextCached("hello", 1.0, stops, cache, nil)
		hexSecond := cache.Hex

		assert.NotEqual(
			t,
			hexFirst,
			hexSecond,
			"different phases should produce different hex values",
		)
	})
}
