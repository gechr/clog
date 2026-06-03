package style

import (
	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/theme"
)

// TOML configures per-token lipgloss styles for TOML syntax highlighting.
// nil fields render the corresponding token unstyled.
//
// Use [DefaultTOML] as a starting point for customization.
type TOML struct {
	BoolFalse   *lipgloss.Style // false
	BoolTrue    *lipgloss.Style // true
	Comment     *lipgloss.Style // # comment text
	DateTime    *lipgloss.Style // date, time, datetime values
	Float       *lipgloss.Style // floating point values
	Integer     *lipgloss.Style // integer values
	Key         *lipgloss.Style // bare and dotted keys
	Punctuation *lipgloss.Style // structural tokens (=, [, ], {, }, ,)
	String      *lipgloss.Style // basic and literal strings
	TableKey    *lipgloss.Style // [table] and [[array]] header keys
}

// DefaultTOML returns lipgloss styles for TOML tokens using clog's default
// dark theme. Terminal-aware light/dark selection is applied by the [Logger];
// see [github.com/gechr/clog.Logger.SetPrintTheme].
func DefaultTOML() *TOML {
	return NewTOML(theme.Dark())
}

// NewTOML returns lipgloss styles for TOML tokens using the given theme.
func NewTOML(th *theme.Theme) *TOML {
	return &TOML{
		BoolFalse:   new(lipgloss.NewStyle().Foreground(th.BoolFalse).Italic(true)),
		BoolTrue:    new(lipgloss.NewStyle().Foreground(th.BoolTrue).Italic(true)),
		Comment:     new(lipgloss.NewStyle().Foreground(th.Comment)),
		DateTime:    new(lipgloss.NewStyle().Foreground(th.Accent)),
		Float:       new(lipgloss.NewStyle().Foreground(th.Number)),
		Integer:     new(lipgloss.NewStyle().Foreground(th.Number)),
		Key:         new(lipgloss.NewStyle().Foreground(th.Key)),
		Punctuation: new(lipgloss.NewStyle().Foreground(th.Foreground)),
		String:      new(lipgloss.NewStyle().Foreground(th.String)),
		TableKey:    new(lipgloss.NewStyle().Foreground(th.Accent)),
	}
}
