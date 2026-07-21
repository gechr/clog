// Package json provides JSON syntax highlighting for clog.
package json

import (
	"bytes"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer"
	"github.com/gechr/clog/style"
)

// JSON token bytes.
const (
	tokenLBrace    = '{'
	tokenRBrace    = '}'
	tokenLBracket  = '['
	tokenRBracket  = ']'
	tokenColon     = ':'
	tokenComma     = ','
	tokenQuote     = '"'
	tokenBackslash = '\\'
	tokenMinus     = '-'
)

// Highlight applies syntax highlighting to s using the provided styles.
// By default inter-token whitespace is stripped, flattening pretty-printed
// JSON to a single line. Set [style.JSON.Indent] to a non-empty string to
// pretty-print with indentation instead. Returns s unchanged when styles is nil.
//
// The scanner is defensive: on any unexpected byte the remaining input is
// emitted unstyled rather than panicking.
func Highlight(s string, styles *style.JSON) string {
	if styles == nil {
		return s
	}

	if styles.Mode == style.JSONModeFlat {
		return renderFlat(s, styles)
	}

	var buf strings.Builder
	buf.Grow(len(s))

	data := []byte(s)
	n := len(data)
	i := 0

	// context stack: tokenLBrace = inside object, tokenLBracket = inside array
	const stackInitCap = 8
	stack := make([]byte, 0, stackInitCap)
	expectKey := false
	hjson := styles.Mode == style.JSONModeHuman
	indent := styles.Indent
	preserve := styles.PreserveFormat
	depth := 0
	justOpened := false // true immediately after { or [ (for empty container detection)

	for i < n {
		c := data[i]

		// Whitespace handling: preserve passes through as-is; otherwise strip
		// (indentation is rebuilt from scratch when indent is non-empty).
		if isSpace(c) {
			if preserve {
				buf.WriteByte(c)
			}
			i++
			continue
		}

		// Emit newline + indent for the first value after an opening container.
		// Skipped for } and ] so empty containers ({}, []) stay compact.
		if indent != "" && justOpened && c != tokenRBrace && c != tokenRBracket {
			buf.WriteByte('\n')
			buf.WriteString(strings.Repeat(indent, depth))
			justOpened = false
		}

		switch {
		case c == tokenLBrace:
			if indent == "" && len(stack) > 0 && styles.Spacing&style.JSONSpacingBeforeObject != 0 {
				buf.WriteByte(' ')
			}
			braceStyle := styles.Brace
			if len(stack) == 0 && styles.BraceRoot != nil {
				braceStyle = styles.BraceRoot
			}
			printer.EmitStyled(&buf, "{", braceStyle)
			stack = append(stack, tokenLBrace)
			depth++
			justOpened = true
			expectKey = true
			i++

		case c == tokenRBrace:
			if depth > 0 {
				depth--
			}
			if indent != "" && !justOpened && len(stack) > 0 {
				buf.WriteByte('\n')
				buf.WriteString(strings.Repeat(indent, depth))
			}
			justOpened = false
			braceStyle := styles.Brace
			if len(stack) == 1 && styles.BraceRoot != nil {
				braceStyle = styles.BraceRoot
			}
			printer.EmitStyled(&buf, "}", braceStyle)
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			expectKey = false
			i++

		case c == tokenLBracket:
			if indent == "" && len(stack) > 0 && styles.Spacing&style.JSONSpacingBeforeArray != 0 {
				buf.WriteByte(' ')
			}
			bracketStyle := styles.Bracket
			if len(stack) == 0 && styles.BracketRoot != nil {
				bracketStyle = styles.BracketRoot
			}
			printer.EmitStyled(&buf, "[", bracketStyle)
			stack = append(stack, tokenLBracket)
			depth++
			justOpened = true
			i++

		case c == tokenRBracket:
			if depth > 0 {
				depth--
			}
			if indent != "" && !justOpened && len(stack) > 0 {
				buf.WriteByte('\n')
				buf.WriteString(strings.Repeat(indent, depth))
			}
			justOpened = false
			bracketStyle := styles.Bracket
			if len(stack) == 1 && styles.BracketRoot != nil {
				bracketStyle = styles.BracketRoot
			}
			printer.EmitStyled(&buf, "]", bracketStyle)
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			i++

		case c == tokenColon:
			printer.EmitStyled(&buf, ":", styles.Colon)
			if indent != "" || styles.Spacing&style.JSONSpacingAfterColon != 0 {
				buf.WriteByte(' ')
			}
			expectKey = false
			i++

		case c == tokenComma:
			if !styles.OmitCommas {
				printer.EmitStyled(&buf, ",", styles.Comma)
			}
			if indent != "" {
				buf.WriteByte('\n')
				buf.WriteString(strings.Repeat(indent, depth))
			} else if styles.Spacing&style.JSONSpacingAfterComma != 0 {
				buf.WriteByte(' ')
			}
			if len(stack) > 0 && stack[len(stack)-1] == tokenLBrace {
				expectKey = true
			}
			i++

		case c == tokenQuote:
			j := scanString(data, i)
			raw := string(data[i:j])
			text, style := resolveStringToken(raw, expectKey, hjson, styles)
			printer.EmitStyled(&buf, text, style)
			if expectKey {
				expectKey = false
			}
			i = j

		case c == 't':
			if i+4 <= n && data[i+1] == 'r' && data[i+2] == 'u' && data[i+3] == 'e' {
				printer.EmitStyled(&buf, "true", styles.BoolTrue)
				i += 4
			} else {
				buf.Write(data[i:])
				return buf.String()
			}

		case c == 'f':
			if i+5 <= n && data[i+1] == 'a' && data[i+2] == 'l' && data[i+3] == 's' &&
				data[i+4] == 'e' {
				printer.EmitStyled(&buf, "false", styles.BoolFalse)
				i += 5
			} else {
				buf.Write(data[i:])
				return buf.String()
			}

		case c == 'n':
			if i+4 <= n && data[i+1] == 'u' && data[i+2] == 'l' && data[i+3] == 'l' {
				printer.EmitStyled(&buf, "null", styles.Null)
				i += 4
			} else {
				buf.Write(data[i:])
				return buf.String()
			}

		case c == tokenMinus || (isDigit(c)):
			j := i
			if data[j] == '-' {
				j++
			}
			for j < n && isDigit(data[j]) {
				j++
			}
			if j < n && data[j] == '.' {
				j++
				for j < n && isDigit(data[j]) {
					j++
				}
			}
			if j < n && (data[j] == 'e' || data[j] == 'E') {
				j++
				if j < n && (data[j] == '+' || data[j] == '-') {
					j++
				}
				for j < n && isDigit(data[j]) {
					j++
				}
			}
			printer.EmitStyled(
				&buf,
				string(data[i:j]),
				resolveNumberStyle(string(data[i:j]), styles),
			)
			i = j

		default:
			// unexpected byte: emit remaining input unstyled
			buf.Write(data[i:])
			return buf.String()
		}
	}

	return buf.String()
}

