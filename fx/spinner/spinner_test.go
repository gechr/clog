package spinner_test

import (
	"testing"
	"time"

	"github.com/gechr/clog/fx/spinner"
	"github.com/stretchr/testify/assert"
)

func TestDefaultStyle(t *testing.T) {
	s := spinner.DefaultStyle()
	assert.NotEmpty(t, s.Frames, "DefaultStyle should have non-empty Frames")
	assert.Greater(t, s.Interval, time.Duration(0), "DefaultStyle should have Interval > 0")
}

func TestDefaultStyleUsesMoon(t *testing.T) {
	s := spinner.DefaultStyle()
	// DefaultStyle uses Moon frames in reverse.
	assert.Equal(t, spinner.Moon.Frames, s.Frames)
	assert.Equal(t, spinner.Moon.Interval, s.Interval)
	assert.True(t, s.Reverse, "DefaultStyle should have Reverse=true")
}

func TestApplyOptionsEmpty(t *testing.T) {
	def := spinner.DefaultStyle()
	got := spinner.ApplyOptions([]spinner.Option{})
	assert.Equal(t, def.Frames, got.Frames)
	assert.Equal(t, def.Interval, got.Interval)
	assert.Equal(t, def.Reverse, got.Reverse)
	assert.Equal(t, def.Boomerang, got.Boomerang)
}

func TestWithStyle(t *testing.T) {
	custom := spinner.Style{
		Frames:   []string{"a", "b", "c"},
		Interval: 200 * time.Millisecond,
	}
	got := spinner.ApplyOptions([]spinner.Option{spinner.WithStyle(custom)})
	assert.Equal(t, custom.Frames, got.Frames)
	assert.Equal(t, custom.Interval, got.Interval)
	assert.Equal(t, custom.Reverse, got.Reverse)
	assert.Equal(t, custom.Boomerang, got.Boomerang)
}

func TestWithInterval(t *testing.T) {
	got := spinner.ApplyOptions([]spinner.Option{spinner.WithInterval(42 * time.Millisecond)})
	assert.Equal(t, 42*time.Millisecond, got.Interval)
}

func TestWithIntervalZeroIsNoOp(t *testing.T) {
	// Values <= 0 are a no-op, so Interval stays at DefaultStyle's Interval.
	def := spinner.DefaultStyle()
	got := spinner.ApplyOptions([]spinner.Option{spinner.WithInterval(0)})
	assert.Equal(t, def.Interval, got.Interval, "WithInterval(0) should be a no-op")
}

func TestWithIntervalNegativeIsNoOp(t *testing.T) {
	def := spinner.DefaultStyle()
	got := spinner.ApplyOptions([]spinner.Option{spinner.WithInterval(-time.Second)})
	assert.Equal(t, def.Interval, got.Interval, "WithInterval(<0) should be a no-op")
}

func TestWithReverse(t *testing.T) {
	// DefaultStyle already has Reverse=true; use a plain WithStyle first to reset it.
	custom := spinner.Style{
		Frames:   []string{"x", "y"},
		Interval: 100 * time.Millisecond,
	}
	got := spinner.ApplyOptions([]spinner.Option{
		spinner.WithStyle(custom),
		spinner.WithReverse(),
	})
	assert.True(t, got.Reverse, "expected Reverse=true after WithReverse")
}

func TestWithBoomerang(t *testing.T) {
	got := spinner.ApplyOptions([]spinner.Option{spinner.WithBoomerang()})
	assert.True(t, got.Boomerang, "expected Boomerang=true after WithBoomerang")
}

func TestWithStyleThenWithInterval(t *testing.T) {
	// Applying WithStyle then WithInterval: Interval should be overridden.
	custom := spinner.Style{
		Frames:   []string{"1", "2"},
		Interval: 50 * time.Millisecond,
	}
	got := spinner.ApplyOptions([]spinner.Option{
		spinner.WithStyle(custom),
		spinner.WithInterval(250 * time.Millisecond),
	})
	assert.Equal(t, custom.Frames, got.Frames)
	assert.Equal(t, 250*time.Millisecond, got.Interval)
}

