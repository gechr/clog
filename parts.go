package clog

import (
	"fmt"
	"sync"

	"github.com/gechr/clog/style"
)

// PartRenderer renders a custom log line part. Returning "" omits the part
// from that line, exactly as a built-in part with nothing to show is omitted.
// styles is the logger's live configuration and must be treated as read-only:
// mutating it changes subsequent output, and retaining it beyond the call is
// unsafe.
//
// A renderer runs while the logger's lock is held: it must not log through the
// same logger, and should avoid slow work - it is called for every line whose
// part order includes it.
type PartRenderer func(e Entry, styles *style.Config, noColor bool) string

var builtinParts = map[Part]bool{
	PartTimestamp: true, PartLevel: true, PartSymbol: true,
	PartMessage: true, PartFields: true,
}

var (
	partsMu       sync.RWMutex
	partRenderers = map[Part]PartRenderer{}
)

// RegisterPart registers a renderer for a custom [Part], making it available
// to [Logger.SetParts] and [Event.Parts]. Any [Part] value that is not
// built-in may be used; a part with no registered renderer is skipped.
//
// Registering over an existing custom part replaces its renderer, and a nil
// renderer unregisters it. It panics if p is a built-in part.
func RegisterPart(p Part, render PartRenderer) {
	if builtinParts[p] {
		panic(fmt.Sprintf("clog: cannot override built-in part %d", int(p)))
	}
	if render == nil {
		UnregisterPart(p)
		return
	}
	partsMu.Lock()
	partRenderers[p] = render
	partsMu.Unlock()
}

// UnregisterPart removes a previously registered custom part renderer.
// It is a no-op for built-in parts.
func UnregisterPart(p Part) {
	if builtinParts[p] {
		return
	}
	partsMu.Lock()
	delete(partRenderers, p)
	partsMu.Unlock()
}

// lookupPartRenderer returns the renderer registered for p, or nil.
func lookupPartRenderer(p Part) PartRenderer {
	partsMu.RLock()
	defer partsMu.RUnlock()
	return partRenderers[p]
}
