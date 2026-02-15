package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
)

func main() {
	clog.SetLevel(clog.TraceLevel)
	clog.SetReportTimestamp(true)

	if os.Getenv("DEMO") == "1" {
		demo()
		return
	}

	header := func(h string) {
		fmt.Println()
		style := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
		fmt.Println(style.Render(h))
	}

	// --- Spinner ---
	header("Spinner")
	_ = clog.Spinner("Loading demo").
		Str("eta", "24 hours").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(1 * time.Second)
			return nil
		}).
		Prefix("✅").
		Msg("Demo loaded")

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
			update.Title("Building image").Send()
			time.Sleep(500 * time.Millisecond)
			update.Title("Pushing image").Str("tag", "v1.2.3").Send()
			time.Sleep(500 * time.Millisecond)
			update.Title("Starting containers").Send()
			time.Sleep(500 * time.Millisecond)
			return nil
		}).
		Prefix("🚀").
		Msg("Deployed")

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
		Dur("timeout", 30*time.Second).
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
	db.Debug().Dur("latency", 2*time.Millisecond).Msg("Query executed")
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
	clog.Warn().Str("query", "SELECT *").Dur("latency", 5*time.Second).Msg("Slow query")
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

	// --- Handler ---
	header("Custom Handler")
	logger := clog.New(nil)
	logger.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
		fmt.Printf("[CUSTOM] level=%s msg=%q fields=%d\n", e.Level, e.Message, len(e.Fields))
	}))
	logger.Info().Str("k", "v").Msg("handled by custom handler")
	logger.Error().Err(errors.New("boom")).Msg("error via handler")
}

func demo() {
	_ = clog.Spinner("Loading demo").
		Str("eta", "Soon™").
		Wait(context.Background(), func(_ context.Context) error {
			time.Sleep(2 * time.Second)
			return nil
		}).
		Prefix("✅").
		Msg("Demo loaded")

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
			update.Title("Building image").Send()
			time.Sleep(1 * time.Second)
			update.Title("Pushing image").Str("tag", "v1.2.3").Send()
			time.Sleep(1 * time.Second)
			update.Title("Starting containers").Send()
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
				progress := min(i+rand.IntN(5)+1, hundred)
				percent := fmt.Sprintf("%d%%", progress)
				update.Title("Applying migrations").Str("progress", percent).Send()
				time.Sleep(30 * time.Millisecond)
			}
			return nil
		}).
		Prefix("✅").
		Msg("Migrations applied")

	_ = clog.Spinner("Downloading artifacts").
		Type(spinner.Dot).
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
