//nolint:dupl // TOML's token-style shape coincidentally mirrors HCL's
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
		BoolFalse:   fgItalic(th.BoolFalse),
		BoolTrue:    fgItalic(th.BoolTrue),
		Comment:     fg(th.Comment),
		DateTime:    fg(th.Accent),
		Float:       fg(th.Number),
		Integer:     fg(th.Number),
		Key:         fg(th.Key),
		Punctuation: fg(th.Foreground),
		String:      fg(th.String),
		TableKey:    fg(th.Accent),
	}
}
