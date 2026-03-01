package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
	"github.com/gechr/clog/style"
)

func main() {
	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

	// Default JSON highlighting
	clog.Error(). Symbol("💥").
		Str("batch", "1/1").
		RawJSON("error", []byte(`{"status":"unprocessable_entity","detail":"rate limit exceeded","code":null}`)).
		Msg("Batch failed")

	clog.Info(). Symbol("📦").
		Str("endpoint", "/api/resources").
		RawJSON("response", []byte(`{"id":"abc123","count":42,"active":true,"tags":["prod","staging"]}`)).
		Msg("Resource fetched")

	// Custom JSON styles
	customStyles := style.DefaultJSON()
	customStyles.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
	customStyles.Null = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Faint(true))
	customStyleSet := clog.DefaultStyles()
	customStyleSet.FieldJSON = customStyles
	clog.SetStyles(customStyleSet)
	clog.Info(). Symbol("🎨").
		RawJSON("payload", []byte(`{"id":"abc123","count":42,"deleted_at":null,"tags":["production"]}`)).
		Msg("Custom styled JSON")
	clog.SetStyles(clog.DefaultStyles())

	// Human mode
	humanStyles := clog.DefaultStyles()
	humanStyles.FieldJSON = style.DefaultJSON()
	humanStyles.FieldJSON.Mode = style.JSONModeHuman
	clog.SetStyles(humanStyles)
	clog.Info(). Symbol("👤").
		RawJSON("response", []byte(`{"status":"ok","count":42,"active":true,"deleted_at":null}`)).
		Msg("Human mode")
	clog.SetStyles(clog.DefaultStyles())

	// Flat mode
	flatStyles := clog.DefaultStyles()
	flatStyles.FieldJSON = style.DefaultJSON()
	flatStyles.FieldJSON.Mode = style.JSONModeFlat
	clog.SetStyles(flatStyles)
	clog.Info(). Symbol("📋").
		RawJSON("response", []byte(`{"user":{"name":"alice","role":"admin"},"tags":["prod","staging"]}`)).
		Msg("Flat mode")
	clog.SetStyles(clog.DefaultStyles())
}
