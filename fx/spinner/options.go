package spinner

import (
	"time"

	"github.com/gechr/clog/internal/gradient"
	"github.com/gechr/clog/style"
)

// Option configures a spinner constructor.
type Option func(*Config)

// Apply returns a copy of [DefaultConfig] with all opts applied in order.
func Apply(opts ...Option) Config {
	return ApplyTo(DefaultConfig(), opts...)
}

// ApplyTo returns a copy of base with all opts applied in order.
func ApplyTo(base Config, opts ...Option) Config {
	for _, o := range opts {
		o(&base)
	}
	return base
}

// WithBoomerang enables ping-pong playback so the animation smoothly
// reverses at each end instead of jumping from the last frame back to the first.
// Can be combined with [WithConfig] (applied after) or used standalone.
func WithBoomerang() Option {
	return func(s *Config) {
		s.Boomerang = true
	}
}

// WithFrames sets the animation frames. No frames is a no-op (keep existing Frames).
func WithFrames(frames ...string) Option {
	return func(s *Config) {
		if len(frames) > 0 {
			s.Frames = frames
		}
	}
}

// WithInterval overrides the frame duration of the current style.
// Values <= 0 are a no-op (keep existing Interval).
func WithInterval(d time.Duration) Option {
	return func(s *Config) {
		if d > 0 {
			s.Interval = d
		}
	}
}

// WithGradient sets the color stops used to animate the spinner glyph
// color, replacing the level symbol style while the spinner runs.
// No stops applies [DefaultGradient].
func WithGradient(stops ...gradient.ColorStop) Option {
	return func(s *Config) {
		if len(stops) == 0 {
			stops = DefaultGradient()
		}
		s.Gradient = stops
	}
}

// WithGradientMode sets how gradient colors transition between stops:
// [style.GradientFade] (smooth, default) or [style.GradientStep] (discrete).
func WithGradientMode(m style.GradientMode) Option {
	return func(s *Config) {
		s.GradientMode = m
	}
}

// WithGradientSpeed sets the number of full gradient cycles per second used
// by [GradientTimeBased]. Values <= 0 are a no-op (keep existing speed).
func WithGradientSpeed(speed float64) Option {
	return func(s *Config) {
		if speed > 0 {
			s.GradientSpeed = speed
		}
	}
}

// WithGradientTiming selects what drives the gradient phase.
func WithGradientTiming(t GradientTiming) Option {
	return func(s *Config) {
		s.GradientTiming = t
	}
}

// WithReverse enables reverse playback of the spinner frames.
// Can be combined with [WithConfig] (applied after) or used standalone.
func WithReverse() Option {
	return func(s *Config) {
		s.Reverse = true
	}
}

// WithConfig sets the spinner animation style.
func WithConfig(s Config) Option {
	return func(dst *Config) {
		*dst = s
	}
}
