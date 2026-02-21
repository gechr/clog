package clog

import (
	"bytes"
	"strconv"
	"strings"
)

// highlightJSON applies syntax highlighting to s using the provided styles.
// Inter-token whitespace is stripped, flattening pretty-printed JSON to a
// single line. Returns s unchanged when styles is nil.
//
// The scanner is defensive: on any unexpected byte the remaining input is
// emitted unstyled rather than panicking.
func highlightJSON(s string, styles *JSONStyles) string {
	if styles == nil {
		return s
	}

	if styles.Mode == JSONModeFlat {
		return renderFlatJSON(s, styles)
	}

	var buf strings.Builder
	buf.Grow(len(s))

	data := []byte(s)
	n := len(data)
	i := 0

	// context stack: '{' = inside object, '[' = inside array
	const stackInitCap = 8
	stack := make([]byte, 0, stackInitCap)
	expectKey := false
	hjson := styles.Mode == JSONModeHuman

	for i < n {
		c := data[i]

		// strip inter-token whitespace to flatten pretty-printed JSON
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			i++
			continue
		}

		switch {
		case c == '{':
			if len(stack) > 0 && styles.Spacing&JSONSpacingBeforeObject != 0 {
				buf.WriteByte(' ')
			}
			braceStyle := styles.Brace
			if len(stack) == 0 && styles.BraceRoot != nil {
				braceStyle = styles.BraceRoot
			}
			emitStyled(&buf, "{", braceStyle)
			stack = append(stack, '{')
			expectKey = true
			i++

		case c == '}':
			braceStyle := styles.Brace
			if len(stack) == 1 && styles.BraceRoot != nil {
				braceStyle = styles.BraceRoot
			}
			emitStyled(&buf, "}", braceStyle)
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			expectKey = false
			i++

		case c == '[':
			if len(stack) > 0 && styles.Spacing&JSONSpacingBeforeArray != 0 {
				buf.WriteByte(' ')
			}
			bracketStyle := styles.Bracket
			if len(stack) == 0 && styles.BracketRoot != nil {
				bracketStyle = styles.BracketRoot
			}
			emitStyled(&buf, "[", bracketStyle)
			stack = append(stack, '[')
			i++

		case c == ']':
			bracketStyle := styles.Bracket
			if len(stack) == 1 && styles.BracketRoot != nil {
				bracketStyle = styles.BracketRoot
			}
			emitStyled(&buf, "]", bracketStyle)
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
			i++

		case c == ':':
			emitStyled(&buf, ":", styles.Colon)
			if styles.Spacing&JSONSpacingAfterColon != 0 {
				buf.WriteByte(' ')
			}
			expectKey = false
			i++

		case c == ',':
			if !styles.OmitCommas {
				emitStyled(&buf, ",", styles.Comma)
			}
			if styles.Spacing&JSONSpacingAfterComma != 0 {
				buf.WriteByte(' ')
			}
			if len(stack) > 0 && stack[len(stack)-1] == '{' {
				expectKey = true
			}
			i++

		case c == '"':
			j := i + 1
			for j < n {
				if data[j] == '\\' {
					j += 2 // skip escaped character
					if j >= n {
						break
					}
					continue
				}
				if data[j] == '"' {
					j++
					break
				}
				j++
			}
			if j > n {
				j = n
			}
			raw := string(data[i:j])
			text, style := resolveStringToken(raw, expectKey, hjson, styles)
			emitStyled(&buf, text, style)
			if expectKey {
				expectKey = false
			}
			i = j

		case c == 't':
			if i+4 <= n && data[i+1] == 'r' && data[i+2] == 'u' && data[i+3] == 'e' {
				emitStyled(&buf, "true", styles.BoolTrue)
				i += 4
			} else {
				buf.Write(data[i:])
				return buf.String()
			}

		case c == 'f':
			if i+5 <= n && data[i+1] == 'a' && data[i+2] == 'l' && data[i+3] == 's' &&
				data[i+4] == 'e' {
				emitStyled(&buf, "false", styles.BoolFalse)
				i += 5
			} else {
				buf.Write(data[i:])
				return buf.String()
			}

		case c == 'n':
			if i+4 <= n && data[i+1] == 'u' && data[i+2] == 'l' && data[i+3] == 'l' {
				emitStyled(&buf, "null", styles.Null)
				i += 4
			} else {
				buf.Write(data[i:])
				return buf.String()
			}

		case c == '-' || (c >= '0' && c <= '9'):
			j := i
			if data[j] == '-' {
				j++
			}
			for j < n && data[j] >= '0' && data[j] <= '9' {
				j++
			}
			if j < n && data[j] == '.' {
				j++
				for j < n && data[j] >= '0' && data[j] <= '9' {
					j++
				}
			}
			if j < n && (data[j] == 'e' || data[j] == 'E') {
				j++
				if j < n && (data[j] == '+' || data[j] == '-') {
					j++
				}
				for j < n && data[j] >= '0' && data[j] <= '9' {
					j++
				}
			}
			emitStyled(&buf, string(data[i:j]), resolveNumberStyle(string(data[i:j]), styles))
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
func resolveStringToken(raw string, isKey, hjson bool, styles *JSONStyles) (string, Style) {
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

// renderFlatJSON flattens nested object keys with dot notation and renders
// the result using human-mode quoting. Arrays are rendered intact.
// Non-object root values fall back to human-mode rendering.
func renderFlatJSON(s string, styles *JSONStyles) string {
	data := bytes.TrimSpace([]byte(s))
	if len(data) == 0 || data[0] != '{' {
		// non-object root: human-mode rendering without flattening
		humanStyles := *styles
		humanStyles.Mode = JSONModeHuman
		return highlightJSON(s, &humanStyles)
	}

	pairs := collectFlatPairs(data, "")

	// value styles: human-mode, no root brace/bracket distinction
	// (since values are rendered as fragments, not root documents)
	valueStyles := *styles
	valueStyles.Mode = JSONModeHuman
	valueStyles.BraceRoot = nil
	valueStyles.BracketRoot = nil

	var buf strings.Builder
	buf.Grow(len(s))

	braceStyle := styles.BraceRoot
	if braceStyle == nil {
		braceStyle = styles.Brace
	}
	emitStyled(&buf, "{", braceStyle)

	for i, p := range pairs {
		if i > 0 {
			if !styles.OmitCommas {
				emitStyled(&buf, ",", styles.Comma)
			}
			if styles.Spacing&JSONSpacingAfterComma != 0 {
				buf.WriteByte(' ')
			}
		}
		emitStyled(&buf, p.key, styles.Key)
		emitStyled(&buf, ":", styles.Colon)
		if styles.Spacing&JSONSpacingAfterColon != 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(highlightJSON(string(p.value), &valueStyles))
	}

	emitStyled(&buf, "}", braceStyle)
	return buf.String()
}

// flatPair holds a dotted key and its raw JSON value extracted during flattening.
type flatPair struct {
	key   string
	value []byte // scalar, null, bool, number, or array — never an object
}

// collectFlatPairs walks a JSON object and returns (dotted_key, raw_value)
// pairs. Nested objects are recursed into; arrays and scalars are kept as-is.
func collectFlatPairs(data []byte, prefix string) []flatPair {
	n := len(data)
	i := 0

	for i < n && isJSONSpace(data[i]) {
		i++
	}
	if i >= n || data[i] != '{' {
		return nil
	}
	i++ // skip '{'

	var pairs []flatPair

	for i < n {
		for i < n && isJSONSpace(data[i]) {
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
		j := i + 1
		for j < n {
			if data[j] == '\\' {
				j += 2
				if j >= n {
					break
				}
				continue
			}
			if data[j] == '"' {
				j++
				break
			}
			j++
		}
		if j > n {
			j = n
		}
		rawKey := string(data[i+1 : j-1]) // unescaped content between quotes
		i = j

		// build dotted key path
		fullKey := rawKey
		if prefix != "" {
			fullKey = prefix + "." + rawKey
		}

		// skip whitespace + colon
		for i < n && isJSONSpace(data[i]) {
			i++
		}
		if i >= n || data[i] != ':' {
			break
		}
		i++
		for i < n && isJSONSpace(data[i]) {
			i++
		}

		// scan value extent
		valueStart := i
		i = scanJSONValueEnd(data, i)
		rawValue := bytes.TrimSpace(data[valueStart:i])

		// recurse into nested objects; keep everything else as a leaf
		if len(rawValue) > 0 && rawValue[0] == '{' {
			pairs = append(pairs, collectFlatPairs(rawValue, fullKey)...)
		} else {
			pairs = append(pairs, flatPair{key: fullKey, value: rawValue})
		}
	}

	return pairs
}

// scanJSONValueEnd returns the index one past the end of the JSON value
// starting at i in data. Handles strings, objects, arrays, and bare literals.
func scanJSONValueEnd(data []byte, i int) int {
	n := len(data)
	if i >= n {
		return i
	}
	switch data[i] {
	case '"':
		i++
		for i < n {
			if data[i] == '\\' {
				i += 2
				if i >= n {
					break
				}
				continue
			}
			if data[i] == '"' {
				return i + 1
			}
			i++
		}
		if i > n {
			i = n
		}
		return i

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
				i++
				for i < n {
					if data[i] == '\\' {
						i += 2
						if i >= n {
							break
						}
						continue
					}
					if data[i] == '"' {
						i++
						break
					}
					i++
				}
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
		for i < n && data[i] != ',' && data[i] != '}' && data[i] != ']' && !isJSONSpace(data[i]) {
			i++
		}
		return i
	}
}

// isJSONSpace reports whether c is a JSON whitespace character.
func isJSONSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// emitStyled writes text to buf, applying style if non-nil.
func emitStyled(buf *strings.Builder, text string, style Style) {
	if style != nil {
		buf.WriteString(style.Render(text))
	} else {
		buf.WriteString(text)
	}
}

// resolveNumberStyle resolves the effective style for a JSON number token.
// Sign-based styles (Negative/Positive/Zero) take priority over type-based
// (Float/Integer), with Number as the ultimate fallback.
// Fallback chains:
//   - negative:  NumberNegative → Number
//   - zero:      NumberZero → NumberPositive → Number
//   - positive:  NumberPositive → Number
//   - float:     NumberFloat → Number  (when no sign style matched)
//   - integer:   NumberInteger → Number (when no sign style matched)
func resolveNumberStyle(val string, styles *JSONStyles) Style {
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
	if last := s[len(s)-1]; last == ' ' || last == '\t' || last == '\n' || last == '\r' {
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
	if s[0] >= '0' && s[0] <= '9' {
		return raw, false
	}
	if s[0] == '-' && len(s) > 1 && s[1] >= '0' && s[1] <= '9' {
		return raw, false
	}
	return s, true
}
