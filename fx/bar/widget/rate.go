package widget

import (
	"fmt"

	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/internal/numfmt"
)

// Rate returns a [bar.Widget] that displays throughput in items per second
// (e.g. "150/s", "1.5k/s"). The result is right-aligned to the widest value
// seen so far to prevent the bar from jumping. Use [WithUnit] to add a label:
// "150 ops/s".
func Rate(opts ...Option) bar.Widget {
	c := config{}
	apply(&c, opts...)

	p := pad()

	return func(s bar.State) string {
		raw := FormatRate(s.Rate, c.unit)
		return p(raw, c.render(raw))
	}
}

// BytesRate returns a [bar.Widget] that displays throughput in SI byte
// units per second (e.g. "82.9 MB/s", "1.5 GB/s"). The result is right-aligned
// to the widest value seen so far to prevent the bar from jumping.
func BytesRate(opts ...Option) bar.Widget {
	return bytesRateWidget(numfmt.Bytes, opts)
}

// IBytesRate returns a [bar.Widget] that displays throughput in IEC byte
// units per second (e.g. "82.9 MiB/s", "1.5 GiB/s"). The result is right-aligned
// to the widest value seen so far to prevent the bar from jumping.
func IBytesRate(opts ...Option) bar.Widget {
	return bytesRateWidget(numfmt.IBytes, opts)
}

// bytesRateWidget is the shared implementation for [BytesRate] and [IBytesRate].
func bytesRateWidget(format func(uint64, int) string, opts []Option) bar.Widget {
	c := config{digits: 3} //nolint:mnd // default significant digits
	apply(&c, opts...)

	p := pad()

	return func(s bar.State) string {
		var raw string
		if s.Rate <= 0 {
			raw = "0 B/s"
		} else {
			raw = format(uint64(s.Rate), c.digits) + "/s"
		}
		return p(raw, c.render(raw))
	}
}

// FormatRate formats items/second as a compact string: "0/s", "150/s", "1.5k/s".
// When unit is non-empty it is inserted before the "/s": "150 ops/s".
func FormatRate(rate float64, unit string) string {
	var num string
	switch {
	case rate <= 0:
		num = "0"
	case rate >= 1_000_000: //nolint:mnd // million threshold
		//nolint:mnd // million
		num = numfmt.TrimDecimalZeros(fmt.Sprintf("%.1f", rate/1_000_000)) + "M"
	case rate >= 1000: //nolint:mnd // kilo threshold
		num = numfmt.TrimDecimalZeros(fmt.Sprintf("%.1f", rate/1000)) + "k" //nolint:mnd // kilo
	case rate >= 1:
		num = numfmt.TrimDecimalZeros(fmt.Sprintf("%.1f", rate))
	default:
		num = numfmt.TrimDecimalZeros(fmt.Sprintf("%.2f", rate))
	}

	if unit != "" {
		return num + " " + unit + "/s"
	}
	return num + "/s"
}
