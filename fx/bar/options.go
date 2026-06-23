package bar

import (
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/gradient"
)

// Option applies a configuration change to a [Config].
type Option func(*Config)

// Apply returns a copy of [DefaultConfig] with all opts applied in order.
// Use [WithConfig] as the first option to start from a different base config.
func Apply(opts ...Option) Config {
	base := DefaultConfig()
	for _, o := range opts {
		o(&base)
	}
	return base
}

// WithCapLeft sets the left cap string (e.g. "[", "│").
func WithCapLeft(s string) Option {
	return func(st *Config) {
		st.CapLeft = s
	}
}

// WithCapRight sets the right cap string (e.g. "]", "│").
func WithCapRight(s string) Option {
	return func(st *Config) {
		st.CapRight = s
	}
}

// WithCapStyle sets the lipgloss style applied to the left and right caps.
func WithCapStyle(ls lipgloss.Style) Option {
	return func(s *Config) {
		s.CapStyle = &ls
	}
}

// WithCharEmpty sets the rune used for fully empty cells.
func WithCharEmpty(r rune) Option {
	return func(s *Config) {
		s.CharEmpty = r
	}
}

// WithCharFill sets the rune used for fully filled cells.
func WithCharFill(r rune) Option {
	return func(s *Config) {
		s.CharFill = r
	}
}

// WithCharHead sets the decorative rune at the leading edge of the filled
// section (1x resolution only). Set to 0 to disable. Ignored when
// [Config.HalfFilled] or [Config.GradientFill] is set.
func WithCharHead(r rune) Option {
	return func(s *Config) {
		s.CharHead = r
	}
}

// WithGradientFill sets the sub-cell fill runes ordered from least to most
// filled. Enables Nx sub-cell resolution where N = len(runes)+1. Overrides
// [Config.HalfFilled] and [Config.CharHead].
func WithGradientFill(runes ...rune) Option {
	return func(s *Config) {
		s.GradientFill = runes
	}
}

// WithHalfEmpty sets the rune shown at the start of the empty section when
// [Config.HalfFilled] is not displayed (2x resolution). Set to 0 to disable.
func WithHalfEmpty(r rune) Option {
	return func(s *Config) {
		s.HalfEmpty = r
	}
}

// WithHalfFilled sets the rune shown at the leading edge of the filled section,
// enabling 2x sub-cell resolution. Set to 0 to disable.
func WithHalfFilled(r rune) Option {
	return func(s *Config) {
		s.HalfFilled = r
	}
}

// WithSeparator sets the string placed between the message, bar, and widget text.
func WithSeparator(sep string) Option {
	return func(s *Config) {
		s.Separator = sep
	}
}

// WithSmoothingMode sets the bar fill smoothing mode.
func WithSmoothingMode(m SmoothingMode) Option {
	return func(s *Config) {
		s.Smoothing = m
	}
}

// WithSmoothingTau sets the exponential decay time constant for [SmoothEase].
// Smaller values converge faster (snappier); larger values are smoother.
// Non-positive values reset to [DefaultSmoothingTau].
func WithSmoothingTau(d time.Duration) Option {
	return func(s *Config) {
		if d <= 0 {
			s.SmoothingTau = 0
			return
		}
		s.SmoothingTau = d
	}
}

// WithPlacement sets the horizontal bar placement mode.
func WithPlacement(p Placement) Option {
	return func(s *Config) {
		s.Placement = p
	}
}

// WithPendingMode sets how the bar behaves before any progress is reported.
func WithPendingMode(m PendingMode) Option {
	return func(s *Config) {
		s.PendingMode = m
	}
}

// WithUpdateInterval samples the timing basis used by ETA, rate, and elapsed
// widgets/dynamic fields at most once per duration. Progress-driven text and
// the bar fill itself still render from live current/total updates.
// Non-positive values disable coalescing.
func WithUpdateInterval(d time.Duration) Option {
	return func(s *Config) {
		if d <= 0 {
			s.UpdateInterval = 0
			return
		}
		s.UpdateInterval = d
	}
}

// WithProgressGradient sets the filled-cell color gradient.
func WithProgressGradient(stops ...gradient.ColorStop) Option {
	return func(s *Config) {
		s.ProgressGradient = stops
	}
}

// WithConfig replaces the entire bar style.
func WithConfig(s Config) Option {
	return func(st *Config) {
		*st = s
	}
}

// WithStyleEmpty sets the lipgloss style for empty cells.
func WithStyleEmpty(ls lipgloss.Style) Option {
	return func(s *Config) {
		s.StyleEmpty = &ls
	}
}

// WithStyleFill sets the lipgloss style for filled cells.
func WithStyleFill(ls lipgloss.Style) Option {
	return func(s *Config) {
		s.StyleFill = &ls
	}
}

// WithTruncationMarker sets the marker appended inside the width budget when
// bar placement needs to truncate surrounding text to fit the terminal. Pass
// an empty string to disable the marker.
func WithTruncationMarker(marker string) Option {
	return func(s *Config) {
		s.TruncationMarker = &marker
	}
}

// WithWidth sets a fixed inner width for the bar.
// When w is 0, the bar auto-sizes from the terminal width.
func WithWidth(w int) Option {
	return func(s *Config) {
		s.Width = w
	}
}

// WithMaxWidth sets the maximum auto-sized inner width.
func WithMaxWidth(w int) Option {
	return func(s *Config) {
		s.WidthMax = w
	}
}

// WithMinWidth sets the minimum auto-sized inner width.
func WithMinWidth(w int) Option {
	return func(s *Config) {
		s.WidthMin = w
	}
}

// WithWidgetLeft sets the widget displayed to the left of the bar.
func WithWidgetLeft(w Widget) Option {
	return func(s *Config) {
		s.WidgetLeft = w
	}
}

// WithWidgetRight sets the widget displayed to the right of the bar.
func WithWidgetRight(w Widget) Option {
	return func(s *Config) {
		s.WidgetRight = w
	}
}
