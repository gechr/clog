//go:build !unix

package clog

import "os"

// notifyResize is a no-op on platforms without a SIGWINCH equivalent
// (Windows, plan9, js/wasm). The resize goroutine is still started and
// torn down normally; it simply never observes a resize event, so callers
// fall back to re-querying terminal size on demand.
func notifyResize(ch chan os.Signal) {}
