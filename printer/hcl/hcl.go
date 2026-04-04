// Package hcl provides HCL syntax highlighting for clog.
package hcl

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer"
	"github.com/gechr/clog/style"
	gohcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// Highlight applies syntax highlighting to s using the provided styles.
// Original formatting is preserved. Returns s unchanged when styles is nil
// or lexing fails.
func Highlight(s string, styles *style.HCL) string {
	if styles == nil || s == "" {
		return s
	}

	tokens, diags := hclsyntax.LexConfig(
		[]byte(s), "", gohcl.Pos{Line: 1, Column: 1},
	)
	if diags.HasErrors() {
		return s
	}

	var buf strings.Builder
	buf.Grow(len(s) * 2) //nolint:mnd // extra capacity for ANSI escapes

	pos := 0   // current byte offset into s
	depth := 0 // brace nesting depth

	for i, tok := range tokens {
		start := tok.Range.Start.Byte
		end := tok.Range.End.Byte

		// Emit any gap (whitespace) between the previous token and this one.
		if start > pos {
			buf.WriteString(s[pos:start])
		}

		// Track brace depth for nested-key detection.
		if tok.Type == hclsyntax.TokenOBrace {
			depth++
		} else if tok.Type == hclsyntax.TokenCBrace && depth > 0 {
			depth--
		}

		st := resolveStyle(tok, tokens, i, depth, styles)
		text := string(tok.Bytes)
		printer.EmitStyled(&buf, text, st)

		pos = end
	}

	// Emit any trailing bytes after the last token.
	if pos < len(s) {
		buf.WriteString(s[pos:])
	}

	return buf.String()
}

// nextNonNewline returns the next token that isn't a newline, or nil.
func nextNonNewline(tokens hclsyntax.Tokens, i int) *hclsyntax.Token {
	for j := i + 1; j < len(tokens); j++ {
		if tokens[j].Type != hclsyntax.TokenNewline {
			return &tokens[j]
		}
	}
	return nil
}

// resolveStyle returns the appropriate style for a token based on its type
// and surrounding context.
func resolveStyle(
	tok hclsyntax.Token,
	tokens hclsyntax.Tokens,
	i, depth int,
	styles *style.HCL,
) *lipgloss.Style {
	switch tok.Type {
	case hclsyntax.TokenIdent:
		return resolveIdentStyle(tok, tokens, i, depth, styles)
	case hclsyntax.TokenNumberLit:
		return styles.Number
	case hclsyntax.TokenOQuote, hclsyntax.TokenCQuote,
		hclsyntax.TokenOHeredoc, hclsyntax.TokenCHeredoc,
		hclsyntax.TokenQuotedLit, hclsyntax.TokenStringLit:
		return styles.String
	case hclsyntax.TokenComment:
		return styles.Comment
	case hclsyntax.TokenEqual, hclsyntax.TokenOBrace, hclsyntax.TokenCBrace,
		hclsyntax.TokenOBrack, hclsyntax.TokenCBrack,
		hclsyntax.TokenOParen, hclsyntax.TokenCParen,
		hclsyntax.TokenComma, hclsyntax.TokenDot, hclsyntax.TokenEllipsis,
		hclsyntax.TokenFatArrow, hclsyntax.TokenQuestion, hclsyntax.TokenColon,
		hclsyntax.TokenDoubleColon,
		hclsyntax.TokenStar, hclsyntax.TokenSlash, hclsyntax.TokenPlus,
		hclsyntax.TokenMinus, hclsyntax.TokenPercent,
		hclsyntax.TokenEqualOp, hclsyntax.TokenNotEqual,
		hclsyntax.TokenLessThan, hclsyntax.TokenLessThanEq,
		hclsyntax.TokenGreaterThan, hclsyntax.TokenGreaterThanEq,
		hclsyntax.TokenAnd, hclsyntax.TokenOr, hclsyntax.TokenBang,
		hclsyntax.TokenBitwiseAnd, hclsyntax.TokenBitwiseOr,
		hclsyntax.TokenBitwiseNot, hclsyntax.TokenBitwiseXor,
		hclsyntax.TokenStarStar, hclsyntax.TokenSemicolon:
		return styles.Punctuation
	case hclsyntax.TokenTemplateInterp, hclsyntax.TokenTemplateControl,
		hclsyntax.TokenTemplateSeqEnd,
		hclsyntax.TokenNewline, hclsyntax.TokenEOF,
		hclsyntax.TokenApostrophe, hclsyntax.TokenBacktick,
		hclsyntax.TokenTabs, hclsyntax.TokenInvalid,
		hclsyntax.TokenBadUTF8, hclsyntax.TokenQuotedNewline,
		hclsyntax.TokenNil:
		return nil
	}
	return nil
}

