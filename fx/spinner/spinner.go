// Package spinner provides spinner animation styles and presets for clog.
package spinner

import "time"

// Style is a set of frames used in animating the spinner.
// Set Reverse to true to play the frames in reverse order.
// Set Boomerang to true to play the frames in a ping-pong loop so the
// animation smoothly reverses at each end instead of jumping from the last
// frame back to the first. For example, frames [a, b, c] play as
// [a, b, c, b, a, b, c, ...]. Boomerang and Reverse can be combined.
type Style struct {
	Boomerang bool
	Interval  time.Duration
	Frames    []string
	Reverse   bool
}

// DefaultStyle returns the default spinner [Style].
// It uses [Moon] in reverse.
func DefaultStyle() Style {
	return Style{
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
