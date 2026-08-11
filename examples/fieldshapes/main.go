package main

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog"
	"github.com/gechr/clog/style"
)

func main() {
	// Shapes decide which tokens a field renders...
	clog.SetFieldShapes(clog.FieldShapeMap{
		"region": {OmitKey: true, Prefix: "(", Suffix: ")"},
		"branch": {OmitKey: true, Prefix: "@"},
		"build":  {Prefix: "#"},
	})

	// ...styles decide their color. The two compose, so the badge below keeps
	// a different color per region - brackets included.
	styles := clog.DefaultStyles()
	styles.KeyValues["region"] = style.KeyValue{
		Values: style.ValueMap{
			"emea": new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))), // green
			"apac": new(lipgloss.NewStyle().Foreground(lipgloss.Color("4"))), // blue
		},
		Default: new(lipgloss.NewStyle().Faint(true)),
	}
	clog.SetStyles(styles)

	clog.Info().Str("region", "emea").Str("queue", "ingest").Msg("Draining queue")
	clog.Info().Str("region", "apac").Str("queue", "ingest").Msg("Draining queue")
	clog.Info().Str("region", "antarctica").Msg("Unlisted value uses the faint default")

	// A prefix alone, and a shape that keeps its key.
	clog.Info().Str("branch", "topic").Int("build", 412).Msg("Pipeline queued")

	// Affixes wrap outside quoting and slice brackets.
	clog.Info().Str("region", "emea north").Msg("Values needing quotes stay wrapped")

	// Shapes are structural: clearing them restores key=value everywhere.
	clog.SetFieldShapes(nil)
	clog.Info().Str("region", "emea").Str("branch", "topic").Msg("Shapes cleared")
}
