package clog

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// ColorStop defines a color at a specific position along a gradient.
// Position is in the range 0.0-1.0.
type ColorStop struct {
	Position float64        // 0.0-1.0
	Color    colorful.Color // from github.com/lucasb-eyer/go-colorful
}

// ThresholdStyle holds optional style overrides for the number and unit
// segments of a quantity or duration value. nil fields keep the default style.
type ThresholdStyle struct {
	Number Style // Override for the number segment (nil = keep default).
	Unit   Style // Override for the unit segment (nil = keep default).
}

// Threshold defines a style override when a quantity's numeric value
// meets or exceeds the given threshold. Thresholds are evaluated in descending
// order — the first match wins.
type Threshold struct {
	Value float64        // Minimum numeric value (inclusive) to trigger this style.
	Style ThresholdStyle // Style overrides for number and unit segments.
}

// Style is a convenience alias for *lipgloss.Style.
type Style = *lipgloss.Style

// StyleMap maps string keys to lipgloss styles (e.g. field key names or unit strings).
type StyleMap = map[string]Style

// Thresholds is a list of [Threshold] entries, evaluated high -> low (first match wins).
type Thresholds = []Threshold

// ThresholdMap maps unit strings to their thresholds (evaluated high -> low).
type ThresholdMap = map[string]Thresholds

// LevelStyleMap maps log levels to lipgloss styles.
type LevelStyleMap = map[Level]Style

// ValueStyleMap maps typed values to lipgloss styles. Keys use Go equality
// (e.g. bool true != string "true").
type ValueStyleMap = map[any]Style

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

// Sort controls how fields are sorted in output.
type Sort int

const (
	// SortNone preserves the insertion order of fields (default).
	SortNone Sort = iota
	// SortAscending sorts fields by key A→Z.
	SortAscending
	// SortDescending sorts fields by key Z→A.
	SortDescending
)

// JSONStyles configures per-token lipgloss styles for JSON syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultJSONStyles] as a starting point for customization.
type JSONStyles struct {
	// Mode controls rendering behaviour.
	// JSONModeJSON (default) preserves standard JSON quoting.
	// JSONModeHuman strips quotes from identifier-like keys and simple string values.
	// JSONModeFlat flattens nested object keys with dot notation; arrays are kept intact.
	Mode JSONMode
	// Spacing controls where spaces are inserted. Zero (default) means no spaces.
	// Use JSONSpacingAll for {"key": "value", "n": 1} style output.
	Spacing JSONSpacing
	// OmitCommas omits the comma between items. JSONSpacingAfterComma still
	// applies and can be used to keep a space separator: {"a":1 "b":2}.
	OmitCommas bool

	Key            Style // Object keys
	String         Style // String values
	Number         Style // Numeric values — base fallback for all number sub-styles
	NumberPositive Style // Positive numbers (with or without explicit sign); falls back to Number
	NumberNegative Style // Negative numbers; falls back to Number
	NumberZero     Style // Zero; falls back to NumberPositive, then Number
	NumberFloat    Style // Floating-point values; falls back to Number
	NumberInteger  Style // Integer values; falls back to Number
	True           Style // true
	False          Style // false
	Null           Style // null
	Brace          Style // { } (nested)
	RootBrace      Style // { } (outermost object; falls back to Brace if nil)
	Bracket        Style // [ ] (nested)
	RootBracket    Style // [ ] (outermost array; falls back to Bracket if nil)
	Colon          Style // :
	Comma          Style // ,
}

// DefaultJSONStyles returns dracula-inspired lipgloss styles for JSON tokens.
// True and False mirror [DefaultValueStyles] (green/red) for consistency.
func DefaultJSONStyles() *JSONStyles {
	return &JSONStyles{
		Spacing: JSONSpacingAfterComma,
		Key: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#bd93f9")), // purple
		),
		String: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f1fa8c")), // yellow
		),
		Number: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#ff79c6")), // pink
		),
		True: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		),
		False: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		),
		Null: new(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8892bf")).
				Italic(true), // muted blue-grey italic
		),
		Brace: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f8f8f2")), // white
		),
		RootBrace: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f8f8f2")).Bold(true), // white bold
		),
		Bracket: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f8f8f2")), // white
		),
		RootBracket: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f8f8f2")).Bold(true), // white bold
		),
		Colon: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f8f8f2")), // white
		),
		Comma: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#f8f8f2")), // white
		),
	}
}

// WithSpacing returns the receiver with the given spacing flags applied.
// It modifies and returns the same pointer for fluent chaining:
//
//	styles.FieldJSON = clog.DefaultJSONStyles().WithSpacing(clog.JSONSpacingAll)
func (s *JSONStyles) WithSpacing(spacing JSONSpacing) *JSONStyles {
	s.Spacing = spacing
	return s
}

