package style

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/theme"
)

// JSONSpacing is a bitmask controlling where spaces are inserted in JSON output.
type JSONSpacing uint

const (
	// JSONSpacingAfterColon inserts a space after each colon: {"key": "value"}.
	JSONSpacingAfterColon JSONSpacing = 1 << iota
	// JSONSpacingAfterComma inserts a space after each comma: {"a": 1, "b": 2}.
	JSONSpacingAfterComma
	// JSONSpacingBeforeObject inserts a space before a nested object value: {"key": {"n":1}}.
	JSONSpacingBeforeObject
	// JSONSpacingBeforeArray inserts a space before a nested array value: {"tags": ["a","b"]}.
	JSONSpacingBeforeArray
	// JSONSpacingAll enables all spacing options.
	JSONSpacingAll = JSONSpacingAfterColon | JSONSpacingAfterComma | JSONSpacingBeforeObject | JSONSpacingBeforeArray
)

// JSONMode controls how JSON is rendered.
type JSONMode int

const (
	// JSONModeJSON renders standard JSON (default).
	JSONModeJSON JSONMode = iota
	// JSONModeHuman renders in HJSON style: keys and simple string values are
	// unquoted, making output more readable at a glance.
	JSONModeHuman
	// JSONModeFlat flattens nested object keys using dot notation and renders
	// scalar values without unnecessary quotes. Arrays are kept intact.
	// Example: {"user":{"name":"alice"},"tags":["a","b"]}
	//       →  {user.name:alice,tags:[a,b]}
	JSONModeFlat
)

// JSON configures per-token lipgloss styles for JSON syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultJSON] as a starting point for customization.
type JSON struct {
	// Mode controls rendering behaviour.
	// JSONModeJSON (default) preserves standard JSON quoting.
	// JSONModeHuman strips quotes from identifier-like keys and simple string values.
	// JSONModeFlat flattens nested object keys with dot notation; arrays are kept intact.
	Mode JSONMode
	// Indent pretty-prints JSON with the given indentation string (e.g. "  "
	// or "\t"). Empty string (default) flattens to a single line.
	Indent string
	// PreserveFormat keeps original whitespace intact instead of stripping
	// it. When true, Indent is ignored.
	PreserveFormat bool
	// OmitCommas omits the comma between items. JSONSpacingAfterComma still
	// applies and can be used to keep a space separator: {"a":1 "b":2}.
	OmitCommas bool
	// Spacing controls where spaces are inserted. Zero (default) means no spaces.
	// Use JSONSpacingAll for {"key": "value", "n": 1} style output.
	Spacing JSONSpacing

	BoolFalse      *lipgloss.Style // false
	BoolTrue       *lipgloss.Style // true
	Key            *lipgloss.Style // Object keys
	Null           *lipgloss.Style // null
	Number         *lipgloss.Style // Numeric values - base fallback for all number sub-styles
	NumberFloat    *lipgloss.Style // Floating-point values; falls back to Number
	NumberInteger  *lipgloss.Style // Integer values; falls back to Number
	NumberNegative *lipgloss.Style // Negative numbers; falls back to Number
	NumberPositive *lipgloss.Style // Positive numbers (with or without explicit sign); falls back to Number
	NumberZero     *lipgloss.Style // Zero; falls back to NumberPositive, then Number
	String         *lipgloss.Style // String values

	Brace       *lipgloss.Style // { } (nested)
	BraceRoot   *lipgloss.Style // { } (outermost object; falls back to Brace if nil)
	Bracket     *lipgloss.Style // [ ] (nested)
	BracketRoot *lipgloss.Style // [ ] (outermost array; falls back to Bracket if nil)
	Colon       *lipgloss.Style // :
	Comma       *lipgloss.Style // ,
}

// DefaultJSON returns lipgloss styles for JSON tokens using clog's default
// dark theme. Terminal-aware light/dark selection is applied by the [Logger];
// see [github.com/gechr/clog.Logger.SetPrintTheme].
func DefaultJSON() *JSON {
	return NewJSON(theme.Dark())
}

// NewJSON returns lipgloss styles for JSON tokens using the given theme.
func NewJSON(th *theme.Theme) *JSON {
	return &JSON{
		Spacing: JSONSpacingAfterComma,

		BoolFalse: new(lipgloss.NewStyle().Foreground(th.BoolFalse).Italic(true)),
		BoolTrue:  new(lipgloss.NewStyle().Foreground(th.BoolTrue).Italic(true)),
		Key:       new(lipgloss.NewStyle().Foreground(th.Key)),
		Null:      new(lipgloss.NewStyle().Foreground(th.Comment).Italic(true)),
		Number:    new(lipgloss.NewStyle().Foreground(th.Number)),
		String:    new(lipgloss.NewStyle().Foreground(th.String)),

		Brace:       new(lipgloss.NewStyle().Foreground(th.Foreground)),
		BraceRoot:   new(lipgloss.NewStyle().Foreground(th.Foreground).Bold(true)),
		Bracket:     new(lipgloss.NewStyle().Foreground(th.Foreground)),
		BracketRoot: new(lipgloss.NewStyle().Foreground(th.Foreground).Bold(true)),
		Colon:       new(lipgloss.NewStyle().Foreground(th.Foreground)),
		Comma:       new(lipgloss.NewStyle().Foreground(th.Foreground)),
	}
}

// WithSpacing returns the receiver with the given spacing flags applied.
// It modifies and returns the same pointer for fluent chaining:
//
//	styles.JSON = style.DefaultJSON().WithSpacing(style.JSONSpacingAll)
func (s *JSON) WithSpacing(spacing JSONSpacing) *JSON {
	s.Spacing = spacing
	return s
}
