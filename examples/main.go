package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
	"github.com/lucasb-eyer/go-colorful"
)

func main() {
	demoFlag := flag.Bool("demo", false, "run the demo")
	quickFlag := flag.Bool("quick", false, "skip animations")
	spinnersFlag := flag.String("spinners", "", "demo spinners (comma-separated names, empty for all)")
	flag.Parse()

	spinnersSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "spinners" {
			spinnersSet = true
		}
	})

	clog.SetLevel(clog.TraceLevel)
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
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
		fmt.Println(style.Render(h))
	}

	if !*quickFlag {
		// --- Spinner ---
		header("Spinner")
		_ = clog.Spinner("Loading demo").
			Str("eta", "Soon™").
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(1 * time.Second)
				return nil
			}).
			Prefix("✅").
			Msg("Demo loaded")

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
				time.Sleep(500 * time.Millisecond)
				update.Msg("Pushing image").Str("tag", "v1.2.3").Send()
				time.Sleep(500 * time.Millisecond)
				update.Msg("Starting containers").Send()
				time.Sleep(500 * time.Millisecond)
				return nil
			}).
			Prefix("🚀").
			Msg("Deployed")

		// --- Pulse ---
		header("Pulse (default gradient)")
		_ = clog.Pulse("Warming up inference engine").
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).
			Prefix("✅").
			Msg("Inference engine ready")

		header("Pulse (custom gradient)")
		_ = clog.Pulse("Replicating data across regions",
			clog.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.2, B: 0.2}},
			clog.ColorStop{Position: 0.5, Color: colorful.Color{R: 1, G: 1, B: 0.3}},
			clog.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.2, B: 0.2}},
		).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).
			Prefix("✅").
			Msg("Data replicated")

		// --- Shimmer ---
		header("Shimmer (default gradient)")
		_ = clog.Shimmer("Indexing documents and rebuilding search catalogue").
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).
			Prefix("✅").
			Msg("Search catalogue rebuilt")

		header("Shimmer (custom gradient)")
		_ = clog.Shimmer("Deploying service to production cluster and running health checks",
			clog.ColorStop{Position: 0, Color: colorful.Color{R: 0.3, G: 0.3, B: 0.8}},
			clog.ColorStop{Position: 0.5, Color: colorful.Color{R: 1, G: 1, B: 1}},
			clog.ColorStop{Position: 1, Color: colorful.Color{R: 0.3, G: 0.3, B: 0.8}},
		).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).
			Prefix("🚀").
			Msg("Service deployed and health checks passed")

		header("Shimmer (middle direction, rainbow)")
		_ = clog.Shimmer("Synchronizing upstream dependencies and rebuilding artifacts",
			clog.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
			clog.ColorStop{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
			clog.ColorStop{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
			clog.ColorStop{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
			clog.ColorStop{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
			clog.ColorStop{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
			clog.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
		).
			ShimmerDirection(clog.DirectionMiddleIn).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).
			Prefix("✅").
			Msg("Dependencies synced and artifacts rebuilt")

		_ = clog.Shimmer("Broadcasting configuration changes to all edge nodes",
			clog.ColorStop{Position: 0, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
			clog.ColorStop{Position: 0.17, Color: colorful.Color{R: 1, G: 0.6, B: 0.2}},
			clog.ColorStop{Position: 0.33, Color: colorful.Color{R: 1, G: 1, B: 0.4}},
			clog.ColorStop{Position: 0.5, Color: colorful.Color{R: 0.3, G: 1, B: 0.5}},
			clog.ColorStop{Position: 0.67, Color: colorful.Color{R: 0.4, G: 0.5, B: 1}},
			clog.ColorStop{Position: 0.83, Color: colorful.Color{R: 0.7, G: 0.3, B: 1}},
			clog.ColorStop{Position: 1, Color: colorful.Color{R: 1, G: 0.3, B: 0.3}},
		).
			ShimmerDirection(clog.DirectionMiddleOut).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(3 * time.Second)
				return nil
			}).
			Prefix("✅").
			Msg("Configuration broadcast complete")

		// --- Elapsed timer ---
		header("Elapsed Timer (respects field ordering)")
		_ = clog.Spinner("Processing batch").
			Str("batch", "1/3").
			Elapsed("elapsed").
			Int("workers", 4).
			Wait(context.Background(), func(_ context.Context) error {
				time.Sleep(2 * time.Second)
				return nil
			}).
			Prefix("✅").
			Msg("Batch processed")
	}

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
	// --- Value colouring ---
	header("Value Colouring")
	clog.Info().
		Bool("enabled", true).
		Bool("cached", false).
		Int("count", 42).
		Msg("Booleans and numbers get coloured automatically")

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
	// --- Custom prefix ---
	header("Custom Prefix")
	clog.Info().Prefix("🎉").Str("version", "1.0.0").Msg("Released")
	clog.Info().Prefix("📦").Str("pkg", "clog").Msg("Installed")
	clog.Warn().Prefix("🐌").Str("query", "SELECT *").Msg("Slow query")
	// --- Sub-loggers ---
	header("Sub-loggers")
	auth := clog.With().Str("component", "auth").Prefix("🔒").Logger()
	auth.Info().Str("user", "alice").Msg("Login successful")
	auth.Warn().Str("user", "bob").Str("reason", "bad password").Msg("Login failed")
	auth.Debug().Str("token", "eyJ...").Msg("Token issued")

	db := clog.With().Str("component", "db").Str("host", "postgres:5432").Logger()
	db.Info().Msg("Connected")
	db.Debug().Duration("latency", 2*time.Millisecond).Msg("Query executed")
	// --- Level alignment ---
	header("Level Alignment (Right, default)")
	clog.SetLevelLabels(clog.LevelMap{
		clog.DebugLevel: "DEBUG",
		clog.InfoLevel:  "I",
		clog.WarnLevel:  "WARNING",
		clog.ErrorLevel: "ERR",
	})
	clog.Debug().Msg("aligned right")
	clog.Info().Msg("aligned right")
	clog.Warn().Msg("aligned right")
	clog.Error().Msg("aligned right")

	header("Level Alignment (Left)")
	clog.SetLevelLabels(clog.LevelMap{
		clog.DebugLevel: "DEBUG",
		clog.InfoLevel:  "I",
		clog.WarnLevel:  "WARNING",
		clog.ErrorLevel: "ERR",
	})
	clog.SetLevelAlign(clog.AlignLeft)
	clog.Debug().Msg("aligned left")
	clog.Info().Msg("aligned left")
	clog.Warn().Msg("aligned left")
	clog.Error().Msg("aligned left")
	clog.SetLevelAlign(clog.AlignRight) // reset

	header("Level Alignment (Center)")
	clog.SetLevelLabels(clog.LevelMap{
		clog.DebugLevel: "DEBUG",
		clog.InfoLevel:  "I",
		clog.WarnLevel:  "WARNING",
		clog.ErrorLevel: "ERR",
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
	clog.SetLevelLabels(clog.LevelMap{
		clog.TraceLevel: "A",
		clog.DebugLevel: "B",
		clog.InfoLevel:  "C",
		clog.DryLevel:   "D",
		clog.WarnLevel:  "E",
		clog.ErrorLevel: "F",
		clog.FatalLevel: "G",
	})
	clog.Debug().Msg("with custom labels")
	clog.Info().Msg("with custom labels")
	clog.Warn().Msg("with custom labels")
	clog.Error().Msg("with custom labels")
	clog.SetLevelLabels(clog.DefaultLabels()) // reset
	// --- Custom prefixes ---
	header("Custom Prefixes")
	clog.SetPrefixes(clog.LevelMap{
		clog.InfoLevel:  ">>",
		clog.WarnLevel:  "!!",
		clog.ErrorLevel: "XX",
	})
	clog.Info().Msg("custom prefix")
	clog.Warn().Msg("custom prefix")
	clog.Error().Msg("custom prefix")
	clog.SetPrefixes(clog.DefaultPrefixes()) // reset
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
	partStyles.Messages[clog.InfoLevel] = new(italic)
	partStyles.Messages[clog.WarnLevel] = new(italic)
	clog.SetStyles(partStyles)
	clog.SetParts(clog.PartTimestamp, clog.PartLevel, clog.PartPrefix, clog.PartFields, clog.PartMessage)
	clog.Info().Str("user", "alice").Int("status", 200).Msg("Request handled")
	clog.Warn().Str("query", "SELECT *").Duration("latency", 5*time.Second).Msg("Slow query")
	clog.SetStyles(clog.DefaultStyles())  // reset
	clog.SetParts(clog.DefaultParts()...) // reset

	header("Hide Log Level")
	clog.SetParts(clog.PartTimestamp, clog.PartPrefix, clog.PartMessage, clog.PartFields)
	clog.Info().Str("user", "alice").Msg("Login")
	clog.Error().Err(errors.New("timeout")).Msg("Request failed")
	clog.SetParts(clog.DefaultParts()...) // reset

	header("Hide Prefix")
	clog.SetParts(clog.PartTimestamp, clog.PartLevel, clog.PartMessage, clog.PartFields)
	clog.Info().Str("status", "ok").Msg("Health check")
	clog.Warn().Str("disk", "92%").Msg("Low disk space")
	clog.SetParts(clog.DefaultParts()...) // reset

	header("Minimal (message only)")
	clog.SetParts(clog.PartMessage)
	minimalStyles := clog.DefaultStyles()
	minimalStyles.Messages[clog.ErrorLevel] = new(lipgloss.NewStyle().Strikethrough(true).Foreground(lipgloss.Color("1")))
	clog.SetStyles(minimalStyles)
	clog.Info().Msg("Just the message, nothing else")
	clog.Error().Msg("Look Ma, an error!")
	clog.SetStyles(clog.DefaultStyles())  // reset
	clog.SetParts(clog.DefaultParts()...) // reset

	header("No Timestamp (default)")
	clog.SetReportTimestamp(false)
	clog.Info().Str("mode", "clean").Msg("No timestamp prefix")
	clog.SetReportTimestamp(true) // reset
	// --- Per-level message styles ---
	header("Per-Level Message Styles")
	styles := clog.DefaultStyles()
	styles.Messages[clog.TraceLevel] = new(lipgloss.NewStyle().Faint(true).Foreground(lipgloss.Color("6"))) // dim cyan
	styles.Messages[clog.DebugLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("6")))             // cyan
	styles.Messages[clog.InfoLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("2")))              // green
	styles.Messages[clog.DryLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("5")))               // magenta
	styles.Messages[clog.WarnLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("3")))              // yellow
	styles.Messages[clog.ErrorLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))             // red
	styles.Messages[clog.FatalLevel] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))             // red
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

	// --- QuoteMode ---
	header("QuoteMode: Never")
	clog.SetQuoteMode(clog.QuoteNever)
	clog.Info().
		Str("msg", "hello world").
		Strs("tags", []string{"has space", "ok"}).
		Msg("Quotes suppressed even for values with spaces")

	header("QuoteMode: Always")
	clog.SetQuoteMode(clog.QuoteAlways)
	clog.Info().
		Str("reason", "timeout").
		Str("msg", "hello world").
		Msg("All string values are quoted")

	header("QuoteMode: Auto (default)")
	clog.SetQuoteMode(clog.QuoteAuto)
	clog.Info().
		Str("reason", "timeout").
		Str("msg", "hello world").
		Msg("Only values that need quoting are quoted")

	// --- Custom Quote Character ---
	header("Custom Quote Character")
	clog.SetQuoteChar('\'')
	clog.Info().
		Str("msg", "hello world").
		Strs("tags", []string{"has space", "ok"}).
		Msg("Single quotes instead of double")
	clog.SetQuoteChar(0) // reset to default

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
	customStyles := clog.DefaultJSONStyles()
	customStyles.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))              // green keys
	customStyles.Null = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5555")).Faint(true)) // red dim null
	customStyleSet := clog.DefaultStyles()
	customStyleSet.FieldJSON = customStyles
	clog.SetStyles(customStyleSet)
	clog.Info().
		RawJSON("payload", []byte(`{"id":"abc123","count":42,"ratio":0.875,"active":true,"archived":false,"deleted_at":null,"tags":["production","staging"],"meta":{"region":"us-east-1","latency_ms":12.5}}`)).
		Msg("Resource fetched")
	clog.SetStyles(clog.DefaultStyles()) // reset

	header("RawJSON (human mode)")
	humanStyles := clog.DefaultStyles()
	humanStyles.FieldJSON = clog.DefaultJSONStyles()
	humanStyles.FieldJSON.Mode = clog.JSONModeHuman
	clog.SetStyles(humanStyles)
	clog.Info().
		RawJSON("response", []byte(`{"status":"ok","count":42,"active":true,"deleted_at":null,"tags":["production","staging"],"meta":{"region":"us-east-1","latency_ms":12.5}}`)).
		Msg("Resource fetched")
	clog.SetStyles(clog.DefaultStyles()) // reset

	header("RawJSON (flat mode)")
	flatStyles := clog.DefaultStyles()
	flatStyles.FieldJSON = clog.DefaultJSONStyles()
	flatStyles.FieldJSON.Mode = clog.JSONModeFlat
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
	noHighlightStyles.FieldJSON = nil
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
	clog.SetElapsedFormatFunc(func(d time.Duration) string {
		return d.Truncate(time.Second).String()
	})
	clog.SetPercentFormatFunc(func(v float64) string {
		return fmt.Sprintf("%.0f/100", v)
	})
	clog.Info().
		Percent("progress", 75).
		Msg("Custom format hooks")
	clog.SetElapsedFormatFunc(nil) // reset
	clog.SetPercentFormatFunc(nil) // reset

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
}

func spinners(filter string) {
	type entry struct {
		name    string
		spinner clog.SpinnerType
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
	clog.SetParts(clog.PartPrefix, clog.PartMessage, clog.PartFields)
	styles := clog.DefaultStyles()
	orange := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styles.Messages[clog.InfoLevel] = &orange
	clog.SetStyles(styles)

	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	check := green.Render("✓")

	ctx := context.Background()
	for _, e := range all {
		cycle := e.spinner.FPS * time.Duration(len(e.spinner.Frames))
		dur := cycle * 2

		_ = clog.Spinner(e.name).
			Type(e.spinner).
			Duration("cycle", cycle).
			Wait(ctx, func(_ context.Context) error {
				time.Sleep(dur)
				return nil
			}).
			Prefix(check).
			Msg(e.name)
	}
}

func demo() {
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
		Type(clog.SpinnerDot).
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
