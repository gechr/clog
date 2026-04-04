package style

import "charm.land/lipgloss/v2"

// TOML configures per-token lipgloss styles for TOML syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultTOML] as a starting point for customization.
type TOML struct {
	BoolFalse *lipgloss.Style // false
	BoolTrue  *lipgloss.Style // true
	Comment   *lipgloss.Style // # comment text
	DateTime  *lipgloss.Style // date, time, datetime values
	Float     *lipgloss.Style // floating point values
	Integer   *lipgloss.Style // integer values
	Key       *lipgloss.Style // bare and dotted keys
	String    *lipgloss.Style // basic and literal strings
	TableKey  *lipgloss.Style // [table] and [[array]] header keys
}

// DefaultTOML returns dracula-inspired lipgloss styles for TOML tokens.
// Colors match the JSON/YAML defaults where tokens overlap.
func DefaultTOML() *TOML {
	return &TOML{
		BoolFalse: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red (matches JSON)
		),
		BoolTrue: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green (matches JSON)
		),
		Comment: new(
			lipgloss.NewStyle().Faint(true),
		),
		DateTime: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd")), // cyan
		),
		Float: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#ff79c6")), // pink (matches JSON Number)
		),
		Integer: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#ff79c6")), // pink (matches JSON Number)
		),
		Key: new(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#bd93f9")), // purple (matches JSON/YAML Key)
		),
		String: new(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f1fa8c")), // yellow (matches JSON/YAML String)
		),
		TableKey: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#bd93f9")).Bold(true), // purple bold
		),
	}
}
