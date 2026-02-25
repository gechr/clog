package clog

import "time"

// SpinnerStyle is a set of frames used in animating the spinner.
// Set Reverse to true to play the frames in reverse order.
// Set Boomerang to true to play the frames in a ping-pong loop so the
// animation smoothly reverses at each end instead of jumping from the last
// frame back to the first. For example, frames [a, b, c] play as
// [a, b, c, b, a, b, c, ...]. Boomerang and Reverse can be combined.
type SpinnerStyle struct {
	Frames    []string
	FPS       time.Duration
	Reverse   bool
	Boomerang bool
}

func (s SpinnerStyle) applyAnimation(b *AnimationBuilder) { b.spinner = s }

// boomerangFrames expands frames into a ping-pong sequence.
// [a, b, c] becomes [a, b, c, b]. Styles with fewer than 3 frames are
// returned unchanged because there is nothing to bounce.
func boomerangFrames(frames []string) []string {
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

// DefaultSpinnerStyle returns the default [SpinnerStyle].
// It uses [SpinnerMoon] in reverse.
func DefaultSpinnerStyle() SpinnerStyle {
	return SpinnerStyle{
		Frames:  SpinnerMoon.Frames,
		FPS:     SpinnerMoon.FPS,
		Reverse: true,
	}
}

// Spinner creates a new [AnimationBuilder] using the [Default] logger with a
// rotating spinner animation.
func Spinner(msg string) *AnimationBuilder { return Default.Spinner(msg) }

// Spinner creates a new [AnimationBuilder] with a rotating spinner animation.
func (l *Logger) Spinner(msg string) *AnimationBuilder {
	b := &AnimationBuilder{
		level:   InfoLevel,
		logger:  l,
		mode:    animationSpinner,
		msg:     msg,
		spinner: DefaultSpinnerStyle(),
	}
	b.initSelf(b)
	return b
}
