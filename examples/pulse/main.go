package main

import (
	"context"
	"time"

	"github.com/gechr/clog"
	"github.com/gechr/clog/fx/pulse"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
)

func main() {
	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

	rainbow := []style.ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
		{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
		{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
		{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
		{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
		{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
		{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
	}

	_ = clog.Pulse("Warming up inference engine", pulse.WithGradient(rainbow...)).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(5 * time.Second)
			return nil
		}). Symbol("✅").
		Msg("Inference engine ready")

	_ = clog.Pulse("Replicating data across regions",
		pulse.WithGradient(
			style.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.2, B: 0.2}},
			style.ColorStop{Position: 0.5, Color: colorful.Color{R: 1, G: 1, B: 0.3}},
			style.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.2, B: 0.2}},
		),
	).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(5 * time.Second)
			return nil
		}). Symbol("✅").
		Msg("Data replicated")
}
