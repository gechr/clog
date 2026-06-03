package style

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/theme"
)

// YAML configures per-token lipgloss styles for YAML syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultYAML] as a starting point for customization.
type YAML struct {
	Alias       *lipgloss.Style // *alias reference
	Anchor      *lipgloss.Style // &anchor name
	BoolFalse   *lipgloss.Style // false, no, off
	BoolTrue    *lipgloss.Style // true, yes, on
	Comment     *lipgloss.Style // # comment text
	Key         *lipgloss.Style // mapping keys
	Null        *lipgloss.Style // null, ~
	Number      *lipgloss.Style // int, float, hex, octal, binary, inf, nan
	Punctuation *lipgloss.Style // structural tokens (:, -, [, ], {, }, ,)
	String      *lipgloss.Style // string values (plain, single-quoted, double-quoted)
	Tag         *lipgloss.Style // !!str, !!int, !custom
}

// DefaultYAML returns lipgloss styles for YAML tokens using clog's default
// dark theme. Terminal-aware light/dark selection is applied by the [Logger];
// see [github.com/gechr/clog.Logger.SetPrintTheme].
func DefaultYAML() *YAML {
	return NewYAML(theme.Dark())
}

// NewYAML returns lipgloss styles for YAML tokens using the given theme.
func NewYAML(th *theme.Theme) *YAML {
	return &YAML{
		Alias:       new(lipgloss.NewStyle().Foreground(th.Accent)),
		Anchor:      new(lipgloss.NewStyle().Foreground(th.Accent)),
		BoolFalse:   new(lipgloss.NewStyle().Foreground(th.BoolFalse).Italic(true)),
		BoolTrue:    new(lipgloss.NewStyle().Foreground(th.BoolTrue).Italic(true)),
		Comment:     new(lipgloss.NewStyle().Foreground(th.Comment)),
		Key:         new(lipgloss.NewStyle().Foreground(th.Key)),
		Null:        new(lipgloss.NewStyle().Foreground(th.Comment).Italic(true)),
		Number:      new(lipgloss.NewStyle().Foreground(th.Number)),
		Punctuation: new(lipgloss.NewStyle().Foreground(th.Foreground)),
		String:      new(lipgloss.NewStyle().Foreground(th.String)),
		Tag:         new(lipgloss.NewStyle().Foreground(th.Comment)),
	}
}
