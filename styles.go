package clog

import "github.com/charmbracelet/lipgloss"

// Styles holds lipgloss styles for the logger's pretty output.
// Pointer fields can be set to nil to disable that style entirely.
type Styles struct {
	DurationUnits           map[string]*lipgloss.Style // Duration unit → style override (e.g. "s" → yellow).
	FieldDurationNumber     *lipgloss.Style            // nil = no styling.
	FieldDurationUnit       *lipgloss.Style            // nil = no styling.
	FieldError              *lipgloss.Style            // nil = no styling.
	FieldNumber             *lipgloss.Style            // nil = no styling.
	FieldQuantityNumber     *lipgloss.Style            // nil = no styling.
	FieldQuantityUnit       *lipgloss.Style            // nil = no styling.
	FieldString             *lipgloss.Style            // nil = no styling.
	FieldTime               *lipgloss.Style            // nil = no styling.
	KeyDefault              *lipgloss.Style            // Style for field key names without a per-key override.
	Keys                    map[string]*lipgloss.Style // Field key name → value style (e.g. "path" → blue).
	Levels                  map[Level]*lipgloss.Style  // Level label style (e.g. "INF", "ERR").
	Messages                map[Level]*lipgloss.Style  // Message text style per level.
	QuantityUnits           map[string]*lipgloss.Style // Unit string → style override (e.g. "km" → green).
	QuantityUnitsIgnoreCase bool                       // Case-insensitive quantity unit matching (default true).
	Separator               *lipgloss.Style            // Style for the key=value separator.
	SeparatorText           string                     // Separator between key and value (default "=").
	Timestamp               *lipgloss.Style            // Style for the timestamp prefix.
	Values                  map[string]*lipgloss.Style // Formatted value → style (e.g. "true" → green).
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
		Keys: make(map[string]*lipgloss.Style),
		Levels: map[Level]*lipgloss.Style{
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
		DurationUnits:           make(map[string]*lipgloss.Style),
		Messages:                DefaultMessageStyles(),
		QuantityUnits:           make(map[string]*lipgloss.Style),
		QuantityUnitsIgnoreCase: true,
		Separator:               new(lipgloss.NewStyle().Faint(true)),
		SeparatorText:           "=",
		Timestamp:               new(lipgloss.NewStyle().Faint(true)),
		Values:                  DefaultValueStyles(),
	}
}

// DefaultMessageStyles returns the default per-level message styles (unstyled).
func DefaultMessageStyles() map[Level]*lipgloss.Style {
	s := new(lipgloss.NewStyle())

	return map[Level]*lipgloss.Style{
		TraceLevel: s,
		DebugLevel: s,
		InfoLevel:  s,
		DryLevel:   s,
		WarnLevel:  s,
		ErrorLevel: s,
		FatalLevel: s,
	}
}

// DefaultValueStyles returns sensible default styles for common value strings.
func DefaultValueStyles() map[string]*lipgloss.Style {
	return map[string]*lipgloss.Style{
		"true":  new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))), // green
		"false": new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))), // red
		nilStr:  new(lipgloss.NewStyle().Faint(true)),                     // grey
		"":      new(lipgloss.NewStyle().Faint(true)),                     // grey
	}
}
