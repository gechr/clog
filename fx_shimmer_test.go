package clog

import (
	"bytes"
	"context"
	"testing"

	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShimmerDefault(t *testing.T) {
	b := Shimmer("test")

	assert.Equal(t, shimmer.DefaultGradient(), b.ShimmerStops)
}

func TestShimmerCustom(t *testing.T) {
	custom := []style.ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}
	b := Shimmer("test", shimmer.WithGradient(custom...))

	assert.Equal(t, custom, b.ShimmerStops)
}

func TestShimmerDirection(t *testing.T) {
	b := Shimmer("test", shimmer.WithDirection(shimmer.Left))

	assert.Equal(t, shimmer.Left, b.ShimmerDir)
}

func TestShimmerDirectionDefault(t *testing.T) {
	b := Shimmer("test")

	assert.Equal(t, shimmer.Right, b.ShimmerDir)
}

func TestShimmerSpeedDefault(t *testing.T) {
	b := Shimmer("test")

	assert.InDelta(t, shimmer.Speed, b.Speed, 1e-9)
}

func TestShimmerSpeedCustom(t *testing.T) {
	b := Shimmer("test", shimmer.WithSpeed(2.0))

	assert.InDelta(t, 2.0, b.Speed, 1e-9)
}

func TestShimmerSpeedZeroFallsBackToDefault(t *testing.T) {
	b := Shimmer("test", shimmer.WithSpeed(0))

	assert.InDelta(t, shimmer.Speed, b.Speed, 1e-9)
}

func TestShimmerSpeedNegativeFallsBackToDefault(t *testing.T) {
	b := Shimmer("test", shimmer.WithSpeed(-1.0))

	assert.InDelta(t, shimmer.Speed, b.Speed, 1e-9)
}

func TestShimmerBuilderSymbol(t *testing.T) {
	b := Shimmer("test").Symbol("🔄")

	assert.Equal(t, "🔄", b.SymbolIcon)
}

func TestShimmerBuilderSymbolDefault(t *testing.T) {
	b := Shimmer("test")

	assert.Empty(t, b.SymbolIcon)
}

func TestShimmerDefaultSymbolInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Shimmer("loading").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "INF ⏳ loading\n", buf.String())
}

func TestShimmerCustomSymbolInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Shimmer(
		"loading",
	).Symbol("🔄").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "INF 🔄 loading\n", buf.String())
}
