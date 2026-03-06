// Package duration provides duration field configuration for clog.
package duration

import (
	"sync/atomic"
	"time"
)

// formatFunc holds the global custom format function.
var formatFunc atomic.Pointer[func(time.Duration) string]

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
	formatFunc *func(time.Duration) string
}

// Save captures the current duration configuration so it can be
// restored later with [Restore]. Typical usage in tests:
//
//	snap := duration.Save()
//	t.Cleanup(func() { duration.Restore(snap) })
func Save() Snapshot {
	return Snapshot{
		formatFunc: formatFunc.Load(),
	}
}

// Restore resets the duration configuration to a previously saved [Snapshot].
func Restore(s Snapshot) {
	formatFunc.Store(s.formatFunc)
}
