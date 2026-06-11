// Package env holds the process-wide environment-variable prefix shared by
// clog's packages, and the "custom prefix first, then CLOG" lookup built on
// it. Root clog and the theme package both delegate here so the prefix can
// never desync between them.
package env

import (
	"os"
	"strings"
	"sync/atomic"
)

// DefaultPrefix is the fallback environment variable prefix.
const DefaultPrefix = "CLOG"

var prefix atomic.Value // stores string; "" means no custom prefix

// SetPrefix sets the custom environment-variable prefix, trimming trailing
// underscores. Pass "" to restore the default-only lookup.
func SetPrefix(p string) {
	prefix.Store(strings.TrimRight(p, "_"))
}

// Prefix returns the current custom prefix ("" when unset).
func Prefix() string {
	p, _ := prefix.Load().(string)
	return p
}

// Lookup reads the environment variable for suffix, checking the custom
// prefix first and falling back to [DefaultPrefix]. It returns the value
// and the name of the variable it came from (the fallback name when
// neither is set), so callers can reference the right variable in error
// messages.
func Lookup(suffix string) (string, string) {
	if p := Prefix(); p != "" {
		name := p + "_" + suffix
		if v := os.Getenv(name); v != "" {
			return v, name
		}
	}
	name := DefaultPrefix + "_" + suffix
	return os.Getenv(name), name
}
