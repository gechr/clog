// Package spinner provides spinner animation styles and presets for clog.
package spinner

import (
	"time"

	"github.com/gechr/clog/internal/gradient"
	"github.com/gechr/clog/style"
)

// Config is a set of frames used in animating the spinner.
// Set Reverse to true to play the frames in reverse order.
// Set Boomerang to true to play the frames in a ping-pong loop so the
// animation smoothly reverses at each end instead of jumping from the last
// frame back to the first. For example, frames [a, b, c] play as
// [a, b, c, b, a, b, c, ...]. Boomerang and Reverse can be combined.
// Set Gradient to animate the glyph color; see [GradientTiming] for what
// drives the color phase.
type Config struct {
	Boomerang bool
	// Gradient colors the glyph from these stops while animating,
	// replacing the level symbol style. nil keeps the level symbol style.
	Gradient []gradient.ColorStop
	// GradientMode selects smooth fades (default) or discrete steps
	// between gradient stops.
	GradientMode style.GradientMode
	// GradientSpeed is the number of full gradient cycles per second in
	// [GradientTimeBased] mode. <= 0 uses [DefaultGradientSpeed].
	GradientSpeed float64
	// GradientTiming selects what drives the gradient phase.
	GradientTiming GradientTiming
	Interval       time.Duration
	Frames         []string
	Reverse        bool
}

// DefaultConfig returns the default spinner [Config].
// It uses [Moon] in reverse.
func DefaultConfig() Config {
	return Config{
		Frames:   Moon.Frames,
		Interval: Moon.Interval,
		Reverse:  true,
	}
}

// BoomerangFrames expands frames into a ping-pong sequence.
// [a, b, c] becomes [a, b, c, b]. Styles with fewer than 3 frames are
// returned unchanged because there is nothing to bounce.
func BoomerangFrames(frames []string) []string {
	if len(frames) < 3 { //nolint:mnd // minimum frames for a bounce
		return frames
	}
	out := make(
		[]string,
		len(frames),
		2*len(frames)-2, //nolint:mnd // forward + reverse minus endpoints
	)
	copy(out, frames)
	for i := len(frames) - 2; i > 0; i-- { //nolint:mnd // skip last frame (already present)
		out = append(out, frames[i])
	}
	return out
}
