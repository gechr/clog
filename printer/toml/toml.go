// Package toml provides TOML syntax highlighting for clog.
package toml

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer"
	"github.com/gechr/clog/style"
)

// Highlight applies syntax highlighting to s using the provided styles.
// Original formatting is preserved. Returns s unchanged when styles is nil.
//
// The scanner is line-oriented and handles comments, table headers,
// key-value pairs, and all TOML value types including multi-line strings.
func Highlight(s string, styles *style.TOML) string {
	if styles == nil || s == "" {
		return s
	}

	var buf strings.Builder
	buf.Grow(len(s) * 2) //nolint:mnd // extra capacity for ANSI escapes

	data := []byte(s)
	n := len(data)
	i := 0

	for i < n {
		// Skip whitespace at start of line.
		lineStart := i
		for i < n && (data[i] == ' ' || data[i] == '\t') {
			i++
		}

		if i >= n {
			buf.Write(data[lineStart:])
			break
		}

		switch data[i] {
		case '#':
			end := scanToEOL(data, i)
			buf.Write(data[lineStart:i])
			printer.EmitStyled(&buf, string(data[i:end]), styles.Comment)
			i = end

		case '[':
			buf.Write(data[lineStart:i])
			i = highlightTableHeader(data, i, n, &buf, styles)

		case '\n':
			buf.WriteByte('\n')
			i++

		default:
			// Key-value pair.
			buf.Write(data[lineStart:i])
			i = highlightKeyValue(data, i, n, &buf, styles)
		}
	}

	return buf.String()
}

// highlightTableHeader highlights [table] or [[array]] headers.
func highlightTableHeader(data []byte, i, n int, buf *strings.Builder, styles *style.TOML) int {
	arrayTable := i+1 < n && data[i+1] == '['

	// Find the closing bracket(s).
	start := i
	depth := 0
	j := i
	for j < n && data[j] != '\n' {
		if data[j] == '[' {
			depth++
		} else if data[j] == ']' {
			depth--
			if depth == 0 {
				j++
				if arrayTable && j < n && data[j] == ']' {
					j++
				}
				break
			}
		}
		j++
	}

	printer.EmitStyled(buf, string(data[start:j]), styles.TableKey)

	// Emit any trailing comment on the same line.
	for j < n && (data[j] == ' ' || data[j] == '\t') {
		buf.WriteByte(data[j])
		j++
	}
	if j < n && data[j] == '#' {
		end := scanToEOL(data, j)
		printer.EmitStyled(buf, string(data[j:end]), styles.Comment)
		j = end
	}

	return j
}

// highlightKeyValue highlights a key = value expression.
func highlightKeyValue(data []byte, i, n int, buf *strings.Builder, styles *style.TOML) int {
	// Scan key (bare or quoted).
	keyStart := i
	i = scanKey(data, i, n)
	printer.EmitStyled(buf, string(data[keyStart:i]), styles.Key)

	// Emit whitespace and '='.
	for i < n && (data[i] == ' ' || data[i] == '\t') {
		buf.WriteByte(data[i])
		i++
	}
	if i < n && data[i] == '=' {
		printer.EmitStyled(buf, "=", styles.Punctuation)
		i++
	}

	// Emit whitespace after '='.
	for i < n && (data[i] == ' ' || data[i] == '\t') {
		buf.WriteByte(data[i])
		i++
	}

	// Highlight value.
	return highlightValue(data, i, n, buf, styles)
}

// scanKey scans a TOML key (bare, quoted, or dotted).
func scanKey(data []byte, i, n int) int {
	for i < n && data[i] != '=' && data[i] != '\n' {
		if data[i] == '"' || data[i] == '\'' {
			i = scanQuotedString(data, i, n)
		} else {
			i++
		}
	}
	// Trim trailing whitespace from key span.
	for i > 0 && (data[i-1] == ' ' || data[i-1] == '\t') {
		i--
	}
	return i
}

// highlightValue highlights a TOML value starting at position i.
func highlightValue(data []byte, i, n int, buf *strings.Builder, styles *style.TOML) int {
	if i >= n || data[i] == '\n' {
		return i
	}

	c := data[i]

	switch {
	case c == '"' || c == '\'':
		start := i
		i = scanQuotedString(data, i, n)
		printer.EmitStyled(buf, string(data[start:i]), styles.String)

	case c == 't' && i+4 <= n && string(data[i:i+4]) == "true":
		printer.EmitStyled(buf, "true", styles.BoolTrue)
		i += 4

	case c == 'f' && i+5 <= n && string(data[i:i+5]) == "false":
		printer.EmitStyled(buf, "false", styles.BoolFalse)
		i += 5

	case c == '[':
		i = highlightArray(data, i, n, buf, styles)

	case c == '{':
		i = highlightInlineTable(data, i, n, buf, styles)

	case isDigit(c) || c == '+' || c == '-' || c == 'i' || c == 'n':
		start := i
		i = scanBareValue(data, i, n)
		val := string(data[start:i])
		st := classifyNumber(val, styles)
		printer.EmitStyled(buf, val, st)

	default:
		end := scanToEOL(data, i)
		buf.Write(data[i:end])
		i = end
	}

	// Trailing whitespace + inline comment.
	for i < n && (data[i] == ' ' || data[i] == '\t') {
		buf.WriteByte(data[i])
		i++
	}
	if i < n && data[i] == '#' {
		end := scanToEOL(data, i)
		printer.EmitStyled(buf, string(data[i:end]), styles.Comment)
		i = end
	}

	return i
}

