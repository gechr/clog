package clog

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/gechr/clog/style"
)

const (
	// DefaultTableColumnGap is the number of spaces between adjacent columns.
	DefaultTableColumnGap   = 2
	defaultTablePaddingTop  = 1
	defaultTablePaddingLeft = 2
)

// TableColumn describes a column in display order. Optional columns are
// dropped from right to left when the table exceeds the output width.
type TableColumn struct {
	Key       string
	Label     string
	Optional  bool
	Separator string // joins independently styled values in one cell
}

// TablePadding sets the blank lines above and below a section and the spaces
// on either side of each content line. Negative sizes count as zero.
type TablePadding struct {
	Top, Right, Bottom, Left int
}

// TableConfig controls the layout and value colors of a LiveTable.
// Treat the configuration, including its style maps, as immutable after construction.
type TableConfig struct {
	Columns        []TableColumn
	Headers        bool
	ColumnGap      *int            // spaces between visible columns; nil uses two
	FooterAlign    Align           // alignment within the visible columns; defaults to left
	SummaryStyle   *lipgloss.Style // nil renders the summary as plain text
	FooterStyle    *lipgloss.Style // nil renders the footer as plain text
	Padding        *TablePadding   // shared by all sections unless overridden
	SummaryPadding *TablePadding
	BodyPadding    *TablePadding // headers and rows
	FooterPadding  *TablePadding
	KeyValues      style.KeyValueMap
	Values         style.ValueMap
}

// TableRow maps column keys to ordered values. Missing cells render empty.
type TableRow map[string]any

// TableValues displays multiple independently styled values in one cell.
type TableValues []any

// TableSnapshot replaces the optional summary, ordered rows, and optional footer.
type TableSnapshot struct {
	Summary string
	Rows    []TableRow
	Footer  string
}

// LiveTable maintains a single replaceable snapshot on an Output.
type LiveTable struct {
	out        *Output
	cfg        TableConfig
	mu         sync.Mutex
	slot       uint64
	stopResize func()
	closed     bool

	current atomic.Pointer[func(int) string]
}

// NewLiveTable creates a table. Update starts its display on the first snapshot.
func NewLiveTable(out *Output, cfg TableConfig) *LiveTable {
	cfg.Columns = slices.Clone(cfg.Columns)
	return &LiveTable{out: out, cfg: cfg}
}

// Update replaces the summary, rows, and footer. The caller may reuse rows after Update returns.
func (t *LiveTable) Update(snapshot TableSnapshot) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	frame := t.cfg.snapshot(snapshot, !t.out.ColorsDisabled())
	t.current.Store(&frame)
	if !t.out.IsTTY() {
		return
	}
	if t.slot == 0 {
		t.stopResize = t.out.ListenResize()
		t.slot = t.out.LiveRegion().Register(func(time.Time) string {
			return (*t.current.Load())(t.out.Width())
		}, time.Second)
	} else {
		t.out.LiveRegion().RenderFrame(time.Now())
	}
}

// Close leaves the final snapshot in scrollback. Redirected output receives
// just this snapshot. Close can be called repeatedly.
func (t *LiveTable) Close() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return
	}
	t.closed = true
	if t.slot != 0 {
		t.out.LiveRegion().Unregister(t.slot)
		t.stopResize()
	}
	if frame := t.current.Load(); frame != nil {
		if s := (*frame)(t.out.Width()); s != "" {
			t.out.WriteLine(s + nl)
		}
	}
}

