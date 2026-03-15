package widget

import (
	"fmt"
	"strings"

	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/internal/numfmt"
)

// Percent returns a [bar.Widget] that displays padded percentage text
// (e.g. "  0%", " 50%", "100%"). The result is right-aligned to a fixed width
// to prevent the bar from jumping as digit count changes. Default digits
// is 0; use [WithDigits] for decimal places (1 -> " 50%", "42.5%").
// Trailing zeros are always stripped ("100.0%" -> "100%").
func Percent(opts ...Option) bar.Widget {
	c := config{digits: 0}
	applyOptions(&c, opts)

	// Unstripped width of "100%" at the given digits for stable padding.
	padWidth := len(fmt.Sprintf("%.*f%%", c.digits, bar.PercentDisplayMax))

	return func(s bar.State) string {
		pct := bar.PercentValue(s.Current, s.Total)
		if c.minPercent > 0 && pct < c.minPercent {
			return ""
		}
		str := numfmt.TrimDecimalZeros(fmt.Sprintf("%.*f", c.digits, pct)) + "%"
		padding := padWidth - len(str)
		if padding > 0 {
			return strings.Repeat(" ", padding) + c.renderProgress(str, s.Current, s.Total)
		}
		return c.renderProgress(str, s.Current, s.Total)
	}
}
