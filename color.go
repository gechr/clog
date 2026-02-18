package clog

import (
	"fmt"
	"os"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var (
	colorsDisabledFlag atomic.Bool
	colorsForced       atomic.Bool
	noColorEnvSet      atomic.Bool
)

// ColorsDisabled returns true if colours should be disabled.
// This is true when NO_COLOR is set, [SetGlobalColorMode]([ColorNever]) was called,
// or stdout is not a terminal -- unless colours were forced via
// [SetGlobalColorMode]([ColorAlways]).
func ColorsDisabled() bool {
	if colorsForced.Load() {
		return false
	}
	return colorsDisabledFlag.Load() || noColorEnvSet.Load() || !IsTerminal()
}

// SetGlobalColorMode sets the colour output mode.
//
//   - [ColorAuto]   -- detect terminal capabilities (default behaviour).
//   - [ColorAlways] -- force colours even when output is not a TTY (overrides NO_COLOR).
//   - [ColorNever]  -- disable all colours and hyperlinks.
//
// Call this early in your application, before creating custom [Styles].
func SetGlobalColorMode(mode ColorMode) {
	switch mode {
	case ColorAlways:
		os.Unsetenv("NO_COLOR")
		colorsForced.Store(true)
		colorsDisabledFlag.Store(false)
		lipgloss.DefaultRenderer().SetOutput(
			termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.TrueColor)),
		)
	case ColorAuto:
		colorsForced.Store(false)
		colorsDisabledFlag.Store(false)
		_, set := os.LookupEnv("NO_COLOR")
		noColorEnvSet.Store(set)
		lipgloss.DefaultRenderer().SetOutput(termenv.NewOutput(os.Stdout))
	case ColorNever:
		_ = os.Setenv("NO_COLOR", "1")
		colorsForced.Store(false)
		colorsDisabledFlag.Store(true)
		hyperlinksEnabled.Store(false)
		lipgloss.DefaultRenderer().SetOutput(
			termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.Ascii)),
		)
	}
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
