package style

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// BacktickMode controls what happens to the backtick delimiters around a
// styled `code` span.
type BacktickMode int

const (
	// BacktickUnset renders like [BacktickStrip]. As the zero value it reads
	// as "not set" to [Config.Merge], so merging a config that leaves the
	// mode unset keeps the current one.
	BacktickUnset BacktickMode = iota
	// BacktickStrip drops the backtick delimiters, shrinking the message by
	// two visible columns per span - the default, intended for prose.
	BacktickStrip
	// BacktickKeep keeps the backtick delimiters, styled with the span, so
	// the message's visible width is exactly what the caller wrote.
	// Use this when logging pre-aligned content such as padded table rows.
	BacktickKeep
)

// RenderBackticks styles s for display: text inside a matched pair of backticks
// is rendered with code and the delimiters removed, while the surrounding text
// is rendered with base. A nil style renders its text unstyled. Equivalent to
// [BacktickStrip.Render].
func RenderBackticks(s string, base, code *lipgloss.Style) string {
	return BacktickStrip.Render(s, base, code)
}

// Render styles s for display: text inside a matched pair of backticks is
// rendered with code, while the surrounding text is rendered with base. The
// delimiters are dropped under [BacktickStrip] (and [BacktickUnset]) or kept
// inside the styled span under [BacktickKeep]. A nil style renders its text
// unstyled.
//
// When code is nil or s carries no backticks, s is rendered whole by base with
// any backticks left intact - the behaviour before backtick styling, and what a
// non-color writer falls back to. An unmatched trailing backtick is not a
// delimiter: the remainder of s (the backtick included) is rendered by base.
func (m BacktickMode) Render(s string, base, code *lipgloss.Style) string {
	if code == nil || !strings.Contains(s, "`") {
		return renderOr(base, s)
	}

	var b strings.Builder
	last := 0
	for i := 0; i < len(s); i++ {
		if s[i] != '`' {
			continue
		}
		rel := strings.IndexByte(s[i+1:], '`')
		if rel < 0 {
			break
		}
		end := i + 1 + rel
		b.WriteString(renderOr(base, s[last:i]))
		if m == BacktickKeep {
			b.WriteString(code.Render(s[i : end+1]))
		} else {
			b.WriteString(code.Render(s[i+1 : end]))
		}
		i = end
		last = end + 1
	}
	b.WriteString(renderOr(base, s[last:]))
	return b.String()
}

// renderOr renders s with style, or returns s unchanged when style is nil or s
// is empty.
func renderOr(style *lipgloss.Style, s string) string {
	if style == nil || s == "" {
		return s
	}
	return style.Render(s)
}
