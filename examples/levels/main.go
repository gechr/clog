package main

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog"
)

// Define a custom level between Dry (2) and Warn (5).
const SuccessLevel clog.Level = clog.LevelDry + 1

func main() {
	clog.RegisterLevel(SuccessLevel, clog.LevelConfig{
		Name:  "success",
		Label: "SCS", Symbol: "✅",
		Style: new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))),
	})

	clog.SetLevel(clog.LevelTrace)

	clog.Info().Msg("Starting build")
	clog.Log(SuccessLevel).Msg("Build completed")
	clog.Warn().Msg("Deprecated flag used")

	// Custom level respects filtering.
	clog.SetLevel(clog.LevelWarn)
	clog.Log(SuccessLevel).Msg("This is hidden because level is set to Warn")
	clog.Warn().Msg("Only warnings and above are visible")
}
