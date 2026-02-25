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
	stylesPage := flag.Int("styles-page", 0, "show a specific page of spinner presets (1-indexed, 10 per page)")
	flag.Parse()

	clog.SetLevel(clog.TraceLevel)
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

func showStyles(page int) {
	type entry struct {
		name    string
		spinner clog.SpinnerStyle
	}

	all := []entry{
		{"Aesthetic", clog.SpinnerAesthetic},
		{"Arc", clog.SpinnerArc},
		{"Arrow2", clog.SpinnerArrow2},
		{"Arrow3", clog.SpinnerArrow3},
		{"Balloon", clog.SpinnerBalloon},
		{"Balloon2", clog.SpinnerBalloon2},
		{"BetaWave", clog.SpinnerBetaWave},
		{"Binary", clog.SpinnerBinary},
		{"BluePulse", clog.SpinnerBluePulse},
		{"BouncingBall", clog.SpinnerBouncingBall},
		{"BoxBounce", clog.SpinnerBoxBounce},
		{"BoxBounce2", clog.SpinnerBoxBounce2},
		{"Christmas", clog.SpinnerChristmas},
		{"Circle", clog.SpinnerCircle},
		{"CircleHalves", clog.SpinnerCircleHalves},
		{"CircleQuarters", clog.SpinnerCircleQuarters},
		{"Dot", clog.SpinnerDot},
		{"Dots", clog.SpinnerDots},
		{"Dots3", clog.SpinnerDots3},
		{"Dots4", clog.SpinnerDots4},
		{"Dots5", clog.SpinnerDots5},
		{"Dots6", clog.SpinnerDots6},
		{"Dots7", clog.SpinnerDots7},
		{"Dots8", clog.SpinnerDots8},
		{"Dots8Bit", clog.SpinnerDots8Bit},
		{"Dots9", clog.SpinnerDots9},
		{"Dots11", clog.SpinnerDots11},
		{"Dots12", clog.SpinnerDots12},
		{"Dots13", clog.SpinnerDots13},
		{"Dots14", clog.SpinnerDots14},
		{"DotsCircle", clog.SpinnerDotsCircle},
		{"Dqpb", clog.SpinnerDqpb},
		{"DwarfFortress", clog.SpinnerDwarfFortress},
		{"Ellipsis", clog.SpinnerEllipsis},
		{"FingerDance", clog.SpinnerFingerDance},
		{"Fish", clog.SpinnerFish},
		{"FistBump", clog.SpinnerFistBump},
		{"Flip", clog.SpinnerFlip},
		{"Globe", clog.SpinnerGlobe},
		{"Grenade", clog.SpinnerGrenade},
		{"GrowHorizontal", clog.SpinnerGrowHorizontal},
		{"GrowVertical", clog.SpinnerGrowVertical},
		{"Hamburger", clog.SpinnerHamburger},
		{"Jump", clog.SpinnerJump},
		{"Layer", clog.SpinnerLayer},
		{"Line", clog.SpinnerLine},
		{"Line2", clog.SpinnerLine2},
		{"Material", clog.SpinnerMaterial},
		{"Meter", clog.SpinnerMeter},
		{"Mindblown", clog.SpinnerMindblown},
		{"MiniDot", clog.SpinnerMiniDot},
		{"Monkey", clog.SpinnerMonkey},
		{"Moon", clog.SpinnerMoon},
		{"Noise", clog.SpinnerNoise},
		{"OrangeBluePulse", clog.SpinnerOrangeBluePulse},
		{"OrangePulse", clog.SpinnerOrangePulse},
		{"Pipe", clog.SpinnerPipe},
		{"Point", clog.SpinnerPoint},
		{"Points", clog.SpinnerPoints},
		{"Pong", clog.SpinnerPong},
		{"Pulse", clog.SpinnerPulse},
		{"RollingLine", clog.SpinnerRollingLine},
		{"Runner", clog.SpinnerRunner},
		{"Sand", clog.SpinnerSand},
		{"Shark", clog.SpinnerShark},
		{"SimpleDots", clog.SpinnerSimpleDots},
		{"SimpleDotsScrolling", clog.SpinnerSimpleDotsScrolling},
		{"Smiley", clog.SpinnerSmiley},
		{"SoccerHeader", clog.SpinnerSoccerHeader},
		{"Speaker", clog.SpinnerSpeaker},
		{"SquareCorners", clog.SpinnerSquareCorners},
		{"Squish", clog.SpinnerSquish},
		{"Star2", clog.SpinnerStar2},
		{"TimeTravel", clog.SpinnerTimeTravel},
		{"Toggle", clog.SpinnerToggle},
		{"Toggle2", clog.SpinnerToggle2},
		{"Toggle3", clog.SpinnerToggle3},
		{"Toggle4", clog.SpinnerToggle4},
		{"Toggle5", clog.SpinnerToggle5},
		{"Toggle6", clog.SpinnerToggle6},
		{"Toggle7", clog.SpinnerToggle7},
		{"Toggle8", clog.SpinnerToggle8},
		{"Toggle9", clog.SpinnerToggle9},
		{"Toggle10", clog.SpinnerToggle10},
		{"Toggle11", clog.SpinnerToggle11},
		{"Toggle12", clog.SpinnerToggle12},
		{"Toggle13", clog.SpinnerToggle13},
		{"Triangle", clog.SpinnerTriangle},
		{"Weather", clog.SpinnerWeather},
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
	clog.SetParts(clog.PartMessage, clog.PartPrefix)
	styles := clog.DefaultStyles()
	orange := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styles.Messages[clog.InfoLevel] = &orange
	clog.SetStyles(styles)

	maxName := 0
	for _, e := range all {
		maxName = max(maxName, len(e.name))
	}

	ctx := context.Background()
	g := clog.NewGroup(ctx)
	results := make([]*clog.TaskResult, len(all))
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
}
