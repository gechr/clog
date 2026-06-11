package pulse

import "github.com/gechr/clog/internal/gradient"

// Option configures a [Config].
type Option func(*Config)

// Apply returns a copy of [DefaultConfig] with all opts applied in order.
func Apply(opts ...Option) Config {
	s := DefaultConfig()
	for _, o := range opts {
		o(&s)
	}
	return s
}

// WithGradient sets custom gradient color stops.
func WithGradient(stops ...gradient.ColorStop) Option {
	return func(s *Config) { s.Gradient = stops }
}

// WithSpeed sets the number of full oscillation cycles per second.
// Values <= 0 are a no-op (keep default).
func WithSpeed(speed float64) Option {
	return func(s *Config) {
		if speed > 0 {
			s.Speed = speed
		}
	}
}
