package printer

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
)

// splitOriginWhitespace splits origin into leading whitespace, visible
// content, and trailing whitespace.
func splitOriginWhitespace(origin string) (string, string, string) {
	trimmed := strings.TrimRightFunc(origin, unicode.IsSpace)
	suffix := origin[len(trimmed):]
	prefixLen := len(trimmed) - len(strings.TrimLeftFunc(trimmed, unicode.IsSpace))
	return origin[:prefixLen], trimmed[prefixLen:], suffix
}

// EmitStyled writes text to buf, applying style if non-nil. Only the visible
// content is styled; surrounding whitespace is emitted verbatim to avoid
// lipgloss padding shorter lines to match the widest line.
func EmitStyled(buf *strings.Builder, text string, style *lipgloss.Style) {
	if style != nil {
		prefix, content, suffix := splitOriginWhitespace(text)
		buf.WriteString(prefix)
		if content != "" {
			buf.WriteString(style.Render(content))
		}
		buf.WriteString(suffix)
	} else {
		buf.WriteString(text)
	}
}
