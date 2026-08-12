package main

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog"
)

// Define a custom level between Dry (20) and Success (30).
const AuditLevel clog.Level = clog.LevelDry + 5

func main() {
	clog.RegisterLevel(AuditLevel, clog.LevelConfig{
		Name:  "audit",
		Label: "AUD", Symbol: "📋",
		Style: new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))),
	})

	clog.SetLevel(clog.LevelTrace)

	clog.Info().Msg("Starting build")
	clog.Log(AuditLevel).Msg("Config changed")
	clog.Success().Msg("Build completed") // built-in success level
	clog.Warn().Msg("Deprecated flag used")

	// Both custom and built-in levels respect filtering.
	clog.SetLevel(clog.LevelWarn)
	clog.Log(AuditLevel).Msg("This is hidden because level is set to Warn")
	clog.Success().Msg("This is also hidden")
	clog.Warn().Msg("Only warnings and above are visible")
}
