package clog

import (
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
// This is true when NO_COLOR is set, [ConfigureColorOutput]("never") was called,
// or stdout is not a terminal — unless colours were forced via
// [ConfigureColorOutput]("always").
func ColorsDisabled() bool {
	if colorsForced.Load() {
		return false
	}

	return colorsDisabledFlag.Load() || noColorEnvSet.Load() || !IsTerminal()
}

// ConfigureColorOutput sets the colour output mode.
//
//   - "auto"   — detect terminal capabilities (default behaviour).
//   - "always" — force colours even when output is not a TTY (overrides NO_COLOR).
//   - "never"  — disable all colours and hyperlinks.
//
// Call this early in your application, before creating custom [Styles].
func ConfigureColorOutput(mode string) {
	switch mode {
	case "always":
		os.Unsetenv("NO_COLOR")
		colorsForced.Store(true)
		colorsDisabledFlag.Store(false)
		lipgloss.DefaultRenderer().SetOutput(
			termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.TrueColor)),
		)
	case "auto":
		colorsForced.Store(false)
		colorsDisabledFlag.Store(false)
		_, set := os.LookupEnv("NO_COLOR")
		noColorEnvSet.Store(set)
		lipgloss.DefaultRenderer().SetOutput(termenv.NewOutput(os.Stdout))
	case "never":
		_ = os.Setenv("NO_COLOR", "1")
		colorsForced.Store(false)
		colorsDisabledFlag.Store(true)
		hyperlinksEnabled.Store(false)
		lipgloss.DefaultRenderer().SetOutput(
			termenv.NewOutput(os.Stdout, termenv.WithProfile(termenv.Ascii)),
		)
	}
}

func init() {
	// Check NO_COLOR per https://no-color.org/ — presence of the variable
	// (regardless of value, including empty) disables colours.
	_, set := os.LookupEnv("NO_COLOR")
	noColorEnvSet.Store(set)
}