// Styles holds lipgloss styles for the logger's pretty output.
// Pointer fields can be set to nil to disable that style entirely.
type Styles struct {
	// Duration unit -> thresholds (evaluated high->low).
	DurationThresholds ThresholdMap
	// Duration unit -> style override (e.g. "s" -> yellow).
	DurationUnits StyleMap
	// Style for the numeric segments of duration values (e.g. "1" in "1m30s") [nil = plain text]
	FieldDurationNumber Style
	// Style for the unit segments of duration values (e.g. "m" in "1m30s") [nil = plain text]
	FieldDurationUnit Style
	// Style for the numeric segments of elapsed-time values [nil = falls back to FieldDurationNumber]
	FieldElapsedNumber Style
	// Style for the unit segments of elapsed-time values [nil = falls back to FieldDurationUnit]
	FieldElapsedUnit Style
	// Style for error field values [nil = plain text]
	FieldError Style
	// Per-token styles for JSON syntax highlighting.
	// nil disables JSON highlighting; use [DefaultJSONStyles] to enable.
	FieldJSON *JSONStyles
	// Style for int/float field values [nil = plain text]
	FieldNumber Style
	// Base style for Percent fields (foreground overridden by gradient). nil = gradient color only.
	FieldPercent Style
	// Style for the numeric part of quantity values (e.g. "5" in "5km") [nil = plain text]
	FieldQuantityNumber Style
	// Style for the unit part of quantity values (e.g. "km" in "5km") [nil = plain text]
	FieldQuantityUnit Style
	// Style for string field values [nil = plain text]
	FieldString Style
	// Style for time.Time field values [nil = plain text]
	FieldTime Style
	// Style for field key names without a per-key override.
	KeyDefault Style
	// Field key name -> value style (e.g. "path" -> blue).
	Keys StyleMap
	// Level label style (e.g. "INF", "ERR").
	Levels LevelStyleMap
	// Message text style per level.
	Messages LevelStyleMap
	// Gradient stops for Percent fields (default: red → yellow → green).
	PercentGradient []ColorStop
	// Quantity unit -> thresholds (evaluated high->low).
	QuantityThresholds ThresholdMap
	// Unit string -> style override (e.g. "km" -> green).
	QuantityUnits StyleMap
	// Style for key/value separator.
	Separator Style
	// Style for the timestamp prefix.
	Timestamp Style
	// Values maps typed values to styles. Keys use Go equality
	// Allows diffentiating between e.g. `true` (bool) and "true" (string)
	Values ValueStyleMap
}

// DefaultStyles returns the default colour styles.
func DefaultStyles() *Styles {
	return &Styles{
		FieldDurationNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldDurationUnit: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldError: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		),
		FieldJSON: DefaultJSONStyles(),
		FieldNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityUnit: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldString: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("15")), // white
		),
		FieldTime: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		KeyDefault: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("4")), // blue
		),
		Keys: make(StyleMap),
		Levels: LevelStyleMap{
			TraceLevel: new(lipgloss.NewStyle().
				Bold(true).
				Faint(true).
				Foreground(lipgloss.Color("6"))), // dim cyan
			DebugLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("6"))), // cyan
			InfoLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("2"))), // green
			DryLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("5"))), // magenta
			WarnLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("3"))), // yellow
			ErrorLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1"))), // red
			FatalLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1"))), // red
		},
		DurationThresholds: make(ThresholdMap),
		DurationUnits:      make(StyleMap),
		Messages:           DefaultMessageStyles(),
		PercentGradient:    DefaultPercentGradient(),
		QuantityThresholds: make(ThresholdMap),
		QuantityUnits:      make(StyleMap),
		Separator:          new(lipgloss.NewStyle().Faint(true)),
		Timestamp:          new(lipgloss.NewStyle().Faint(true)),
		Values:             DefaultValueStyles(),
	}
}

// DefaultMessageStyles returns the default per-level message styles (unstyled).
func DefaultMessageStyles() LevelStyleMap {
	return LevelStyleMap{
		TraceLevel: new(lipgloss.NewStyle()),
		DebugLevel: new(lipgloss.NewStyle()),
		InfoLevel:  new(lipgloss.NewStyle()),
		DryLevel:   new(lipgloss.NewStyle()),
		WarnLevel:  new(lipgloss.NewStyle()),
		ErrorLevel: new(lipgloss.NewStyle()),
		FatalLevel: new(lipgloss.NewStyle()),
	}
}

// DefaultPercentGradient returns the default red → yellow → green gradient
// used for [Styles.PercentGradient].
func DefaultPercentGradient() []ColorStop {
	start, middle, end := 0.0, 0.5, 1.0
	return []ColorStop{
		{
			Position: start,
			Color:    colorful.Color{R: 1, G: 0, B: 0}, // red
		},
		{
			Position: middle,
			Color:    colorful.Color{R: 1, G: 1, B: 0}, // yellow
		},
		{
			Position: end,
			Color:    colorful.Color{R: 0, G: 1, B: 0}, // green
		},
	}
}

// DefaultValueStyles returns sensible default styles for common value strings.
func DefaultValueStyles() ValueStyleMap {
	return ValueStyleMap{
		true:  new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))), // green
		false: new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))), // red
		nil:   new(lipgloss.NewStyle().Faint(true)),
		Nil: new(
			lipgloss.NewStyle().Faint(true),
		), // "<nil>" string (from Stringers with nil elements)
		"": new(lipgloss.NewStyle().Faint(true)),
	}
}
