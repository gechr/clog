package bar

import (
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/gradient"
)

// Option applies a configuration change to a [Style].
type Option func(*Style)

// ApplyOptions returns a copy of [DefaultStyle] with all opts applied in order.
// Use [WithStyle] as the first option to start from a different base style.
func ApplyOptions(opts []Option) Style {
	base := DefaultStyle()
	for _, o := range opts {
		o(&base)
	}
	return base
}

// WithCapLeft sets the left cap string (e.g. "[", "│").
func WithCapLeft(s string) Option {
	return func(st *Style) {
		st.CapLeft = s
	}
}

// WithCapRight sets the right cap string (e.g. "]", "│").
func WithCapRight(s string) Option {
	return func(st *Style) {
		st.CapRight = s
	}
}

// WithCapStyle sets the lipgloss style applied to the left and right caps.
func WithCapStyle(ls *lipgloss.Style) Option {
	return func(s *Style) {
		s.CapStyle = ls
	}
}

// WithCharEmpty sets the rune used for fully empty cells.
func WithCharEmpty(r rune) Option {
	return func(s *Style) {
		s.CharEmpty = r
	}
}

// WithCharFill sets the rune used for fully filled cells.
func WithCharFill(r rune) Option {
	return func(s *Style) {
		s.CharFill = r
	}
}

// WithCharHead sets the decorative rune at the leading edge of the filled
// section (1x resolution only). Set to 0 to disable. Ignored when
// [Style.HalfFilled] or [Style.GradientFill] is set.
func WithCharHead(r rune) Option {
	return func(s *Style) {
		s.CharHead = r
	}
}

// WithGradientFill sets the sub-cell fill runes ordered from least to most
// filled. Enables Nx sub-cell resolution where N = len(runes)+1. Overrides
// [Style.HalfFilled] and [Style.CharHead].
func WithGradientFill(runes []rune) Option {
	return func(s *Style) {
		s.GradientFill = runes
	}
}

// WithHalfEmpty sets the rune shown at the start of the empty section when
// [Style.HalfFilled] is not displayed (2x resolution). Set to 0 to disable.
func WithHalfEmpty(r rune) Option {
	return func(s *Style) {
		s.HalfEmpty = r
	}
}

// WithHalfFilled sets the rune shown at the leading edge of the filled section,
// enabling 2x sub-cell resolution. Set to 0 to disable.
func WithHalfFilled(r rune) Option {
	return func(s *Style) {
		s.HalfFilled = r
	}
}

// WithSeparator sets the string placed between the message, bar, and widget text.
func WithSeparator(sep string) Option {
	return func(s *Style) {
		s.Separator = sep
	}
}

// WithPlacement sets the horizontal bar placement mode.
func WithPlacement(p Placement) Option {
	return func(s *Style) {
		s.Placement = p
	}
}

// WithPendingMode sets how the bar behaves before any progress is reported.
func WithPendingMode(m PendingMode) Option {
	return func(s *Style) {
		s.PendingMode = m
	}
}

// WithUpdateInterval coalesces visible bar updates so the displayed state
// changes at most once per duration. Non-positive values disable coalescing.
func WithUpdateInterval(d time.Duration) Option {
	return func(s *Style) {
		if d <= 0 {
			s.UpdateInterval = 0
			return
		}
		s.UpdateInterval = d
	}
}

// WithProgressGradient sets the filled-cell color gradient.
func WithProgressGradient(stops ...gradient.ColorStop) Option {
	return func(s *Style) {
		s.ProgressGradient = stops
	}
}

// WithStyle replaces the entire bar style.
func WithStyle(s Style) Option {
	return func(st *Style) {
		*st = s
	}
}

// WithStyleEmpty sets the lipgloss style for empty cells.
func WithStyleEmpty(ls *lipgloss.Style) Option {
	return func(s *Style) {
		s.StyleEmpty = ls
	}
}

// WithStyleFill sets the lipgloss style for filled cells.
func WithStyleFill(ls *lipgloss.Style) Option {
	return func(s *Style) {
		s.StyleFill = ls
	}
}

// WithWidth sets a fixed inner width for the bar.
// When w is 0, the bar auto-sizes from the terminal width.
func WithWidth(w int) Option {
	return func(s *Style) {
		s.Width = w
	}
}

// WithWidthMax sets the maximum auto-sized inner width.
func WithWidthMax(w int) Option {
	return func(s *Style) {
		s.WidthMax = w
	}
}

// WithWidthMin sets the minimum auto-sized inner width.
func WithWidthMin(w int) Option {
	return func(s *Style) {
		s.WidthMin = w
	}
}

// WithWidgetLeft sets the widget displayed to the left of the bar.
func WithWidgetLeft(w Widget) Option {
	return func(s *Style) {
		s.WidgetLeft = w
	}
}

// WithWidgetRight sets the widget displayed to the right of the bar.
func WithWidgetRight(w Widget) Option {
	return func(s *Style) {
		s.WidgetRight = w
	}
}
