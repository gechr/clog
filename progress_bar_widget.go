package clog

import (
	"fmt"
	"strings"
	"time"
)

// BarState holds progress information passed to [BarWidget] functions.
type BarState struct {
	Current int
	Total   int
	Elapsed time.Duration
	Rate    float64 // items per second (0 when elapsed or current is 0)
}

// BarWidget renders a text label from the current bar state.
// Return "" to display nothing for this tick.
type BarWidget func(BarState) string

// WidgetOption configures a bar widget constructor.
type WidgetOption func(*widget)

// widget holds resolved options for widget constructors.
type widget struct {
	digits int    // significant digits for bytes; decimal places for percent
	style  Style  // optional lipgloss style applied to the widget's output
	unit   string // unit label for rate widgets (e.g. "ops", "files")
}

func applyWidgetOpts(w *widget, opts []WidgetOption) {
	for _, o := range opts {
		o(w)
	}
}

// render applies the widget's style to s, or returns s unchanged when no style
// is set or s is empty (preserving the empty signal used by [Widgets]).
func (w widget) render(s string) string {
	if s == "" || w.style == nil {
		return s
	}
	return w.style.Render(s)
}

// WithDigits configures the precision of formatted values.
//   - [WidgetBytes] / [WidgetIBytes]: significant digits (default 3).
//     3 → "82.9 MB", "1 GB"; 2 → "83 MB", "5.5 GB"
//   - [WidgetPercent]: decimal places (default 0).
//     0 → "42%"; 1 → "42.5%"
func WithDigits(n int) WidgetOption {
	return func(c *widget) { c.digits = n }
}

// WithStyle sets a lipgloss style applied to the widget's output string.
// Accepted by all built-in widgets. Empty outputs (e.g. [WidgetETA] when
// complete) are never styled.
func WithStyle(style Style) WidgetOption {
	return func(c *widget) { c.style = style }
}

// WithUnit sets a unit label for rate widgets. For example, WithUnit("ops")
// produces "150 ops/s" instead of "150/s".
func WithUnit(unit string) WidgetOption {
	return func(c *widget) { c.unit = unit }
}

// WidgetNone is a [BarWidget] that always returns "".
// Use it to explicitly suppress the default percent display.
var WidgetNone BarWidget = func(BarState) string { return "" }

// Widgets combines multiple [BarWidget] functions into a single [BarWidget].
// The outputs are joined with a space separator; empty outputs are skipped.
//
//	style.WidgetRight = clog.Widgets(clog.WidgetETA(), clog.WidgetRate())
func Widgets(widgets ...BarWidget) BarWidget {
	return func(s BarState) string {
		var parts []string
		for _, w := range widgets {
			if text := w(s); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, " ")
	}
}

// WidgetSeparator returns a [BarWidget] that always renders the given string.
// Use it inside [Widgets] to place a visual divider between other widgets:
//
//	style.WidgetRight = clog.Widgets(clog.WidgetETA(), clog.WidgetSeparator("│"), clog.WidgetRate())
//
// Pass [WithStyle] to apply a lipgloss style to the separator:
//
//	sep := new(lipgloss.NewStyle().Faint(true))
//	style.WidgetRight = clog.Widgets(clog.WidgetETA(), clog.WidgetSeparator("│", clog.WithStyle(sep)), clog.WidgetRate())
func WidgetSeparator(s string, opts ...WidgetOption) BarWidget {
	w := widget{}
	applyWidgetOpts(&w, opts)
	return func(BarState) string { return w.render(s) }
}

// WidgetBytes returns a [BarWidget] that displays download-style progress
// in human-readable SI byte units (e.g. "1.5 GB / 2 GB"). The [BarState]
// Current and Total fields are interpreted as byte counts.
// Uses base-1000 units (kB, MB, GB). See [WidgetIBytes] for base-1024 (KiB, MiB, GiB).
// Default digits is 3; use [WithDigits] to change.
func WidgetBytes(opts ...WidgetOption) BarWidget {
	return widgetBytes(humanBytes, humanBytesWidth, opts)
}

// WidgetIBytes returns a [BarWidget] that displays download-style progress
// in human-readable IEC byte units (e.g. "1.5 GiB / 2 GiB"). The [BarState]
// Current and Total fields are interpreted as byte counts.
// Uses base-1024 units (KiB, MiB, GiB). See [WidgetBytes] for base-1000 (kB, MB, GB).
// Default digits is 3; use [WithDigits] to change.
func WidgetIBytes(opts ...WidgetOption) BarWidget {
	return widgetBytes(humanIBytes, humanIBytesWidth, opts)
}

// WidgetPercent returns a [BarWidget] that displays padded percentage text
// (e.g. "  0%", " 50%", "100%"). The result is right-aligned to a fixed width
// to prevent the bar from jumping as digit count changes. Default digits
// is 0; use [WithDigits] for decimal places (1 → " 50%", "42.5%").
// Trailing zeros are always stripped ("100.0%" → "100%").
func WidgetPercent(opts ...WidgetOption) BarWidget {
	w := widget{digits: 0}
	applyWidgetOpts(&w, opts)

	// Unstripped width of "100%" at the given digits for stable padding.
	padWidth := len(fmt.Sprintf("%.*f%%", w.digits, percentMax))

	return func(s BarState) string {
		pct := barPercentValue(s.Current, s.Total)
		str := trimDecimalZeros(fmt.Sprintf("%.*f", w.digits, pct)) + "%"
		padding := padWidth - len(str)
		if padding > 0 {
			return strings.Repeat(" ", padding) + w.render(str)
		}
		return w.render(str)
	}
}

