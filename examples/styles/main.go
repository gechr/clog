package main

import (
	"errors"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
	"github.com/gechr/clog/field/percent"
)

func main() {
	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

	styles := clog.DefaultStyles()

	// Per-level message styles
	styles.Messages[clog.LevelTrace] = new(lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("6")))
	styles.Messages[clog.LevelDebug] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("6")))
	styles.Messages[clog.LevelInfo] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("2")))
	styles.Messages[clog.LevelWarn] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("3")))
	styles.Messages[clog.LevelError] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))

	// Per-level symbol styles (works with any symbol, not just emojis)
	styles.Symbols[clog.LevelWarn] = new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")))

	// Custom key style
	styles.KeyDefault = new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")))

	// Custom value styles
	styles.Values["production"] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))
	styles.Values["staging"] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("3")))

	clog.SetStyles(styles)

	clog.Trace().Symbol("🔍").Str("module", "auth").Msg("Token validation started")
	clog.Debug().Symbol("🐛").Str("query", "SELECT *").Duration("latency", 2*time.Millisecond).Msg("Query executed")
	clog.Info().Symbol("🚀").Str("env", "production").Int("port", 8080).Msg("Server started")
	clog.Warn().Symbol("!!").Percent("usage", 95, percent.WithReverseGradient()).Msg("Low disk space")
	clog.Error().Symbol("💥").Err(errors.New("connection refused")).Str("host", "db.internal").Msg("Connection failed")
}
