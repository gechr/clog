package clog

import "github.com/charmbracelet/lipgloss"

// Styles holds lipgloss styles for the logger's pretty output.
type Styles struct {
	Levels        map[Level]lipgloss.Style
	Messages      map[Level]lipgloss.Style // Per-level message styles.
	Key           lipgloss.Style
	Separator     lipgloss.Style
	SeparatorText string // Separator between key and value (default "=").
	Timestamp     lipgloss.Style
	String        *lipgloss.Style           // Style for string values. Nil = no styling.
	Number        *lipgloss.Style           // Style for numeric values (int, uint64, float64). Nil = no styling.
	Error         *lipgloss.Style           // Style for error values. Nil = no styling.
	KeyStyles     map[string]lipgloss.Style // Field key name → value style (e.g. "path" → blue).
	ValueStyles   map[string]lipgloss.Style // Formatted value → style (e.g. "true" → green).
}

// DefaultStyles returns the default colour styles.
func DefaultStyles() *Styles {
	return &Styles{
		Levels: map[Level]lipgloss.Style{
			TraceLevel: lipgloss.NewStyle().
				Bold(true).
				Faint(true).
				Foreground(lipgloss.Color("6")),
			// dim cyan
			DebugLevel: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("6")),
			// cyan
			InfoLevel: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("2")),
			// green
			DryLevel: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("5")),
			// magenta
			WarnLevel: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("3")),
			// yellow
			ErrorLevel: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1")),
			// red
			FatalLevel: lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1")),
			// red
		},

		Separator:     lipgloss.NewStyle().Faint(true),
		SeparatorText: "=",

		Error:     new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))), // red
		Messages:  DefaultMessageStyles(),
		Number:    new(lipgloss.NewStyle().Foreground(lipgloss.Color("5"))),  // magenta
		String:    new(lipgloss.NewStyle().Foreground(lipgloss.Color("15"))), // white
		Key:       lipgloss.NewStyle().Foreground(lipgloss.Color("4")),       // blue
		Timestamp: lipgloss.NewStyle().Faint(true),

		KeyStyles:   make(map[string]lipgloss.Style),
		ValueStyles: DefaultValueStyles(),
	}
}

// DefaultMessageStyles returns the default per-level message styles (unstyled).
func DefaultMessageStyles() map[Level]lipgloss.Style {
	s := lipgloss.NewStyle()

	return map[Level]lipgloss.Style{
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
func DefaultValueStyles() map[string]lipgloss.Style {
	return map[string]lipgloss.Style{
		"true":  lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green
		"false": lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
		"<nil>": lipgloss.NewStyle().Faint(true),                     // grey
		"":      lipgloss.NewStyle().Faint(true),                     // grey
	}
}
