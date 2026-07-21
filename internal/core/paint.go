package core

import (
	"strings"

	xansi "github.com/gechr/x/ansi"
)

// nl is the newline string written between block lines and after the last
// line of a painted block.
const nl = "\n"

// FrameRows returns the number of physical terminal rows the rendered line
// occupies once wrapping at termWidth is accounted for. ANSI escape codes
// are stripped before measuring. Falls back to 1 when the width is unknown
// (e.g. Output.Width() == 0).
func FrameRows(line string, termWidth int) int {
	if termWidth <= 0 {
		return 1
	}
	w := xansi.StringWidth(line)
	if w <= 0 {
		return 1
	}
	return (w + termWidth - 1) / termWidth
}

// BlockRows returns the total number of physical terminal rows a block of
// lines occupies once wrapping at termWidth is accounted for.
func BlockRows(lines []string, termWidth int) int {
	rows := 0
	for _, line := range lines {
		rows += FrameRows(line, termWidth)
	}
	return rows
}

// AppendRepaint appends an overwrite-in-place repaint of lines over the
// previously rendered block of prevRows physical rows, bracketed by DEC 2026
// synchronized-output markers. The block is not erased up front - that would
// flash blank on terminals without synchronized output - instead each line
// clears only the rows it touches, and rows the new frame no longer covers
// are erased at the end. Returns the physical row count of the new frame.
func AppendRepaint(buf *strings.Builder, lines []string, prevRows, width int) int {
	rows := BlockRows(lines, width)
	buf.WriteString(xansi.EnableSyncOutput)
	if prevRows > 0 {
		buf.WriteString(xansi.CursorUp(prevRows))
		buf.WriteString(xansi.CursorHorizontalAbsolute(1))
		if width <= 0 {
			// Wrap math is unreliable without a known width; fall back to
			// erasing the whole block before repainting.
			buf.WriteString(xansi.EraseScreenBelow)
		}
	}
	for i, line := range lines {
		if i > 0 {
			// Literal newline so the terminal advances (and scrolls when
			// the block reaches the viewport bottom). CursorNextLine would
			// clamp at the bottom row and silently fail to advance, leaving
			// the row arithmetic out of sync with reality. xansi.ClearLine
			// ends with "\r" so the column is reset before the next line is
			// written.
			buf.WriteString(nl)
		}
		buf.WriteString(xansi.ClearLine)
		buf.WriteString(line)
		// ClearLine only cleared the first physical row of a wrapped line;
		// intermediate rows are fully overwritten by the content itself,
		// but a partial final row keeps its stale tail. Trim it unless the
		// row is exactly full: the cursor then sits in the deferred-wrap
		// state, where EL0 would erase the last glyph instead.
		if w := xansi.StringWidth(line); width > 0 && w > width && w%width != 0 {
			buf.WriteString(xansi.EraseLineRight)
		}
	}
	// Park the cursor one line below the block only while a block is still
	// rendered, so zero-line frames don't leave a blank gap. Literal newline
	// for the same scroll-at-bottom reason as above.
	if len(lines) > 0 {
		buf.WriteString(nl)
	}
	// Rows the new frame no longer covers (a shrinking block, or a zero-line
	// frame replacing a previous block) are erased below the park position.
	// Steady-state frames skip the erase entirely.
	if width > 0 && rows != prevRows {
		buf.WriteString(xansi.CursorHorizontalAbsolute(1))
		buf.WriteString(xansi.EraseScreenBelow)
	}
	buf.WriteString(xansi.DisableSyncOutput)
	return rows
}
