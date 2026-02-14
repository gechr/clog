package clog

import "github.com/charmbracelet/lipgloss"

// Styles holds lipgloss styles for the logger's pretty output.
// Pointer fields can be set to nil to disable that style entirely.
type Styles struct {
	FieldDuration *lipgloss.Style // Nil = no styling.
	FieldError    *lipgloss.Style // Nil = no styling.
	FieldNumber   *lipgloss.Style // Nil = no styling.
	FieldString   *lipgloss.Style // Nil = no styling.
	FieldTime     *lipgloss.Style // Nil = no styling.
	KeyDefault    *lipgloss.Style
	Keys          map[string]*lipgloss.Style // Field key name → value style (e.g. "path" → blue).
	Levels        map[Level]*lipgloss.Style
	Messages      map[Level]*lipgloss.Style
	Separator     *lipgloss.Style
	SeparatorText string // Separator between key and value (default "=").
	Timestamp     *lipgloss.Style
	Values        map[string]*lipgloss.Style // Formatted value → style (e.g. "true" → green).
}

// DefaultStyles returns the default colour styles.
func DefaultStyles() *Styles {
	return &Styles{
		FieldDuration: new(lipgloss.NewStyle().Foreground(lipgloss.Color("5"))),  // magenta
		FieldError:    new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))),  // red
		FieldNumber:   new(lipgloss.NewStyle().Foreground(lipgloss.Color("5"))),  // magenta
		FieldString:   new(lipgloss.NewStyle().Foreground(lipgloss.Color("15"))), // white
		FieldTime:     new(lipgloss.NewStyle().Foreground(lipgloss.Color("5"))),  // magenta
		KeyDefault:    new(lipgloss.NewStyle().Foreground(lipgloss.Color("4"))),  // blue
		Keys:          make(map[string]*lipgloss.Style),
		Levels: map[Level]*lipgloss.Style{
			TraceLevel: new(lipgloss.NewStyle().
				Bold(true).
				Faint(true).
				Foreground(lipgloss.Color("6"))),
			// dim cyan
			DebugLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("6"))),
			// cyan
			InfoLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("2"))),
			// green
			DryLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("5"))),
			// magenta
			WarnLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("3"))),
			// yellow
			ErrorLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1"))),
			// red
			FatalLevel: new(lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("1"))),
			// red
		},
		Messages:      DefaultMessageStyles(),
		Separator:     new(lipgloss.NewStyle().Faint(true)),
		SeparatorText: "=",
		Timestamp:     new(lipgloss.NewStyle().Faint(true)),
		Values:        DefaultValueStyles(),
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
