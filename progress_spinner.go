package clog

import "time"

// SpinnerStyle is a set of frames used in animating the spinner.
// Set Reverse to true to play the frames in reverse order.
type SpinnerStyle struct {
	Frames  []string
	FPS     time.Duration
	Reverse bool
}

func (s SpinnerStyle) applyAnimation(b *AnimationBuilder) { b.spinner = s }

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
