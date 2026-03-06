// Package duration provides duration field configuration for clog.
package duration

import (
	"sync/atomic"
	"time"
)

// formatFunc holds the global custom format function.
var formatFunc atomic.Pointer[func(time.Duration) string]

// gradientMax holds the max duration (in nanoseconds) for gradient mapping.
// 0 means gradient is disabled.
var gradientMax atomic.Int64

// SetGradientMax sets the max duration for duration gradient coloring.
// The gradient maps 0 → max onto the configured color stops.
// Set to 0 (the default) to disable gradient coloring.
func SetGradientMax(d time.Duration) {
	gradientMax.Store(int64(d))
}

// GradientMax returns the current gradient max duration.
// 0 means gradient is disabled.
func GradientMax() time.Duration {
	return time.Duration(gradientMax.Load())
}

// SetFormatFunc configures a custom format function for [time.Duration] and
// elapsed-time fields. When set, this function is called instead of the
// default formatter for both [Event.Duration] fields and [Event.Elapsed]
// fields. For elapsed fields, [elapsed.SetFormatFunc] takes priority when
// both are configured. Pass nil to restore the default.
func SetFormatFunc(fn func(time.Duration) string) {
	if fn == nil {
		formatFunc.Store(nil)
	} else {
		formatFunc.Store(&fn)
	}
}

// FormatFunc returns the current custom format function, or nil if using default.
func FormatFunc() func(time.Duration) string {
	p := formatFunc.Load()
	if p == nil {
		return nil
	}
	return *p
}

// Snapshot captures the current state of all duration configuration.
// Use [Restore] to reset the state in test cleanup.
type Snapshot struct {
	formatFunc  *func(time.Duration) string
	gradientMax int64
}

// Save captures the current duration configuration so it can be
// restored later with [Restore]. Typical usage in tests:
//
//	snap := duration.Save()
//	t.Cleanup(func() { duration.Restore(snap) })
func Save() Snapshot {
	return Snapshot{
		formatFunc:  formatFunc.Load(),
		gradientMax: gradientMax.Load(),
	}
}

// Restore resets the duration configuration to a previously saved [Snapshot].
func Restore(s Snapshot) {
	formatFunc.Store(s.formatFunc)
	gradientMax.Store(s.gradientMax)
}
