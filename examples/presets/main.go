package main

import "github.com/gechr/clog"

func main() {
	// One call replaces dozens of imperative Set* calls.
	clog.ApplyPreset(clog.TersePreset())

	clog.Info().Str("service", "billing").Msg("Starting")
	clog.Success().Int("routes", 42).Msg("Mounted")
	clog.Notice().Str("path", "~/src/code").Msg("Skipping existing checkout")
	clog.Dry().Str("cmd", "rm -rf ./cache").Msg("Would remove")
	clog.Warn().Str("region", "emea").Msg("Degraded upstream")
	clog.Error().Str("host", "db-1").Msg("Connection refused")

	// Presets layer: a sparse preset only changes the fields it sets.
	clog.ApplyPreset(&clog.Preset{
		Symbols: clog.LabelMap{clog.LevelInfo: ">>"},
	})
	clog.Info().Msg("Symbol changed, everything else kept")
}
