package spinner_test

import (
	"testing"
	"time"

	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
)

func twoStopGradient() []style.ColorStop {
	return []style.ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
		{Position: 1, Color: colorful.Color{R: 0, G: 0, B: 1}},
	}
}

func TestDefaultGradientCyclic(t *testing.T) {
	stops := spinner.DefaultGradient()
	assert.NotEmpty(t, stops)
	assert.InDelta(t, 0.0, stops[0].Position, 1e-9)
	assert.InDelta(t, 1.0, stops[len(stops)-1].Position, 1e-9)
	// The last stop repeats the first so the sweep wraps seamlessly.
	assert.Equal(t, stops[0].Color, stops[len(stops)-1].Color)
}

func TestGradientPhaseFrameSynced(t *testing.T) {
	c := spinner.Config{Gradient: twoStopGradient()}
	assert.InDelta(t, 0.0, spinner.GradientPhase(c, 0, 4, 0), 1e-9)
	assert.InDelta(t, 0.25, spinner.GradientPhase(c, 1, 4, 0), 1e-9)
	assert.InDelta(t, 0.5, spinner.GradientPhase(c, 2, 4, 0), 1e-9)
	assert.InDelta(t, 0.75, spinner.GradientPhase(c, 3, 4, 0), 1e-9)
	// Wraps after one full revolution.
	assert.InDelta(t, 0.0, spinner.GradientPhase(c, 4, 4, 0), 1e-9)
}

func TestGradientPhaseFrameSyncedIgnoresReverse(t *testing.T) {
	// Reverse only affects glyph playback order; the gradient still sweeps
	// forward, keyed off the raw pre-Reverse tick.
	forward := spinner.Config{Gradient: twoStopGradient()}
	reversed := forward
	reversed.Reverse = true
	for tick := range 8 {
		assert.InDelta(t,
			spinner.GradientPhase(forward, tick, 4, 0),
			spinner.GradientPhase(reversed, tick, 4, 0),
			1e-9,
		)
	}
}

func TestGradientPhaseFrameSyncedZeroFrames(t *testing.T) {
	c := spinner.Config{Gradient: twoStopGradient()}
	assert.InDelta(t, 0.0, spinner.GradientPhase(c, 3, 0, 0), 1e-9)
}

func TestGradientPhaseTimeBased(t *testing.T) {
	c := spinner.Config{
		Gradient:       twoStopGradient(),
		GradientSpeed:  1.0,
		GradientTiming: spinner.GradientTimeBased,
	}
	assert.InDelta(t, 0.25, spinner.GradientPhase(c, 0, 4, 250*time.Millisecond), 1e-9)
	// Tick and frame count are irrelevant in time-based mode.
	assert.InDelta(t, 0.25, spinner.GradientPhase(c, 7, 10, 250*time.Millisecond), 1e-9)
	// Wraps past one full cycle.
	assert.InDelta(t, 0.25, spinner.GradientPhase(c, 0, 4, 1250*time.Millisecond), 1e-9)
}

func TestGradientPhaseTimeBasedDefaultSpeed(t *testing.T) {
	// Speed <= 0 falls back to DefaultGradientSpeed (0.5 cycles/sec).
	c := spinner.Config{
		Gradient:       twoStopGradient(),
		GradientTiming: spinner.GradientTimeBased,
	}
	assert.InDelta(t, 0.5, spinner.GradientPhase(c, 0, 4, time.Second), 1e-9)
}

func TestBuildStyleLUTNilWithoutGradient(t *testing.T) {
	assert.Nil(t, spinner.BuildStyleLUT(spinner.Config{}, 10))
}

func TestBuildStyleLUTNilWithZeroFrames(t *testing.T) {
	c := spinner.Config{Gradient: twoStopGradient()}
	assert.Nil(t, spinner.BuildStyleLUT(c, 0))
}

func TestBuildStyleLUTFrameSyncedSize(t *testing.T) {
	c := spinner.Config{Gradient: twoStopGradient()}
	lut := spinner.BuildStyleLUT(c, 10)
	assert.NotNil(t, lut)
	assert.Equal(t, 10, lut.Len())
}

func TestBuildStyleLUTTimeBasedSize(t *testing.T) {
	c := spinner.Config{
		Gradient:       twoStopGradient(),
		GradientTiming: spinner.GradientTimeBased,
	}
	lut := spinner.BuildStyleLUT(c, 10)
	assert.NotNil(t, lut)
	assert.Equal(t, 64, lut.Len())
}

func TestStyleLUTAtDistinctColors(t *testing.T) {
	c := spinner.Config{Gradient: twoStopGradient()}
	lut := spinner.BuildStyleLUT(c, 4)
	start := lut.At(0).Render("x")
	mid := lut.At(0.5).Render("x")
	assert.NotEqual(t, start, mid, "phases 0 and 0.5 should render different colors")
}

func TestStyleLUTAtWraps(t *testing.T) {
	c := spinner.Config{Gradient: twoStopGradient()}
	lut := spinner.BuildStyleLUT(c, 4)
	assert.Equal(t, lut.At(0).Render("x"), lut.At(1.0).Render("x"))
	assert.Equal(t, lut.At(0.25).Render("x"), lut.At(1.25).Render("x"))
	// Negative phases wrap into [0, 1) too.
	assert.Equal(t, lut.At(0.75).Render("x"), lut.At(-0.25).Render("x"))
}

func TestStyleLUTStepModeUsesOnlyStopColors(t *testing.T) {
	// Step stops at 0 and 0.5: cyclic sampling never reaches t=1, so a
	// second stop must sit strictly below 1 to ever be selected.
	c := spinner.Config{
		Gradient: []style.ColorStop{
			{Position: 0, Color: colorful.Color{R: 1, G: 0, B: 0}},
			{Position: 0.5, Color: colorful.Color{R: 0, G: 0, B: 1}},
		},
		GradientMode: style.GradientStep,
	}
	lut := spinner.BuildStyleLUT(c, 8)
	first := lut.At(0).Render("x")
	last := lut.At(0.99).Render("x")
	for _, phase := range []float64{0, 0.2, 0.4, 0.6, 0.8, 0.99} {
		got := lut.At(phase).Render("x")
		assert.Contains(t, []string{first, last}, got,
			"step mode should only use stop colors (phase %v)", phase)
	}
	assert.NotEqual(t, first, last)
}

func TestStyleLUTFadeModeInterpolates(t *testing.T) {
	c := spinner.Config{Gradient: twoStopGradient()}
	lut := spinner.BuildStyleLUT(c, 8)
	first := lut.At(0).Render("x")
	mid := lut.At(0.5).Render("x")
	last := lut.At(0.99).Render("x")
	assert.NotEqual(t, first, mid, "fade mode should produce intermediate colors")
	assert.NotEqual(t, last, mid, "fade mode should produce intermediate colors")
}
