package style

import "charm.land/lipgloss/v2"

// HCL configures per-token lipgloss styles for HCL syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultHCL] as a starting point for customization.
type HCL struct {
	BlockType *lipgloss.Style // block type identifiers (resource, variable, data, etc.)
	BoolFalse *lipgloss.Style // false
	BoolTrue  *lipgloss.Style // true
	Comment   *lipgloss.Style // # and // and /* */ comments
	Key       *lipgloss.Style // attribute keys (identifier before =)
	Null      *lipgloss.Style // null
	Number    *lipgloss.Style // numeric literals
	String    *lipgloss.Style // string values (quoted literals and quote markers)
}

// DefaultHCL returns dracula-inspired lipgloss styles for HCL tokens.
// Colors match the JSON/YAML defaults where tokens overlap.
func DefaultHCL() *HCL {
	return &HCL{
		BlockType: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8be9fd")), // cyan
		),
		BoolFalse: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red (matches JSON BoolFalse)
		),
		BoolTrue: new(
			lipgloss.NewStyle().Foreground(lipgloss.Color("2")), // green (matches JSON BoolTrue)
		),
		Comment: new(
			lipgloss.NewStyle().Faint(true),
		),
		Key: new(
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#bd93f9")), // purple (matches JSON/YAML Key)
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
	}
}
