package clog

import (
	"io"
	"testing"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/gechr/clog/internal/env"
	"github.com/stretchr/testify/assert"
)

func saveEnvPrefix(t *testing.T) {
	t.Helper()

	orig := env.Prefix()

	t.Cleanup(func() { env.SetPrefix(orig) })
}

func TestGetEnvDefaultPrefix(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("CLOG_LOG_LEVEL", "debug")
	env.SetPrefix("")

	assert.Equal(t, "debug", getEnv(envLogLevel))
}

func TestGetEnvCustomPrefix(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("MYAPP_LOG_LEVEL", "trace")
	t.Setenv("CLOG_LOG_LEVEL", "info")
	env.SetPrefix("MYAPP")

	// Custom prefix takes precedence.
	assert.Equal(t, "trace", getEnv(envLogLevel))
}

func TestGetEnvCustomPrefixFallback(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("MYAPP_LOG_LEVEL", "")
	t.Setenv("CLOG_LOG_LEVEL", "warn")
	env.SetPrefix("MYAPP")

	// Empty custom prefix value falls back to CLOG.
	assert.Equal(t, "warn", getEnv(envLogLevel))
}

func TestGetEnvNoPrefix(t *testing.T) {
	saveEnvPrefix(t)

	t.Setenv("CLOG_LOG_LEVEL", "")
	env.SetPrefix("")

	assert.Empty(t, getEnv(envLogLevel))
}

func TestSetEnvPrefix(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	saveEnvPrefix(t)

	SetDefault(NewWriter(io.Discard))
	t.Setenv("MYAPP_LOG_LEVEL", "debug")
	t.Setenv("CLOG_LOG_LEVEL", "")

	SetEnvPrefix("MYAPP")

	assert.Equal(t, LevelDebug, Default().level)
	assert.True(t, Default().reportTimestamp)
}

func TestSetEnvPrefixFallbackToClog(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	saveEnvPrefix(t)

	SetDefault(NewWriter(io.Discard))
	t.Setenv("MYAPP_LOG_LEVEL", "")
	t.Setenv("CLOG_LOG_LEVEL", "warn")

	SetEnvPrefix("MYAPP")

	assert.Equal(t, LevelWarn, Default().level)
}

func TestSetEnvPrefixTrimsUnderscores(t *testing.T) {
	saveEnvPrefix(t)

	SetEnvPrefix("MYAPP___")

	assert.Equal(t, "MYAPP", env.Prefix())
}

func TestEnvLogLevelWhitespaceTrimming(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	saveEnvPrefix(t)

	for _, tt := range []struct {
		name  string
		value string
		want  Level
	}{
		{"leading space", " debug", LevelDebug},
		{"trailing space", "debug ", LevelDebug},
		{"both spaces", " debug ", LevelDebug},
		{"tabs", "\tdebug\t", LevelDebug},
		{"newline", "warn\n", LevelWarn},
	} {
		t.Run(tt.name, func(t *testing.T) {
			SetDefault(NewWriter(io.Discard))
			t.Setenv("CLOG_LOG_LEVEL", tt.value)
			env.SetPrefix("")

			loadLogLevelFromEnv()

			assert.Equal(t, tt.want, Default().level)
		})
	}
}

func TestEnvLoadAllFromEnvReChecksNoColor(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	saveEnvPrefix(t)

	SetDefault(NewWriter(io.Discard))

	// Start with NO_COLOR unset.
	t.Setenv("NO_COLOR", "")
	// Remove it so LookupEnv returns false.
	// t.Setenv registers cleanup, but we need it unset now.
	// We'll use a different approach: set it, call loadAllFromEnv, check.

	// First: NO_COLOR is set -> should be true after loadAllFromEnv.
	t.Setenv("NO_COLOR", "1")
	loadAllFromEnv()
	assert.True(t, noColorEnvSet.Load(), "noColorEnvSet should be true when NO_COLOR is set")
}

func TestEnvHyperlinkPresetApplied(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	saveEnvPrefix(t)

	SetDefault(NewWriter(io.Discard))
	t.Setenv("CLOG_HYPERLINK_FORMAT", "vscode")
	// Individual format vars override the preset.
	t.Setenv("CLOG_HYPERLINK_PATH_FORMAT", "custom://{path}")
	env.SetPrefix("")

	loadHyperlinkFormatsFromEnv()

	formats := Default().FieldFormats()
	assert.Equal(t, "custom://{path}", formats.HyperlinkPathFormat)
	assert.Equal(t, "vscode://file{path}:{line}", formats.HyperlinkLineFormat)
	assert.Equal(t, "vscode://file{path}:{line}:{column}", formats.HyperlinkColumnFormat)
}

func TestSetEnvPrefixHyperlinkFormats(t *testing.T) {
	origDefault := Default()
	defer func() { SetDefault(origDefault) }()

	saveEnvPrefix(t)

	SetDefault(NewWriter(io.Discard))
	t.Setenv("MYAPP_HYPERLINK_PATH_FORMAT", "vscode://file{path}")
	t.Setenv("MYAPP_HYPERLINK_LINE_FORMAT", "vscode://file{path}:{line}")
	t.Setenv("CLOG_HYPERLINK_PATH_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_LINE_FORMAT", "")

	SetEnvPrefix("MYAPP")

	formats := Default().FieldFormats()
	assert.Equal(t, "vscode://file{path}", formats.HyperlinkPathFormat)
	assert.Equal(t, "vscode://file{path}:{line}", formats.HyperlinkLineFormat)

	cfg := hyperlink.Config{
		Enabled:    true,
		PathFormat: formats.HyperlinkPathFormat,
		LineFormat: formats.HyperlinkLineFormat,
	}

	// Path format applied to file-only URL.
	got := cfg.ResolvePathURL("/test/file.go", 0, 0)
	assert.Equal(t, "vscode://file/test/file.go", got)

	// Line format applied to file+line URL.
	got = cfg.ResolvePathURL("/test/file.go", 42, 0)
	assert.Equal(t, "vscode://file/test/file.go:42", got)
}
