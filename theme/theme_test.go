package theme_test

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/theme"
	"github.com/stretchr/testify/require"
)

func resetThemeEnvPrefix(t *testing.T) {
	t.Helper()
	theme.SetEnvPrefix("")
	t.Cleanup(func() {
		theme.SetEnvPrefix("")
	})
}

func TestDarkPreservesPreviousDefaultPalette(t *testing.T) {
	th := theme.Dark()

	require.Equal(t, "dark", th.Name())
	require.Equal(t, theme.BackgroundDark, th.Background)
	require.Equal(t, lipgloss.Color("#8be9fd"), th.Accent)
	require.Equal(t, lipgloss.Color("#ff5555"), th.BoolFalse)
	require.Equal(t, lipgloss.Color("#50fa7b"), th.BoolTrue)
	require.Equal(t, lipgloss.Color("#6272a4"), th.Comment)
	require.Equal(t, lipgloss.Color("#f8f8f2"), th.Foreground)
	require.Equal(t, lipgloss.Color("#bd93f9"), th.Key)
	require.Equal(t, lipgloss.Color("#ff79c6"), th.Number)
	require.Equal(t, lipgloss.Color("#ffb86c"), th.Secondary)
	require.Equal(t, lipgloss.Color("#f1fa8c"), th.String)
	requireSamePalette(t, th, theme.Dracula())
}

func TestLightPalette(t *testing.T) {
	th := theme.Light()

	require.Equal(t, "light", th.Name())
	require.Equal(t, theme.BackgroundLight, th.Background)
	require.Equal(t, lipgloss.Color("#006d75"), th.Accent)
	require.Equal(t, lipgloss.Color("#a11d33"), th.BoolFalse)
	require.Equal(t, lipgloss.Color("#256d1b"), th.BoolTrue)
	require.Equal(t, lipgloss.Color("#5f6368"), th.Comment)
	require.Equal(t, lipgloss.Color("#3c4043"), th.Foreground)
	require.Equal(t, lipgloss.Color("#2459b3"), th.Key)
	require.Equal(t, lipgloss.Color("#9a4d00"), th.Number)
	require.Equal(t, lipgloss.Color("#7047b5"), th.Secondary)
	require.Equal(t, lipgloss.Color("#256d1b"), th.String)
}

func TestAutoUsesFallbackWhenDetectionUnavailable(t *testing.T) {
	resetThemeEnvPrefix(t)

	require.Equal(t, theme.Dark().Name(), theme.Auto(nil).Name())
	require.Equal(
		t,
		theme.Light().Name(),
		theme.DefaultPair(theme.WithFallback(theme.BackgroundLight)).Auto(nil).Name(),
	)
}

func TestDefaultPairForBackground(t *testing.T) {
	pair := theme.DefaultPair()

	require.Equal(t, theme.Light().Name(), pair.ForBackground(theme.BackgroundLight).Name())
	require.Equal(t, theme.Dark().Name(), pair.ForBackground(theme.BackgroundDark).Name())
}

func TestPairFromEnv(t *testing.T) {
	resetThemeEnvPrefix(t)
	t.Setenv("CLOG_THEME_LIGHT", "light")
	t.Setenv("CLOG_THEME_DARK", "dracula")

	pair, err := theme.PairFromEnv()
	require.NoError(t, err)
	require.Equal(t, theme.Light().Name(), pair.Light.Name())
	require.Equal(t, theme.Dracula().Name(), pair.Dark.Name())
}

func TestPairFromEnvCustomEnvPrefixTakesPrecedence(t *testing.T) {
	resetThemeEnvPrefix(t)
	theme.SetEnvPrefix("MYAPP")
	t.Setenv("MYAPP_THEME_LIGHT", "catppuccin-latte")
	t.Setenv("MYAPP_THEME_DARK", "monokai")
	t.Setenv("CLOG_THEME_LIGHT", "light")
	t.Setenv("CLOG_THEME_DARK", "dark")

	pair, err := theme.PairFromEnv()
	require.NoError(t, err)
	require.Equal(t, theme.CatppuccinLatte().Name(), pair.Light.Name())
	require.Equal(t, theme.Monokai().Name(), pair.Dark.Name())
}

func TestPairFromEnvRequiresBothThemes(t *testing.T) {
	resetThemeEnvPrefix(t)
	t.Setenv("CLOG_THEME_LIGHT", "light")

	_, err := theme.PairFromEnv()
	require.EqualError(t, err, "CLOG_THEME_DARK is required")
}

func TestPairFromEnvRejectsWrongBackground(t *testing.T) {
	resetThemeEnvPrefix(t)
	t.Setenv("CLOG_THEME_LIGHT", "dracula")
	t.Setenv("CLOG_THEME_DARK", "catppuccin-latte")

	_, err := theme.PairFromEnv()
	require.EqualError(t, err, `light theme must declare background "light", got "dark"`)
}

