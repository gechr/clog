package clog

import (
	"bytes"
	"context"
	"testing"

	"github.com/gechr/clog/fx/pulse"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPulseDefault(t *testing.T) {
	b := Pulse("test")

	assert.Equal(t, pulse.DefaultGradient(), b.PulseGradient())
}

func TestPulseCustom(t *testing.T) {
	custom := []style.ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}
	b := Pulse("test", pulse.WithGradient(custom...))

	assert.Equal(t, custom, b.PulseGradient())
}

func TestPulseBuilderSymbol(t *testing.T) {
	b := Pulse("test").Symbol("🔄")

	assert.Equal(t, "🔄", b.SymbolOverride())
}

func TestPulseBuilderSymbolDefault(t *testing.T) {
	b := Pulse("test")

	assert.Empty(t, b.SymbolOverride())
}

func TestPulseSpeedDefault(t *testing.T) {
	b := Pulse("test")

	assert.InDelta(t, pulse.Speed, b.AnimationSpeed(), 1e-9)
}

func TestPulseSpeedCustom(t *testing.T) {
	b := Pulse("test", pulse.WithSpeed(2.0))

	assert.InDelta(t, 2.0, b.AnimationSpeed(), 1e-9)
}

func TestPulseSpeedZeroFallsBackToDefault(t *testing.T) {
	b := Pulse("test", pulse.WithSpeed(0))

	assert.InDelta(t, pulse.Speed, b.AnimationSpeed(), 1e-9)
}

func TestPulseSpeedNegativeFallsBackToDefault(t *testing.T) {
	b := Pulse("test", pulse.WithSpeed(-1.0))

	assert.InDelta(t, pulse.Speed, b.AnimationSpeed(), 1e-9)
}

func TestPulseDefaultSymbolInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Pulse("loading").Wait(context.Background(), func(_ context.Context) error {
		return nil
	})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "INF ⏳ loading\n", buf.String())
}

func TestPulseCustomSymbolInOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	result := Pulse(
		"loading",
	).Symbol("🔄").
		Wait(context.Background(), func(_ context.Context) error {
			return nil
		})

	require.NoError(t, result.TaskErr)
	assert.Equal(t, "INF 🔄 loading\n", buf.String())
}
