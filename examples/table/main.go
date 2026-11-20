// Run with go run ./examples/table.
package main

import (
	"flag"
	"fmt"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog"
	"github.com/gechr/clog/style"
)

type stage string

const (
	queued  stage = "queued"
	packed  stage = "packed"
	shipped stage = "shipped"
)

type order struct {
	title   string
	format  string
	history clog.TableValues
}

func main() {
	bordered := flag.Bool("border", false, "draw a border around the table columns")
	var border *clog.TableBorder
	flag.Func("border-color", "draw the border in a hex or ANSI palette color", func(value string) error {
		c := lipgloss.Color(value)
		if _, invalid := c.(lipgloss.NoColor); invalid {
			return fmt.Errorf("invalid border color %q", value)
		}
		border = &clog.TableBorder{Color: c}
		return nil
	})
	columnGap := flag.Int("column-gap", clog.DefaultTableColumnGap, "spaces between adjacent columns")
	headers := flag.Bool("headers", true, "show column headers")
	prompt := flag.Bool("prompt", false, "pause for Enter before the first shipment")
	interval := flag.Duration("interval", time.Second, "time between updates on a terminal")
	footerAlign := clog.AlignLeft
	flag.Func("footer-align", "align the shipment count (left, center, right)", func(value string) error {
		switch value {
		case "left":
			footerAlign = clog.AlignLeft
		case "center":
			footerAlign = clog.AlignCenter
		case "right":
			footerAlign = clog.AlignRight
		default:
			return fmt.Errorf("unknown footer alignment %q (use left, center, or right)", value)
		}
		return nil
	})
	flag.Parse()

	out := clog.Default().Output()
	if *bordered && border == nil {
		border = &clog.TableBorder{}
	}
	table := clog.NewLiveTable(out, clog.TableConfig{
		Headers:      *headers,
		Border:       border,
		ColumnGap:    columnGap,
		FooterAlign:  footerAlign,
		SummaryStyle: new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Magenta)),
		FooterStyle:  new(lipgloss.NewStyle().Foreground(lipgloss.Cyan)),
		Columns: []clog.TableColumn{
			{Key: "title", Label: "TITLE"},
			{Key: "format", Label: "FORMAT", Optional: true},
			{Key: "history", Label: "ORDER HISTORY", Separator: " → "},
		},
		KeyValues: style.KeyValueMap{
			"format": {Values: style.ValueMap{
				"hardcover": new(lipgloss.NewStyle().Foreground(lipgloss.Blue)),
				"paperback": new(lipgloss.NewStyle().Foreground(lipgloss.Magenta)),
				"pocket":    new(lipgloss.NewStyle().Foreground(lipgloss.Cyan)),
			}},
			"history": {Values: style.ValueMap{
				queued:  new(lipgloss.NewStyle().Faint(true)),
				packed:  new(lipgloss.NewStyle().Foreground(lipgloss.Yellow)),
				shipped: new(lipgloss.NewStyle().Foreground(lipgloss.Green)),
			}}},
	})
	defer table.Close()

	orders := []order{
		{title: "Forest Atlas", format: "hardcover", history: clog.TableValues{queued}},
		{title: "Night Walks", format: "paperback", history: clog.TableValues{queued}},
		{title: "Pocket Almanac", format: "pocket", history: clog.TableValues{queued}},
	}
	shippedCount := 0
	update := func() {
		rows := make([]clog.TableRow, 0, len(orders))
		for _, o := range orders {
			rows = append(rows, clog.TableRow{"title": o.title, "format": o.format, "history": o.history})
		}
		table.Update(clog.TableSnapshot{
			Summary: "Book orders",
			Rows:    rows,
			Footer:  fmt.Sprintf("%d/%d shipped", shippedCount, len(orders)),
		})
	}

	update()
	for _, next := range []stage{packed, shipped} {
		if next == shipped && *prompt {
			if _, err := clog.Input("Press Enter to ship the orders "); err != nil {
				return
			}
		}
		for i := range orders {
			if out.IsTTY() {
				time.Sleep(*interval)
			}
			orders[i].history = append(orders[i].history, next)
			if next == shipped {
				shippedCount++
			}
			update()
		}
	}
}