// resolveStringToken returns the text and style to use for a quoted JSON string
// token. When hjson is true, quotes are stripped if the HJSON spec permits it.
func resolveStringToken(
	raw string,
	isKey, hjson bool,
	styles *style.JSON,
) (string, *lipgloss.Style) {
	if isKey {
		text, unquoted := hjsonUnquoteKey(raw, hjson)
		if unquoted {
			return text, styles.Key
		}
		return raw, styles.Key
	}
	text, unquoted := hjsonUnquoteValue(raw, hjson)
	if unquoted {
		return text, styles.String
	}
	return raw, styles.String
}

// renderFlat flattens nested object keys with dot notation and renders
// the result using human-mode quoting. Arrays are rendered intact.
// Non-object root values fall back to human-mode rendering.
func renderFlat(s string, styles *style.JSON) string {
	data := bytes.TrimSpace([]byte(s))
	if len(data) == 0 || data[0] != '{' {
		// non-object root: human-mode rendering without flattening
		humanStyles := *styles
		humanStyles.Mode = style.JSONModeHuman
		return Highlight(s, &humanStyles)
	}

	pairs := collectFlatPairs(data, "")

	// value styles: human-mode, no root brace/bracket distinction
	// (since values are rendered as fragments, not root documents)
	valueStyles := *styles
	valueStyles.Mode = style.JSONModeHuman
	valueStyles.BraceRoot = nil
	valueStyles.BracketRoot = nil

	var buf strings.Builder
	buf.Grow(len(s))

	braceStyle := styles.BraceRoot
	if braceStyle == nil {
		braceStyle = styles.Brace
	}
	printer.EmitStyled(&buf, "{", braceStyle)

	for i, p := range pairs {
		if i > 0 {
			if !styles.OmitCommas {
				printer.EmitStyled(&buf, ",", styles.Comma)
			}
			if styles.Spacing&style.JSONSpacingAfterComma != 0 {
				buf.WriteByte(' ')
			}
		}
		printer.EmitStyled(&buf, p.Key, styles.Key)
		printer.EmitStyled(&buf, ":", styles.Colon)
		if styles.Spacing&style.JSONSpacingAfterColon != 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(Highlight(string(p.Value), &valueStyles))
	}

	printer.EmitStyled(&buf, "}", braceStyle)
	return buf.String()
}

