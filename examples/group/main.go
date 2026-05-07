package main

import (
	"context"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog"
	"github.com/gechr/clog/fx/bar"
	"github.com/gechr/clog/fx/bar/widget"
)

func main() {
	clog.SetLevel(clog.LevelTrace)
	clog.SetReportTimestamp(true)

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
	downloadBar.WidgetLeft = widget.Percent(widget.WithDigits(0))
	downloadBar.WidgetRight = widget.None()

	gradientBar := bar.Smooth
	gradientBar.ProgressGradient = bar.DefaultGradient()
	gradientBar.WidgetLeft = widget.Percent(widget.WithDigits(0))
	gradientBar.WidgetRight = widget.None()

	fileSize := 150 * 1000 * 1000
	bytesBar := bar.Smooth
	bytesBar.StyleFill = new(lipgloss.NewStyle().Foreground(lipgloss.Color("6")))
	bytesBar.StyleEmpty = new(lipgloss.NewStyle().Foreground(lipgloss.Color("8")))
	bytesBar.WidgetRight = widget.Bytes(widget.WithDigits(0))

	g := clog.Group(context.Background())
	g.Add(clog.Bar("Downloading", 1000, bar.WithStyle(downloadBar)).
		Str("file", "release.tar.gz").Elapsed("elapsed")).
		Progress(barFill)
	g.Add(clog.Bar("Building", 1000, bar.WithStyle(gradientBar)).
		Str("target", "release").Elapsed("elapsed")).
		Progress(barFill)
	g.Add(clog.Bar("Fetching", fileSize, bar.WithStyle(bytesBar)).
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
			time.Sleep(5 * time.Second)
			return nil
		})
	g.Add(clog.Pulse("Indexing search catalogue")).
		Run(func(_ context.Context) error {
			time.Sleep(5 * time.Second)
			return nil
		})
	if err := g.Wait().Symbol("✅").Msg("All tasks complete"); err != nil {
		clog.Fatal().Err(err).Msg("group demo failed")
	}
}
