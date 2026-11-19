package clog

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/gechr/clog/style"
	xansi "github.com/gechr/x/ansi"
	"github.com/stretchr/testify/require"
)

func ExampleLiveTable() {
	var out bytes.Buffer
	table := NewLiveTable(
		TestOutput(&out),
		TableConfig{
			Padding: &TablePadding{},
			Columns: []TableColumn{{Key: "item"}, {Key: "state"}},
		},
	)
	rows := []TableRow{{"item": "book", "state": "queued"}}
	table.Update(TableSnapshot{Summary: "Orders", Rows: rows})
	rows[0]["state"] = "packed"
	table.Update(TableSnapshot{Summary: "Orders", Rows: rows})
	table.Close()
	fmt.Print(out.String())
	// Output:
	// Orders
	// book  packed
}

func TestTableSnapshot(t *testing.T) {
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	cfg := TableConfig{
		Headers: true,
		Columns: []TableColumn{
			{Key: "item", Label: "Item"},
			{Key: "state", Label: "State", Separator: " / "},
			{Key: "detail", Label: "Detail", Optional: true},
		},
		KeyValues: style.KeyValueMap{"state": {Values: style.ValueMap{"done": &green}}},
	}
	rows := []TableRow{{"item": "A", "state": TableValues{"new", "done"}, "detail": "sample"}}
	frame := NewLiveTable(nil, cfg).snapshot(TableSnapshot{Summary: "Orders", Rows: rows}, false)
	bold := lipgloss.NewStyle().Bold(true)
	header := bold.Render("Item") + "  " + bold.Render("State") + "       " + bold.Render("Detail")
	require.Equal(
		t,
		"\n  Orders\n\n  Item  State       Detail\n  A     new / done  sample",
		frame.full(80),
	)
	require.Equal(t, "\n  Orders\n\n  Item  State\n  A     new / done", frame.full(18))
	require.Equal(
		t,
		"\n  "+header+"\n  A     new / "+green.Render("done")+"  sample",
		NewLiveTable(nil, cfg).snapshot(TableSnapshot{Rows: rows}, true).full(80),
	)
}

func TestLiveTableSharesTerminal(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	table := NewLiveTable(out, TableConfig{Columns: []TableColumn{{Key: "item"}}})
	table.Update(TableSnapshot{Summary: "", Rows: []TableRow{{"item": "first"}}})
	out.LiveRegion().Suspend(true)
	out.WriteLine("message\n")
	table.Update(TableSnapshot{Summary: "", Rows: []TableRow{{"item": "second"}}})
	out.LiveRegion().Resume()
	table.Close()
	table.Close()
	require.True(t, strings.HasSuffix(buf.String(), "second\n"))
	require.Equal(t, 1, strings.Count(buf.String(), "message\n"))
}

func TestTableTypedStyles(t *testing.T) {
	type stage string
	const ready stage = "ready"
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	bold := lipgloss.NewStyle().Bold(true)
	cfg := TableConfig{
		Columns:   []TableColumn{{Key: "a"}, {Key: "b", Separator: "/"}, {Key: "c"}},
		Values:    style.ValueMap{true: &green, ready: &green},
		KeyValues: style.KeyValueMap{"b": {Values: style.ValueMap{true: &green}, Default: &bold}},
	}
	row := TableRow{"a": ready, "b": TableValues{true, ready}, "c": struct{ V any }{[]int{1}}}
	require.Equal(
		t,
		"\n  "+green.Render("ready")+"  "+green.Render("true")+"/"+bold.Render("ready")+"  {[1]}",
		NewLiveTable(nil, cfg).snapshot(TableSnapshot{Rows: []TableRow{row}}, true).full(80),
	)
}

func TestTableControls(t *testing.T) {
	cfg := TableConfig{
		Padding: &TablePadding{},
		Columns: []TableColumn{{Key: "a"}, {Key: "empty"}, {Key: "b"}},
	}
	rows := []TableRow{{"a": "x\ty", "b": "m\nn"}, {"a": "x\ry", "b": "\x1b[2Jz"}}
	require.Equal(
		t,
		"first\nsecond\nx y  m n\nx y  z",
		NewLiveTable(
			nil,
			cfg,
		).snapshot(TableSnapshot{Summary: "first\nsecond", Rows: rows}, false).full(
			8,
		),
	)
}

func TestTableViewport(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.width, out.height, out.sizeDone = 40, 5, true
	table := NewLiveTable(out, TableConfig{Columns: []TableColumn{{Key: "id"}}})
	var rows []TableRow
	for i := range 10 {
		rows = append(rows, TableRow{"id": i})
	}
	table.Update(TableSnapshot{Rows: rows})
	slot := out.LiveRegion().Register(func(time.Time) string { return "status" }, time.Hour)
	out.LiveRegion().Suspend(true)
	// The status slot and overflow indicator share the available four lines.
	require.Positive(t, strings.Count(buf.String(), "+9 more lines"))
	require.Positive(t, strings.Count(buf.String(), "status"))
	require.Zero(t, strings.Count(buf.String(), "\x1b[11A"))
	out.LiveRegion().Resume()
	out.LiveRegion().Unregister(slot)
	table.Close()
	table.Update(TableSnapshot{Summary: "late"})
	require.True(
		t,
		strings.HasSuffix(buf.String(), "\n  0\n  1\n  2\n  3\n  4\n  5\n  6\n  7\n  8\n  9\n"),
	)
}

