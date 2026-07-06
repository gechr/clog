package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/field/fraction"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
	"github.com/gechr/clog/fx/pulse"
	"github.com/gechr/clog/fx/shimmer"
	"github.com/gechr/clog/fx/spinner"
	"github.com/gechr/clog/style"
	"github.com/lucasb-eyer/go-colorful"
)

func main() {
	demoFlag := flag.Bool("demo", false, "run the demo")
	quickFlag := flag.Bool("quick", false, "skip animations")
	spinnersFlag := flag.String(
		"spinners",
		"",
		"demo spinners (comma-separated names, empty for all)",
	)
	flag.Parse()

	spinnersSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "spinners" {
			spinnersSet = true
		}
	})

	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

	if spinnersSet {
		spinners(*spinnersFlag)
		return
	}

	if *demoFlag {
		demo()
		return
	}

	header := func(h string) {
		fmt.Println()
		s := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
		fmt.Println(s.Render(h))
	}

	if !*quickFlag {
		// --- Shimmer (all directions, rainbow) ---
		header("Shimmer")
		rainbow := []style.ColorStop{
			{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
			{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
			{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
			{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
			{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
			{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
			{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
		}
		sleep3s := func(_ context.Context) error {
			time.Sleep(5 * time.Second)
			return nil
		}
		shimmerGroup := clog.Group(context.Background())
		shimmerGroup.Add(clog.Shimmer("Shimmer right: streaming data to downstream services",
			shimmer.WithGradient(rainbow...),
			shimmer.WithDirection(shimmer.Right))).
			Run(sleep3s)
		shimmerGroup.Add(clog.Shimmer("Shimmer left: rewinding transaction log to checkpoint",
			shimmer.WithGradient(rainbow...),
			shimmer.WithDirection(shimmer.Left))).
			Run(sleep3s)
		shimmerGroup.Add(clog.Shimmer("Middle in: synchronizing upstream dependencies and rebuilding",
			shimmer.WithGradient(rainbow...),
			shimmer.WithDirection(shimmer.MiddleIn))).
			Run(sleep3s)
		shimmerGroup.Add(clog.Shimmer("Middle out: broadcasting configuration changes to edge nodes",
			shimmer.WithGradient(rainbow...),
			shimmer.WithDirection(shimmer.MiddleOut))).
			Run(sleep3s)
		shimmerGroup.Add(clog.Shimmer("Bounce in: converging replicas and verifying quorum",
			shimmer.WithGradient(rainbow...),
			shimmer.WithDirection(shimmer.BounceIn))).
			Run(sleep3s)
		shimmerGroup.Add(clog.Shimmer("Bounce out: propagating cache invalidation to all locations",
			shimmer.WithGradient(rainbow...),
			shimmer.WithDirection(shimmer.BounceOut))).
			Run(sleep3s)
		shimmerGroup.Add(clog.Shimmer("Fast shimmer: rapid gradient cycle at 2× speed",
			shimmer.WithGradient(rainbow...),
			shimmer.WithSpeed(1.0))).
			Run(sleep3s)
		if err := shimmerGroup.Wait().Symbol("✅").Msg("Shimmer demo complete"); err != nil {
			clog.Fatal().Err(err).Msg("shimmer demo failed")
		}

		// --- Group (bar styles + spinner + pulse running concurrently) ---
		header("Bar")

		thinColored := bar.Thin
		thinColored.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("2")))
		thinColored.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
		thinColored.WidgetLeft = widget.Percent()
		thinColored.WidgetRight = widget.None()

		gradientBar := bar.Smooth
		gradientBar.ProgressGradient = bar.DefaultGradient()
		gradientBar.WidgetLeft = widget.Percent()
		gradientBar.WidgetRight = widget.None()

		barFill := func(_ context.Context, p *clog.Update) error {
			for i := range 1001 {
				p.SetProgress(i).Send()
				time.Sleep(3 * time.Millisecond)
			}
			return nil
		}

		downloadBar := bar.Thin
		downloadBar.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))
		downloadBar.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
		downloadBar.WidgetLeft = widget.Percent()
		downloadBar.WidgetRight = widget.None()

		g := clog.Group(context.Background())
		g.Add(clog.Bar("Downloading", 1000, bar.WithConfig(downloadBar)).
			Str("file", "release.tar.gz").Elapsed("elapsed")).
			Progress(barFill)
		g.Add(clog.Bar("Installing", 1000, bar.WithConfig(thinColored)).
			Str("pkg", "clog").BarPercent("progress").Elapsed("elapsed")).
			Progress(barFill)
		g.Add(clog.Bar("Building", 1000, bar.WithConfig(gradientBar)).
			Str("target", "release").Elapsed("elapsed")).
			Progress(barFill)
		g.Add(clog.Bar("Syncing", 1000, bar.WithConfig(bar.Config{
			Placement:    bar.PlaceInline,
			CapLeft:      "│",
			CapRight:     "│",
			CharEmpty:    ' ',
			CharFill:     '█',
			GradientFill: []rune{'▏', '▎', '▍', '▌', '▋', '▊', '▉'},
			Separator:    " ",
			StyleEmpty:   new(lipgloss.NewStyle().Foreground(lipgloss.Color("8"))),
			StyleFill:    new(lipgloss.NewStyle().Foreground(lipgloss.Color("3"))),
			WidthMax:     40,
			WidthMin:     10,
		})).Str("region", "us-east-1").Elapsed("elapsed")).
			Progress(barFill)
		fileSize := 150 * 1000 * 1000 // 150 MB
		bytesBar := bar.Smooth
		bytesBar.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("6")))
		bytesBar.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
		bytesBar.WidgetRight = widget.Bytes()
		g.Add(clog.Bar("Fetching", fileSize, bar.WithConfig(bytesBar)).
			Str("file", "model.bin").Elapsed("elapsed")).
			Progress(func(_ context.Context, p *clog.Update) error {
				steps := 1000
				for i := range steps + 1 {
					p.SetProgress(fileSize * i / steps).Send()
					time.Sleep(3 * time.Millisecond)
				}
				return nil
			})
		g.Add(clog.Spinner("Processing data").Int("workers", 4)).
			Run(func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			})
		g.Add(clog.Pulse("Indexing search catalogue")).
			Run(func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			})
		if err := g.Wait().Symbol("✅").Msg("Group demo complete"); err != nil {
			clog.Fatal().Err(err).Msg("group demo failed")
		}

		// --- Spinner ---
		header("Spinner")
		_ = clog.Spinner("Loading demo").
			Str("eta", "Soon™").
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(1 * time.Second)
				return nil
			}).Symbol("✅").
			Msg("Demo loaded")

		_ = clog.Spinner("Running migrations").
			Str("db", "postgres").
			Progress(context.Background(), func(_ context.Context, update *clog.Update) error {
				steps := 100
				for i := range steps {
					update.Msg("Applying migrations").
						Percent("progress", float64(i+1)/float64(steps)).
						Send()
					time.Sleep(30 * time.Millisecond)
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
				time.Sleep(500 * time.Millisecond)
				update.Msg("Pushing image").Str("tag", "v1.2.3").Send()
				time.Sleep(500 * time.Millisecond)
				update.Msg("Starting containers").Send()
				time.Sleep(500 * time.Millisecond)
				return nil
			}).Symbol("🚀").
			Msg("Deployed")

		// --- Pulse ---
		header("Pulse")
		_ = clog.Pulse("Warming up inference engine").
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).Symbol("✅").
			Msg("Inference engine ready")

		header("Pulse (custom gradient)")
		_ = clog.Pulse("Replicating data across regions",
			pulse.WithGradient(
				style.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.2, B: 0.2}},
				style.ColorStop{Position: 0.5, Color: colorful.Color{R: 1, G: 1, B: 0.3}},
				style.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.2, B: 0.2}},
			),
		).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).Symbol("✅").
			Msg("Data replicated")

		// --- Shimmer ---
		header("Shimmer (default gradient)")
		_ = clog.Shimmer("Indexing documents and rebuilding search catalogue").
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).Symbol("✅").
			Msg("Search catalogue rebuilt")

		header("Shimmer (custom gradient)")
		_ = clog.Shimmer("Deploying service to production cluster and running health checks",
			shimmer.WithGradient(
				style.ColorStop{Position: 0, Color: colorful.Color{R: 0.3, G: 0.3, B: 0.8}},
				style.ColorStop{Position: 0.5, Color: colorful.Color{R: 1, G: 1, B: 1}},
				style.ColorStop{Position: 1, Color: colorful.Color{R: 0.3, G: 0.3, B: 0.8}},
			),
		).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).Symbol("🚀").
			Msg("Service deployed and health checks passed")

		header("Shimmer (middle direction, rainbow)")
		_ = clog.Shimmer("Synchronizing upstream dependencies and rebuilding artifacts",
			shimmer.WithGradient(
				style.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
				style.ColorStop{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
				style.ColorStop{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
				style.ColorStop{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
				style.ColorStop{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
				style.ColorStop{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
				style.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
			),
			shimmer.WithDirection(shimmer.MiddleIn),
		).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).Symbol("✅").
			Msg("Dependencies synced and artifacts rebuilt")

		_ = clog.Shimmer("Broadcasting configuration changes to all edge nodes",
			shimmer.WithGradient(
				style.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
				style.ColorStop{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
				style.ColorStop{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
				style.ColorStop{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
				style.ColorStop{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
				style.ColorStop{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
				style.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
			),
			shimmer.WithDirection(shimmer.MiddleOut),
		).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).Symbol("✅").
			Msg("Configuration broadcast complete")

		// --- Elapsed timer ---
		header("Elapsed Timer")
		_ = clog.Spinner("Processing batch").
			Str("batch", "1/3").
			Elapsed("elapsed").
			Int("workers", 4).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			}).Symbol("✅").
			Msg("Batch processed")
	}

	// --- Elapsed operations (no animation) ---
	header("Elapsed Operations")
	e := clog.Info().Elapsed("elapsed")
	time.Sleep(2 * time.Second)
	e.Msg("database migration")

	e = clog.Error().Elapsed("elapsed").Str("target", "release")
	time.Sleep(1 * time.Second)
	e.Err(errors.New("syntax error in main.go")).Msg("compile")

	e = clog.Info().Str("workers", "4").Elapsed("elapsed")
	time.Sleep(1 * time.Second)
	e.Int("items", 150).Msg("processed all items")

	// --- Elapsed gradient ---
	header("Elapsed Gradient")
	formats := clog.DefaultFieldFormats()
	formats.ElapsedGradientMax = 3 * time.Second
	clog.SetFieldFormats(formats)
	e = clog.Info().Elapsed("elapsed")
	time.Sleep(1 * time.Second)
	e.Msg("fast operation (green)")
	e = clog.Info().Elapsed("elapsed")
	time.Sleep(2 * time.Second)
	e.Msg("medium operation (yellow)")
	e = clog.Info().Elapsed("elapsed")
	time.Sleep(3 * time.Second)
	e.Msg("slow operation (red)")
	// Per-field override: this field maxes out at 1s regardless of the
	// logger's 3s default, so a 1s operation still renders red.
	e = clog.Info().Elapsed("elapsed", elapsed.WithGradientMax(1*time.Second))
	time.Sleep(1 * time.Second)
	e.Msg("per-field gradient max override (red)")
	clog.SetFieldFormats(clog.DefaultFieldFormats()) // reset

	// --- Context propagation ---
	header("Context Propagation")
	ctxLogger := clog.With().Str("request_id", "abc-123").Logger()
	ctx := ctxLogger.WithContext(context.Background())
	handleRequest(ctx)

	// --- Basic levels ---
	header("Levels")
	clog.Trace().Msg("Trace message")
	clog.Debug().Msg("Debug message")
	clog.Info().Msg("Info message")
	clog.Dry().Msg("Dry-run message")
	clog.Warn().Msg("Warning message")
	clog.Error().Msg("Error message")
	// --- Dry-run ---
	header("Dry-Run Mode")
	clog.Dry().Str("file", "config.yaml").Msg("Would overwrite")
	clog.Dry().Str("user", "admin").Msg("Would delete account")
	clog.Dry().Str("table", "users").Int("rows", 1500).Msg("Would truncate")
	// --- Typed fields ---
	header("Typed Fields")
	clog.Info().
		Str("host", "localhost").
		Int("port", 8080).
		Bool("tls", true).
		Msg("Server started")

	clog.Info().
		Float64("latency_ms", 12.345).
		Uint64("request_id", 9876543210).
		Duration("timeout", 30*time.Second).
		Quantity("cooldown", "5m").
		Quantity("distance", "5.1km").
		Quantities("limits", []string{"100MB", "5m", "10 req"}).
		Time("started", time.Now().Add(-30*time.Second)).
		Msg("Request handled")

	clog.Error().
		Err(errors.New("connection refused")).
		Str("host", "db.internal").
		Int("retries", 3).
		Msg("Database connection failed")

	// --- Conditional fields ---
	header("Conditional Fields (When)")
	detailed := true
	clog.Info().
		Str("user", "alice").
		When(detailed, func(e *clog.Event) {
			e.Str("role", "admin").Int("login_count", 42)
		}).
		Msg("User authenticated")
	// --- Value coloring ---
	header("Value Coloring")
	clog.Info().
		Bool("enabled", true).
		Bool("cached", false).
		Int("count", 42).
		Msg("Booleans and numbers get colored automatically")

	clog.Info().
		Any("value", nil).
		Str("name", "").
		Msg("Nil and empty values render as grey")
	// --- Slice fields ---
	header("Slice Fields")
	clog.Info().
		Strs("tags", []string{"api", "v2", "production"}).
		Msg("String slice")

	clog.Info().
		Strs("args", []string{"hello world", "simple", "key=val"}).
		Msg("String slice with per-element quoting")

	clog.Info().
		Ints("ports", []int{80, 443, 8080}).
		Msg("Int slice")

	clog.Info().
		Uints64("ids", []uint64{100, 200, 300}).
		Msg("Uint64 slice")

	clog.Info().
		Floats64("temps", []float64{36.6, 37.2, 38.1}).
		Msg("Float64 slice")

	clog.Info().
		Bools("flags", []bool{true, false, true}).
		Msg("Bool slice")

	clog.Info().
		Durations("latencies", []time.Duration{5 * time.Second, 2*time.Minute + 30*time.Second, 500 * time.Millisecond}).
		Msg("Duration slice")
	// --- Formatted messages ---
	header("Formatted Messages")
	clog.Info().Msgf("Processed %d items in %s", 150, 2*time.Second)
	clog.Info().Str("status", "ok").Send()
	// --- Custom symbol ---
	header("Custom Symbol")
	clog.Info().Symbol("🎉").Str("version", "1.0.0").Msg("Released")
	clog.Info().Symbol("📦").Str("pkg", "clog").Msg("Installed")
	clog.Warn().Symbol("🐌").Str("query", "SELECT *").Msg("Slow query")
	// --- Sub-loggers ---
	header("Sub-loggers")
	auth := clog.With().Str("component", "auth").Symbol("🔒").Logger()
	auth.Info().Str("user", "alice").Msg("Login successful")
	auth.Warn().Str("user", "bob").Str("reason", "bad password").Msg("Login failed")
	auth.Debug().Str("token", "eyJ...").Msg("Token issued")

	db := clog.With().Str("component", "db").Str("host", "postgres:5432").Logger()
	db.Info().Msg("Connected")
	db.Debug().Duration("latency", 2*time.Millisecond).Msg("Query executed")
	// --- Level alignment ---
	header("Level Alignment (Right, default)")
	clog.SetLabels(clog.LabelMap{
		clog.LevelDebug: "DEBUG",
		clog.LevelInfo:  "I",
		clog.LevelWarn:  "WARNING",
		clog.LevelError: "ERR",
	})
	clog.Debug().Msg("aligned right")
	clog.Info().Msg("aligned right")
	clog.Warn().Msg("aligned right")
	clog.Error().Msg("aligned right")

	header("Level Alignment (Left)")
	clog.SetLabels(clog.LabelMap{
		clog.LevelDebug: "DEBUG",
		clog.LevelInfo:  "I",
		clog.LevelWarn:  "WARNING",
		clog.LevelError: "ERR",
	})
	clog.SetLevelAlign(clog.AlignLeft)
	clog.Debug().Msg("aligned left")
	clog.Info().Msg("aligned left")
	clog.Warn().Msg("aligned left")
	clog.Error().Msg("aligned left")
	clog.SetLevelAlign(clog.AlignRight) // reset

	header("Level Alignment (Center)")
	clog.SetLabels(clog.LabelMap{
		clog.LevelDebug: "DEBUG",
		clog.LevelInfo:  "I",
		clog.LevelWarn:  "WARNING",
		clog.LevelError: "ERR",
	})
	clog.SetLevelAlign(clog.AlignCenter)
	clog.Debug().Msg("centered")
	clog.Info().Msg("centered")
	clog.Warn().Msg("centered")
	clog.Error().Msg("centered")
	clog.SetLevelAlign(clog.AlignRight) // reset

	header("Level Alignment (None)")
	clog.SetLevelAlign(clog.AlignNone)
	clog.Debug().Msg("no alignment")
	clog.Info().Msg("no alignment")
	clog.Warn().Msg("no alignment")
	clog.Error().Msg("no alignment")
	clog.SetLevelAlign(clog.AlignRight) // reset
	// --- Custom labels ---
	header("Custom Labels")
	clog.SetLabels(clog.LabelMap{
		clog.LevelTrace: "A",
		clog.LevelDebug: "B",
		clog.LevelInfo:  "C",
		clog.LevelDry:   "D",
		clog.LevelWarn:  "E",
		clog.LevelError: "F",
		clog.LevelFatal: "G",
	})
	clog.Debug().Msg("with custom labels")
	clog.Info().Msg("with custom labels")
	clog.Warn().Msg("with custom labels")
	clog.Error().Msg("with custom labels")
	clog.SetLabels(clog.DefaultLabels()) // reset
	// --- Custom symbols ---
	header("Custom Symbols")
	clog.SetSymbols(clog.LabelMap{
		clog.LevelInfo:  ">>",
		clog.LevelWarn:  "!!",
		clog.LevelError: "XX",
	})
	clog.Info().Msg("custom symbol")
	clog.Warn().Msg("custom symbol")
	clog.Error().Msg("custom symbol")
	clog.SetSymbols(clog.DefaultSymbols()) // reset
	// --- Hyperlinks ---
	header("Hyperlinks")
	clog.Info().
		Path("dir", "src/").
		Msg("Clickable directory via Path()")

	clog.Info().
		Line("file", "examples/main.go", 42).
		Msg("Clickable file:line via Line()")

	clog.Info().
		Str("file", clog.PathLink("examples/main.go", 42)).
		Msg("Clickable file path via PathLink()")

	clog.Info().
		Str("docs", clog.Hyperlink("https://github.com/gechr/clog", "clog")).
		Msg("Clickable URL")
	// --- Part ordering ---
	header("Custom Part Order (fields before message)")
	partStyles := clog.DefaultStyles()
	italic := lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("4"))
	partStyles.Messages[clog.LevelInfo] = new(italic)
	partStyles.Messages[clog.LevelWarn] = new(italic)
	clog.SetStyles(partStyles)
	clog.SetParts(
		clog.PartTimestamp,
		clog.PartLevel,
		clog.PartSymbol,
		clog.PartFields,
		clog.PartMessage,
	)
	clog.Info().Str("user", "alice").Int("status", 200).Msg("Request handled")
	clog.Warn().Str("query", "SELECT *").Duration("latency", 5*time.Second).Msg("Slow query")
	clog.SetStyles(clog.DefaultStyles())  // reset
	clog.SetParts(clog.DefaultParts()...) // reset

	header("Hide Log Level")
	clog.SetParts(clog.PartTimestamp, clog.PartSymbol, clog.PartMessage, clog.PartFields)
	clog.Info().Str("user", "alice").Msg("Login")
	clog.Error().Err(errors.New("timeout")).Msg("Request failed")
	clog.SetParts(clog.DefaultParts()...) // reset

	header("Hide Symbol")
	clog.SetParts(clog.PartTimestamp, clog.PartLevel, clog.PartMessage, clog.PartFields)
	clog.Info().Str("status", "ok").Msg("Health check")
	clog.Warn().Str("disk", "92%").Msg("Low disk space")
	clog.SetParts(clog.DefaultParts()...) // reset

	header("Minimal (message only)")
	clog.SetParts(clog.PartMessage)
	minimalStyles := clog.DefaultStyles()
	minimalStyles.Messages[clog.LevelError] = new(
		lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("1")),
	)
	clog.SetStyles(minimalStyles)
	clog.Info().Msg("Just the message, nothing else")
	clog.Error().Msg("Look Ma, an error!")
	clog.SetStyles(clog.DefaultStyles())  // reset
	clog.SetParts(clog.DefaultParts()...) // reset

	header("No Timestamp (default)")
	clog.SetReportTimestamp(false)
	clog.Info().Str("mode", "clean").Msg("No timestamp symbol")
	clog.SetReportTimestamp(true) // reset
	// --- Per-level message styles ---
	header("Per-Level Message Styles")
	styles := clog.DefaultStyles()
	styles.Messages[clog.LevelTrace] = new(
		lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("6")),
	) // dim cyan
	styles.Messages[clog.LevelDebug] = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("6")),
	) // cyan
	styles.Messages[clog.LevelInfo] = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("2")),
	) // green
	styles.Messages[clog.LevelDry] = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("5")),
	) // magenta
	styles.Messages[clog.LevelWarn] = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
	) // yellow
	styles.Messages[clog.LevelError] = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	) // red
	styles.Messages[clog.LevelFatal] = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
	) // red
	clog.SetStyles(styles)
	clog.Trace().Msg("Trace message is dim cyan")
	clog.Debug().Msg("Debug message is cyan")
	clog.Info().Msg("Info message is green")
	clog.Dry().Msg("Dry-run message is magenta")
	clog.Warn().Msg("Warning message is yellow")
	clog.Error().Msg("Error message is red")
	clog.SetStyles(clog.DefaultStyles()) // reset
	// --- OmitEmpty ---
	header("OmitEmpty")
	clog.SetOmitEmpty(true)
	clog.Info().
		Str("name", "alice").
		Str("nickname", "").
		Any("role", nil).
		Int("age", 0).
		Bool("admin", false).
		Msg("Empty string and nil omitted; zero int and false kept")
	clog.SetOmitEmpty(false) // reset

	// --- OmitZero ---
	header("OmitZero")
	clog.SetOmitZero(true)
	clog.Info().
		Str("name", "alice").
		Str("nickname", "").
		Any("role", nil).
		Int("age", 0).
		Bool("admin", false).
		Msg("All zero/empty values omitted")
	clog.SetOmitZero(false) // reset

	// --- Quote ---
	header("Quote: Never")
	clog.SetQuote(clog.QuoteNever)
	clog.Info().
		Str("msg", "hello world").
		Strs("tags", []string{"has space", "ok"}).
		Msg("Quotes suppressed even for values with spaces")

	header("Quote: Always")
	clog.SetQuote(clog.QuoteAlways)
	clog.Info().
		Str("reason", "timeout").
		Str("msg", "hello world").
		Msg("All string values are quoted")

	header("Quote: Auto (default)")
	clog.SetQuote(clog.QuoteAuto)
	clog.Info().
		Str("reason", "timeout").
		Str("msg", "hello world").
		Msg("Only values that need quoting are quoted")

	// --- Custom Quote Character ---
	header("Custom Quote Character")
	clog.SetQuoteChars('\'', '\'')
	clog.Info().
		Str("msg", "hello world").
		Strs("tags", []string{"has space", "ok"}).
		Msg("Single quotes instead of double")
	clog.SetQuoteChars(0, 0) // reset to default

	// --- Asymmetric Quote Characters ---
	header("Asymmetric Quote Characters")
	clog.SetQuoteChars('«', '»')
	clog.Info().
		Str("msg", "hello world").
		Msg("French-style guillemets")
	clog.SetQuoteChars(0, 0) // reset to default

	// --- Bytes ---
	header("Bytes")
	clog.Info().
		Bytes("body", []byte(`{"status":"ok","count":42}`)).
		Msg("JSON bytes get syntax highlighting")
	clog.Info().
		Bytes("raw", []byte("plain text content")).
		Msg("Non-JSON bytes stored as string")

	// --- JSON Highlighting ---
	header("RawJSON (default)")
	clog.Error().
		Str("batch", "1/1").
		Uint("retries", 1).
		RawJSON("error", []byte(`{"errors":[{"status":"unprocessable_entity","detail":"API rate limit exceeded, retry after 30s","code":null}]}`)).
		Msg("Batch failed")

	// All JSON value types: string, int, float, bool (true/false), null, array, nested object
	clog.Info().
		Str("endpoint", "/api/resources").
		RawJSON("response", []byte(`{"id":"abc123","count":42,"ratio":0.875,"active":true,"archived":false,"deleted_at":null,"tags":["production","staging"],"meta":{"region":"us-east-1","latency_ms":12.5}}`)).
		Msg("Resource fetched")

	header("RawJSON (custom styles)")
	customStyles := style.DefaultJSON()
	customStyles.Key = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")),
	) // green keys
	customStyles.Null = new(
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Faint(true),
	) // red dim null
	customStyleSet := clog.DefaultStyles()
	customStyleSet.JSON = customStyles
	clog.SetStyles(customStyleSet)
	clog.Info().
		RawJSON("payload", []byte(`{"id":"abc123","count":42,"ratio":0.875,"active":true,"archived":false,"deleted_at":null,"tags":["production","staging"],"meta":{"region":"us-east-1","latency_ms":12.5}}`)).
		Msg("Resource fetched")
	clog.SetStyles(clog.DefaultStyles()) // reset

	header("RawJSON (human mode)")
	humanStyles := clog.DefaultStyles()
	humanStyles.JSON = style.DefaultJSON()
	humanStyles.JSON.Mode = style.JSONModeHuman
	clog.SetStyles(humanStyles)
	clog.Info().
		RawJSON("response", []byte(`{"status":"ok","count":42,"active":true,"deleted_at":null,"tags":["production","staging"],"meta":{"region":"us-east-1","latency_ms":12.5}}`)).
		Msg("Resource fetched")
	clog.SetStyles(clog.DefaultStyles()) // reset

	header("RawJSON (flat mode)")
	flatStyles := clog.DefaultStyles()
	flatStyles.JSON = style.DefaultJSON()
	flatStyles.JSON.Mode = style.JSONModeFlat
	clog.SetStyles(flatStyles)
	clog.Error().
		Str("batch", "1/1").
		RawJSON("error", []byte(`{"errors":[{"status":"unprocessable_entity","detail":"API rate limit exceeded, retry after 30s","code":null}],"meta":{"region":"us-east-1","request_id":"abc123"}}`)).
		Msg("Batch failed")
	clog.Info().
		RawJSON("response", []byte(`{"user":{"name":"alice","role":"admin"},"session":{"token":"abc","expires_in":3600},"tags":["production","staging"]}`)).
		Msg("Authenticated")
	clog.SetStyles(clog.DefaultStyles()) // reset

	header("RawJSON (no highlighting)")
	noHighlightStyles := clog.DefaultStyles()
	noHighlightStyles.JSON = nil
	clog.SetStyles(noHighlightStyles)
	clog.Info().
		RawJSON("payload", []byte(`{"id":"abc123","count":42,"ratio":0.875,"active":true,"archived":false,"deleted_at":null,"tags":["production","staging"],"meta":{"region":"us-east-1","latency_ms":12.5}}`)).
		Msg("Resource fetched")
	clog.SetStyles(clog.DefaultStyles()) // reset

	// --- Handler ---
	header("Custom Handler")
	logger := clog.New(nil)
	logger.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
		fmt.Printf("[CUSTOM] level=%s msg=%q fields=%d\n", e.Level, e.Message, len(e.Fields))
	}))
	logger.Info().Str("k", "v").Msg("handled by custom handler")
	logger.Error().Err(errors.New("boom")).Msg("error via handler")

	// --- Format hooks ---
	header("Format Hooks")
	formats = clog.DefaultFieldFormats()
	formats.ElapsedFormat = func(d time.Duration) string {
		return d.Truncate(time.Second).String()
	}
	formats.PercentFormat = func(v float64) string {
		return fmt.Sprintf("%.0f/100", v)
	}
	clog.SetFieldFormats(formats)
	clog.Info().
		Percent("progress", 0.75).
		Msg("Custom format hooks")
	clog.SetFieldFormats(clog.DefaultFieldFormats()) // reset

	// --- Number formatting ---
	header("Number Formatting")
	clog.Info().Int("rows", 1234567).Fraction("processed", 1234567, 9999999).Msg("plain (default)")
	clog.SetNumberFormat(clog.NumberGrouped)
	clog.Info().Int("rows", 1234567).Fraction("processed", 1234567, 9999999).Msg("grouped")
	clog.SetNumberFormat(clog.NumberCompact)
	clog.Info().Int("rows", 1234567).Fraction("processed", 1234567, 9999999).Msg("compact")
	clog.SetNumberFormat(clog.NumberPlain)
	clog.Info().
		Int("rows", 1234567).
		Fraction("processed", 1234567, 9999999, fraction.WithFormat(clog.NumberCompact)).
		Msg("per-field compact fraction, plain int")
	clog.SetFieldFormats(clog.DefaultFieldFormats()) // reset

	// --- Field sort order ---
	header("Field Sort Order (Ascending)")
	clog.SetFieldSort(clog.SortAscending)
	clog.Info().
		Str("zoo", "animals").
		Int("count", 42).
		Str("alpha", "first").
		Msg("Fields sorted A→Z")
	clog.SetFieldSort(clog.SortNone) // reset

	header("Field Sort Order (Descending)")
	clog.SetFieldSort(clog.SortDescending)
	clog.Info().
		Str("alpha", "first").
		Int("count", 42).
		Str("zoo", "animals").
		Msg("Fields sorted Z→A")
	clog.SetFieldSort(clog.SortNone) // reset

	// --- Dividers ---
	header("Dividers")
	clog.Divider().Send()
	clog.Divider().Msg("Build Phase")
	clog.Divider().Char('═').Msg("Deployment")
	clog.Divider().Align(clog.AlignCenter).Msg("Test Results")
	clog.Divider().Align(clog.AlignRight).Msg("Summary")
}

