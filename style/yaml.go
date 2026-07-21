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
		Alias:       fg(th.Accent),
		Anchor:      fg(th.Accent),
		BoolFalse:   fgItalic(th.BoolFalse),
		BoolTrue:    fgItalic(th.BoolTrue),
		Comment:     fg(th.Comment),
		Key:         fg(th.Key),
		Null:        fgItalic(th.Comment),
		Number:      fg(th.Number),
		Punctuation: fg(th.Foreground),
		String:      fg(th.String),
		Tag:         fg(th.Comment),
	}
}