// resolveIdentStyle classifies an identifier token by context:
//   - followed by '=' → attribute key
//   - followed by '"' or '{' → block type
//   - value "true"/"false" → bool
//   - value "null" → null
func resolveIdentStyle(
	tok hclsyntax.Token,
	tokens hclsyntax.Tokens,
	i, depth int,
	styles *style.HCL,
) *lipgloss.Style {
	val := string(tok.Bytes)

	// Bool and null literals.
	switch val {
	case "true":
		return styles.BoolTrue
	case "false":
		return styles.BoolFalse
	case "null":
		return styles.Null
	}

	// Look at the next meaningful token for context.
	next := nextNonNewline(tokens, i)
	if next == nil {
		return nil
	}

	switch next.Type {
	case hclsyntax.TokenEqual:
		if depth >= 2 && styles.NestedKey != nil {
			return styles.NestedKey
		}
		return styles.Key
	case hclsyntax.TokenOQuote, hclsyntax.TokenOBrace:
		return styles.BlockType
	case hclsyntax.TokenCBrace, hclsyntax.TokenOBrack, hclsyntax.TokenCBrack,
		hclsyntax.TokenOParen, hclsyntax.TokenCParen,
		hclsyntax.TokenCQuote, hclsyntax.TokenOHeredoc, hclsyntax.TokenCHeredoc,
		hclsyntax.TokenStar, hclsyntax.TokenSlash, hclsyntax.TokenPlus,
		hclsyntax.TokenMinus, hclsyntax.TokenPercent,
		hclsyntax.TokenEqualOp, hclsyntax.TokenNotEqual,
		hclsyntax.TokenLessThan, hclsyntax.TokenLessThanEq,
		hclsyntax.TokenGreaterThan, hclsyntax.TokenGreaterThanEq,
		hclsyntax.TokenAnd, hclsyntax.TokenOr, hclsyntax.TokenBang,
		hclsyntax.TokenDot, hclsyntax.TokenComma, hclsyntax.TokenDoubleColon,
		hclsyntax.TokenEllipsis, hclsyntax.TokenFatArrow,
		hclsyntax.TokenQuestion, hclsyntax.TokenColon,
		hclsyntax.TokenTemplateInterp, hclsyntax.TokenTemplateControl,
		hclsyntax.TokenTemplateSeqEnd,
		hclsyntax.TokenQuotedLit, hclsyntax.TokenStringLit,
		hclsyntax.TokenNumberLit, hclsyntax.TokenIdent,
		hclsyntax.TokenComment, hclsyntax.TokenNewline, hclsyntax.TokenEOF,
		hclsyntax.TokenBitwiseAnd, hclsyntax.TokenBitwiseOr,
		hclsyntax.TokenBitwiseNot, hclsyntax.TokenBitwiseXor,
		hclsyntax.TokenStarStar, hclsyntax.TokenApostrophe,
		hclsyntax.TokenBacktick, hclsyntax.TokenSemicolon,
		hclsyntax.TokenTabs, hclsyntax.TokenInvalid,
		hclsyntax.TokenBadUTF8, hclsyntax.TokenQuotedNewline,
		hclsyntax.TokenNil:
		return nil
	}

	return nil
}