func spinners(filter string) {
	type entry struct {
		name string
		s    spinner.Config
	}

	all := []entry{
		{"Aesthetic", spinner.Aesthetic},
		{"Arc", spinner.Arc},
		{"Arrow3", spinner.Arrow3},
		{"BetaWave", spinner.BetaWave},
		{"BouncingBall", spinner.BouncingBall},
		{"Circle", spinner.Circle},
		{"Dots", spinner.Dots},
		{"Dots8Bit", spinner.Dots8Bit},
		{"Dots12", spinner.Dots12},
		{"Flip", spinner.Flip},
		{"Globe", spinner.Globe},
		{"GrowVertical", spinner.GrowVertical},
		{"Layer", spinner.Layer},
		{"Line", spinner.Line},
		{"Material", spinner.Material},
		{"Meter", spinner.Meter},
		{"Moon", spinner.Moon},
		{"Noise", spinner.Noise},
		{"Pipe", spinner.Pipe},
		{"Pong", spinner.Pong},
		{"Runner", spinner.Runner},
		{"Shark", spinner.Shark},
		{"Smiley", spinner.Smiley},
		{"SoccerHeader", spinner.SoccerHeader},
		{"SquareCorners", spinner.SquareCorners},
		{"Toggle", spinner.Toggle},
		{"Triangle", spinner.Triangle},
		{"Weather", spinner.Weather},
	}

	if filter != "" {
		names := make(map[string]bool)
		for n := range strings.SplitSeq(filter, ",") {
			names[strings.ToLower(strings.TrimSpace(n))] = true
		}
		filtered := make([]entry, 0, len(names))
		for _, e := range all {
			if names[strings.ToLower(e.name)] {
				filtered = append(filtered, e)
			}
		}
		all = filtered
	}

	clog.SetReportTimestamp(false)
	clog.SetParts(clog.PartMessage, clog.PartSymbol)
	styles := clog.DefaultStyles()
	orange := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styles.Messages[clog.LevelInfo] = &orange
	clog.SetStyles(styles)

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	check := green.Render("✓")

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
	for i, e := range all {
		if err := results[i].Symbol(check).Msg(e.name); err != nil {
			clog.Fatal().Err(err).Msg("spinner demo failed")
		}
	}
}