// flatPair holds a dotted key and its raw JSON value extracted during flattening.
type flatPair struct {
	Key   string
	Value []byte // scalar, null, bool, number, or array - never an object
}

// collectFlatPairs walks a JSON object and returns (dotted_key, raw_value)
// pairs. Nested objects are recursed into; arrays and scalars are kept as-is.
func collectFlatPairs(data []byte, prefix string) []flatPair {
	n := len(data)
	i := 0

	for i < n && isSpace(data[i]) {
		i++
	}
	if i >= n || data[i] != '{' {
		return nil
	}
	i++ // skip '{'

	var pairs []flatPair

	for i < n {
		for i < n && isSpace(data[i]) {
			i++
		}
		if i >= n || data[i] == '}' {
			break
		}
		if data[i] == ',' {
			i++
			continue
		}
		if data[i] != '"' {
			break // malformed
		}

		// scan key string
		j := scanString(data, i)
		rawKey := string(data[i+1 : j-1]) // unescaped content between quotes
		i = j

		// build dotted key path
		fullKey := rawKey
		if prefix != "" {
			fullKey = prefix + "." + rawKey
		}

		// skip whitespace + colon
		for i < n && isSpace(data[i]) {
			i++
		}
		if i >= n || data[i] != ':' {
			break
		}
		i++
		for i < n && isSpace(data[i]) {
			i++
		}

		// scan value extent
		valueStart := i
		i = scanValueEnd(data, i)
		rawValue := bytes.TrimSpace(data[valueStart:i])

		// recurse into nested objects; keep everything else as a leaf
		if len(rawValue) > 0 && rawValue[0] == '{' {
			pairs = append(pairs, collectFlatPairs(rawValue, fullKey)...)
		} else {
			pairs = append(pairs, flatPair{Key: fullKey, Value: rawValue})
		}
	}

	return pairs
}

// scanString returns the index one past the closing quote of the JSON string
// starting at i in data (i must point at the opening quote), or len(data)
// when the string is unterminated. Backslash escapes are honored.
func scanString(data []byte, i int) int {
	n := len(data)
	j := i + 1
	for j < n {
		if data[j] == tokenBackslash {
			j += 2 // skip escaped character
			if j >= n {
				break
			}
			continue
		}
		if data[j] == tokenQuote {
			j++
			break
		}
		j++
	}
	return min(j, n)
}

// scanValueEnd returns the index one past the end of the JSON value
// starting at i in data. Handles strings, objects, arrays, and bare literals.
func scanValueEnd(data []byte, i int) int {
	n := len(data)
	if i >= n {
		return i
	}
	switch data[i] {
	case '"':
		return scanString(data, i)

	case '{', '[':
		openByte := data[i]
		closeByte := byte('}')
		if openByte == '[' {
			closeByte = ']'
		}
		depth := 0
		for i < n {
			c := data[i]
			if c == '"' {
				i = scanString(data, i)
				continue
			}
			switch c {
			case openByte:
				depth++
			case closeByte:
				depth--
				if depth == 0 {
					return i + 1
				}
			}
			i++
		}
		if i > n {
			i = n
		}
		return i

	default:
		// number, true, false, null
		for i < n && data[i] != ',' && data[i] != '}' && data[i] != ']' && !isSpace(data[i]) {
			i++
		}
		return i
	}
}

