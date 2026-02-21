package clog

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// noColorEnvSet is loaded eagerly during package var init (before Default)
// from the NO_COLOR environment variable.
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

// ColorsDisabled returns true if colours are disabled on the [Default] logger.
func ColorsDisabled() bool {
	return Default.Output().ColorsDisabled()
}
