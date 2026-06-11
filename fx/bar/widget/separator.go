package widget

import "github.com/gechr/clog/fx/bar"

// Separator returns a [bar.Widget] that always renders the given string.
// Use it inside [Widgets] to place a visual divider between other widgets:
//
//	style.WidgetRight = widget.Widgets(widget.ETA(), widget.Separator("|"), widget.Rate())
//
// Pass [WithStyle] to apply a lipgloss style to the separator.
func Separator(s string, opts ...Option) bar.Widget {
	c := config{}
	apply(&c, opts...)
	return func(bar.State) string { return c.render(s) }
}
