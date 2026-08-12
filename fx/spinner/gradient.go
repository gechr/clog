package spinner

import (
	"math"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/gradient"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
)

// GradientTiming selects what drives the gradient phase.
type GradientTiming int

const (
	// GradientFrameSynced derives the color from the frame index, so the
	// gradient completes one full sweep per spinner revolution (default).
	GradientFrameSynced GradientTiming = iota
	// GradientTimeBased derives the color from elapsed time multiplied by
	// [Config.GradientSpeed], independent of the frame count or interval.
	GradientTimeBased
)

const (
	// DefaultGradientSpeed is the default number of full gradient cycles
	// per second in [GradientTimeBased] mode.
	DefaultGradientSpeed = 0.5

	// GradientTickRate is the repaint interval targeted for smooth color
	// transitions in [GradientTimeBased] mode (~30fps). It is a target, not
	// a guarantee: the logger's animation interval floor is applied last, so
	// the effective cadence is the slower of the two.
	GradientTickRate = 33 * time.Millisecond

	// gradientLUTSize is the number of pre-computed styles in
	// [GradientTimeBased] mode.
	gradientLUTSize = 64
)

// DefaultGradient returns a saturated red-to-green-to-blue cycle whose last
// stop repeats the first, so the sweep wraps seamlessly in both timing modes.
func DefaultGradient() []gradient.ColorStop {
	rR, rG, rB := 1.0, 0.35, 0.35
	gR, gG, gB := 0.35, 1.0, 0.5
	bR, bG, bB := 0.4, 0.6, 1.0
	third, twoThirds := 0.33, 0.67
	red := colorful.Color{R: rR, G: rG, B: rB}
	green := colorful.Color{R: gR, G: gG, B: gB}
	blue := colorful.Color{R: bR, G: bG, B: bB}
	return []gradient.ColorStop{
		{Position: 0, Color: red},
		{Position: third, Color: green},
		{Position: twoThirds, Color: blue},
		{Position: 1.0, Color: red},
	}
}

// StyleLUT is a pre-computed table of glyph styles, one per gradient sample.
// Reusing styles across frames eliminates per-frame style allocations.
type StyleLUT struct {
	styles []lipgloss.Style
}

// BuildStyleLUT pre-computes the glyph styles for c. It returns nil when c
// has no gradient, letting callers skip gradient styling entirely.
// frameCount sizes the table in [GradientFrameSynced] mode so each frame
// maps to exactly one style; [GradientTimeBased] mode uses a fixed size.
func BuildStyleLUT(c Config, frameCount int) *StyleLUT {
	if len(c.Gradient) == 0 {
		return nil
	}
	size := gradientLUTSize
	if c.GradientTiming == GradientFrameSynced {
		size = frameCount
	}
	if size <= 0 {
		return nil
	}
	styles := make([]lipgloss.Style, size)
	for i := range styles {
		t := float64(i) / float64(size)
		var col colorful.Color
		switch c.GradientMode {
		case style.GradientFade:
			col = style.InterpolateGradient(t, c.Gradient)
		case style.GradientStep:
			col = style.StepGradient(t, c.Gradient)
		}
		styles[i] = lipgloss.NewStyle().
			Foreground(lipgloss.Color(col.Clamped().Hex()))
	}
	return &StyleLUT{styles: styles}
}

// At returns the style for phase t, wrapping t into [0, 1).
func (l *StyleLUT) At(t float64) lipgloss.Style {
	n := len(l.styles)
	t = math.Mod(t, 1.0)
	if t < 0 {
		t++
	}
	idx := min(int(t*float64(n)), n-1)
	return l.styles[idx]
}

// Len returns the number of pre-computed styles.
func (l *StyleLUT) Len() int { return len(l.styles) }

// GradientPhase returns the gradient phase in [0, 1) for the given raw frame
// tick (before any [Config.Reverse] mapping) and elapsed animation duration.
// In [GradientFrameSynced] mode the phase advances one step per frame so the
// gradient always sweeps forward regardless of frame playback order.
func GradientPhase(c Config, tick, frameCount int, elapsed time.Duration) float64 {
	switch c.GradientTiming {
	case GradientFrameSynced:
		if frameCount <= 0 {
			return 0
		}
		return float64(tick%frameCount) / float64(frameCount)
	case GradientTimeBased:
		speed := c.GradientSpeed
		if speed <= 0 {
			speed = DefaultGradientSpeed
		}
		return math.Mod(elapsed.Seconds()*speed, 1.0)
	}
	return 0
}