func TestPresetsValid(t *testing.T) {
	presets := map[string]spinner.Style{
		"Aesthetic":           spinner.Aesthetic,
		"Arc":                 spinner.Arc,
		"Arrow2":              spinner.Arrow2,
		"Arrow3":              spinner.Arrow3,
		"Balloon":             spinner.Balloon,
		"Balloon2":            spinner.Balloon2,
		"BetaWave":            spinner.BetaWave,
		"Binary":              spinner.Binary,
		"BluePulse":           spinner.BluePulse,
		"BouncingBall":        spinner.BouncingBall,
		"BoxBounce":           spinner.BoxBounce,
		"BoxBounce2":          spinner.BoxBounce2,
		"Christmas":           spinner.Christmas,
		"Circle":              spinner.Circle,
		"CircleHalves":        spinner.CircleHalves,
		"CircleQuarters":      spinner.CircleQuarters,
		"Dot":                 spinner.Dot,
		"Dots":                spinner.Dots,
		"Dots11":              spinner.Dots11,
		"Dots12":              spinner.Dots12,
		"Dots13":              spinner.Dots13,
		"Dots14":              spinner.Dots14,
		"Dots3":               spinner.Dots3,
		"Dots4":               spinner.Dots4,
		"Dots5":               spinner.Dots5,
		"Dots6":               spinner.Dots6,
		"Dots7":               spinner.Dots7,
		"Dots8":               spinner.Dots8,
		"Dots8Bit":            spinner.Dots8Bit,
		"Dots9":               spinner.Dots9,
		"DotsCircle":          spinner.DotsCircle,
		"Dqpb":                spinner.Dqpb,
		"DwarfFortress":       spinner.DwarfFortress,
		"Ellipsis":            spinner.Ellipsis,
		"FingerDance":         spinner.FingerDance,
		"Fish":                spinner.Fish,
		"FistBump":            spinner.FistBump,
		"Flip":                spinner.Flip,
		"Globe":               spinner.Globe,
		"Grenade":             spinner.Grenade,
		"GrowHorizontal":      spinner.GrowHorizontal,
		"GrowVertical":        spinner.GrowVertical,
		"Hamburger":           spinner.Hamburger,
		"Jump":                spinner.Jump,
		"Layer":               spinner.Layer,
		"Line":                spinner.Line,
		"Line2":               spinner.Line2,
		"Material":            spinner.Material,
		"Meter":               spinner.Meter,
		"Mindblown":           spinner.Mindblown,
		"MiniDot":             spinner.MiniDot,
		"Monkey":              spinner.Monkey,
		"Moon":                spinner.Moon,
		"Noise":               spinner.Noise,
		"OrangeBluePulse":     spinner.OrangeBluePulse,
		"OrangePulse":         spinner.OrangePulse,
		"Pipe":                spinner.Pipe,
		"Point":               spinner.Point,
		"Points":              spinner.Points,
		"Pong":                spinner.Pong,
		"Pulse":               spinner.Pulse,
		"RollingLine":         spinner.RollingLine,
		"Runner":              spinner.Runner,
		"Sand":                spinner.Sand,
		"Shark":               spinner.Shark,
		"SimpleDots":          spinner.SimpleDots,
		"SimpleDotsScrolling": spinner.SimpleDotsScrolling,
		"Smiley":              spinner.Smiley,
		"SoccerHeader":        spinner.SoccerHeader,
		"Speaker":             spinner.Speaker,
		"SquareCorners":       spinner.SquareCorners,
		"Squish":              spinner.Squish,
		"Star2":               spinner.Star2,
		"TimeTravel":          spinner.TimeTravel,
		"Toggle":              spinner.Toggle,
		"Toggle10":            spinner.Toggle10,
		"Toggle11":            spinner.Toggle11,
		"Toggle12":            spinner.Toggle12,
		"Toggle13":            spinner.Toggle13,
		"Toggle2":             spinner.Toggle2,
		"Toggle3":             spinner.Toggle3,
		"Toggle4":             spinner.Toggle4,
		"Toggle5":             spinner.Toggle5,
		"Toggle6":             spinner.Toggle6,
		"Toggle7":             spinner.Toggle7,
		"Toggle8":             spinner.Toggle8,
		"Toggle9":             spinner.Toggle9,
		"Triangle":            spinner.Triangle,
		"Weather":             spinner.Weather,
	}

	for name, s := range presets {
		t.Run(name, func(t *testing.T) {
			assert.NotEmpty(t, s.Frames, "%s: Frames should be non-empty", name)
			assert.Greater(t, s.Interval, time.Duration(0), "%s: Interval should be > 0", name)
		})
	}
}
