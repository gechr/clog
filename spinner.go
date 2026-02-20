package clog

import "github.com/charmbracelet/bubbles/spinner"

// DefaultSpinner is the default spinner animation.
var DefaultSpinner = spinner.Spinner{
	Frames: []string{"🌔", "🌓", "🌒", "🌑", "🌘", "🌗", "🌖", "🌕"},
	FPS:    spinner.Moon.FPS,
}

// Spinner creates a new [AnimationBuilder] with a rotating spinner animation.
func Spinner(title string) *AnimationBuilder {
	b := &AnimationBuilder{
		mode:    animModeSpinner,
		spinner: DefaultSpinner,
		title:   title,
	}
	b.initSelf(b)
	return b
}
