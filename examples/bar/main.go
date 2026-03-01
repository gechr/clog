package main

import (
	"context"
	"flag"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
)

func main() {
	stylesFlag := flag.Bool("styles", false, "show all bar style presets")
	etaFlag := flag.Bool("eta", false, "show ETA, rate, and dynamic total demos")
	flag.Parse()

	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

	if *stylesFlag {
		showStyles()
		return
	}
	if *etaFlag {
		showETA()
		return
	}

	downloadBar := bar.Thin
	downloadBar.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))
	downloadBar.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
	downloadBar.WidgetLeft = widget.Percent()
	downloadBar.WidgetRight = widget.None()

	_ = clog.Bar("Downloading", 1000, bar.WithStyle(downloadBar)).
		Str("file", "release.tar.gz").
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *clog.Update) error {
			for i := range 1001 {
				p.SetProgress(i).Send()
				time.Sleep(3 * time.Millisecond)
			}
			return nil
		}).Symbol("✅").
		Msg("Download complete")

	fileSize := 150 * 1000 * 1000
	bytesBar := bar.Smooth
	bytesBar.ProgressGradient = bar.DefaultGradient()
	bytesBar.WidgetRight = widget.Bytes()

	_ = clog.Bar("Fetching", fileSize, bar.WithStyle(bytesBar)).
		Str("file", "model.bin").
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *clog.Update) error {
			steps := 1000
			for i := range steps + 1 {
				p.SetProgress(fileSize * i / steps).Send()
				time.Sleep(3 * time.Millisecond)
			}
			return nil
		}).Symbol("✅").
		Msg("Model downloaded")
}

func showETA() {
	// ETA countdown with item rate composed on the right.
	etaStyle := bar.Thin
	etaStyle.WidgetRight = widget.Widgets(
		widget.ETA(),
		widget.Separator("│"),
		widget.Rate(widget.WithUnit("items")),
	)

	_ = clog.Bar("Processing", 500, bar.WithStyle(etaStyle)).
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *clog.Update) error {
			for i := range 501 {
				p.SetProgress(i).Send()
				time.Sleep(5 * time.Millisecond)
			}
			return nil
		}).Symbol("✅").
		Msg("Processing complete")

	// ETA with byte throughput for a download.
	downloadStyle := bar.Smooth
	downloadStyle.ProgressGradient = bar.DefaultGradient()
	downloadStyle.WidgetLeft = widget.ETA()
	downloadStyle.WidgetRight = widget.BytesRate()

	totalBytes := 200 * 1000 * 1000
	_ = clog.Bar("Downloading", totalBytes, bar.WithStyle(downloadStyle)).
		Str("file", "dataset.tar.gz").
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *clog.Update) error {
			steps := 1000
			for i := range steps + 1 {
				p.SetProgress(totalBytes * i / steps).Send()
				time.Sleep(3 * time.Millisecond)
			}
			return nil
		}).Symbol("✅").
		Msg("Download complete")

	// Dynamic total with AddTotal - discovers more work mid-task.
	scanStyle := bar.Block
	scanStyle.WidgetLeft = widget.ETA()
	scanStyle.WidgetRight = widget.Percent()

	_ = clog.Bar("Scanning", 100, bar.WithStyle(scanStyle)).
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *clog.Update) error {
			for i := range 101 {
				p.SetProgress(i).Send()
				time.Sleep(5 * time.Millisecond)
				// Halfway through, discover 50 more items.
				if i == 50 {
					p.AddTotal(50)
				}
			}
			// Process the extra items.
			for i := 101; i <= 150; i++ {
				p.SetProgress(i).Send()
				time.Sleep(5 * time.Millisecond)
			}
			return nil
		}).Symbol("✅").
		Msg("Scan complete")
}

func showStyles() {
	type barEntry struct {
		name  string
		style bar.Style
	}

	all := []barEntry{
		{"BarBasic", bar.Basic},
		{"BarBraille", bar.Braille},
		{"BarDash", bar.Dash},
		{"BarThin", bar.Thin},
		{"BarBlock", bar.Block},
		{"BarGradient", bar.Gradient},
		{"BarSmooth", bar.Smooth},
	}

	fill := func(_ context.Context, p *clog.Update) error {
		for i := range 1001 {
			p.SetProgress(i).Send()
			time.Sleep(8 * time.Millisecond)
		}
		return nil
	}

	g := clog.Group(context.Background())
	for _, e := range all {
		g.Add(clog.Bar(e.name, 1000, bar.WithStyle(e.style))).Progress(fill)
	}
	g.Wait().Symbol("✅").Msg("All bar styles complete")
}
