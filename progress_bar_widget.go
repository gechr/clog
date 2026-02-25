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
	unit   string // unit label for rate widgets (e.g. "ops", "files")
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

// WithUnit sets a unit label for rate widgets. For example, WithUnit("ops")
// produces "150 ops/s" instead of "150/s".
func WithUnit(unit string) WidgetOption {
	return func(c *widget) { c.unit = unit }
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

// widgetPad returns a function that right-aligns strings to a stable width.
// The width ratchets up to the longest string seen so far, preventing the
// bar from jumping as formatted values change length.
func widgetPad() func(string) string {
	var w int
	return func(s string) string {
		w = max(w, len(s))
		return fmt.Sprintf("%*s", w, s)
	}
}

// WidgetETA returns a [BarWidget] that displays the estimated time remaining
// based on elapsed time and current progress (e.g. "ETA 2m30s", "ETA 5s").
// The result is right-aligned to the widest value seen so far to prevent the
// bar from jumping as the ETA shrinks. Returns "" when the bar is complete
// (current >= total), "ETA --" when the rate is zero (no progress yet).
func WidgetETA(_ ...WidgetOption) BarWidget {
	pad := widgetPad()

	return func(s BarState) string {
		if s.Total > 0 && s.Current >= s.Total {
			return ""
		}
		if s.Rate <= 0 {
			return pad("ETA --")
		}
		remaining := float64(s.Total-s.Current) / s.Rate
		d := time.Duration(remaining * float64(time.Second))
		return pad("ETA " + formatETA(d))
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
		return pad(formatRate(s.Rate, w.unit))
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
		if s.Rate <= 0 {
			return pad("0 B/s")
		}
		return pad(humanBytes(uint64(s.Rate), w.digits) + "/s")
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
		if s.Rate <= 0 {
			return pad("0 B/s")
		}
		return pad(humanIBytes(uint64(s.Rate), w.digits) + "/s")
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
