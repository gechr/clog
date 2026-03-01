package spinner

import "time"

// Option configures a spinner constructor.
type Option func(*Style)

// ApplyOptions applies the given options to the default style and returns the result.
func ApplyOptions(opts []Option) Style {
	s := DefaultStyle()
	for _, o := range opts {
		o(&s)
	}
	return s
}

// WithBoomerang enables ping-pong playback so the animation smoothly
// reverses at each end instead of jumping from the last frame back to the first.
// Can be combined with [WithStyle] (applied after) or used standalone.
func WithBoomerang() Option {
	return func(s *Style) {
		s.Boomerang = true
	}
}

// WithFrames sets the animation frames. Values <= 0 are a no-op (keep existing Frames).
func WithFrames(frames []string) Option {
	return func(s *Style) {
		if len(frames) > 0 {
			s.Frames = frames
		}
	}
}

// WithInterval overrides the frame duration of the current style.
// Values <= 0 are a no-op (keep existing Interval).
func WithInterval(d time.Duration) Option {
	return func(s *Style) {
		if d > 0 {
			s.Interval = d
		}
	}
}

// WithReverse enables reverse playback of the spinner frames.
// Can be combined with [WithStyle] (applied after) or used standalone.
func WithReverse() Option {
	return func(s *Style) {
		s.Reverse = true
	}
}

// WithStyle sets the spinner animation style.
func WithStyle(s Style) Option {
	return func(dst *Style) {
		*dst = s
	}
}
