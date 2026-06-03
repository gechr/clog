// Package theme provides color palettes for syntax highlighting.
package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme defines a named color palette for syntax highlighting.
type Theme struct {
	name string

	// Background declares the terminal background this theme is designed for.
	Background Background

	Accent     color.Color // type-level identifiers (block types, table headers, datetime, anchors)
	BoolFalse  color.Color // false, no, off
	BoolTrue   color.Color // true, yes, on
	Comment    color.Color // comments, null, tags
	Foreground color.Color // punctuation, structural tokens
	Key        color.Color // mapping/attribute keys
	Number     color.Color // numeric values
	Secondary  color.Color // secondary keys (nested keys)
	String     color.Color // string values
}

// Dark returns clog's default dark-background color theme.
//
// This intentionally preserves the colors that clog used before light/dark
// theme selection was added.
func Dark() *Theme {
	return &Theme{
		name:       themeNameDark,
		Background: BackgroundDark,
		Accent:     lipgloss.Color("#8be9fd"), // cyan
		BoolFalse:  lipgloss.Color("#ff5555"), // red
		BoolTrue:   lipgloss.Color("#50fa7b"), // green
		Comment:    lipgloss.Color("#6272a4"), // comment
		Foreground: lipgloss.Color("#f8f8f2"), // foreground
		Key:        lipgloss.Color("#bd93f9"), // purple
		Number:     lipgloss.Color("#ff79c6"), // pink
		Secondary:  lipgloss.Color("#ffb86c"), // orange
		String:     lipgloss.Color("#f1fa8c"), // yellow
	}
}

// Light returns clog's default light-background color theme.
func Light() *Theme {
	return &Theme{
		name:       themeNameLight,
		Background: BackgroundLight,
		Accent:     lipgloss.Color("#006d75"), // teal
		BoolFalse:  lipgloss.Color("#a11d33"), // red
		BoolTrue:   lipgloss.Color("#256d1b"), // green
		Comment:    lipgloss.Color("#5f6368"), // gray
		Foreground: lipgloss.Color("#3c4043"), // dark gray
		Key:        lipgloss.Color("#2459b3"), // blue
		Number:     lipgloss.Color("#9a4d00"), // orange
		Secondary:  lipgloss.Color("#7047b5"), // purple
		String:     lipgloss.Color("#256d1b"), // green
	}
}

// Monokai returns the Monokai color theme.
func Monokai() *Theme {
	return &Theme{
		name:       themeNameMonokai,
		Background: BackgroundDark,
		Accent:     lipgloss.Color("#66d9ef"), // cyan
		BoolFalse:  lipgloss.Color("#f92672"), // pink
		BoolTrue:   lipgloss.Color("#a6e22e"), // green
		Comment:    lipgloss.Color("#88846f"), // comment
		Foreground: lipgloss.Color("#f8f8f2"), // foreground
		Key:        lipgloss.Color("#ae81ff"), // purple
		Number:     lipgloss.Color("#f92672"), // pink
		Secondary:  lipgloss.Color("#fd971f"), // orange
		String:     lipgloss.Color("#e6db74"), // yellow
	}
}

// CatppuccinLatte returns the Catppuccin Latte (light) color theme.
func CatppuccinLatte() *Theme {
	return &Theme{
		name:       themeNameCatppuccinLatte,
		Background: BackgroundLight,
		Accent:     lipgloss.Color("#179299"), // teal
		BoolFalse:  lipgloss.Color("#d20f39"), // red
		BoolTrue:   lipgloss.Color("#40a02b"), // green
		Comment:    lipgloss.Color("#7c7f93"), // overlay2
		Foreground: lipgloss.Color("#4c4f69"), // text
		Key:        lipgloss.Color("#1e66f5"), // blue
		Number:     lipgloss.Color("#fe640b"), // peach
		Secondary:  lipgloss.Color("#dc8a78"), // rosewater
		String:     lipgloss.Color("#40a02b"), // green
	}
}

// CatppuccinFrappe returns the Catppuccin Frappe (dark) color theme.
func CatppuccinFrappe() *Theme {
	return &Theme{
		name:       themeNameCatppuccinFrappe,
		Background: BackgroundDark,
		Accent:     lipgloss.Color("#81c8be"), // teal
		BoolFalse:  lipgloss.Color("#e78284"), // red
		BoolTrue:   lipgloss.Color("#a6d189"), // green
		Comment:    lipgloss.Color("#949cbb"), // overlay2
		Foreground: lipgloss.Color("#c6d0f5"), // text
		Key:        lipgloss.Color("#8caaee"), // blue
		Number:     lipgloss.Color("#ef9f76"), // peach
		Secondary:  lipgloss.Color("#f2d5cf"), // rosewater
		String:     lipgloss.Color("#a6d189"), // green
	}
}

// CatppuccinMacchiato returns the Catppuccin Macchiato (dark) color theme.
func CatppuccinMacchiato() *Theme {
	return &Theme{
		name:       themeNameCatppuccinMacchiato,
		Background: BackgroundDark,
		Accent:     lipgloss.Color("#8bd5ca"), // teal
		BoolFalse:  lipgloss.Color("#ed8796"), // red
		BoolTrue:   lipgloss.Color("#a6da95"), // green
		Comment:    lipgloss.Color("#939ab7"), // overlay2
		Foreground: lipgloss.Color("#cad3f5"), // text
		Key:        lipgloss.Color("#8aadf4"), // blue
		Number:     lipgloss.Color("#f5a97f"), // peach
		Secondary:  lipgloss.Color("#f4dbd6"), // rosewater
		String:     lipgloss.Color("#a6da95"), // green
	}
}

// CatppuccinMocha returns the Catppuccin Mocha (dark) color theme.
func CatppuccinMocha() *Theme {
	return &Theme{
		name:       themeNameCatppuccinMocha,
		Background: BackgroundDark,
		Accent:     lipgloss.Color("#94e2d5"), // teal
		BoolFalse:  lipgloss.Color("#f38ba8"), // red
		BoolTrue:   lipgloss.Color("#a6e3a1"), // green
		Comment:    lipgloss.Color("#9399b2"), // overlay2
		Foreground: lipgloss.Color("#cdd6f4"), // text
		Key:        lipgloss.Color("#89b4fa"), // blue
		Number:     lipgloss.Color("#fab387"), // peach
		Secondary:  lipgloss.Color("#f5e0dc"), // rosewater
		String:     lipgloss.Color("#a6e3a1"), // green
	}
}

// Dracula returns the Dracula color theme.
func Dracula() *Theme {
	th := Dark()
	th.name = themeNameDracula
	return th
}