// isDigit reports whether c is an ASCII digit.
func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// isSpace reports whether c is a JSON whitespace character.
func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// resolveNumberStyle resolves the effective style for a JSON number token.
// Sign-based styles (Negative/Positive/Zero) take priority over type-based
// (Float/Integer), with Number as the ultimate fallback.
// Fallback chains:
//   - negative:  NumberNegative -> Number
//   - zero:      NumberZero -> NumberPositive -> Number
//   - positive:  NumberPositive -> Number
//   - float:     NumberFloat -> Number  (when no sign style matched)
//   - integer:   NumberInteger -> Number (when no sign style matched)
func resolveNumberStyle(val string, styles *style.JSON) *lipgloss.Style {
	isNeg := len(val) > 0 && val[0] == '-'
	isFloat := strings.ContainsAny(val, ".eE")

	f, err := strconv.ParseFloat(val, 64)
	isZero := err == nil && f == 0

	// Sign-based resolution (higher priority).
	switch {
	case isZero:
		if styles.NumberZero != nil {
			return styles.NumberZero
		}
		if styles.NumberPositive != nil {
			return styles.NumberPositive
		}
	case isNeg:
		if styles.NumberNegative != nil {
			return styles.NumberNegative
		}
	default:
		if styles.NumberPositive != nil {
			return styles.NumberPositive
		}
	}

	// Type-based resolution.
	if isFloat {
		if styles.NumberFloat != nil {
			return styles.NumberFloat
		}
	} else {
		if styles.NumberInteger != nil {
			return styles.NumberInteger
		}
	}

	return styles.Number
}

// hjsonUnquoteKey returns the unquoted key and true if hjson is enabled and
// the key doesn't require quoting per the HJSON spec (needsEscapeName rules:
// must not contain ,{}[]\s:#"' or the sequences // or /*).
// Empty keys and keys with escape sequences are always kept quoted.
func hjsonUnquoteKey(raw string, hjson bool) (string, bool) {
	if !hjson || len(raw) < 2 {
		return raw, false
	}
	s := raw[1 : len(raw)-1]
	if len(s) == 0 {
		return raw, false
	}
	if strings.IndexByte(s, '\\') >= 0 {
		return raw, false
	}
	for i, c := range s {
		switch c {
		case ',', '{', '[', '}', ']', ':', '#', '"', '\'':
			return raw, false
		case '/':
			if i+1 < len(s) && (s[i+1] == '/' || s[i+1] == '*') {
				return raw, false
			}
		default:
			if c <= ' ' {
				return raw, false
			}
		}
	}
	return s, true
}

// hjsonUnquoteValue returns the unquoted value and true if hjson is enabled
// and the string value doesn't require quoting per the HJSON spec:
//   - not empty (empty string must remain "")
//   - no escape sequences
//   - doesn't start with whitespace, ", ', #, {, }, [, ], :, ,, //, or /*
//   - doesn't end with whitespace
//   - contains no control characters
//   - not ambiguous as a keyword (true/false/null) or number
func hjsonUnquoteValue(raw string, hjson bool) (string, bool) {
	if !hjson || len(raw) < 2 {
		return raw, false
	}
	s := raw[1 : len(raw)-1]
	if len(s) == 0 {
		return raw, false // empty string must remain ""
	}
	if strings.IndexByte(s, '\\') >= 0 {
		return raw, false
	}
	// first-character checks (needsQuotes)
	switch s[0] {
	case ' ', '\t', '"', '\'', '#', '{', '}', '[', ']', ':', ',':
		return raw, false
	case '/':
		if len(s) > 1 && (s[1] == '/' || s[1] == '*') {
			return raw, false
		}
	}
	// last character must not be whitespace
	if isSpace(s[len(s)-1]) {
		return raw, false
	}
	// no control characters within the value
	for _, c := range s {
		if c < ' ' {
			return raw, false
		}
	}
	// ambiguous as keyword
	for _, kw := range []string{"true", "false", "null"} {
		if strings.HasPrefix(s, kw) {
			rest := s[len(kw):]
			if rest == "" || rest[0] == ' ' || rest[0] == '\t' ||
				rest[0] == ',' || rest[0] == ']' || rest[0] == '}' ||
				rest[0] == '#' || rest[0] == '/' {
				return raw, false
			}
		}
	}
	// ambiguous as number: starts with digit or '-' followed by digit
	if isDigit(s[0]) {
		return raw, false
	}
	if s[0] == '-' && len(s) > 1 && isDigit(s[1]) {
		return raw, false
	}
	return s, true
}
