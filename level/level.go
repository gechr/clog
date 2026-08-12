// Package level defines the Level type and built-in level constants for clog.
//
// This package exists so that both the root clog package and subpackages
// like style can reference Level without circular imports.
package level

import (
	"errors"
	"fmt"
	"maps"
	"strings"
	"sync"
)

// ErrUnknownLevel is returned when a level string or value is not recognized.
var ErrUnknownLevel = errors.New("unknown level")

// Level represents a log level. Lower values are more verbose.
//
// Level implements [encoding.TextMarshaler] and [encoding.TextUnmarshaler],
// so it works directly with [flag.TextVar] and most flag libraries.
type Level int

const (
	Trace   Level = -20
	Debug   Level = -10
	Info    Level = 0
	Hint    Level = 10
	Dry     Level = 20
	Success Level = 30
	Notice  Level = 35
	Warn    Level = 40
	Error   Level = 50
	Fatal   Level = 60

	// Unset is passed to [clog.SetNonTTYLevel] to disable the non-TTY level
	// filter. Its value is intentionally below all real log levels so the
	// check e.level < nonTTYLevel is always false, meaning no restriction.
	Unset Level = -1 << 30
)

// Level name strings.
const (
	TraceValue   = "trace"
	DebugValue   = "debug"
	InfoValue    = "info"
	HintValue    = "hint"
	DryValue     = "dry"
	SuccessValue = "success"
	NoticeValue  = "notice"
	WarnValue    = "warn"
	ErrorValue   = "error"
	FatalValue   = "fatal"
)

var builtins = map[Level]bool{
	Trace: true, Debug: true, Info: true, Hint: true, Dry: true,
	Success: true, Notice: true, Warn: true, Error: true, Fatal: true,
}

var labels = map[Level]string{
	Trace: "TRC", Debug: "DBG", Info: "INF", Hint: "HNT", Dry: "DRY",
	Success: "OK", Notice: "NTC", Warn: "WRN", Error: "ERR", Fatal: "FTL",
}

var names = map[Level]string{
	Trace:   TraceValue,
	Debug:   DebugValue,
	Info:    InfoValue,
	Hint:    HintValue,
	Dry:     DryValue,
	Success: SuccessValue,
	Notice:  NoticeValue,
	Warn:    WarnValue,
	Error:   ErrorValue,
	Fatal:   FatalValue,
}

var mu sync.RWMutex

// Register adds a custom level with the given name and short label.
// It panics if l conflicts with a built-in level.
func Register(l Level, name, label string) {
	if builtins[l] {
		panic(fmt.Sprintf("level: cannot override built-in level %d", int(l)))
	}
	mu.Lock()
	names[l] = name
	labels[l] = label
	mu.Unlock()
}

// Unregister removes a previously registered custom level.
// It is a no-op for built-in levels.
func Unregister(l Level) {
	if builtins[l] {
		return
	}
	mu.Lock()
	delete(names, l)
	delete(labels, l)
	mu.Unlock()
}

// Name returns the canonical name for the level (e.g. "info", "error"),
// or "" for unknown levels.
func (l Level) Name() string {
	mu.RLock()
	defer mu.RUnlock()
	return names[l]
}

// String returns the short label for the level (e.g. "INF", "ERR").
func (l Level) String() string {
	mu.RLock()
	defer mu.RUnlock()
	if s, ok := labels[l]; ok {
		return s
	}
	return fmt.Sprintf("LVL(%d)", int(l))
}

// MarshalText implements [encoding.TextMarshaler].
func (l Level) MarshalText() ([]byte, error) {
	mu.RLock()
	defer mu.RUnlock()
	if name, ok := names[l]; ok {
		return []byte(name), nil
	}
	return nil, fmt.Errorf("%w '%d'", ErrUnknownLevel, l)
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (l *Level) UnmarshalText(text []byte) error {
	parsed, err := Parse(string(text))
	if err != nil {
		return err
	}
	*l = parsed
	return nil
}

// parseNames maps accepted built-in name strings to their levels: the
// canonical names plus the aliases "warning" and "critical".
var parseNames = map[string]Level{
	TraceValue:   Trace,
	DebugValue:   Debug,
	InfoValue:    Info,
	HintValue:    Hint,
	DryValue:     Dry,
	SuccessValue: Success,
	NoticeValue:  Notice,
	WarnValue:    Warn,
	"warning":    Warn,
	ErrorValue:   Error,
	FatalValue:   Fatal,
	"critical":   Fatal,
}

// Parse maps a level name string to a [Level] value.
// It accepts the canonical names ("trace", "debug", "info", "hint", "dry",
// "success", "notice", "warn", "error", "fatal") plus aliases
// ("warning" → Warn, "critical" → Fatal). Matching is case-insensitive.
func Parse(s string) (Level, error) {
	lower := strings.ToLower(s)
	if lvl, ok := parseNames[lower]; ok {
		return lvl, nil
	}

	// Check custom levels.
	mu.RLock()
	defer mu.RUnlock()
	for lvl, name := range names {
		if builtins[lvl] {
			continue
		}
		if strings.ToLower(name) == lower {
			return lvl, nil
		}
	}
	return 0, fmt.Errorf("%w %q", ErrUnknownLevel, s)
}

// Labels returns a copy of all level labels (short display strings like "INF", "ERR").
func Labels() map[Level]string {
	mu.RLock()
	defer mu.RUnlock()
	return maps.Clone(labels)
}

// All returns all registered level names (built-in + custom) mapped to their [Level] values.
func All() map[string]Level {
	mu.RLock()
	defer mu.RUnlock()
	m := make(map[string]Level, len(names))
	for l, name := range names {
		m[name] = l
	}
	return m
}
