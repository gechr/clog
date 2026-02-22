package main

import (
	"errors"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
)

func main() {
	clog.SetLevel(clog.TraceLevel)
	clog.SetReportTimestamp(true)

	styles := clog.DefaultStyles()

	// Per-level message styles
	styles.Messages[clog.TraceLevel] = new(lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("6")))
	styles.Messages[clog.DebugLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("6")))
	styles.Messages[clog.InfoLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("2")))
	styles.Messages[clog.WarnLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("3")))
	styles.Messages[clog.ErrorLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))

	// Custom key style
	styles.KeyDefault = new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")))

	// Custom value styles
	styles.Values["production"] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))
	styles.Values["staging"] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("3")))

	clog.SetStyles(styles)

	clog.Trace().Prefix("🔍").Str("module", "auth").Msg("Token validation started")
	clog.Debug().Prefix("🐛").Str("query", "SELECT *").Duration("latency", 2*time.Millisecond).Msg("Query executed")
	clog.Info().Prefix("🚀").Str("env", "production").Int("port", 8080).Msg("Server started")
	clog.Warn().Prefix("⚡").Str("disk", "92%").Msg("Low disk space")
	clog.Error().Prefix("💥").Err(errors.New("connection refused")).Str("host", "db.internal").Msg("Connection failed")
}
