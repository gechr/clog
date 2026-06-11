package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog"
	"github.com/gechr/clog/fx/spinner"
)

func main() {
	stylesFlag := flag.Bool("styles", false, "show all spinner presets")
	stylesPage := flag.Int(
		"styles-page",
		0,
		"show a specific page of spinner presets (1-indexed, 10 per page)",
	)
	flag.Parse()

	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

	if *stylesFlag || *stylesPage > 0 {
		showStyles(*stylesPage)
		return
	}

	_ = clog.Spinner("Loading configuration").
		Str("file", "config.yaml").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(2 * time.Second)
			return nil
		}).Symbol("✅").
		Msg("Configuration loaded")

	_ = clog.Spinner("Running migrations").
		Str("db", "postgres").
		Progress(context.Background(), func(_ context.Context, update *clog.Update) error {
			for i := range 100 {
				update.Msg("Applying migrations").Percent("progress", float64(i+1)).Send()
				time.Sleep(20 * time.Millisecond)
			}
			return nil
		}).Symbol("✅").
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
		Progress(context.Background(), func(_ context.Context, update *clog.Update) error {
			update.Msg("Building image").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Pushing image").Str("tag", "v1.2.3").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Starting containers").Send()
			time.Sleep(500 * time.Millisecond)
			return nil
		}).Symbol("🚀").
		Msg("Deployed")
}

func showStyles(page int) {
	type entry struct {
		name string
		s    spinner.Config
	}

	all := []entry{
		{"Aesthetic", spinner.Aesthetic},
		{"Arc", spinner.Arc},
		{"Arrow2", spinner.Arrow2},
		{"Arrow3", spinner.Arrow3},
		{"Balloon", spinner.Balloon},
		{"Balloon2", spinner.Balloon2},
		{"BetaWave", spinner.BetaWave},
		{"Binary", spinner.Binary},
		{"BluePulse", spinner.BluePulse},
		{"BouncingBall", spinner.BouncingBall},
		{"BoxBounce", spinner.BoxBounce},
		{"BoxBounce2", spinner.BoxBounce2},
		{"Christmas", spinner.Christmas},
		{"Circle", spinner.Circle},
		{"CircleHalves", spinner.CircleHalves},
		{"CircleQuarters", spinner.CircleQuarters},
		{"Dot", spinner.Dot},
		{"Dots", spinner.Dots},
		{"Dots3", spinner.Dots3},
		{"Dots4", spinner.Dots4},
		{"Dots5", spinner.Dots5},
		{"Dots6", spinner.Dots6},
		{"Dots7", spinner.Dots7},
		{"Dots8", spinner.Dots8},
		{"Dots8Bit", spinner.Dots8Bit},
		{"Dots9", spinner.Dots9},
		{"Dots11", spinner.Dots11},
		{"Dots12", spinner.Dots12},
		{"Dots13", spinner.Dots13},
		{"Dots14", spinner.Dots14},
		{"DotsCircle", spinner.DotsCircle},
		{"Dqpb", spinner.Dqpb},
		{"DwarfFortress", spinner.DwarfFortress},
		{"Ellipsis", spinner.Ellipsis},
		{"FingerDance", spinner.FingerDance},
		{"Fish", spinner.Fish},
		{"FistBump", spinner.FistBump},
		{"Flip", spinner.Flip},
		{"Globe", spinner.Globe},
		{"Grenade", spinner.Grenade},
		{"GrowHorizontal", spinner.GrowHorizontal},
		{"GrowVertical", spinner.GrowVertical},
		{"Hamburger", spinner.Hamburger},
		{"Jump", spinner.Jump},
		{"Layer", spinner.Layer},
		{"Line", spinner.Line},
		{"Line2", spinner.Line2},
		{"Material", spinner.Material},
		{"Meter", spinner.Meter},
		{"Mindblown", spinner.Mindblown},
		{"MiniDot", spinner.MiniDot},
		{"Monkey", spinner.Monkey},
		{"Moon", spinner.Moon},
		{"Noise", spinner.Noise},
		{"OrangeBluePulse", spinner.OrangeBluePulse},
		{"OrangePulse", spinner.OrangePulse},
		{"Pipe", spinner.Pipe},
		{"Point", spinner.Point},
		{"Points", spinner.Points},
		{"Pong", spinner.Pong},
		{"Pulse", spinner.Pulse},
		{"RollingLine", spinner.RollingLine},
		{"Runner", spinner.Runner},
		{"Sand", spinner.Sand},
		{"Shark", spinner.Shark},
		{"SimpleDots", spinner.SimpleDots},
		{"SimpleDotsScrolling", spinner.SimpleDotsScrolling},
		{"Smiley", spinner.Smiley},
		{"SoccerHeader", spinner.SoccerHeader},
		{"Speaker", spinner.Speaker},
		{"SquareCorners", spinner.SquareCorners},
		{"Squish", spinner.Squish},
		{"Stars", spinner.Stars},
		{"TimeTravel", spinner.TimeTravel},
		{"Toggle", spinner.Toggle},
		{"Toggle2", spinner.Toggle2},
		{"Toggle3", spinner.Toggle3},
		{"Toggle4", spinner.Toggle4},
		{"Toggle5", spinner.Toggle5},
		{"Toggle6", spinner.Toggle6},
		{"Toggle7", spinner.Toggle7},
		{"Toggle8", spinner.Toggle8},
		{"Toggle9", spinner.Toggle9},
		{"Toggle10", spinner.Toggle10},
		{"Toggle11", spinner.Toggle11},
		{"Toggle12", spinner.Toggle12},
		{"Toggle13", spinner.Toggle13},
		{"Triangle", spinner.Triangle},
		{"Weather", spinner.Weather},
	}

	const pageSize = 10
	if page > 0 {
		start := (page - 1) * pageSize
		if start >= len(all) {
			return
		}
		end := min(start+pageSize, len(all))
		all = all[start:end]
	}

	clog.SetReportTimestamp(false)
	clog.SetParts(clog.PartMessage, clog.PartSymbol)
	styles := clog.DefaultStyles()
	orange := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styles.Messages[clog.LevelInfo] = &orange
	clog.SetStyles(styles)

	maxName := 0
	for _, e := range all {
		maxName = max(maxName, len(e.name))
	}

	ctx := context.Background()
	g := clog.Group(ctx)
	results := make([]*clog.TaskResult, len(all))
	for i, e := range all {
		name := fmt.Sprintf("%-*s", maxName, e.name)
		results[i] = g.Add(clog.Spinner(name, spinner.WithConfig(e.s))).
			Run(func(_ context.Context) error {
				time.Sleep(10 * time.Second)
				return nil
			})
	}
	g.Wait()
}
