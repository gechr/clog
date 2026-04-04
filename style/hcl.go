package style

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/theme"
)

// HCL configures per-token lipgloss styles for HCL syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultHCL] as a starting point for customization.
type HCL struct {
	BlockType   *lipgloss.Style // block type identifiers (resource, variable, data, etc.)
	BoolFalse   *lipgloss.Style // false
	BoolTrue    *lipgloss.Style // true
	Comment     *lipgloss.Style // # and // and /* */ comments
	Key         *lipgloss.Style // attribute keys (identifier before =)
	NestedKey   *lipgloss.Style // attribute keys inside nested blocks (depth >= 2); falls back to Key
	Null        *lipgloss.Style // null
	Number      *lipgloss.Style // numeric literals
	Punctuation *lipgloss.Style // structural tokens (=, {, }, [, ])
	String      *lipgloss.Style // string values (quoted literals and quote markers)
}

// DefaultHCL returns Dracula-themed lipgloss styles for HCL tokens.
func DefaultHCL() *HCL {
	return NewHCL(theme.Dracula())
}

// NewHCL returns lipgloss styles for HCL tokens using the given theme.
func NewHCL(th theme.Theme) *HCL {
	return &HCL{
		BlockType:   new(lipgloss.NewStyle().Foreground(th.Accent)),
		BoolFalse:   new(lipgloss.NewStyle().Foreground(th.BoolFalse).Italic(true)),
		BoolTrue:    new(lipgloss.NewStyle().Foreground(th.BoolTrue).Italic(true)),
		Comment:     new(lipgloss.NewStyle().Foreground(th.Comment)),
		Key:         new(lipgloss.NewStyle().Foreground(th.Key)),
		NestedKey:   new(lipgloss.NewStyle().Foreground(th.Secondary)),
		Null:        new(lipgloss.NewStyle().Foreground(th.Comment).Italic(true)),
		Number:      new(lipgloss.NewStyle().Foreground(th.Number)),
		Punctuation: new(lipgloss.NewStyle().Foreground(th.Foreground)),
		String:      new(lipgloss.NewStyle().Foreground(th.String)),
	}
}
