package main

import (
	"context"
	"flag"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
)

func main() {
	stylesFlag := flag.Bool("styles", false, "show all bar style presets")
	flag.Parse()

	clog.SetLevel(clog.TraceLevel)
	clog.SetReportTimestamp(true)

	if *stylesFlag {
		showStyles()
		return
	}

	downloadBar := clog.BarThin
	downloadBar.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))
	downloadBar.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
	downloadBar.WidgetLeft = clog.WidgetPercent()
	downloadBar.WidgetRight = clog.WidgetNone

	_ = clog.Bar("Downloading", 1000).
		Style(downloadBar).
		Str("file", "release.tar.gz").
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *clog.ProgressUpdate) error {
			for i := range 1001 {
				p.SetProgress(i).Send()
				time.Sleep(3 * time.Millisecond)
			}
			return nil
		}).
		Prefix("✅").
		Msg("Download complete")

	fileSize := 150 * 1000 * 1000
	bytesBar := clog.BarSmooth
	bytesBar.ProgressGradient = clog.DefaultBarGradient()
	bytesBar.WidgetRight = clog.WidgetBytes()

	_ = clog.Bar("Fetching", fileSize).
		Style(bytesBar).
		Str("file", "model.bin").
		Elapsed("elapsed").
		Progress(context.Background(), func(_ context.Context, p *clog.ProgressUpdate) error {
			steps := 1000
			for i := range steps + 1 {
				p.SetProgress(fileSize * i / steps).Send()
				time.Sleep(3 * time.Millisecond)
			}
			return nil
		}).
		Prefix("✅").
		Msg("Model downloaded")
}

func showStyles() {
	type barEntry struct {
		name  string
		style clog.BarStyle
	}

	all := []barEntry{
		{"BarBasic", clog.BarBasic},
		{"BarDash", clog.BarDash},
		{"BarThin", clog.BarThin},
		{"BarBlock", clog.BarBlock},
		{"BarGradient", clog.BarGradient},
		{"BarSmooth", clog.BarSmooth},
	}

	fill := func(_ context.Context, p *clog.ProgressUpdate) error {
		for i := range 1001 {
			p.SetProgress(i).Send()
			time.Sleep(8 * time.Millisecond)
		}
		return nil
	}

	g := clog.NewGroup(context.Background())
	for _, e := range all {
		g.Add(clog.Bar(e.name, 1000).Style(e.style)).Progress(fill)
	}
	g.Wait().Prefix("✅").Msg("All bar styles complete")
}
