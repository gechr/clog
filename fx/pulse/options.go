package pulse

import "github.com/gechr/clog/internal/gradient"

// Option configures a [Style].
type Option func(*Style)

// ApplyOptions applies options over [DefaultStyle] and returns the resolved config.
func ApplyOptions(opts []Option) Style {
	s := DefaultStyle()
	for _, o := range opts {
		o(&s)
	}
	return s
}

// WithGradient sets custom gradient color stops.
func WithGradient(stops ...gradient.ColorStop) Option {
	return func(s *Style) { s.Gradient = stops }
}

// WithSpeed sets the number of full oscillation cycles per second.
// Values <= 0 are a no-op (keep default).
func WithSpeed(speed float64) Option {
	return func(s *Style) {
		if speed > 0 {
			s.Speed = speed
		}
	}
}
