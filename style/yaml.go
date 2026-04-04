package style

import "charm.land/lipgloss/v2"

// YAML configures per-token lipgloss styles for YAML syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultYAML] as a starting point for customization.
type YAML struct {
	Anchor  *lipgloss.Style // &anchor name
	Alias   *lipgloss.Style // *alias reference
	Bool    *lipgloss.Style // true, false, yes, no, on, off
	Comment *lipgloss.Style // # comment text
	Key     *lipgloss.Style // mapping keys
	Null    *lipgloss.Style // null, ~
	Number  *lipgloss.Style // int, float, hex, octal, binary, inf, nan
	String  *lipgloss.Style // string values (plain, single-quoted, double-quoted)
	Tag     *lipgloss.Style // !!str, !!int, !custom
}

// DefaultYAML returns dracula-inspired lipgloss styles for YAML tokens.
// Colors match the JSON defaults where tokens overlap.
func DefaultYAML() *YAML {
	return &YAML{
		Anchor: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd")), // cyan
		),
		Alias: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd")), // cyan
		),
		Bool: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#ff79c6")), // pink
		),
		Comment: new(
			lipgloss.NewStyle().Faint(true),
		),
		Key: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#bd93f9")), // purple (matches JSON Key)
		),
		Null: new(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#8892bf")).
				Italic(true), // muted blue-grey italic (matches JSON Null)
		),
		Number: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#ff79c6")), // pink (matches JSON Number)
		),
		String: new(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#f1fa8c")), // yellow (matches JSON String)
		),
		Tag: new(
			lipgloss.NewStyle().Faint(true),
		),
	}
}
