package clog

import "time"

// SpinnerType is a set of frames used in animating the spinner.
// Set Reverse to true to play the frames in reverse order.
type SpinnerType struct {
	Frames  []string
	FPS     time.Duration
	Reverse bool
}

// DefaultSpinner is the default spinner animation.
var DefaultSpinner = SpinnerType{
	Frames:  SpinnerMoon.Frames,
	FPS:     SpinnerMoon.FPS,
	Reverse: true,
}

// Spinner creates a new [AnimationBuilder] with a rotating spinner animation.
func Spinner(msg string) *AnimationBuilder {
	b := &AnimationBuilder{
		level:   InfoLevel,
		mode:    animModeSpinner,
		msg:     msg,
		spinner: DefaultSpinner,
	}
	b.initSelf(b)
	return b
}
