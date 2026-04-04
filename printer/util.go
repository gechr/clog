package printer

import (
	"strings"
	"unicode"
)

// SplitOriginWhitespace splits Origin into leading whitespace, visible
// content, and trailing whitespace. This avoids passing newlines into
// lipgloss.Render, which pads shorter lines to match the widest line.
func SplitOriginWhitespace(origin string) (string, string, string) {
	trimmed := strings.TrimRightFunc(origin, unicode.IsSpace)
	suffix := origin[len(trimmed):]
	prefixLen := len(trimmed) - len(strings.TrimLeftFunc(trimmed, unicode.IsSpace))
	return origin[:prefixLen], trimmed[prefixLen:], suffix
}