func TestTableCustomPadding(t *testing.T) {
	cfg := TableConfig{
		Columns: []TableColumn{{Key: "id"}, {Key: "detail", Optional: true}},
		Padding: &TablePadding{Top: 2, Left: 3, Right: 1, Bottom: 1},
	}
	frame := NewLiveTable(
		nil,
		cfg,
	).snapshot(TableSnapshot{Rows: []TableRow{{"id": "a", "detail": "more"}}}, false)
	require.Equal(t, "\n\n   a  more \n", frame.full(11))
	require.Equal(t, "\n\n   a \n", frame.full(10))
	require.Equal(t, "\n\na\n", frame.full(1))
	require.Empty(t, NewLiveTable(nil, cfg).snapshot(TableSnapshot{}, false).full(20))
}

func TestTableFooterAlignment(t *testing.T) {
	snapshot := TableSnapshot{Rows: []TableRow{{"id": "123456", "details": "xx"}}, Footer: "界\nend"}
	cfg := TableConfig{Columns: []TableColumn{{Key: "id"}, {Key: "details", Optional: true}}}
	for _, tc := range []struct {
		name         string
		align        Align
		wide, narrow string
	}{
		{"default", AlignNone, "  界\n  end", "  界\n  end"},
		{"left", AlignLeft, "  界\n  end", "  界\n  end"},
		{"center", AlignCenter, "      界\n     end", "    界\n   end"},
		{"right", AlignRight, "          界\n         end", "      界\n     end"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg.FooterAlign = tc.align
			frame := NewLiveTable(nil, cfg).snapshot(snapshot, false)
			require.Equal(t, "\n  123456  xx\n\n"+tc.wide, frame.full(80))
			require.Equal(t, "\n  123456\n\n"+tc.narrow, frame.full(8))
		})
	}
	require.Equal(
		t,
		"\n  done",
		NewLiveTable(nil, cfg).snapshot(TableSnapshot{Footer: "done"}, false).full(80),
	)
}

func TestTableSummaryAndFooterStyles(t *testing.T) {
	summary := lipgloss.NewStyle().Bold(true)
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	cfg := TableConfig{
		Columns:      []TableColumn{{Key: "item"}},
		SummaryStyle: &summary,
		FooterStyle:  &footer,
		FooterAlign:  AlignRight,
	}
	snapshot := TableSnapshot{
		Summary: "Orders",
		Rows:    []TableRow{{"item": "Notebook"}},
		Footer:  "sent\n界",
	}
	require.Equal(
		t,
		"\n  "+summary.Render(
			"Orders",
		)+"\n  Notebook\n\n      "+footer.Render(
			"sent",
		)+"\n        "+footer.Render(
			"界",
		),
		NewLiveTable(nil, cfg).snapshot(snapshot, true).full(20),
	)
	require.Equal(
		t,
		"\n  Orders\n  Notebook\n\n      sent\n        界",
		NewLiveTable(nil, cfg).snapshot(snapshot, false).full(20),
	)
}

func TestTableSectionPadding(t *testing.T) {
	cfg := TableConfig{
		Headers: true,
		Columns: []TableColumn{{Key: "item", Label: "Item"}},
		Padding: &TablePadding{Top: 1, Right: 1, Left: 2},
	}
	snapshot := TableSnapshot{Summary: "Title", Rows: []TableRow{{"item": "book"}}, Footer: "end"}
	require.Equal(
		t,
		"\n  Title \n\n  Item \n  book \n\n  end ",
		NewLiveTable(nil, cfg).snapshot(snapshot, false).full(30),
	)

	cfg.SummaryPadding = &TablePadding{}
	cfg.BodyPadding = &TablePadding{Left: 5}
	cfg.FooterPadding = &TablePadding{Top: 2, Bottom: 1, Left: 1, Right: 2}
	cfg.FooterAlign = AlignRight
	frame := NewLiveTable(nil, cfg).snapshot(snapshot, false)
	require.Equal(t, "Title\n     Item\n     book\n\n\n    end  \n", frame.full(30))
	require.Equal(t, "Tit…\n   …\n   …\n\n\n …  \n", frame.full(4))

	cfg.Headers = false
	require.Equal(
		t,
		"Title",
		NewLiveTable(nil, cfg).snapshot(TableSnapshot{Summary: "Title"}, false).full(30),
	)
	require.Equal(
		t,
		"\n\n end  \n",
		NewLiveTable(nil, cfg).snapshot(TableSnapshot{Footer: "end"}, false).full(30),
	)
}