// highlightArray highlights a TOML array [...].
func highlightArray(data []byte, i, n int, buf *strings.Builder, styles *style.TOML) int {
	printer.EmitStyled(buf, "[", styles.Punctuation)
	i++ // skip '['

	for i < n {
		// Skip whitespace and newlines.
		for i < n && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
			buf.WriteByte(data[i])
			i++
		}
		if i >= n {
			break
		}
		if data[i] == ']' {
			printer.EmitStyled(buf, "]", styles.Punctuation)
			i++
			return i
		}
		if data[i] == ',' {
			printer.EmitStyled(buf, ",", styles.Punctuation)
			i++
			continue
		}
		if data[i] == '#' {
			end := scanToEOL(data, i)
			printer.EmitStyled(buf, string(data[i:end]), styles.Comment)
			i = end
			continue
		}
		i = highlightValue(data, i, n, buf, styles)
	}
	return i
}

// highlightInlineTable highlights a TOML inline table {...}.
func highlightInlineTable(data []byte, i, n int, buf *strings.Builder, styles *style.TOML) int {
	printer.EmitStyled(buf, "{", styles.Punctuation)
	i++ // skip '{'

	for i < n {
		for i < n && (data[i] == ' ' || data[i] == '\t') {
			buf.WriteByte(data[i])
			i++
		}
		if i >= n {
			break
		}
		if data[i] == '}' {
			printer.EmitStyled(buf, "}", styles.Punctuation)
			i++
			return i
		}
		if data[i] == ',' {
			printer.EmitStyled(buf, ",", styles.Punctuation)
			i++
			continue
		}
		// Key = value inside inline table.
		i = highlightKeyValue(data, i, n, buf, styles)
	}
	return i
}

// scanQuotedString scans a basic or literal string (including multi-line).
func scanQuotedString(data []byte, i, n int) int {
	quote := data[i]

	// Multi-line: """ or '''
	const tripleQuoteLen = 3
	if i+2 < n && data[i+1] == quote && data[i+2] == quote {
		i += tripleQuoteLen
		for i < n {
			if data[i] == '\\' && quote == '"' {
				i += 2 // skip escape in basic strings
				if i > n {
					i = n
				}
				continue
			}
			if data[i] == quote && i+2 < n && data[i+1] == quote && data[i+2] == quote {
				return i + tripleQuoteLen
			}
			i++
		}
		return i
	}

	// Single-line.
	i++ // skip opening quote
	for i < n {
		if data[i] == '\\' && quote == '"' {
			i += 2 // skip escape in basic strings
			if i > n {
				i = n
			}
			continue
		}
		if data[i] == quote {
			return i + 1
		}
		if data[i] == '\n' {
			return i // unterminated
		}
		i++
	}
	return i
}

// scanBareValue scans a bare value (number, date, inf, nan) until a delimiter.
func scanBareValue(data []byte, i, n int) int {
	for i < n {
		c := data[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' ||
			c == ',' || c == ']' || c == '}' || c == '#' {
			break
		}
		i++
	}
	return i
}

// scanToEOL returns the index past the end of the current line (including \n).
func scanToEOL(data []byte, i int) int {
	for i < len(data) && data[i] != '\n' {
		i++
	}
	if i < len(data) {
		i++ // include the \n
	}
	return i
}

// classifyNumber determines if a bare value is an integer, float, or datetime
// and returns the appropriate style.
func classifyNumber(val string, styles *style.TOML) *lipgloss.Style {
	if val == "inf" || val == "+inf" || val == "-inf" ||
		val == "nan" || val == "+nan" || val == "-nan" {
		return styles.Float
	}

	// DateTime heuristic: YYYY-MM-DD (len >= 10, dash at pos 4)
	// or HH:MM (len >= 5, colon at pos 2).
	const (
		minDateLen = 10 // YYYY-MM-DD
		dateDash   = 4  // position of first dash
		minTimeLen = 5  // HH:MM
		timeColon  = 2  // position of first colon
	)
	if len(val) >= minDateLen && val[dateDash] == '-' {
		return styles.DateTime
	}
	if len(val) >= minTimeLen && val[timeColon] == ':' {
		return styles.DateTime
	}

	// Float: contains '.', 'e', or 'E' (but not hex 0x which also has letters).
	if !strings.HasPrefix(val, "0x") && !strings.HasPrefix(val, "0o") &&
		!strings.HasPrefix(val, "0b") {
		if strings.ContainsAny(val, ".eE") {
			return styles.Float
		}
	}

	return styles.Integer
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
