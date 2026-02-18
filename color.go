package clog

import (
	"fmt"
	"sync/atomic"
)

// noColorEnvSet is loaded once at init time from the NO_COLOR environment variable.
var noColorEnvSet atomic.Bool

// ColorsDisabled returns true if colours are disabled on the [Default] logger.
func ColorsDisabled() bool {
	return Default.output.ColorsDisabled()
}

// MarshalText implements [encoding.TextMarshaler].
func (m ColorMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (m *ColorMode) UnmarshalText(text []byte) error {
	switch string(text) {
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
