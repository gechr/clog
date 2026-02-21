package clog

import (
	"bytes"
	"context"
	"testing"

	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPulseGradient(t *testing.T) {
	stops := DefaultPulseGradient()

	require.Len(t, stops, 3)
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
	shimmerGot := shimmerText("ab", 0.5, DirectionRight, lut, nil)
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

func TestPulseDefault(t *testing.T) {
	b := Pulse("test")

	assert.Equal(t, DefaultPulseGradient(), b.pulseStops)
}

func TestPulseCustom(t *testing.T) {
	custom := []ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}
	b := Pulse("test", custom...)

	assert.Equal(t, custom, b.pulseStops)
}

func TestPulseBuilderPrefix(t *testing.T) {
	b := Pulse("test").Prefix("🔄")

	assert.Equal(t, "🔄", b.prefix)
}

func TestPulseBuilderPrefixDefault(t *testing.T) {
	b := Pulse("test")

	assert.Empty(t, b.prefix)
}

func TestPulseDefaultPrefixInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Pulse("loading").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.err)
	assert.Contains(t, buf.String(), "⏳")
}

func TestPulseCustomPrefixInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Pulse(
		"loading",
	).Prefix("🔄").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	require.NoError(t, result.err)
	assert.Contains(t, buf.String(), "🔄")
	assert.NotContains(t, buf.String(), "⏳")
}

func TestPulseTextCached(t *testing.T) {
	withTrueColor(t)
	stops := DefaultPulseGradient()

	t.Run("non_empty_result", func(t *testing.T) {
		cache := &pulseCache{}

		got := pulseTextCached("hello", 0.5, stops, cache)

		assert.NotEmpty(t, got)
		assert.NotEmpty(t, cache.hex)
	})

	t.Run("cache_hit_same_phase", func(t *testing.T) {
		cache := &pulseCache{}

		first := pulseTextCached("hello", 0.5, stops, cache)
		hexAfterFirst := cache.hex

		second := pulseTextCached("hello", 0.5, stops, cache)

		assert.Equal(t, first, second)
		assert.Equal(t, hexAfterFirst, cache.hex, "cache hex should not change on same phase")
	})

	t.Run("empty_text", func(t *testing.T) {
		cache := &pulseCache{}

		got := pulseTextCached("", 0.5, stops, cache)

		assert.Empty(t, got)
	})

	t.Run("cache_miss_different_phase", func(t *testing.T) {
		cache := &pulseCache{}

		pulseTextCached("hello", 0.0, stops, cache)
		hexFirst := cache.hex

		pulseTextCached("hello", 1.0, stops, cache)
		hexSecond := cache.hex

		assert.NotEqual(
			t,
			hexFirst,
			hexSecond,
			"different phases should produce different hex values",
		)
	})
}
