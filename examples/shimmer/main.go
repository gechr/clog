package main

import (
	"context"
	"flag"
	"time"

	"github.com/gechr/clog"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
)

func main() {
	directionsFlag := flag.Bool("directions", false, "show all shimmer directions")
	flag.Parse()

	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

	if *directionsFlag {
		showDirections()
		return
	}

	_ = clog.Shimmer("Indexing documents and rebuilding search catalogue").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(3 * time.Second)
			return nil
		}). Symbol("✅").
		Msg("Search catalogue rebuilt")

	_ = clog.Shimmer("Deploying service to production cluster",
		shimmer.WithGradient(
			style.ColorStop{Position: 0, Color: colorful.Color{R: 0.3, G: 0.3, B: 0.8}},
			style.ColorStop{Position: 0.5, Color: colorful.Color{R: 1, G: 1, B: 1}},
			style.ColorStop{Position: 1, Color: colorful.Color{R: 0.3, G: 0.3, B: 0.8}},
		),
		shimmer.WithDirection(shimmer.MiddleIn),
	).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(3 * time.Second)
			return nil
		}). Symbol("🚀").
		Msg("Service deployed")
}

func showDirections() {
	rainbow := []style.ColorStop{
		{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
		{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
		{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
		{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
		{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
		{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
		{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
	}

	sleep := func(_ context.Context) error {
		time.Sleep(10 * time.Second)
		return nil
	}

	g := clog.Group(context.Background())
	g.Add(clog.Shimmer("Right: streaming data to downstream services",
		shimmer.WithGradient(rainbow...),
		shimmer.WithDirection(shimmer.Right))).
		Run(sleep)
	g.Add(clog.Shimmer("Left: rewinding transaction log to checkpoint",
		shimmer.WithGradient(rainbow...),
		shimmer.WithDirection(shimmer.Left))).
		Run(sleep)
	g.Add(clog.Shimmer("Middle in: synchronizing upstream dependencies",
		shimmer.WithGradient(rainbow...),
		shimmer.WithDirection(shimmer.MiddleIn))).
		Run(sleep)
	g.Add(clog.Shimmer("Middle out: broadcasting config changes to nodes",
		shimmer.WithGradient(rainbow...),
		shimmer.WithDirection(shimmer.MiddleOut))).
		Run(sleep)
	g.Add(clog.Shimmer("Bounce in: converging replicas and verifying quorum",
		shimmer.WithGradient(rainbow...),
		shimmer.WithDirection(shimmer.BounceIn))).
		Run(sleep)
	g.Add(clog.Shimmer("Bounce out: propagating cache invalidation",
		shimmer.WithGradient(rainbow...),
		shimmer.WithDirection(shimmer.BounceOut))).
		Run(sleep)
	g.Wait().Symbol("✅").Msg("All directions complete")
}
