// Package yaml provides YAML syntax highlighting for clog.
package yaml

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer"
	"github.com/gechr/clog/style"
	"github.com/goccy/go-yaml/lexer"
	"github.com/goccy/go-yaml/token"
)

// Highlight applies syntax highlighting to s using the provided styles.
// Highlighting is best-effort: if the lexer does not round-trip the input
// exactly, s is returned unchanged to preserve the caller's formatting.
// Returns s unchanged when styles is nil.
func Highlight(s string, styles *style.YAML) string {
	if styles == nil {
		return s
	}

	tokens := lexer.Tokenize(s)
	if len(tokens) == 0 {
		return s
	}

	// Round-trip guard: verify the lexer faithfully reproduces the input
	// before applying styles. Compare unstyled Origins against the original.
	var plain strings.Builder
	plain.Grow(len(s))
	for _, tok := range tokens {
		plain.WriteString(tok.Origin)
	}
	if stripTrailingNewlines(plain.String()) != stripTrailingNewlines(s) {
		return s
	}

	var buf strings.Builder
	buf.Grow(len(s) * 2) //nolint:mnd // extra capacity for ANSI escapes

	for i, tok := range tokens {
		st := resolveStyle(tok, tokens, i, styles)
		printer.EmitStyled(&buf, tok.Origin, st)
	}

	return buf.String()
}

// stripTrailingNewlines removes trailing newline characters for round-trip
// comparison, since the lexer may drop a final trailing newline.
func stripTrailingNewlines(s string) string {
	return strings.TrimRight(s, "\n")
}

// isKeyType reports whether the token type can represent a mapping key.
func isKeyType(t token.Type) bool {
	switch t {
	case token.StringType, token.SingleQuoteType, token.DoubleQuoteType, token.MergeKeyType:
		return true
	case token.UnknownType, token.DocumentHeaderType, token.DocumentEndType,
		token.SequenceEntryType, token.MappingKeyType, token.MappingValueType,
		token.CollectEntryType,
		token.SequenceStartType, token.SequenceEndType,
		token.MappingStartType, token.MappingEndType,
		token.CommentType, token.AnchorType, token.AliasType, token.TagType,
		token.LiteralType, token.FoldedType,
		token.DirectiveType, token.SpaceType,
		token.NullType, token.ImplicitNullType,
		token.InfinityType, token.NanType,
		token.IntegerType, token.BinaryIntegerType, token.OctetIntegerType,
		token.HexIntegerType, token.FloatType,
		token.BoolType, token.InvalidType:
		return false
	}
	return false
}

// resolveStyle returns the appropriate style for a token based on its type
// and context (e.g. a string before ':' is a key, a string after '&' is an
// anchor name).
func resolveStyle(
	tok *token.Token,
	tokens token.Tokens,
	i int,
	styles *style.YAML,
) *lipgloss.Style {
	// Anchor/alias name: previous token is Anchor or Alias.
	if i > 0 {
		switch tokens[i-1].Type {
		case token.AnchorType:
			return styles.Anchor
		case token.AliasType:
			return styles.Alias
		case token.UnknownType, token.DocumentHeaderType, token.DocumentEndType,
			token.SequenceEntryType, token.MappingKeyType, token.MappingValueType,
			token.MergeKeyType, token.CollectEntryType,
			token.SequenceStartType, token.SequenceEndType,
			token.MappingStartType, token.MappingEndType,
			token.CommentType, token.TagType,
			token.LiteralType, token.FoldedType,
			token.SingleQuoteType, token.DoubleQuoteType,
			token.DirectiveType, token.SpaceType,
			token.NullType, token.ImplicitNullType,
			token.InfinityType, token.NanType,
			token.IntegerType, token.BinaryIntegerType, token.OctetIntegerType,
			token.HexIntegerType, token.FloatType,
			token.StringType, token.BoolType, token.InvalidType:
		}
	}

	// Key: any key-eligible type followed by MappingValue.
	if isKeyType(tok.Type) && i+1 < len(tokens) && tokens[i+1].Type == token.MappingValueType {
		return styles.Key
	}

	switch tok.Type {
	case token.AnchorType:
		return styles.Anchor
	case token.AliasType:
		return styles.Alias
	case token.BoolType:
		switch tok.Value {
		case "y", "Y", "yes", "Yes", "YES",
			"true", "True", "TRUE",
			"on", "On", "ON":
			return styles.BoolTrue
		default:
			return styles.BoolFalse
		}
	case token.CommentType:
		return styles.Comment
	case token.NullType, token.ImplicitNullType:
		return styles.Null
	case token.IntegerType, token.BinaryIntegerType, token.OctetIntegerType,
		token.HexIntegerType, token.FloatType, token.InfinityType, token.NanType:
		return styles.Number
	case token.StringType, token.SingleQuoteType, token.DoubleQuoteType,
		token.LiteralType, token.FoldedType:
		return styles.String
	case token.TagType:
		return styles.Tag
	case token.MappingValueType, token.MappingKeyType,
		token.SequenceEntryType,
		token.MappingStartType, token.MappingEndType,
		token.SequenceStartType, token.SequenceEndType,
		token.CollectEntryType,
		token.DocumentHeaderType, token.DocumentEndType:
		return styles.Punctuation
	case token.UnknownType, token.MergeKeyType,
		token.DirectiveType, token.SpaceType,
		token.InvalidType:
		return nil
	}
	return nil
}
