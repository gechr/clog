package clog

import (
	"bytes"
	"testing"

	"github.com/gechr/clog/style"
	"github.com/gechr/clog/theme"
	"github.com/stretchr/testify/require"
)

// resetThemePrefix clears the theme package's env prefix so theme.FromEnv reads
// the default CLOG_* variables regardless of test ordering.
func resetThemePrefix(t *testing.T) {
	t.Helper()
	theme.SetEnvPrefix("")
	t.Cleanup(func() { theme.SetEnvPrefix("") })
}

func TestNewDefersPrintThemeDetection(t *testing.T) {
	// A freshly constructed logger must not have queried the terminal yet; the
	// printer theme stays pending until the first colored write.
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	require.True(t, l.printThemeDirty)
	require.Nil(t, l.printThemePair)
}

func TestResolvePrintThemeUsesFallbackBackground(t *testing.T) {
	// ColorAlways on a non-terminal writer: detection fails, so the pair's
	// fallback background decides the theme.
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetTheme(theme.DefaultPair(theme.WithFallback(theme.BackgroundLight)))

	l.mu.Lock()
	l.resolvePrintThemeLocked()
	l.mu.Unlock()

	require.False(t, l.printThemeDirty)
	require.Equal(t, style.BacktickFor(theme.BackgroundLight), l.styles.Backtick)
	require.Equal(t, style.NewJSON(theme.Light()), l.styles.JSON)
}

func TestResolvePrintThemeDeferredWhenColorsDisabled(t *testing.T) {
	// Non-TTY / colors disabled: never query the terminal, stay pending.
	l := New(TestOutput(&bytes.Buffer{}))

	l.mu.Lock()
	l.resolvePrintThemeLocked()
	l.mu.Unlock()

	require.True(t, l.printThemeDirty)
}

func TestSetThemeSingleAppliesRegardlessOfBackground(t *testing.T) {
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetTheme(theme.Single(theme.Monokai()))
	resolved(t, l)

	require.False(t, l.printThemeDirty)
	require.Equal(t, style.NewJSON(theme.Monokai()), l.styles.JSON)
}

func TestSetThemeNilRestoresDefaultPair(t *testing.T) {
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetTheme(theme.Single(theme.Monokai()))
	l.SetTheme(nil)

	require.True(t, l.printThemeDirty)
	require.Nil(t, l.printThemePair)
}

func TestSetStylesNilReenablesAutoDetection(t *testing.T) {
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetTheme(theme.Single(theme.Monokai()))
	l.SetStyles(nil)

	require.True(t, l.printThemeDirty)
	require.Nil(t, l.printThemePair)
}

// lightLogger returns a logger whose background resolves to light (detection is
// unavailable on a buffer, so the pair fallback decides).
func lightLogger(t *testing.T) *Logger {
	t.Helper()
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetTheme(theme.DefaultPair(theme.WithFallback(theme.BackgroundLight)))
	return l
}

func resolved(t *testing.T, l *Logger) {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.resolvePrintThemeLocked()
}

func TestSetStylesDefaultPassthroughStillAdapts(t *testing.T) {
	// Passing DefaultStyles() through (as apps do when tweaking unrelated
	// fields) must NOT freeze the gradient/printer styles at their dark
	// defaults: they still adapt to the terminal background.
	l := lightLogger(t)
	l.SetStyles(DefaultStyles())
	resolved(t, l)

	require.Equal(t, style.BacktickFor(theme.BackgroundLight), l.styles.Backtick)
	require.Equal(t, style.ElapsedGradientFor(theme.BackgroundLight), l.styles.ElapsedGradient)
	require.Equal(t, style.ElapsedGradientFor(theme.BackgroundLight), l.styles.DurationGradient)
	require.Equal(t, style.PercentGradientFor(theme.BackgroundLight), l.styles.PercentGradient)
	require.Equal(t, style.NewJSON(theme.Light()), l.styles.JSON)
}

func TestSetStylesCustomBacktickPreserved(t *testing.T) {
	l := lightLogger(t)
	custom := style.BacktickFor(theme.BackgroundDark).Bold(true)
	l.SetStyles(&style.Config{Backtick: &custom})
	resolved(t, l)

	require.Equal(t, &custom, l.styles.Backtick)
}

func TestSetStylesCustomGradientPreserved(t *testing.T) {
	// A gradient that differs from the default is a deliberate override and
	// survives background resolution.
	l := lightLogger(t)
	custom := []style.ColorStop{{Position: 0, Color: style.DefaultElapsedGradient()[2].Color}}
	l.SetStyles(&style.Config{ElapsedGradient: custom})
	resolved(t, l)

	// Overriding any gradient opts the gradient set out of adaptation, so the
	// custom value is kept verbatim rather than replaced by the light stops.
	require.Equal(t, custom, l.styles.ElapsedGradient)
}

func TestSetStylesPerStyleOverrideIsGranular(t *testing.T) {
	// Overriding one printer style must not opt the others out of adaptation.
	l := lightLogger(t)
	customJSON := style.NewJSON(theme.Monokai())
	l.SetStyles(&style.Config{JSON: customJSON})
	resolved(t, l)

	require.Equal(t, customJSON, l.styles.JSON)
	require.Equal(t, style.NewYAML(theme.Light()), l.styles.YAML)
	require.Equal(t, style.NewHCL(theme.Light()), l.styles.HCL)
}

func TestLoadThemeFromEnvExplicit(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()
	saveEnvPrefix(t)
	resetThemePrefix(t)

	Default = New(NewOutput(&bytes.Buffer{}, ColorAlways))
	t.Setenv("CLOG_THEME", "monokai")
	// The pair vars are present but the explicit theme must win.
	t.Setenv("CLOG_THEME_LIGHT", "light")
	t.Setenv("CLOG_THEME_DARK", "dark")

	loadThemeFromEnv()
	resolved(t, Default)

	// The explicit theme is wrapped via theme.Single, so both backgrounds
	// render Monokai.
	require.False(t, Default.printThemeDirty)
	require.NotNil(t, Default.printThemePair)
	require.Equal(t, style.NewJSON(theme.Monokai()), Default.styles.JSON)
}

func TestLoadThemeFromEnvPair(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()
	saveEnvPrefix(t)
	resetThemePrefix(t)

	Default = New(NewOutput(&bytes.Buffer{}, ColorAlways))
	t.Setenv("CLOG_THEME_LIGHT", "catppuccin-latte")
	t.Setenv("CLOG_THEME_DARK", "dracula")

	loadThemeFromEnv()

	require.True(t, Default.printThemeDirty)
	require.NotNil(t, Default.printThemePair)
	require.Equal(t, theme.CatppuccinLatte().Name(), Default.printThemePair.Light.Name())
	require.Equal(t, theme.Dracula().Name(), Default.printThemePair.Dark.Name())
}

func TestLoadThemeFromEnvUnsetLeavesDefault(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()
	saveEnvPrefix(t)
	resetThemePrefix(t)

	Default = New(NewOutput(&bytes.Buffer{}, ColorAlways))
	Default.SetTheme(theme.Single(theme.Monokai()))
	resolved(t, Default)

	loadThemeFromEnv()

	// No theme env vars set: the explicitly chosen theme is left intact.
	require.False(t, Default.printThemeDirty)
	require.Equal(t, style.NewJSON(theme.Monokai()), Default.styles.JSON)
}