// widgetBytes is the shared implementation for [WidgetBytes] and [WidgetIBytes].
// maxWidth returns the unstripped formatted width for stable right-alignment.
func widgetBytes(
	format func(uint64, int) string,
	maxWidth func(uint64, int) int,
	opts []WidgetOption,
) BarWidget {
	w := widget{digits: 3} //nolint:mnd // default significant digits
	applyWidgetOpts(&w, opts)

	// Cache the formatted total to avoid re-computing every tick.
	var cachedTotal int
	var cachedTotalStr string
	var cachedWidth int

	return func(s BarState) string {
		if s.Total != cachedTotal || cachedTotalStr == "" {
			cachedTotal = s.Total
			tot := uint64(max(s.Total, 0))
			cachedTotalStr = format(tot, w.digits)
			cachedWidth = maxWidth(tot, w.digits)
		}
		cur := format(uint64(max(s.Current, 0)), w.digits)
		padding := cachedWidth - len(cur)
		if padding > 0 {
			return strings.Repeat(" ", padding) + w.render(cur+" / "+cachedTotalStr)
		}
		return w.render(cur + " / " + cachedTotalStr)
	}
}

// widgetPad returns a function that right-aligns content to a stable width.
// raw is the unstyled string used for width tracking; content is what is
// actually displayed (potentially styled). Padding spaces are always plain so
// that background colours do not bleed into alignment space.
func widgetPad() func(raw, content string) string {
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

// WidgetETA returns a [BarWidget] that displays the estimated time remaining
// based on elapsed time and current progress (e.g. "ETA 2m30s", "ETA 5s").
// The result is right-aligned to the widest value seen so far to prevent the
// bar from jumping as the ETA shrinks. Returns "" when the bar is complete
// (current >= total), "ETA ∞" when the rate is zero (no progress yet).
func WidgetETA(opts ...WidgetOption) BarWidget {
	w := widget{}
	applyWidgetOpts(&w, opts)
	pad := widgetPad()

	return func(s BarState) string {
		if s.Total > 0 && s.Current >= s.Total {
			return ""
		}
		var raw string
		if s.Rate <= 0 {
			raw = "ETA ∞"
		} else {
			remaining := float64(s.Total-s.Current) / s.Rate
			d := time.Duration(remaining * float64(time.Second))
			raw = "ETA " + formatETA(d)
		}
		return pad(raw, w.render(raw))
	}
}

// WidgetRate returns a [BarWidget] that displays throughput in items per second
// (e.g. "150/s", "1.5k/s"). The result is right-aligned to the widest value
// seen so far to prevent the bar from jumping. Use [WithUnit] to add a label:
// "150 ops/s".
func WidgetRate(opts ...WidgetOption) BarWidget {
	w := widget{}
	applyWidgetOpts(&w, opts)

	pad := widgetPad()

	return func(s BarState) string {
		raw := formatRate(s.Rate, w.unit)
		return pad(raw, w.render(raw))
	}
}

// WidgetBytesRate returns a [BarWidget] that displays throughput in SI byte
// units per second (e.g. "82.9 MB/s", "1.5 GB/s"). The result is right-aligned
// to the widest value seen so far to prevent the bar from jumping. Reuses the
// SI formatting from [humanBytes].
func WidgetBytesRate(opts ...WidgetOption) BarWidget {
	w := widget{digits: 3} //nolint:mnd // default significant digits
	applyWidgetOpts(&w, opts)

	pad := widgetPad()

	return func(s BarState) string {
		var raw string
		if s.Rate <= 0 {
			raw = "0 B/s"
		} else {
			raw = humanBytes(uint64(s.Rate), w.digits) + "/s"
		}
		return pad(raw, w.render(raw))
	}
}

// WidgetIBytesRate returns a [BarWidget] that displays throughput in IEC byte
// units per second (e.g. "82.9 MiB/s", "1.5 GiB/s"). The result is right-aligned
// to the widest value seen so far to prevent the bar from jumping. Reuses the
// IEC formatting from [humanIBytes].
func WidgetIBytesRate(opts ...WidgetOption) BarWidget {
	w := widget{digits: 3} //nolint:mnd // default significant digits
	applyWidgetOpts(&w, opts)

	pad := widgetPad()

	return func(s BarState) string {
		var raw string
		if s.Rate <= 0 {
			raw = "0 B/s"
		} else {
			raw = humanIBytes(uint64(s.Rate), w.digits) + "/s"
		}
		return pad(raw, w.render(raw))
	}
}

// formatRate formats items/second as a compact string: "0/s", "150/s", "1.5k/s".
// When unit is non-empty it is inserted before the "/s": "150 ops/s".
func formatRate(rate float64, unit string) string {
	var num string
	switch {
	case rate <= 0:
		num = "0"
	case rate >= 1_000_000: //nolint:mnd // million threshold
		num = trimDecimalZeros(fmt.Sprintf("%.1f", rate/1_000_000)) + "M" //nolint:mnd // million
	case rate >= 1000: //nolint:mnd // kilo threshold
		num = trimDecimalZeros(fmt.Sprintf("%.1f", rate/1000)) + "k" //nolint:mnd // kilo
	case rate >= 1:
		num = trimDecimalZeros(fmt.Sprintf("%.1f", rate))
	default:
		num = trimDecimalZeros(fmt.Sprintf("%.2f", rate))
	}

	if unit != "" {
		return num + " " + unit + "/s"
	}
	return num + "/s"
}

// barPercentValue computes the clamped percentage as a float64.
func barPercentValue(current, total int) float64 {
	if total <= 0 {
		return 0
	}
	pct := float64(current) / float64(total) * percentMax
	return min(pct, percentMax)
}
