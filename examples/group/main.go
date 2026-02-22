package main

import (
	"context"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gechr/clog"
)

func main() {
	clog.SetLevel(clog.TraceLevel)
	clog.SetReportTimestamp(true)

	barFill := func(_ context.Context, p *clog.ProgressUpdate) error {
		for i := range 1001 {
			p.SetProgress(i).Send()
			time.Sleep(3 * time.Millisecond)
		}
		return nil
	}

	downloadBar := clog.BarThin
	downloadBar.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))
	downloadBar.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
	downloadBar.WidgetLeft = clog.WidgetPercent(clog.WithDigits(0))
	downloadBar.WidgetRight = clog.WidgetNone

	gradientBar := clog.BarSmooth
	gradientBar.ProgressGradient = clog.DefaultBarGradient()
	gradientBar.WidgetLeft = clog.WidgetPercent(clog.WithDigits(0))
	gradientBar.WidgetRight = clog.WidgetNone

	fileSize := 150 * 1000 * 1000
	bytesBar := clog.BarSmooth
	bytesBar.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("6")))
	bytesBar.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
	bytesBar.WidgetRight = clog.WidgetBytes(clog.WithDigits(0))

	g := clog.NewGroup(context.Background())
	g.Add(clog.Bar("Downloading", 1000).
		Style(downloadBar).Str("file", "release.tar.gz").Elapsed("elapsed")).
		Progress(barFill)
	g.Add(clog.Bar("Building", 1000).
		Style(gradientBar).Str("target", "release").Elapsed("elapsed")).
		Progress(barFill)
	g.Add(clog.Bar("Fetching", fileSize).
		Style(bytesBar).Str("file", "model.bin").Elapsed("elapsed")).
		Progress(func(_ context.Context, p *clog.ProgressUpdate) error {
			steps := 1000
			for i := range steps + 1 {
				p.SetProgress(fileSize * i / steps).Send()
				time.Sleep(3 * time.Millisecond)
			}
			return nil
		})
	g.Add(clog.Spinner("Processing data").Str("workers", "4")).
		Run(func(_ context.Context) error {
			time.Sleep(5 * time.Second)
			return nil
		})
	g.Add(clog.Pulse("Indexing search catalogue")).
		Run(func(_ context.Context) error {
			time.Sleep(5 * time.Second)
			return nil
		})
	g.Wait().Prefix("✅").Msg("All tasks complete")
}
