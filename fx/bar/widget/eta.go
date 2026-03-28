package widget

import (
	"strconv"
	"time"

	"github.com/gechr/clog/fx/bar"
)

// ETA returns a [bar.Widget] that displays the estimated time remaining
// based on elapsed time and current progress (e.g. "ETA 2m30s", "ETA 5s").
// The result is right-aligned to the widest value seen so far to prevent the
// bar from jumping as the ETA shrinks. Returns "" when the bar is complete
// (current >= total), "ETA ∞" when the rate is zero (no progress yet).
//
// Use [WithPrefix] to change or remove the "ETA " prefix:
//
//	widget.ETA(widget.WithPrefix(""))   // "2m30s" instead of "ETA 2m30s"
//	widget.ETA(widget.WithPrefix("~"))  // "~2m30s"
func ETA(opts ...Option) bar.Widget {
	c := config{}
	applyOptions(&c, opts)

	prefix := "ETA "
	if c.prefix != nil {
		prefix = *c.prefix
	}

	p := pad()

	return func(s bar.State) string {
		if s.Total > 0 && s.Current >= s.Total {
			return ""
		}
		var raw string
		if s.Rate <= 0 {
			raw = prefix + "\u221e"
		} else {
			remaining := float64(s.Total-s.Current) / s.Rate
			d := time.Duration(remaining * float64(time.Second))
			raw = prefix + formatETA(d)
		}
		return p(raw, c.render(raw))
	}
}

// formatETA formats a duration as a compact ETA string, always rounded to
// whole seconds. Uses the same composite format as elapsed formatting:
//   - >= 1h: "1h2m"
//   - >= 1m: "2m30s"
//   - < 1m: "5s", minimum "1s" (never "0s")
func formatETA(d time.Duration) string {
	if d < 0 {
		d = -d
	}

	// Round to whole seconds.
	d = d.Round(time.Second)

	if d >= time.Hour {
		h := int(d / time.Hour)
		remainder := d - time.Duration(h)*time.Hour
		m := int(remainder / time.Minute)
		if m == 0 {
			return strconv.Itoa(h) + "h"
		}
		return strconv.Itoa(h) + "h" + strconv.Itoa(m) + "m"
	}

	if d >= time.Minute {
		m := int(d / time.Minute)
		remainder := d - time.Duration(m)*time.Minute
		s := int(remainder / time.Second)
		if s == 0 {
			return strconv.Itoa(m) + "m"
		}
		return strconv.Itoa(m) + "m" + strconv.Itoa(s) + "s"
	}

	s := max(int(d/time.Second), 1)
	return strconv.Itoa(s) + "s"
}
