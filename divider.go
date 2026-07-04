package clog

import (
	"strings"

	"github.com/gechr/clog/style"
	xansi "github.com/gechr/x/ansi"
)

const (
	defaultDividerWidth = 80
	defaultDividerChar  = '─'
	dividerMinLeader    = 3
)

// DividerBuilder configures and renders a horizontal divider line.
// Create one with [Logger.Divider] or the package-level [Divider] function.
//
// Call [DividerBuilder.Send] to render a plain line, or [DividerBuilder.Msg]
// to render a line with an embedded title:
//
//	clog.Divider().Send()
//	clog.Divider().Msg("Build Phase")
//	clog.Divider().Char('═').Align(AlignCenter).Msg("Results")
type DividerBuilder struct {
	logger     *Logger
	char       rune
	titleAlign Align
	width      int
}

// Divider returns a new [DividerBuilder] for rendering a horizontal rule.
func (l *Logger) Divider() *DividerBuilder {
	return &DividerBuilder{logger: l}
}

// Char sets the character used for the divider line.
// Default is '─'.
func (b *DividerBuilder) Char(char rune) *DividerBuilder {
	b.char = char
	return b
}

// Align sets the title alignment within the divider line.
// Default is [AlignLeft].
func (b *DividerBuilder) Align(align Align) *DividerBuilder {
	b.titleAlign = align
	return b
}

// Width overrides the divider width in columns.
// By default the terminal width is used, falling back to 80 for non-TTY output.
func (b *DividerBuilder) Width(width int) *DividerBuilder {
	b.width = width
	return b
}

// Msg renders the divider with the given title text.
func (b *DividerBuilder) Msg(title string) {
	b.render(title)
}

// Send renders a plain divider line without a title.
func (b *DividerBuilder) Send() {
	b.render("")
}

func (b *DividerBuilder) render(title string) {
	l := b.logger
	l.mu.Lock()
	defer l.mu.Unlock()

	width := b.width
	if width <= 0 {
		width = l.output.Width()
	}
	if width <= 0 {
		width = defaultDividerWidth
	}

	noColor := l.colorsDisabled()

	char := b.char
	if char == 0 {
		char = defaultDividerChar
	}

	var line string
	if title != "" {
		line = renderDividerWithTitle(title, char, width, b.titleAlign, noColor, l.styles)
	} else {
		line = renderDividerLine(char, width, noColor, l.styles)
	}

	l.output.WriteLine(line + nl)
}

func renderDividerLine(char rune, width int, noColor bool, styles *style.Config) string {
	line := strings.Repeat(string(char), width)
	if !noColor && styles.DividerLine != nil {
		return styles.DividerLine.Render(line)
	}
	return line
}

func renderDividerWithTitle(
	title string,
	char rune,
	width int,
	align Align,
	noColor bool,
	styles *style.Config,
) string {
	styledTitle := title
	if !noColor && styles.DividerTitle != nil {
		styledTitle = styles.DividerTitle.Render(title)
	}

	titleWidth := xansi.StringWidth(title)
	padding := 1
	lineCharsAvailable := width - titleWidth - (padding * 2) //nolint:mnd // both sides

	if lineCharsAvailable <= 0 {
		return styledTitle
	}

	var leftCount, rightCount int

	switch align {
	case AlignLeft, AlignNone:
		leftCount = min(dividerMinLeader, lineCharsAvailable)
		rightCount = lineCharsAvailable - leftCount
	case AlignRight:
		rightCount = min(dividerMinLeader, lineCharsAvailable)
		leftCount = lineCharsAvailable - rightCount
	case AlignCenter:
		leftCount = lineCharsAvailable / 2 //nolint:mnd // split evenly
		rightCount = lineCharsAvailable - leftCount
	}

	charStr := string(char)
	leftLine := strings.Repeat(charStr, leftCount)
	rightLine := strings.Repeat(charStr, rightCount)

	if !noColor && styles.DividerLine != nil {
		leftLine = styles.DividerLine.Render(leftLine)
		rightLine = styles.DividerLine.Render(rightLine)
	}

	pad := strings.Repeat(" ", padding)

	return leftLine + pad + styledTitle + pad + rightLine
}
