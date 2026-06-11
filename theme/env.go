package theme

import (
	"fmt"
	"strings"

	"github.com/gechr/clog/internal/env"
)

// DefaultEnvPrefix is the default environment variable prefix.
const DefaultEnvPrefix = "CLOG"

const (
	envTheme      = "THEME"
	envThemeDark  = "THEME_DARK"
	envThemeLight = "THEME_LIGHT"
)

// SetEnvPrefix sets a custom environment variable prefix. The prefix is
// process-wide and shared with root clog.
//
//	theme.SetEnvPrefix("MYAPP")
//	// Now checks MYAPP_THEME_LIGHT/MYAPP_THEME_DARK first, then CLOG_THEME_LIGHT/CLOG_THEME_DARK
func SetEnvPrefix(prefix string) {
	env.SetPrefix(prefix)
}

// PairFromEnv builds a theme pair from <PREFIX>_THEME_LIGHT and <PREFIX>_THEME_DARK.
func PairFromEnv(opts ...PairOption) (*Pair, error) {
	lightName, lightEnv := lookupEnv(envThemeLight)
	darkName, darkEnv := lookupEnv(envThemeDark)
	lightName = strings.TrimSpace(lightName)
	darkName = strings.TrimSpace(darkName)
	if lightName == "" {
		return nil, fmt.Errorf("%s is required", lightEnv)
	}
	if darkName == "" {
		return nil, fmt.Errorf("%s is required", darkEnv)
	}

	var light Theme
	if err := light.UnmarshalText([]byte(lightName)); err != nil {
		return nil, fmt.Errorf("%s: %w", lightEnv, err)
	}
	var dark Theme
	if err := dark.UnmarshalText([]byte(darkName)); err != nil {
		return nil, fmt.Errorf("%s: %w", darkEnv, err)
	}
	return NewPair(&light, &dark, opts...)
}

// FromEnv resolves theme configuration from the environment.
//
// Precedence:
//  1. <PREFIX>_THEME selects an explicit theme, wrapped via [Single] so the
//     same theme applies regardless of the terminal background.
//  2. <PREFIX>_THEME_LIGHT and <PREFIX>_THEME_DARK select a light/dark pair.
//
// It returns (nil, nil) when no theme variables are set. A set-but-invalid
// value returns an error.
func FromEnv(opts ...PairOption) (*Pair, error) {
	if value, envVar := lookupEnv(envTheme); strings.TrimSpace(value) != "" {
		var t Theme
		if err := t.UnmarshalText([]byte(strings.TrimSpace(value))); err != nil {
			return nil, fmt.Errorf("%s: %w", envVar, err)
		}
		return Single(&t), nil
	}

	lightName, _ := lookupEnv(envThemeLight)
	darkName, _ := lookupEnv(envThemeDark)
	if strings.TrimSpace(lightName) == "" && strings.TrimSpace(darkName) == "" {
		return nil, nil //nolint:nilnil // no theme configured is a valid absence, not an error
	}

	return PairFromEnv(opts...)
}

func lookupEnv(suffix string) (string, string) {
	return env.Lookup(suffix)
}
