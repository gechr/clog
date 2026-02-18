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
	Number *lipgloss.Style // Override for the number segment (nil = keep default).
	Unit   *lipgloss.Style // Override for the unit segment (nil = keep default).
}

// Threshold defines a style override when a quantity's numeric value
// meets or exceeds the given threshold. Thresholds are evaluated in descending
// order — the first match wins.
type Threshold struct {
	Value float64        // Minimum numeric value (inclusive) to trigger this style.
	Style ThresholdStyle // Style overrides for number and unit segments.
}

// StyleMap maps string keys to lipgloss styles (e.g. field key names or unit strings).
type StyleMap = map[string]*lipgloss.Style

// Thresholds is a list of [Threshold] entries, evaluated high -> low (first match wins).
type Thresholds = []Threshold

// ThresholdMap maps unit strings to their thresholds (evaluated high -> low).
type ThresholdMap = map[string]Thresholds

// LevelStyleMap maps log levels to lipgloss styles.
type LevelStyleMap = map[Level]*lipgloss.Style

// ValueStyleMap maps typed values to lipgloss styles. Keys use Go equality
// (e.g. bool true != string "true").
type ValueStyleMap = map[any]*lipgloss.Style

// Styles holds lipgloss styles for the logger's pretty output.
// Pointer fields can be set to nil to disable that style entirely.
type Styles struct {
	// Duration unit -> thresholds (evaluated high->low).
	DurationThresholds ThresholdMap
	// Duration unit -> style override (e.g. "s" -> yellow).
	DurationUnits StyleMap
	// Style for the numeric segments of duration values (e.g. "1" in "1m30s") [nil = plain text]
	FieldDurationNumber *lipgloss.Style
	// Style for the unit segments of duration values (e.g. "m" in "1m30s") [nil = plain text]
	FieldDurationUnit *lipgloss.Style
	// Style for error field values [nil = plain text]
	FieldError *lipgloss.Style
	// Style for int/float field values [nil = plain text]
	FieldNumber *lipgloss.Style
	// Base style for Percent fields (foreground overridden by gradient). nil = gradient color only.
	FieldPercent *lipgloss.Style
	// Style for the numeric part of quantity values (e.g. "5" in "5km") [nil = plain text]
	FieldQuantityNumber *lipgloss.Style
	// Style for the unit part of quantity values (e.g. "km" in "5km") [nil = plain text]
	FieldQuantityUnit *lipgloss.Style
	// Style for string field values [nil = plain text]
	FieldString *lipgloss.Style
	// Style for time.Time field values [nil = plain text]
	FieldTime *lipgloss.Style
	// Style for field key names without a per-key override.
	KeyDefault *lipgloss.Style
	// Field key name -> value style (e.g. "path" -> blue).
	Keys StyleMap
	// Level label style (e.g. "INF", "ERR").
	Levels LevelStyleMap
	// Message text style per level.
	Messages LevelStyleMap
	// Gradient stops for Percent fields (default: red → yellow → green).
	PercentGradient []ColorStop
	// Decimal places for Percent display (default 0 = "75%", 1 -> "75.0%", etc).
	PercentPrecision int
	// Quantity unit -> thresholds (evaluated high->low).
	QuantityThresholds ThresholdMap
	// Unit string -> style override (e.g. "km" -> green).
	QuantityUnits StyleMap
	// Case-insensitive quantity unit matching (default true).
	QuantityUnitsIgnoreCase bool
	// Style for key/value [SeparatorText].
	Separator *lipgloss.Style
	// Separator between key and value (default "=").
	SeparatorText string
	// Style for the timestamp prefix.
	Timestamp *lipgloss.Style
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
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Faint(true), // magenta dim
		),
		FieldError: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		),
		FieldNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityNumber: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")), // magenta
		),
		FieldQuantityUnit: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Faint(true), // magenta dim
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
		DurationThresholds:      make(ThresholdMap),
		DurationUnits:           make(StyleMap),
		Messages:                DefaultMessageStyles(),
		PercentGradient:         DefaultPercentGradient(),
		QuantityThresholds:      make(ThresholdMap),
		QuantityUnits:           make(StyleMap),
		QuantityUnitsIgnoreCase: true,
		Separator:               new(lipgloss.NewStyle().Faint(true)),
		SeparatorText:           "=",
		Timestamp:               new(lipgloss.NewStyle().Faint(true)),
		Values:                  DefaultValueStyles(),
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
