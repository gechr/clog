package main

import "github.com/gechr/clog"

func main() {
	clog.Info().Msg("Project")

	src := clog.With().Tree(clog.TreeMiddle).Logger()
	src.Info().Msg("src/")

	main := src.With().Tree(clog.TreeMiddle).Logger()
	main.Info().Msg("main.go")

	util := src.With().Tree(clog.TreeLast).Logger()
	util.Info().Msg("util.go")

	mod := clog.With().Tree(clog.TreeLast).Logger()
	mod.Info().Msg("go.mod")
}