func TestTableColumnGap(t *testing.T) {
	cfg := TableConfig{
		Padding:     &TablePadding{},
		Columns:     []TableColumn{{Key: "a"}, {Key: "empty"}, {Key: "b", Optional: true}},
		FooterAlign: AlignRight,
	}
	snapshot := TableSnapshot{Rows: []TableRow{{"a": "界", "b": "B"}}, Footer: "x"}
	for _, tc := range []struct {
		name string
		gap  *int
		want string
	}{
		{"default", nil, "界  B\n    x"},
		{"zero", new(0), "界B\n  x"},
		{"negative", new(-1), "界B\n  x"},
		{"wide", new(4), "界    B\n      x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg.ColumnGap = tc.gap
			frame := NewLiveTable(nil, cfg).snapshot(snapshot, false)
			require.Equal(t, tc.want, frame.full(7))
			require.Equal(t, "界\n x", frame.full(2))
		})
	}
}

func TestTableWidthsAcrossUpdates(t *testing.T) {
	for _, tc := range []struct {
		name        string
		minWidth    int
		retain      bool
		first, last string
	}{
		{"default", 0, false, "a  x", "b  x"},
		{"minimum", 3, false, "a    x", "b    x"},
		{"retain", 0, true, "a  x", "b     x"},
		{"both", 3, true, "a    x", "b     x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			table := NewLiveTable(TestOutput(&buf), TableConfig{
				Columns: []TableColumn{
					{Key: "a", MinWidth: tc.minWidth},
					{Key: "b", Optional: true},
				},
				RetainWidths: tc.retain,
				Padding:      &TablePadding{},
			})
			update := func(s string) func(int) string {
				table.Update(TableSnapshot{Rows: []TableRow{{"a": s, "b": "x"}}})
				return table.current.Load().full
			}
			first := update("a")
			require.Equal(t, tc.first, first(20))
			wide := update("界界")
			require.Equal(t, "界界  x", wide(20))
			require.Equal(t, "界界", wide(5))
			last := update("b")
			require.Equal(t, tc.last, last(20))
			require.Equal(t, tc.first, first(21))
			table.Close()
			require.Equal(t, tc.last+nl, buf.String())
		})
	}
}

func TestLiveTableCloseModes(t *testing.T) {
	for _, tc := range []struct {
		name  string
		mode  TableCloseMode
		final string
	}{
		{"keep", TableCloseKeep, "Orders\nItem\nBook\nDone\n"},
		{"summary", TableCloseSummary, "Orders\nDone\n"},
		{"clear", TableCloseClear, ""},
	} {
		for _, tty := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/tty=%t", tc.name, tty), func(t *testing.T) {
				var buf bytes.Buffer
				out := TestOutput(&buf)
				out.isTTY = tty
				table := NewLiveTable(out, TableConfig{
					CloseMode: tc.mode,
					Columns:   []TableColumn{{Key: "item", Label: "Item"}},
					Headers:   true,
					Padding:   &TablePadding{},
				})
				table.Update(
					TableSnapshot{
						Summary: "Orders",
						Rows:    []TableRow{{"item": "Book"}},
						Footer:  "Done",
					},
				)
				// Inspect terminal output after stopping the live render loop.
				if tty {
					out.LiveRegion().Suspend(true)
					require.NotEmpty(t, buf.Bytes())
					buf.Reset()
				} else {
					require.Empty(t, buf.String())
				}
				table.Close()
				table.Close()
				require.Equal(t, tc.final, ansi.Strip(buf.String()))
			})
		}
	}
}

func TestLiveTableClearErasesPadding(t *testing.T) {
	var buf bytes.Buffer
	out := TestOutput(&buf)
	out.isTTY = true
	out.width, out.height, out.sizeDone = 40, 25, true
	table := NewLiveTable(out, TableConfig{
		CloseMode: TableCloseClear,
		Headers:   true,
		Columns:   []TableColumn{{Key: "item", Label: "Item"}},
		Padding:   &TablePadding{Top: 1, Bottom: 1, Left: 2, Right: 3},
	})
	out.WriteLine("before" + nl)
	table.Update(TableSnapshot{
		Summary: "Orders",
		Rows:    []TableRow{{"item": "Book"}, {"item": "Map"}},
		Footer:  "Done",
	})
	table.Close()
	out.WriteLine("after" + nl)

	// Three summary lines, five body lines, and three footer lines, including padding.
	erase := xansi.EnableSyncOutput + xansi.CursorUp(11) + xansi.CursorHorizontalAbsolute(1) +
		xansi.CursorHorizontalAbsolute(
			1,
		) + xansi.EraseScreenBelow + xansi.DisableSyncOutput + xansi.ShowCursor
	require.True(
		t,
		strings.HasSuffix(buf.String(), erase+"after"+nl),
		"close must erase back to the first padding line",
	)
}
