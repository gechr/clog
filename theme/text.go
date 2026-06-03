package theme

import (
	"fmt"
	"strings"
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

const (
	themeKeyDark                = "dark"
	themeKeyLight               = "light"
	themeKeyCatppuccinFrappe    = "catppuccinfrappe"
	themeKeyCatppuccinLatte     = "catppuccinlatte"
	themeKeyCatppuccinMacchiato = "catppuccinmacchiato"
	themeKeyCatppuccinMocha     = "catppuccinmocha"
	themeKeyDracula             = "dracula"
	themeKeyMonokai             = "monokai"
)

var validThemeNames = []string{
	themeNameDark,
	themeNameLight,
	themeNameCatppuccinFrappe,
	themeNameCatppuccinLatte,
	themeNameCatppuccinMacchiato,
	themeNameCatppuccinMocha,
	themeNameDracula,
	themeNameMonokai,
}

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

	switch normalizePresetName(string(text)) {
	case themeKeyDark:
		*t = *Dark()
	case themeKeyLight:
		*t = *Light()
	case themeKeyMonokai:
		*t = *Monokai()
	case themeKeyCatppuccinLatte:
		*t = *CatppuccinLatte()
	case themeKeyCatppuccinFrappe:
		*t = *CatppuccinFrappe()
	case themeKeyCatppuccinMacchiato:
		*t = *CatppuccinMacchiato()
	case themeKeyCatppuccinMocha:
		*t = *CatppuccinMocha()
	case themeKeyDracula:
		*t = *Dracula()
	default:
		return fmt.Errorf("unknown theme %q (valid: %s)", text, strings.Join(validThemeNames, ", "))
	}
	return nil
}