func (cfg TableConfig) snapshot(snapshot TableSnapshot, color bool) func(int) string {
	columnGap := DefaultTableColumnGap
	if cfg.ColumnGap != nil {
		columnGap = max(0, *cfg.ColumnGap)
	}
	styles := style.Config{KeyValues: cfg.KeyValues}
	var cells [][]string
	if cfg.Headers {
		headerStyle := lipgloss.NewStyle().Bold(true)
		line := make([]string, len(cfg.Columns))
		for i, c := range cfg.Columns {
			line[i] = tableText(c.Label)
			if color && line[i] != "" {
				line[i] = headerStyle.Render(line[i])
			}
		}
		cells = append(cells, line)
	}
	for _, row := range snapshot.Rows {
		line := make([]string, len(cfg.Columns))
		for i, c := range cfg.Columns {
			value, ok := row[c.Key]
			if !ok || value == nil {
				continue
			}
			values, multiple := value.(TableValues)
			if !multiple {
				values = TableValues{value}
			}
			text := make([]string, len(values))
			for j, v := range values {
				part := tableText(fmt.Sprint(v))
				if color {
					st, matched := lookupKeyValueStyle(c.Key, v, &styles)
					if !matched {
						st = lookupValueStyle(v, cfg.Values)
					}
					if st != nil {
						part = st.Inline(true).Render(part)
					}
				}
				text[j] = part
			}
			line[i] = strings.Join(text, tableText(c.Separator))
		}
		cells = append(cells, line)
	}
	widths := make([]int, len(cfg.Columns))
	for _, line := range cells {
		for i, s := range line {
			widths[i] = max(widths[i], ansi.StringWidth(s))
		}
	}
	summary := tableTextLines(snapshot.Summary, cfg.SummaryStyle, color)
	footer := tableTextLines(snapshot.Footer, cfg.FooterStyle, color)
	summaryPad := cfg.sectionPadding(cfg.SummaryPadding, true)
	bodyGap := len(summary) == 0 ||
		(cfg.Headers && len(cfg.Columns) > 0 && summary[len(summary)-1] != "")
	bodyPad := cfg.sectionPadding(cfg.BodyPadding, bodyGap)

	var last string
	lastWidth := -1
	return func(width int) string {
		if width == lastWidth {
			return last
		}
		pad, bodyWidth := bodyPad.fit(width)
		rows, columnsWidth := cfg.renderRows(cells, widths, bodyWidth, columnGap)
		var sections []string
		if len(summary) > 0 {
			p, w := summaryPad.fit(width)
			sections = append(sections, p.render(summary, w, AlignNone, 0))
		}
		if len(rows) > 0 {
			sections = append(sections, pad.render(rows, bodyWidth, AlignNone, 0))
		}
		if len(footer) > 0 {
			previous := rows
			if len(previous) == 0 {
				previous = summary
			}
			blankAbove := len(previous) == 0 || previous[len(previous)-1] != ""
			p, w := cfg.sectionPadding(cfg.FooterPadding, blankAbove).fit(width)
			// Align inside the body's visible footprint using the footer's own insets.
			alignWidth := 0
			if columnsWidth > 0 {
				alignWidth = pad.Left + columnsWidth + pad.Right - p.Left - p.Right
			}
			sections = append(sections, p.render(footer, w, cfg.FooterAlign, alignWidth))
		}
		last = strings.Join(sections, nl)
		lastWidth = width
		return last
	}
}

func (cfg TableConfig) renderRows(cells [][]string, widths []int, width, gap int) ([]string, int) {
	if width > 0 {
		gap = min(gap, width)
	}
	sizes := slices.Clone(widths)
	total, visible := 0, 0
	for _, size := range sizes {
		if size > 0 {
			total += size
			visible++
		}
	}
	total += max(0, visible-1) * gap
	for i := len(sizes) - 1; i >= 0 && width > 0 && total > width; i-- {
		if sizes[i] > 0 && cfg.Columns[i].Optional {
			total -= sizes[i]
			visible--
			if visible > 0 {
				total -= gap
			}
			sizes[i] = 0
		}
	}
	if visible == 0 {
		return nil, 0
	}
	var rows []string
	separator := ""
	if visible > 1 {
		separator = strings.Repeat(" ", gap)
	}
	for _, line := range cells {
		var fields []string
		for i, s := range line {
			if sizes[i] > 0 {
				fields = append(fields, s+strings.Repeat(" ", sizes[i]-ansi.StringWidth(s)))
			}
		}
		rows = append(rows, strings.TrimRight(strings.Join(fields, separator), " "))
	}
	if width > 0 {
		total = min(total, width)
	}
	return rows, total
}

func (cfg TableConfig) sectionPadding(override *TablePadding, blankAbove bool) TablePadding {
	if override != nil {
		return *override
	}
	if cfg.Padding != nil {
		return *cfg.Padding
	}
	p := TablePadding{Left: defaultTablePaddingLeft}
	if blankAbove {
		p.Top = defaultTablePaddingTop
	}
	return p
}

// fit preserves at least one visible character on a narrow terminal. A zero width is unbounded.
func (p TablePadding) fit(width int) (TablePadding, int) {
	p.Left, p.Right = max(0, p.Left), max(0, p.Right)
	if width > 0 {
		p.Left = min(p.Left, width-1)
		p.Right = min(p.Right, width-p.Left-1)
		width -= p.Left + p.Right
	}
	return p, width
}

func (p TablePadding) render(lines []string, width int, align Align, alignWidth int) string {
	left, right := strings.Repeat(" ", p.Left), strings.Repeat(" ", p.Right)
	var b strings.Builder
	b.WriteString(strings.Repeat(nl, max(0, p.Top)))
	for i, line := range lines {
		if i > 0 {
			b.WriteString(nl)
		}
		if line == "" {
			continue
		}
		if width > 0 {
			line = ansi.Truncate(line, width, "…")
		}
		b.WriteString(left)
		space := max(0, alignWidth-ansi.StringWidth(line))
		switch align {
		case AlignNone, AlignLeft:
		case AlignRight:
			b.WriteString(strings.Repeat(" ", space))
		case AlignCenter:
			b.WriteString(strings.Repeat(" ", space/2)) //nolint:mnd // split spare columns evenly
		}
		b.WriteString(line)
		b.WriteString(right)
	}
	b.WriteString(strings.Repeat(nl, max(0, p.Bottom)))
	return b.String()
}

func tableTextLines(s string, st *lipgloss.Style, color bool) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, nl)
	for i, line := range lines {
		lines[i] = tableText(line)
		if color && st != nil && lines[i] != "" {
			lines[i] = st.Inline(true).Render(lines[i])
		}
	}
	return lines
}

// Keep table geometry independent of terminal controls in caller text.
func tableText(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, ansi.Strip(s))
}
