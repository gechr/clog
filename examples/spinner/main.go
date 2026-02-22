package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
)

func main() {
	stylesFlag := flag.Bool("styles", false, "show all spinner presets")
	flag.Parse()

	clog.SetLevel(clog.TraceLevel)
	clog.SetReportTimestamp(true)

	if *stylesFlag {
		showStyles()
		return
	}

	_ = clog.Spinner("Loading configuration").
		Str("file", "config.yaml").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(2 * time.Second)
			return nil
		}).
		Prefix("✅").
		Msg("Configuration loaded")

	_ = clog.Spinner("Running migrations").
		Str("db", "postgres").
		Progress(context.Background(), func(_ context.Context, update *clog.ProgressUpdate) error {
			for i := range 100 {
				update.Msg("Applying migrations").Percent("progress", float64(i+1)).Send()
				time.Sleep(20 * time.Millisecond)
			}
			return nil
		}).
		Prefix("✅").
		Msg("Migrations applied")

	_ = clog.Spinner("Connecting to database").
		Str("host", "db.internal").
		Int("port", 5432).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(1 * time.Second)
			return errors.New("connection refused")
		}).
		Msg("Connected")

	_ = clog.Spinner("Deploying").
		Str("env", "production").
		Progress(context.Background(), func(_ context.Context, update *clog.ProgressUpdate) error {
			update.Msg("Building image").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Pushing image").Str("tag", "v1.2.3").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Starting containers").Send()
			time.Sleep(500 * time.Millisecond)
			return nil
		}).
		Prefix("🚀").
		Msg("Deployed")
}

func showStyles() {
	type entry struct {
		name    string
		spinner clog.SpinnerStyle
	}

	all := []entry{
		{"Aesthetic", clog.SpinnerAesthetic},
		{"Arc", clog.SpinnerArc},
		{"Arrow3", clog.SpinnerArrow3},
		{"BetaWave", clog.SpinnerBetaWave},
		{"BouncingBall", clog.SpinnerBouncingBall},
		{"Circle", clog.SpinnerCircle},
		{"Dots", clog.SpinnerDots},
		{"Dots8Bit", clog.SpinnerDots8Bit},
		{"Dots12", clog.SpinnerDots12},
		{"Flip", clog.SpinnerFlip},
		{"Globe", clog.SpinnerGlobe},
		{"GrowVertical", clog.SpinnerGrowVertical},
		{"Layer", clog.SpinnerLayer},
		{"Line", clog.SpinnerLine},
		{"Material", clog.SpinnerMaterial},
		{"Meter", clog.SpinnerMeter},
		{"Moon", clog.SpinnerMoon},
		{"Noise", clog.SpinnerNoise},
		{"Pipe", clog.SpinnerPipe},
		{"Pong", clog.SpinnerPong},
		{"Runner", clog.SpinnerRunner},
		{"Shark", clog.SpinnerShark},
		{"Smiley", clog.SpinnerSmiley},
		{"SoccerHeader", clog.SpinnerSoccerHeader},
		{"SquareCorners", clog.SpinnerSquareCorners},
		{"Star2", clog.SpinnerStar2},
		{"Toggle", clog.SpinnerToggle},
		{"Triangle", clog.SpinnerTriangle},
		{"Weather", clog.SpinnerWeather},
	}

	clog.SetReportTimestamp(false)
	clog.SetParts(clog.PartMessage, clog.PartPrefix)
	styles := clog.DefaultStyles()
	orange := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styles.Messages[clog.InfoLevel] = &orange
	clog.SetStyles(styles)

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	check := green.Render("✓")

	maxName := 0
	for _, e := range all {
		maxName = max(maxName, len(e.name))
	}

	ctx := context.Background()
	g := clog.NewGroup(ctx)
	results := make([]*clog.SlotResult, len(all))
	for i, e := range all {
		name := fmt.Sprintf("%-*s", maxName, e.name)
		results[i] = g.Add(clog.Spinner(name).
			Style(e.spinner)).
			Run(func(_ context.Context) error {
				time.Sleep(10 * time.Second)
				return nil
			})
	}
	g.Wait()
	for i, e := range all {
		results[i].Prefix(check).Msg(e.name)
	}
}