func TestFromEnvExplicitTakesPrecedence(t *testing.T) {
	resetThemeEnvPrefix(t)
	t.Setenv("CLOG_THEME", "monokai")
	t.Setenv("CLOG_THEME_LIGHT", "light")
	t.Setenv("CLOG_THEME_DARK", "dark")

	pair, err := theme.FromEnv()
	require.NoError(t, err)
	require.NotNil(t, pair)
	// The explicit theme wins and is applied on both backgrounds.
	require.Equal(t, theme.Monokai().Name(), pair.Light.Name())
	require.Equal(t, theme.Monokai().Name(), pair.Dark.Name())
}

func TestFromEnvPair(t *testing.T) {
	resetThemeEnvPrefix(t)
	t.Setenv("CLOG_THEME_LIGHT", "catppuccin-latte")
	t.Setenv("CLOG_THEME_DARK", "dracula")

	pair, err := theme.FromEnv()
	require.NoError(t, err)
	require.NotNil(t, pair)
	require.Equal(t, theme.CatppuccinLatte().Name(), pair.Light.Name())
	require.Equal(t, theme.Dracula().Name(), pair.Dark.Name())
}

func TestFromEnvUnset(t *testing.T) {
	resetThemeEnvPrefix(t)

	pair, err := theme.FromEnv()
	require.NoError(t, err)
	require.Nil(t, pair)
}

func TestFromEnvExplicitInvalid(t *testing.T) {
	resetThemeEnvPrefix(t)
	t.Setenv("CLOG_THEME", "bogus")

	_, err := theme.FromEnv()
	require.EqualError(t, err, "CLOG_THEME: "+unknownThemeError("bogus"))
}

func TestFromEnvCustomPrefix(t *testing.T) {
	resetThemeEnvPrefix(t)
	theme.SetEnvPrefix("MYAPP")
	t.Setenv("MYAPP_THEME", "monokai")
	t.Setenv("CLOG_THEME", "light")

	pair, err := theme.FromEnv()
	require.NoError(t, err)
	require.NotNil(t, pair)
	require.Equal(t, theme.Monokai().Name(), pair.Light.Name())
}

func TestThemeUnmarshalText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  *theme.Theme
	}{
		{name: "dark", input: "dark", want: theme.Dark()},
		{name: "light", input: "LIGHT", want: theme.Light()},
		{name: "monokai", input: "MONOKAI", want: theme.Monokai()},
		{name: "catppuccin hyphen", input: "catppuccin-mocha", want: theme.CatppuccinMocha()},
		{
			name:  "catppuccin underscore",
			input: "catppuccin_macchiato",
			want:  theme.CatppuccinMacchiato(),
		},
		{name: "catppuccin compact", input: "catppuccinfrappe", want: theme.CatppuccinFrappe()},
		{name: "dracula", input: "dracula", want: theme.Dracula()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got theme.Theme
			require.NoError(t, got.UnmarshalText([]byte(tt.input)))
			require.Equal(t, tt.want.Name(), got.Name())
			requireSamePalette(t, tt.want, &got)
		})
	}
}

func TestThemeMarshalText(t *testing.T) {
	got, err := theme.Light().MarshalText()
	require.NoError(t, err)
	require.Equal(t, "light", string(got))
}

func TestThemeMarshalTextCustom(t *testing.T) {
	_, err := (&theme.Theme{Background: theme.BackgroundDark}).MarshalText()
	require.EqualError(t, err, "cannot marshal custom theme")
}

func TestThemeUnmarshalTextInvalid(t *testing.T) {
	var got theme.Theme
	err := got.UnmarshalText([]byte("bogus"))
	require.EqualError(t, err, unknownThemeError("bogus"))
}

func unknownThemeError(input string) string {
	valid := []string{
		"dark",
		"light",
		"catppuccin-frappe",
		"catppuccin-latte",
		"catppuccin-macchiato",
		"catppuccin-mocha",
		"dracula",
		"monokai",
	}
	return fmt.Sprintf("unknown theme %q (valid: %s)", input, strings.Join(valid, ", "))
}

func requireSamePalette(t *testing.T, a, b *theme.Theme) {
	t.Helper()

	require.Equal(t, a.Background, b.Background)
	require.Equal(t, a.Accent, b.Accent)
	require.Equal(t, a.BoolFalse, b.BoolFalse)
	require.Equal(t, a.BoolTrue, b.BoolTrue)
	require.Equal(t, a.Comment, b.Comment)
	require.Equal(t, a.Foreground, b.Foreground)
	require.Equal(t, a.Key, b.Key)
	require.Equal(t, a.Number, b.Number)
	require.Equal(t, a.Secondary, b.Secondary)
	require.Equal(t, a.String, b.String)
}
