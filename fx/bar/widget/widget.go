// Package widget provides composable text widgets for progress bars.
//
// Widgets render dynamic text labels (percentage, ETA, throughput, etc.)
// from the current [bar.State]. They are assigned to [bar.Config.WidgetLeft]
// or [bar.Config.WidgetRight] to appear alongside the bar.
//
//	style.WidgetRight = widget.Percent()
//	style.WidgetLeft  = widget.ETA()
package widget

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/internal/gradient"
)

// Option configures a widget constructor.
type Option func(*config)

// config holds resolved options for widget constructors.
type config struct {
	digits           int                  // significant digits for bytes; decimal places for percent
	minPercent       float64              // minimum percent value to display; below this the widget returns ""
	prefix           *string              // optional prefix override (nil = use widget default)
	progressGradient []gradient.ColorStop // when set, colors widget text based on progress
	style            *lipgloss.Style      // optional lipgloss style applied to the widget's output
	unit             string               // unit label for rate widgets (e.g. "ops", "files")
}

func apply(c *config, opts ...Option) {
	for _, o := range opts {
		o(c)
	}
}

// render applies the widget's style to s, or returns s unchanged when no style
// is set or s is empty (preserving the empty signal used by [Widgets]).
func (c config) render(s string) string {
	if s == "" || c.style == nil {
		return s
	}
	return c.style.Render(s)
}

// renderProgress applies gradient coloring based on progress, falling back to
// the static style when no gradient is configured. current/total are the raw
// progress values from [bar.State].
func (c config) renderProgress(s string, current, total int) string {
	if s == "" {
		return s
	}
	if len(c.progressGradient) > 0 && total > 0 {
		t := float64(current) / float64(total)
		if t > 1 {
			t = 1
		}
		clr := gradient.Interpolate(t, c.progressGradient)
		st := lipgloss.NewStyle().Foreground(lipgloss.Color(clr.Clamped().Hex()))
		return st.Render(s)
	}
	return c.render(s)
}

// None returns a [bar.Widget] that always returns "".
// Use it to explicitly suppress the default percent display.
func None() bar.Widget {
	return func(bar.State) string {
		return ""
	}
}

// WithDigits configures the precision of formatted values.
//   - [Bytes] / [IBytes]: significant digits (default 3).
//     3 -> "82.9 MB", "1 GB"; 2 -> "83 MB", "5.5 GB"
//   - [Percent]: decimal places (default 0).
//     0 -> "42%"; 1 -> "42.5%"
func WithDigits(n int) Option {
	return func(c *config) { c.digits = n }
}

// WithMinimumPercent sets the minimum percentage threshold for display.
// When the current progress is below this value, the widget returns "".
// Accepted by [Percent]. For example, WithMinimumPercent(1) hides "0%".
func WithMinimumPercent(pct float64) Option {
	return func(c *config) { c.minPercent = pct }
}

// WithStyle sets a lipgloss style applied to the widget's output string.
// Accepted by all built-in widgets. Empty outputs (e.g. [ETA] when
// complete) are never styled.
func WithStyle(style lipgloss.Style) Option {
	return func(c *config) { c.style = &style }
}

// WithProgressGradient colors the widget's output text based on the current
// progress, using the given gradient color stops. The gradient position is
// derived from current/total in [bar.State]. When set, this overrides
// [WithStyle]. Accepted by progress-aware widgets such as [Percent].
func WithProgressGradient(stops ...gradient.ColorStop) Option {
	return func(c *config) { c.progressGradient = stops }
}

// WithPrefix overrides the default prefix prepended to widget text.
// Accepted by [ETA]. For example, WithPrefix("") removes the "ETA " prefix,
// producing "2m30s" instead of "ETA 2m30s".
func WithPrefix(prefix string) Option {
	return func(c *config) { c.prefix = &prefix }
}

// WithUnit sets a unit label for rate widgets. For example, WithUnit("ops")
// produces "150 ops/s" instead of "150/s".
func WithUnit(unit string) Option {
	return func(c *config) { c.unit = unit }
}

// Widgets combines multiple [bar.Widget] functions into a single [bar.Widget].
// The outputs are joined with a space separator; empty outputs are skipped.
//
//	style.WidgetRight = widget.Widgets(widget.ETA(), widget.Rate())
func Widgets(widgets ...bar.Widget) bar.Widget {
	return func(s bar.State) string {
		var parts []string
		for _, w := range widgets {
			if text := w(s); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	}
}

// pad returns a function that right-aligns content to a stable width.
// raw is the unstyled string used for width tracking; content is what is
// actually displayed (potentially styled). Padding spaces are always plain so
// that background colors do not bleed into alignment space.
func pad() func(raw, content string) string {
	var maxW int
	return func(raw, content string) string {
		maxW = max(maxW, len(raw))
		padding := maxW - len(raw)
		if padding > 0 {
			return strings.Repeat(" ", padding) + content
		}
		return content
	}
}
