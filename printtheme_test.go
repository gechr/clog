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
	l.setPrintPair(theme.DefaultPair(theme.WithFallback(theme.BackgroundLight)))

	l.mu.Lock()
	l.resolvePrintThemeLocked()
	l.mu.Unlock()

	require.False(t, l.printThemeDirty)
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

func TestSetPrintThemeDisablesAutoDetection(t *testing.T) {
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetPrintTheme(theme.Monokai())

	require.False(t, l.printThemeDirty)
	require.Nil(t, l.printThemePair)
	require.Equal(t, style.NewJSON(theme.Monokai()), l.styles.JSON)
}

func TestSetPrintThemeNilRestoresAutoDetection(t *testing.T) {
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetPrintTheme(theme.Monokai())
	l.SetPrintTheme(nil)

	require.True(t, l.printThemeDirty)
	require.Nil(t, l.printThemePair)
}

func TestSetStylesNilReenablesAutoDetection(t *testing.T) {
	l := New(NewOutput(&bytes.Buffer{}, ColorAlways))
	l.SetPrintTheme(theme.Monokai())
	l.SetStyles(nil)

	require.True(t, l.printThemeDirty)
	require.Nil(t, l.printThemePair)
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

	require.False(t, Default.printThemeDirty)
	require.Nil(t, Default.printThemePair)
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
	Default.SetPrintTheme(theme.Monokai())

	loadThemeFromEnv()

	// No theme env vars set: the explicitly chosen theme is left intact.
	require.False(t, Default.printThemeDirty)
	require.Equal(t, style.NewJSON(theme.Monokai()), Default.styles.JSON)
}
