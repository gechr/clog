package main

import (
	"github.com/gechr/clog"
)

func main() {
	name, err := clog.Input("Name: ")
	if err != nil {
		clog.Fatal().Err(err).Msg("Failed to read name")
	}
	clog.Info().Str("name", name).Msg("Got name")

	pass, err := clog.Password("Password: ")
	if err != nil {
		clog.Fatal().Err(err).Msg("Failed to read password")
	}
	clog.Info().Int("length", len(pass)).Msg("Got password")

	// Password is Input with clog.WithSensitive(true) applied; the option is also
	// available directly.
	token, err := clog.Input("Token: ", clog.WithSensitive(true))
	if err != nil {
		clog.Fatal().Err(err).Msg("Failed to read token")
	}
	clog.Info().Int("length", len(token)).Msg("Got token")
}
