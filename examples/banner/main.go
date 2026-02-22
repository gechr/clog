package main

import (
	"context"
	"errors"
	"time"

	"github.com/gechr/clog"
	"github.com/lucasb-eyer/go-colorful"
)

func main() {
	clog.SetLevel(clog.TraceLevel)
	clog.SetReportTimestamp(true)

	_ = clog.Shimmer("Initializing environment and loading configuration modules",
		clog.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
		clog.ColorStop{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
		clog.ColorStop{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
		clog.ColorStop{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
		clog.ColorStop{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
		clog.ColorStop{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
		clog.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
	).
		ShimmerDirection(clog.DirectionMiddleIn).
		Str("eta", "Soon™").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(3 * time.Second)
			return nil
		}).
		Prefix("✅").
		Msg("Environment initialized")

	_ = clog.Spinner("Validating config").
		Str("file", "app.toml").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(1 * time.Second)
			return errors.New("missing required field: port")
		}).
		Err()

	_ = clog.Spinner("Deploying").
		Str("env", "production").
		Progress(context.Background(), func(_ context.Context, update *clog.ProgressUpdate) error {
			update.Msg("Building image").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Pushing image").Str("tag", "v1.2.3").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Starting containers").Send()
			time.Sleep(1 * time.Second)
			return nil
		}).
		Prefix("🚀").
		Msg("Deployed")

	_ = clog.Spinner("Running migrations").
		Str("db", "postgres").
		Progress(context.Background(), func(_ context.Context, update *clog.ProgressUpdate) error {
			hundred := 100
			for i := range hundred {
				progress := min(i+1, hundred)
				update.Msg("Applying migrations").Percent("progress", float64(progress)).Send()
				time.Sleep(30 * time.Millisecond)
			}
			return nil
		}).
		Prefix("✅").
		Msg("Migrations applied")

	_ = clog.Spinner("Downloading artifacts").
		Style(clog.SpinnerDot).
		Str("repo", "gechr/clog").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(2 * time.Second)
			return nil
		}).
		Prefix("📦").
		Msg("Artifacts downloaded")

	_ = clog.Spinner("Connecting to database").
		Str("host", "db.internal").
		Int("port", 5432).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(2 * time.Second)
			return errors.New("connection refused")
		}).
		OnErrorLevel(clog.FatalLevel).
		Send()
}
