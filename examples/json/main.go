package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
)

func main() {
	clog.SetLevel(clog.TraceLevel)
	clog.SetReportTimestamp(true)

	// Default JSON highlighting
	clog.Error().
		Prefix("💥").
		Str("batch", "1/1").
		RawJSON("error", []byte(`{"status":"unprocessable_entity","detail":"rate limit exceeded","code":null}`)).
		Msg("Batch failed")

	clog.Info().
		Prefix("📦").
		Str("endpoint", "/api/resources").
		RawJSON("response", []byte(`{"id":"abc123","count":42,"active":true,"tags":["prod","staging"]}`)).
		Msg("Resource fetched")

	// Custom JSON styles
	customStyles := clog.DefaultJSONStyles()
	customStyles.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
	customStyles.Null = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Faint(true))
	customStyleSet := clog.DefaultStyles()
	customStyleSet.FieldJSON = customStyles
	clog.SetStyles(customStyleSet)
	clog.Info().
		Prefix("🎨").
		RawJSON("payload", []byte(`{"id":"abc123","count":42,"deleted_at":null,"tags":["production"]}`)).
		Msg("Custom styled JSON")
	clog.SetStyles(clog.DefaultStyles())

	// Human mode
	humanStyles := clog.DefaultStyles()
	humanStyles.FieldJSON = clog.DefaultJSONStyles()
	humanStyles.FieldJSON.Mode = clog.JSONModeHuman
	clog.SetStyles(humanStyles)
	clog.Info().
		Prefix("👤").
		RawJSON("response", []byte(`{"status":"ok","count":42,"active":true,"deleted_at":null}`)).
		Msg("Human mode")
	clog.SetStyles(clog.DefaultStyles())

	// Flat mode
	flatStyles := clog.DefaultStyles()
	flatStyles.FieldJSON = clog.DefaultJSONStyles()
	flatStyles.FieldJSON.Mode = clog.JSONModeFlat
	clog.SetStyles(flatStyles)
	clog.Info().
		Prefix("📋").
		RawJSON("response", []byte(`{"user":{"name":"alice","role":"admin"},"tags":["prod","staging"]}`)).
		Msg("Flat mode")
	clog.SetStyles(clog.DefaultStyles())
}
