package clog

import (
	"fmt"
	"time"
)

// BarState holds progress information passed to [BarWidget] functions.
type BarState struct {
	Current int
	Total   int
	Elapsed time.Duration
}

// BarWidget renders a text label from the current bar state.
// Return "" to display nothing for this tick.
type BarWidget func(BarState) string

// WidgetOption configures a bar widget constructor.
type WidgetOption func(*widget)

// widget holds resolved options for widget constructors.
type widget struct {
	digits int // significant digits for bytes; decimal places for percent
}

func applyWidgetOpts(w *widget, opts []WidgetOption) {
	for _, o := range opts {
		o(w)
	}
}

// WithDigits configures the precision of formatted values.
//   - [WidgetBytes] / [WidgetIBytes]: significant digits (default 3).
//     3 → "82.9 MB", "1 GB"; 2 → "83 MB", "5.5 GB"
//   - [WidgetPercent]: decimal places (default 0).
//     0 → "42%"; 1 → "42.5%"
func WithDigits(n int) WidgetOption {
	return func(c *widget) { c.digits = n }
}

// WidgetNone is a [BarWidget] that always returns "".
// Use it to explicitly suppress the default percent display.
var WidgetNone BarWidget = func(BarState) string { return "" }

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
		return fmt.Sprintf("%*s", padWidth, str)
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
		return fmt.Sprintf("%*s / %s", cachedWidth, cur, cachedTotalStr)
	}
}

// barPercentValue computes the clamped percentage as a float64.
func barPercentValue(current, total int) float64 {
	if total <= 0 {
		return 0
	}
	pct := float64(current) / float64(total) * percentMax
	return min(pct, percentMax)
}
