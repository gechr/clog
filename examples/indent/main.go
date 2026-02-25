package main

import "github.com/gechr/clog"

func main() {
	clog.Info().Msg("Building project")

	build := clog.With().Indent().Logger()
	build.Info().Msg("Compiling main.go")
	build.Info().Msg("Compiling util.go")

	link := build.With().Indent().Logger()
	link.Info().Msg("Linking binary")
	link.Info().Msg("Stripping symbols")

	clog.Info().Msg("Build complete")
}
