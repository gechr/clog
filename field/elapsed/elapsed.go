// Package elapsed provides elapsed time field configuration for clog.
package elapsed

import (
	"sync/atomic"
	"time"
)

// formatFunc holds the global custom format function.
var formatFunc atomic.Pointer[func(time.Duration) string]

// minimum is the minimum duration (in nanoseconds) below which elapsed is hidden.
var minimum atomic.Int64

// precision holds the decimal precision for formatting.
var precision atomic.Int32

// round holds the rounding duration (in nanoseconds).
var round atomic.Int64

func init() {
	minimum.Store(int64(time.Second))
	round.Store(int64(time.Second))
}

// SetFormatFunc configures a custom format function for elapsed durations.
// When set, this function is used instead of the default formatter.
// Pass nil to restore the default.
func SetFormatFunc(fn func(time.Duration) string) {
	if fn == nil {
		formatFunc.Store(nil)
	} else {
		formatFunc.Store(&fn)
	}
}

// SetMinimum sets the minimum duration below which elapsed fields are hidden.
// Defaults to [time.Second]. Set to 0 to show all values.
func SetMinimum(d time.Duration) {
	minimum.Store(int64(d))
}

// SetPrecision sets the decimal precision for elapsed time formatting.
// For example, 0 = "3s", 1 = "3.2s", 2 = "3.21s". Defaults to 0.
func SetPrecision(n int) {
	precision.Store(int32(n)) //nolint:gosec // precision is a small positive integer
}

// SetRound sets the rounding duration for elapsed time values.
// Defaults to [time.Second]. Set to 0 to disable rounding.
func SetRound(d time.Duration) {
	round.Store(int64(d))
}

// FormatFunc returns the current custom format function, or nil if using default.
func FormatFunc() func(time.Duration) string {
	p := formatFunc.Load()
	if p == nil {
		return nil
	}
	return *p
}

// Minimum returns the minimum duration threshold.
func Minimum() time.Duration {
	return time.Duration(minimum.Load())
}

// Precision returns the current decimal precision.
func Precision() int {
	return int(precision.Load())
}

// Round returns the current rounding duration.
func Round() time.Duration {
	return time.Duration(round.Load())
}

// Snapshot captures the current state of all elapsed configuration.
// Use [Restore] to reset the state in test cleanup.
type Snapshot struct {
	formatFunc *func(time.Duration) string
	minimum    int64
	precision  int32
	round      int64
}

// Save captures the current elapsed configuration so it can be
// restored later with [Restore]. Typical usage in tests:
//
//	snap := elapsed.Save()
//	t.Cleanup(func() { elapsed.Restore(snap) })
func Save() Snapshot {
	return Snapshot{
		formatFunc: formatFunc.Load(),
		minimum:    minimum.Load(),
		precision:  precision.Load(),
		round:      round.Load(),
	}
}

// Restore resets the elapsed configuration to a previously saved [Snapshot].
func Restore(s Snapshot) {
	formatFunc.Store(s.formatFunc)
	minimum.Store(s.minimum)
	precision.Store(s.precision)
	round.Store(s.round)
}
