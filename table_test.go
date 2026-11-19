package clog

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/style"
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
	frame := cfg.snapshot(TableSnapshot{Summary: "Orders", Rows: rows}, false)
	bold := lipgloss.NewStyle().Bold(true)
	header := bold.Render("Item") + "  " + bold.Render("State") + "       " + bold.Render("Detail")
	require.Equal(
		t,
		"\n  Orders\n\n  Item  State       Detail\n  A     new / done  sample",
		frame(80),
	)
	require.Equal(t, "\n  Orders\n\n  Item  State\n  A     new / done", frame(18))
	require.Equal(
		t,
		"\n  "+header+"\n  A     new / "+green.Render("done")+"  sample",
		cfg.snapshot(TableSnapshot{Rows: rows}, true)(80),
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
		cfg.snapshot(TableSnapshot{Rows: []TableRow{row}}, true)(80),
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
		cfg.snapshot(TableSnapshot{Summary: "first\nsecond", Rows: rows}, false)(8),
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
	frame := cfg.snapshot(TableSnapshot{Rows: []TableRow{{"id": "a", "detail": "more"}}}, false)
	require.Equal(t, "\n\n   a  more \n", frame(11))
	require.Equal(t, "\n\n   a \n", frame(10))
	require.Equal(t, "\n\na\n", frame(1))
	require.Empty(t, cfg.snapshot(TableSnapshot{}, false)(20))
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
			frame := cfg.snapshot(snapshot, false)
			require.Equal(t, "\n  123456  xx\n\n"+tc.wide, frame(80))
			require.Equal(t, "\n  123456\n\n"+tc.narrow, frame(8))
		})
	}
	require.Equal(t, "\n  done", cfg.snapshot(TableSnapshot{Footer: "done"}, false)(80))
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
		cfg.snapshot(snapshot, true)(20),
	)
	require.Equal(
		t,
		"\n  Orders\n  Notebook\n\n      sent\n        界",
		cfg.snapshot(snapshot, false)(20),
	)
}

func TestTableSectionPadding(t *testing.T) {
	cfg := TableConfig{
		Headers: true,
		Columns: []TableColumn{{Key: "item", Label: "Item"}},
		Padding: &TablePadding{Top: 1, Right: 1, Left: 2},
	}
	snapshot := TableSnapshot{Summary: "Title", Rows: []TableRow{{"item": "book"}}, Footer: "end"}
	require.Equal(t, "\n  Title \n\n  Item \n  book \n\n  end ", cfg.snapshot(snapshot, false)(30))

	cfg.SummaryPadding = &TablePadding{}
	cfg.BodyPadding = &TablePadding{Left: 5}
	cfg.FooterPadding = &TablePadding{Top: 2, Bottom: 1, Left: 1, Right: 2}
	cfg.FooterAlign = AlignRight
	frame := cfg.snapshot(snapshot, false)
	require.Equal(t, "Title\n     Item\n     book\n\n\n    end  \n", frame(30))
	require.Equal(t, "Tit…\n   …\n   …\n\n\n …  \n", frame(4))

	cfg.Headers = false
	require.Equal(t, "Title", cfg.snapshot(TableSnapshot{Summary: "Title"}, false)(30))
	require.Equal(t, "\n\n end  \n", cfg.snapshot(TableSnapshot{Footer: "end"}, false)(30))
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
			frame := cfg.snapshot(snapshot, false)
			require.Equal(t, tc.want, frame(7))
			require.Equal(t, "界\n x", frame(2))
		})
	}
}
