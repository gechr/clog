package theme

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// DefaultEnvPrefix is the default environment variable prefix.
const DefaultEnvPrefix = "CLOG"

const (
	envTheme      = "THEME"
	envThemeDark  = "THEME_DARK"
	envThemeLight = "THEME_LIGHT"
)

var envPrefix atomic.Value // stores string; "" means no custom prefix

// SetEnvPrefix sets a custom environment variable prefix.
//
//	theme.SetEnvPrefix("MYAPP")
//	// Now checks MYAPP_THEME_LIGHT/MYAPP_THEME_DARK first, then CLOG_THEME_LIGHT/CLOG_THEME_DARK
func SetEnvPrefix(prefix string) {
	envPrefix.Store(strings.TrimRight(prefix, "_"))
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
//  1. <PREFIX>_THEME selects an explicit theme, applied without background
//     detection.
//  2. <PREFIX>_THEME_LIGHT and <PREFIX>_THEME_DARK select a light/dark pair.
//
// It returns (theme, nil, nil) for case 1, (nil, pair, nil) for case 2, and
// (nil, nil, nil) when no theme variables are set. A set-but-invalid value
// returns an error.
func FromEnv(opts ...PairOption) (*Theme, *Pair, error) {
	if name, env := lookupEnv(envTheme); strings.TrimSpace(name) != "" {
		var t Theme
		if err := t.UnmarshalText([]byte(strings.TrimSpace(name))); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", env, err)
		}
		return &t, nil, nil
	}

	lightName, _ := lookupEnv(envThemeLight)
	darkName, _ := lookupEnv(envThemeDark)
	if strings.TrimSpace(lightName) == "" && strings.TrimSpace(darkName) == "" {
		return nil, nil, nil
	}

	pair, err := PairFromEnv(opts...)
	if err != nil {
		return nil, nil, err
	}
	return nil, pair, nil
}

func lookupEnv(suffix string) (string, string) {
	if p, ok := envPrefix.Load().(string); ok && p != "" {
		name := p + "_" + suffix
		if v := os.Getenv(name); v != "" {
			return v, name
		}
	}
	name := DefaultEnvPrefix + "_" + suffix
	return os.Getenv(name), name
}