func handleRequest(ctx context.Context) {
	clog.Ctx(ctx).Info().Str("step", "validate").Msg("Handling request")
	clog.Ctx(ctx).Info().Str("step", "process").Msg("Processing request")
}

func demo() {
	_ = clog.Shimmer("Initializing environment and loading configuration modules",
		shimmer.WithGradient(
			style.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
			style.ColorStop{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
			style.ColorStop{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
			style.ColorStop{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
			style.ColorStop{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
			style.ColorStop{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
			style.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
		),
		shimmer.WithDirection(shimmer.MiddleIn),
	).
		Str("eta", "Soon™").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(3 * time.Second)
			return nil
		}).Symbol("✅").
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
		Progress(context.Background(), func(_ context.Context, update *clog.Update) error {
			update.Msg("Building image").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Pushing image").Str("tag", "v1.2.3").Send()
			time.Sleep(1 * time.Second)
			update.Msg("Starting containers").Send()
			time.Sleep(1 * time.Second)
			return nil
		}).Symbol("🚀").
		Msg("Deployed")

	_ = clog.Spinner("Running migrations").
		Str("db", "postgres").
		Progress(context.Background(), func(_ context.Context, update *clog.Update) error {
			steps := 100
			for i := range steps {
				update.Msg("Applying migrations").
					Percent("progress", float64(i+1)/float64(steps)).
					Send()
				time.Sleep(30 * time.Millisecond)
			}
			return nil
		}).Symbol("✅").
		Msg("Migrations applied")

	_ = clog.Spinner("Downloading artifacts", spinner.WithConfig(spinner.Dot)).
		Str("repo", "gechr/clog").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(2 * time.Second)
			return nil
		}).Symbol("📦").
		Msg("Artifacts downloaded")

	_ = clog.Spinner("Connecting to database").
		Str("host", "db.internal").
		Int("port", 5432).
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(2 * time.Second)
			return errors.New("connection refused")
		}).
		OnErrorLevel(clog.LevelFatal).
		Send()
}
