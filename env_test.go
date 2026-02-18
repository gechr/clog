package clog

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func saveEnvPrefix(t *testing.T) {
	t.Helper()

	orig, _ := envPrefix.Load().(string)

	t.Cleanup(func() {
		if orig == "" {
			envPrefix.Store("")
		} else {
			envPrefix.Store(orig)
		}
	})
}

func TestGetEnvDefaultPrefix(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("CLOG_LOG_LEVEL", "debug")
	envPrefix.Store("")

	assert.Equal(t, "debug", getEnv(envLogLevel))
}

func TestGetEnvCustomPrefix(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("MYAPP_LOG_LEVEL", "trace")
	t.Setenv("CLOG_LOG_LEVEL", "info")
	envPrefix.Store("MYAPP")

	// Custom prefix takes precedence.
	assert.Equal(t, "trace", getEnv(envLogLevel))
}

func TestGetEnvCustomPrefixFallback(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("MYAPP_LOG_LEVEL", "")
	t.Setenv("CLOG_LOG_LEVEL", "warn")
	envPrefix.Store("MYAPP")

	// Empty custom prefix value falls back to CLOG.
	assert.Equal(t, "warn", getEnv(envLogLevel))
}

func TestGetEnvNoPrefix(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("CLOG_LOG_LEVEL", "")
	envPrefix.Store("")

	assert.Empty(t, getEnv(envLogLevel))
}

func TestSetEnvPrefix(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	saveEnvPrefix(t)

	Default = NewWriter(io.Discard)
	t.Setenv("MYAPP_LOG_LEVEL", "debug")
	t.Setenv("CLOG_LOG_LEVEL", "")

	SetEnvPrefix("MYAPP")

	assert.Equal(t, DebugLevel, Default.level)
	assert.True(t, Default.reportTimestamp)
}

func TestSetEnvPrefixFallbackToClog(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	saveEnvPrefix(t)

	Default = NewWriter(io.Discard)
	t.Setenv("MYAPP_LOG_LEVEL", "")
	t.Setenv("CLOG_LOG_LEVEL", "warn")

	SetEnvPrefix("MYAPP")

	assert.Equal(t, WarnLevel, Default.level)
}

func TestSetEnvPrefixTrimsUnderscores(t *testing.T) {
	saveEnvPrefix(t)

	SetEnvPrefix("MYAPP___")

	got, _ := envPrefix.Load().(string)
	assert.Equal(t, "MYAPP", got)
}

func TestSetEnvPrefixHyperlinkFormats(t *testing.T) {
	saveEnvPrefix(t)
	saveFormats(t)

	hyperlinkPathFormat.Store(nil)
	hyperlinkLineFormat.Store(nil)

	t.Setenv("MYAPP_HYPERLINK_PATH_FORMAT", "vscode://file{path}")
	t.Setenv("MYAPP_HYPERLINK_LINE_FORMAT", "vscode://file{path}:{line}")
	t.Setenv("CLOG_HYPERLINK_PATH_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_LINE_FORMAT", "")

	SetEnvPrefix("MYAPP")

	gotPath := hyperlinkPathFormat.Load()
	if assert.NotNil(t, gotPath) {
		assert.Equal(t, "vscode://file{path}", *gotPath)
	}

	gotLine := hyperlinkLineFormat.Load()
	if assert.NotNil(t, gotLine) {
		assert.Equal(t, "vscode://file{path}:{line}", *gotLine)
	}
}
