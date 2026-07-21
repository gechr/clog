package theme

import (
	"fmt"
	"strings"

	xslices "github.com/gechr/x/slices"
)

const (
	themeNameDark                = "dark"
	themeNameLight               = "light"
	themeNameCatppuccinFrappe    = "catppuccin-frappe"
	themeNameCatppuccinLatte     = "catppuccin-latte"
	themeNameCatppuccinMacchiato = "catppuccin-macchiato"
	themeNameCatppuccinMocha     = "catppuccin-mocha"
	themeNameDracula             = "dracula"
	themeNameMonokai             = "monokai"
)

// preset pairs a built-in theme's name with its constructor.
type preset struct {
	name string
	make func() *Theme
}

// presets lists the built-in themes in the order shown by error messages.
// Adding a theme is one entry here (plus its themeName* constant).
var presets = []preset{
	{themeNameDark, Dark},
	{themeNameLight, Light},
	{themeNameCatppuccinFrappe, CatppuccinFrappe},
	{themeNameCatppuccinLatte, CatppuccinLatte},
	{themeNameCatppuccinMacchiato, CatppuccinMacchiato},
	{themeNameCatppuccinMocha, CatppuccinMocha},
	{themeNameDracula, Dracula},
	{themeNameMonokai, Monokai},
}

var validThemeNames = xslices.Map(presets, func(p preset) string { return p.name })

func normalizePresetName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	replacer := strings.NewReplacer("-", "", "_", "", " ", "")
	return replacer.Replace(name)
}

// Name returns the preset name for built-in themes, or "custom" for themes
// that were modified programmatically.
func (t *Theme) Name() string {
	if t == nil || t.name == "" {
		return "custom"
	}
	return t.name
}

// MarshalText implements [encoding.TextMarshaler].
func (t *Theme) MarshalText() ([]byte, error) {
	if t == nil {
		return nil, fmt.Errorf("cannot marshal nil theme")
	}
	if t.name == "" {
		return nil, fmt.Errorf("cannot marshal custom theme")
	}
	return []byte(t.name), nil
}

// UnmarshalText implements [encoding.TextUnmarshaler].
func (t *Theme) UnmarshalText(text []byte) error {
	if t == nil {
		return fmt.Errorf("cannot unmarshal theme into nil receiver")
	}

	key := normalizePresetName(string(text))
	for _, p := range presets {
		if normalizePresetName(p.name) == key {
			*t = *p.make()
			return nil
		}
	}
	return fmt.Errorf("unknown theme %q (valid: %s)", text, strings.Join(validThemeNames, ", "))
}
