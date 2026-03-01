package widget

import (
	"strings"

	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/internal/numfmt"
)

// Bytes returns a [bar.Widget] that displays download-style progress
// in human-readable SI byte units (e.g. "1.5 GB / 2 GB"). The [bar.State]
// Current and Total fields are interpreted as byte counts.
// Uses base-1000 units (kB, MB, GB). See [IBytes] for base-1024 (KiB, MiB, GiB).
// Default digits is 3; use [WithDigits] to change.
func Bytes(opts ...Option) bar.Widget {
	return bytesWidget(numfmt.Bytes, numfmt.BytesWidth, opts)
}

// IBytes returns a [bar.Widget] that displays download-style progress
// in human-readable IEC byte units (e.g. "1.5 GiB / 2 GiB"). The [bar.State]
// Current and Total fields are interpreted as byte counts.
// Uses base-1024 units (KiB, MiB, GiB). See [Bytes] for base-1000 (kB, MB, GB).
// Default digits is 3; use [WithDigits] to change.
func IBytes(opts ...Option) bar.Widget {
	return bytesWidget(numfmt.IBytes, numfmt.IBytesWidth, opts)
}

// bytesWidget is the shared implementation for [Bytes] and [IBytes].
// maxWidth returns the unstripped formatted width for stable right-alignment.
func bytesWidget(
	format func(uint64, int) string,
	maxWidth func(uint64, int) int,
	opts []Option,
) bar.Widget {
	c := config{digits: 3} //nolint:mnd // default significant digits
	applyOptions(&c, opts)

	// Cache the formatted total to avoid re-computing every tick.
	var cachedTotal int
	var cachedTotalStr string
	var cachedWidth int

	return func(s bar.State) string {
		if s.Total != cachedTotal || cachedTotalStr == "" {
			cachedTotal = s.Total
			tot := uint64(max(s.Total, 0))
			cachedTotalStr = format(tot, c.digits)
			cachedWidth = maxWidth(tot, c.digits)
		}
		cur := format(uint64(max(s.Current, 0)), c.digits)
		padding := cachedWidth - len(cur)
		if padding > 0 {
			return strings.Repeat(" ", padding) + c.render(cur+" / "+cachedTotalStr)
		}
		return c.render(cur + " / " + cachedTotalStr)
	}
}
