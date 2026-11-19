# Live tables

Create a table with `clog.NewLiveTable(output, config)`. Each `Update` takes a complete `clog.TableSnapshot` with ordered rows and an optional summary and footer. The table shares the output's live region with animations and log lines. By default, `Close` leaves the final snapshot in scrollback. Set `TableConfig.CloseMode` to `clog.TableCloseSummary` to keep only the summary and footer, or `clog.TableCloseClear` to erase the table and its padding. When the live region is empty, the next log line starts where the cleared block began. With redirected output, only the selected final content is written.

```go
type Stage string
const packed Stage = "packed"

green := lipgloss.NewStyle().Foreground(lipgloss.Green)
table := clog.NewLiveTable(clog.Default().Output(), clog.TableConfig{
    Columns: []clog.TableColumn{
        {Key: "item", Label: "Item"},
        {Key: "steps", Label: "Steps", Separator: " → "},
    },
    Headers: true,
    SummaryStyle: new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Magenta)),
    FooterStyle: &green,
    FooterAlign: clog.AlignRight,
    KeyValues: style.KeyValueMap{
        "steps": {Values: style.ValueMap{packed: &green}},
    },
})
defer table.Close()
table.Update(clog.TableSnapshot{
    Summary: "Orders",
    Footer: "1 order packed",
    Rows: []clog.TableRow{
        {"item": "Book", "steps": clog.TableValues{"queued", packed}},
    },
})
```

Headers are optional and appear in bold when terminal styling is enabled. Columns marked `Optional` drop from right to left as available width shrinks. Cells accept scalar values or a `TableValues` sequence. Styling uses the original Go value as the lookup key. The renderer formats it as single-line text and preserves line breaks in the summary.

Set a column's `MinWidth` to reserve terminal cells before the first update, for example `TableColumn{Key: "format", MinWidth: 12}`. Set `TableConfig.RetainWidths` to keep each column at its widest observed value for the life of that table. Wider values can still expand it, and optional columns can still disappear when the terminal is narrow. With both options unset, each update sizes columns to the current snapshot.

By default the table starts with one blank line, indents each section by two spaces, and leaves two spaces between visible columns. Set `ColumnGap: new(4)` for a wider gap or `ColumnGap: new(0)` for touching columns. The gap also counts toward the width used to fit optional columns.

Set `TableConfig.Padding` to apply the same four sides to every section. `SummaryPadding`, `BodyPadding` (headers and rows), and `FooterPadding` each replace the shared padding for that section. `Top` and `Bottom` count blank lines. `Left` and `Right` count spaces. For example, `Padding: &clog.TablePadding{Top: 1, Left: 4}, FooterPadding: &clog.TablePadding{}` indents the summary and rows and places the footer flush left. An absent section adds no padding.

With the default padding, a blank line separates the summary from the headers and the footer from the rows. Set `TableConfig.FooterAlign` to `clog.AlignLeft` (the default), `clog.AlignCenter`, or `clog.AlignRight`. Each footer line aligns within the width of the visible columns, inside the footer's padding. Use `SummaryStyle` and `FooterStyle` to style their text independently. Both respect the output's color mode.

Use `out.LightDark(light, dark)` when a custom color needs more contrast on a light background. For example, `lipgloss.NewStyle().Foreground(out.LightDark(lipgloss.Color("25"), lipgloss.Color("117")))` chooses deep blue on light terminals and pale blue on dark terminals. The output caches its background detection and uses the dark variant when detection is unavailable.

Logger input prompts suspend the live region for the read and restore its previous state afterwards.

## Try the example

Run the book-order example from the repository root. Three orders move from queued to packed to shipped. The `format` column uses `KeyValues` to assign preset colors to hardcover, paperback, and pocket. The footer counts shipments, and each row keeps its colored history. Narrow the terminal to see the optional format column disappear.

```sh
go run ./examples/table
go run ./examples/table -footer-align=right -column-gap=4
go run ./examples/table -headers=false -prompt
go run ./examples/table | cat
```

With `-prompt`, the table pauses before the first shipment. Press Enter to bring it back and continue. Use `-interval=2s` to slow the transitions. Piping the output skips the delays and prints the completed table once.
