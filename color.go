package clog

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// ColorMode controls how a [Logger] determines color and hyperlink output.
//
// ColorMode implements [encoding.TextMarshaler] and [encoding.TextUnmarshaler],
// so it works directly with [flag.TextVar] and most flag libraries.
//
//go:generate go tool golang.org/x/tools/cmd/stringer -type=ColorMode -linecomment
type ColorMode int

const (
	// ColorAuto uses global detection (terminal, NO_COLOR, etc.). This is the default.
	ColorAuto ColorMode = iota // auto
	// ColorAlways forces colors and hyperlinks, even when output is not a TTY.
	ColorAlways // always
	// ColorNever disables colors and hyperlinks.
	ColorNever // never
)

// noColorEnvSet tracks whether the NO_COLOR environment variable is set.
// Must be populated during var init (before Default) so that the default
// logger's renderer respects NO_COLOR. Re-checked by loadNoColorFromEnv
// when SetEnvPrefix is called.
var noColorEnvSet = func() *atomic.Bool {
	var b atomic.Bool
	_, set := os.LookupEnv("NO_COLOR")
	b.Store(set)
	return &b
}()

// MarshalText implements [encoding.TextMarshaler].
func (m ColorMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (m *ColorMode) UnmarshalText(text []byte) error {
	switch strings.ToLower(string(text)) {
	case ColorAuto.String():
		*m = ColorAuto
	case ColorAlways.String():
		*m = ColorAlways
	case ColorNever.String():
		*m = ColorNever
	default:
		return fmt.Errorf("unknown color mode: %q (valid: %q, %q, %q)",
			text, ColorAuto, ColorAlways, ColorNever)
	}
	return nil
}

// ColorsDisabled returns true if colors are disabled on the [Default] logger.
func ColorsDisabled() bool {
	return Default().Output().ColorsDisabled()
}
