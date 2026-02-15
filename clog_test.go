package clog

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)

	assert.Equal(t, InfoLevel, l.level)
	assert.Nil(t, l.prefix)
	assert.NotNil(t, l.mu)
	assert.Nil(t, l.handler)
	assert.Equal(t, "15:04:05.000", l.timeFormat)
	assert.False(t, l.reportTimestamp)
	assert.NotNil(t, l.styles)
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{TraceLevel, "TRC"},
		{DebugLevel, "DBG"},
		{InfoLevel, "INF"},
		{DryLevel, "DRY"},
		{WarnLevel, "WRN"},
		{ErrorLevel, "ERR"},
		{FatalLevel, "FTL"},
		{Level(99), "LVL(99)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func TestLevelFiltering(t *testing.T) {
	tests := []struct {
		name     string
		logLevel Level
		method   func(*Logger) *Event
		wantNil  bool
	}{
		{"trace_at_trace", TraceLevel, (*Logger).Trace, false},
		{"trace_at_debug", DebugLevel, (*Logger).Trace, true},
		{"trace_at_info", InfoLevel, (*Logger).Trace, true},
		{"debug_at_info", InfoLevel, (*Logger).Debug, true},
		{"info_at_info", InfoLevel, (*Logger).Info, false},
		{"dry_at_info", InfoLevel, (*Logger).Dry, false},
		{"warn_at_info", InfoLevel, (*Logger).Warn, false},
		{"error_at_info", InfoLevel, (*Logger).Error, false},
		{"fatal_at_info", InfoLevel, (*Logger).Fatal, false},
		{"debug_at_trace", TraceLevel, (*Logger).Debug, false},
		{"debug_at_debug", DebugLevel, (*Logger).Debug, false},
		{"info_at_warn", WarnLevel, (*Logger).Info, true},
		{"dry_at_warn", WarnLevel, (*Logger).Dry, true},
		{"warn_at_warn", WarnLevel, (*Logger).Warn, false},
		{"error_at_error", ErrorLevel, (*Logger).Error, false},
		{"warn_at_error", ErrorLevel, (*Logger).Warn, true},
		{"error_at_fatal", FatalLevel, (*Logger).Error, true},
		{"fatal_at_fatal", FatalLevel, (*Logger).Fatal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(io.Discard)
			l.SetLevel(tt.logLevel)

			e := tt.method(l)
			if tt.wantNil {
				assert.Nil(t, e, "expected nil event")
			} else {
				assert.NotNil(t, e, "expected non-nil event")
			}
		})
	}
}

func TestSetLevel(t *testing.T) {
	l := New(io.Discard)

	l.SetLevel(DebugLevel)
	assert.Equal(t, DebugLevel, l.level)

	l.SetLevel(ErrorLevel)
	assert.Equal(t, ErrorLevel, l.level)
}

func TestSetLevelFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		wantLevel     Level
		wantTimestamp bool
	}{
		{"trace", "trace", TraceLevel, true},
		{"debug", "debug", DebugLevel, true},
		{"info", "info", InfoLevel, false},
		{"dry", "dry", DryLevel, false},
		{"warn", "warn", WarnLevel, false},
		{"warning", "warning", WarnLevel, false},
		{"error", "error", ErrorLevel, false},
		{"fatal", "fatal", FatalLevel, false},
		{"case_insensitive", "DEBUG", DebugLevel, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origDefault := Default
			defer func() { Default = origDefault }()

			Default = New(io.Discard)
			t.Setenv("TEST_CLOG_LEVEL", tt.value)
			SetLevelFromEnv("TEST_CLOG_LEVEL")

			assert.Equal(t, tt.wantLevel, Default.level)
			assert.Equal(t, tt.wantTimestamp, Default.reportTimestamp)
		})
	}
}

func TestSetLevelFromEnvNotSet(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	Default.SetLevel(WarnLevel)

	SetLevelFromEnv("CLOG_TEST_NONEXISTENT_VAR")

	assert.Equal(t, WarnLevel, Default.level)
}

func TestGetLevel(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	Default.SetLevel(WarnLevel)

	assert.Equal(t, WarnLevel, GetLevel())
}

func TestIsVerbose(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)

	Default.SetLevel(InfoLevel)
	assert.False(t, IsVerbose(), "expected IsVerbose() false at InfoLevel")

	Default.SetLevel(DebugLevel)
	assert.True(t, IsVerbose(), "expected IsVerbose() true at DebugLevel")

	Default.SetLevel(TraceLevel)
	assert.True(t, IsVerbose(), "expected IsVerbose() true at TraceLevel")
}

func TestResolvePrefix(t *testing.T) {
	tests := []struct {
		name         string
		loggerPrefix *string
		eventPrefix  *string
		level        Level
		want         string
	}{
		{name: "default_info", level: InfoLevel, want: "ℹ️"},
		{name: "default_trace", level: TraceLevel, want: "🔬"},
		{name: "default_debug", level: DebugLevel, want: "🔍"},
		{name: "default_warn", level: WarnLevel, want: "⚠️"},
		{name: "default_error", level: ErrorLevel, want: "❌"},
		{name: "default_fatal", level: FatalLevel, want: "💥"},
		{name: "default_dry", level: DryLevel, want: "🚧"},
		{
			name:         "logger_prefix",
			loggerPrefix: new("LOG"),
			level:        InfoLevel,
			want:         "LOG",
		},
		{
			name:         "event_overrides_logger",
			loggerPrefix: new("LOG"),
			eventPrefix:  new("EVT"),
			level:        InfoLevel,
			want:         "EVT",
		},
		{
			name:        "event_overrides_default",
			eventPrefix: new("EVT"),
			level:       InfoLevel,
			want:        "EVT",
		},
		{
			name:         "empty_logger_prefix",
			loggerPrefix: new(""),
			level:        InfoLevel,
			want:         "",
		},
		{
			name:        "empty_event_prefix",
			eventPrefix: new(""),
			level:       InfoLevel,
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(io.Discard)
			l.prefix = tt.loggerPrefix

			e := &Event{logger: l, level: tt.level, prefix: tt.eventPrefix}

			got := l.resolvePrefix(e)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigure(t *testing.T) {
	t.Run("verbose", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)
		Configure(&Config{Verbose: true})

		assert.Equal(t, DebugLevel, Default.level)
		assert.True(t, Default.reportTimestamp)
	})

	t.Run("output", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)

		var buf bytes.Buffer

		Configure(&Config{Output: &buf})

		Default.mu.Lock()
		out := Default.out
		Default.mu.Unlock()

		assert.Equal(t, &buf, out)
	})

	t.Run("styles", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)
		styles := DefaultStyles()
		Configure(&Config{Styles: styles})

		Default.mu.Lock()
		got := Default.styles
		Default.mu.Unlock()

		assert.Equal(t, styles, got)
	})

	t.Run("nil_config", func(_ *testing.T) {
		Configure(nil)
	})

	t.Run("non_verbose_without_env", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)
		Default.SetLevel(DebugLevel)
		Default.SetReportTimestamp(true)
		t.Setenv(DefaultEnvLogLevel, "")

		Configure(&Config{Verbose: false})

		assert.Equal(t, InfoLevel, Default.level)
		assert.False(t, Default.reportTimestamp)
	})

	t.Run("non_verbose_with_env", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)
		Default.SetLevel(DebugLevel)
		t.Setenv(DefaultEnvLogLevel, "debug")

		Configure(&Config{Verbose: false})

		assert.Equal(t, DebugLevel, Default.level)
	})
}

func TestConfigureVerbose(t *testing.T) {
	t.Run("enable", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)
		ConfigureVerbose(true)

		assert.Equal(t, DebugLevel, Default.level)
		assert.True(t, Default.reportTimestamp)
	})

	t.Run("disable_without_env", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)
		Default.SetLevel(DebugLevel)
		t.Setenv(DefaultEnvLogLevel, "")

		ConfigureVerbose(false)

		assert.Equal(t, InfoLevel, Default.level)
	})

	t.Run("disable_with_env", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = New(io.Discard)
		Default.SetLevel(DebugLevel)
		t.Setenv(DefaultEnvLogLevel, "debug")

		ConfigureVerbose(false)

		assert.Equal(t, DebugLevel, Default.level)
	})
}

func TestPackageLevelConvenienceFunctions(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	Default.SetLevel(TraceLevel)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	tests := []struct {
		name  string
		fn    func() *Event
		level Level
	}{
		{"Trace", Trace, TraceLevel},
		{"Debug", Debug, DebugLevel},
		{"Info", Info, InfoLevel},
		{"Dry", Dry, DryLevel},
		{"Warn", Warn, WarnLevel},
		{"Error", Error, ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn().Msg("test")

			assert.Equal(t, tt.level, got.Level)
			assert.Equal(t, "test", got.Message)
		})
	}
}

func TestPackageLevelWith(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)

	ctx := With()
	assert.NotNil(t, ctx, "expected non-nil context from With()")
}

func TestPackageLevelSetters(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)

	SetLevel(WarnLevel)
	assert.Equal(t, WarnLevel, Default.level)

	SetReportTimestamp(true)
	assert.True(t, Default.reportTimestamp)

	SetTimeFormat("2006-01-02")
	assert.Equal(t, "2006-01-02", Default.timeFormat)

	h := HandlerFunc(func(_ Entry) {})
	SetHandler(h)
	assert.NotNil(t, Default.handler)

	var buf bytes.Buffer

	SetOutput(&buf)

	Default.mu.Lock()
	out := Default.out
	Default.mu.Unlock()

	assert.Equal(t, &buf, out)

	styles := DefaultStyles()
	SetStyles(styles)

	Default.mu.Lock()
	gotStyles := Default.styles
	Default.mu.Unlock()

	assert.Equal(t, styles, gotStyles)

	var exitCode int

	SetExitFunc(func(code int) { exitCode = code })

	Default.mu.Lock()
	fn := Default.exitFunc
	Default.mu.Unlock()

	require.NotNil(t, fn)

	fn(2)

	assert.Equal(t, 2, exitCode)
}

func TestCustomHandlerReceivesEntries(t *testing.T) {
	l := New(io.Discard)

	var entries []Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		entries = append(entries, e)
	}))

	l.Info().Str("a", "1").Msg("first")
	l.Warn().Str("b", "2").Msg("second")

	require.Len(t, entries, 2)
	assert.Equal(t, InfoLevel, entries[0].Level)
	assert.Equal(t, "first", entries[0].Message)
	assert.Equal(t, WarnLevel, entries[1].Level)
	assert.Equal(t, "second", entries[1].Message)
}

func TestCustomHandlerNoBufferOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetHandler(HandlerFunc(func(_ Entry) {}))

	l.Info().Msg("intercepted")

	assert.Zero(t, buf.Len(), "expected no output to buffer when handler is set")
}

func TestSubLoggerWithWith(t *testing.T) {
	l := New(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	sub := l.With().Str("component", "auth").Logger()
	sub.Info().Str("user", "john").Msg("login")

	assert.Equal(t, "login", got.Message)
	require.Len(t, got.Fields, 2)
	assert.Equal(t, "component", got.Fields[0].Key)
	assert.Equal(t, "auth", got.Fields[0].Value)
	assert.Equal(t, "user", got.Fields[1].Key)
	assert.Equal(t, "john", got.Fields[1].Value)
}

func TestWithSharesMutex(t *testing.T) {
	l := New(io.Discard)
	sub := l.With().Str("k", "v").Logger()

	assert.Same(t, l.mu, sub.mu, "sub-logger should share parent's mutex")
}

func TestWithCopiesFields(t *testing.T) {
	l := New(io.Discard)
	l.fields = []Field{{Key: "parent", Value: "yes"}}

	ctx := l.With()
	ctx.Str("child", "added")

	assert.Len(t, l.fields, 1, "parent fields should not be modified")
}

func TestEventFieldsDoNotModifyLogger(t *testing.T) {
	l := New(io.Discard)
	l.fields = []Field{{Key: "ctx", Value: "val"}}

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Str("event", "field").Msg("test")

	assert.Len(t, l.fields, 1, "logger fields should not be modified")
	assert.Len(t, got.Fields, 2)
}

func TestLogFormattedOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ hello\n", buf.String())
}

func TestLogFormattedOutputWithFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Str("key", "val").Msg("hello")

	assert.Equal(t, "INF ℹ️ hello key=val\n", buf.String())
}

func TestLogFormattedOutputCustomPrefix(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Prefix(">>>").Msg("hello")

	assert.Equal(t, "INF >>> hello\n", buf.String())
}

func TestLogFormattedOutputEmptyPrefix(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Prefix("").Msg("hello")

	assert.Equal(t, "INF hello\n", buf.String())
}

func TestLogFormattedOutputWithTimestamp(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetReportTimestamp(true)
	l.Info().Msg("hello")

	got := buf.String()

	assert.Contains(t, got, "INF")
	assert.Contains(t, got, "hello")
	assert.True(t, strings.HasSuffix(got, "\n"))
	// Timestamp format "HH:MM:SS.mmm" = 12 chars, plus trailing space.
	assert.GreaterOrEqual(t, len(got), 12, "output too short for timestamp")
}

func TestLogFormattedOutputQuotedFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=\"hello world\"\n", buf.String())
}

func TestLogFormattedOutputMultipleFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Str("a", "1").Int("b", 2).Bool("c", true).Msg("test")

	assert.Equal(t, "INF ℹ️ test a=1 b=2 c=true\n", buf.String())
}

func TestSetLevelFromEnvDry(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	t.Setenv("TEST_CLOG_LEVEL_DRY", "dry")
	SetLevelFromEnv("TEST_CLOG_LEVEL_DRY")

	assert.Equal(t, DryLevel, Default.level)
}

func TestSetLevelFromEnvFatal(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	t.Setenv("TEST_CLOG_LEVEL_FATAL", "fatal")
	SetLevelFromEnv("TEST_CLOG_LEVEL_FATAL")

	assert.Equal(t, FatalLevel, Default.level)
}

func TestSetLevelFromEnvUnrecognised(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	t.Setenv("TEST_CLOG_LEVEL_BAD", "bogus")

	// Should not change the level, just print to stderr.
	SetLevelFromEnv("TEST_CLOG_LEVEL_BAD")

	assert.Equal(t, InfoLevel, Default.level)
}

func TestSetSeparatorFromEnv(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	t.Setenv("TEST_CLOG_SEP", ":")
	SetSeparatorFromEnv("TEST_CLOG_SEP")

	Default.mu.Lock()
	got := Default.styles.SeparatorText
	Default.mu.Unlock()

	assert.Equal(t, ":", got)
}

func TestSetSeparatorFromEnvNotSet(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetSeparatorFromEnv("CLOG_TEST_NONEXISTENT_SEP")

	Default.mu.Lock()
	got := Default.styles.SeparatorText
	Default.mu.Unlock()

	assert.Equal(t, "=", got)
}

func TestSetLabels(t *testing.T) {
	l := New(io.Discard)
	l.SetLabels(LevelMap{WarnLevel: "WARN"}) //nolint:exhaustive // intentionally partial

	assert.Equal(t, "WARN", l.labels[WarnLevel])
	// Other labels should retain defaults.
	assert.Equal(t, "INF", l.labels[InfoLevel])
}

func TestSetLevelAlign(t *testing.T) {
	l := New(io.Discard)
	l.SetLevelAlign(AlignLeft)

	assert.Equal(t, AlignLeft, l.levelAlign)
}

func TestFormatLabelAlignNone(t *testing.T) {
	l := New(io.Discard)
	l.SetLevelAlign(AlignNone)

	assert.Equal(t, "INF", l.formatLabel(InfoLevel))
}

func TestFormatLabelAlignLeft(t *testing.T) {
	l := New(io.Discard)
	l.SetLabels(LevelMap{ //nolint:exhaustive // intentionally partial
		InfoLevel:  "INF",
		WarnLevel:  "WARN",
		ErrorLevel: "ERROR",
	})
	l.SetLevelAlign(AlignLeft)

	// maxLabelWidth is 5 (ERROR), so INF should be left-padded to 5 chars.
	assert.Equal(t, "INF  ", l.formatLabel(InfoLevel))
}

func TestFormatLabelAlignRight(t *testing.T) {
	l := New(io.Discard)
	l.SetLabels(LevelMap{ //nolint:exhaustive // intentionally partial
		InfoLevel:  "INF",
		WarnLevel:  "WARN",
		ErrorLevel: "ERROR",
	})
	l.SetLevelAlign(AlignRight)

	// maxLabelWidth is 5 (ERROR), so INF should be right-padded.
	assert.Equal(t, "  INF", l.formatLabel(InfoLevel))
}

func TestFormatLabelAlignCenter(t *testing.T) {
	l := New(io.Discard)
	l.SetLabels(LevelMap{ //nolint:exhaustive // intentionally partial
		InfoLevel:  "INF",
		WarnLevel:  "WARN",
		ErrorLevel: "ERROR",
	})
	l.SetLevelAlign(AlignCenter)

	// maxLabelWidth is 5 (ERROR), so INF (3) gets 1 left + 1 right padding.
	assert.Equal(t, " INF ", l.formatLabel(InfoLevel))
	// WARN (4) gets 0 left + 1 right.
	assert.Equal(t, "WARN ", l.formatLabel(WarnLevel))
	// ERROR (5) fits exactly.
	assert.Equal(t, "ERROR", l.formatLabel(ErrorLevel))
}

func TestFormatLabelUnknownAlign(t *testing.T) {
	l := New(io.Discard)
	l.levelAlign = LevelAlign(99) // invalid value

	assert.Equal(t, "INF", l.formatLabel(InfoLevel))
}

func TestSetPrefixes(t *testing.T) {
	l := New(io.Discard)
	l.SetPrefixes(LevelMap{InfoLevel: ">>>"}) //nolint:exhaustive // intentionally partial

	assert.Equal(t, ">>>", l.prefixes[InfoLevel])
	// Other prefixes should retain defaults.
	assert.Equal(t, "🔍", l.prefixes[DebugLevel])
}

func TestPackageLevelSetPrefixes(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetPrefixes(LevelMap{InfoLevel: ">>>"}) //nolint:exhaustive // intentionally partial

	assert.Equal(t, ">>>", Default.prefixes[InfoLevel])
}

func TestSetTimeLocation(t *testing.T) {
	l := New(io.Discard)
	loc := time.UTC
	l.SetTimeLocation(loc)

	assert.Equal(t, loc, l.timeLocation)
}

func TestPackageLevelSetTimeLocation(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	loc := time.UTC
	SetTimeLocation(loc)

	Default.mu.Lock()
	got := Default.timeLocation
	Default.mu.Unlock()

	assert.Equal(t, loc, got)
}

func TestDefaultPrefixes(t *testing.T) {
	p := DefaultPrefixes()

	assert.Equal(t, "ℹ️", p[InfoLevel])
	assert.Equal(t, "🔬", p[TraceLevel])
	assert.Equal(t, "🔍", p[DebugLevel])

	// Modifying the returned map should not affect defaults.
	p[InfoLevel] = "CHANGED"

	p2 := DefaultPrefixes()
	assert.Equal(t, "ℹ️", p2[InfoLevel], "DefaultPrefixes should return a copy")
}

func TestResolvePrefixUsesCustomPrefixes(t *testing.T) {
	l := New(io.Discard)
	l.SetPrefixes(LevelMap{InfoLevel: "CUSTOM"}) //nolint:exhaustive // intentionally partial

	e := &Event{logger: l, level: InfoLevel}
	assert.Equal(t, "CUSTOM", l.resolvePrefix(e))
}

func TestPackageLevelSetLabels(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetLevelLabels(LevelMap{WarnLevel: "WARN"}) //nolint:exhaustive // intentionally partial

	assert.Equal(t, "WARN", Default.labels[WarnLevel])
}

func TestPackageLevelSetLevelAlign(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetLevelAlign(AlignNone)

	assert.Equal(t, AlignNone, Default.levelAlign)
}

func TestSetColorMode(t *testing.T) {
	l := New(io.Discard)

	l.SetColorMode(ColorAlways)
	assert.False(t, l.colorsDisabled())

	l.SetColorMode(ColorNever)
	assert.True(t, l.colorsDisabled())

	l.SetColorMode(ColorAuto)
	// Auto falls through to global detection.
	assert.Equal(t, ColorsDisabled(), l.colorsDisabled())

	// Invalid color mode falls through to global detection.
	l.colorMode = ColorMode(99)
	assert.Equal(t, ColorsDisabled(), l.colorsDisabled())
}

func TestPackageLevelSetColorMode(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetColorMode(ColorAlways)

	assert.Equal(t, ColorAlways, Default.colorMode)
}

func TestPackageLevelFatal(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	// Fatal() should return non-nil event (FatalLevel is always >= any level).
	e := Fatal()

	assert.NotNil(t, e, "expected non-nil event from Fatal()")
}

func TestLogFormattedOutputColored(t *testing.T) {
	origForced := colorsForced.Load()
	defer colorsForced.Store(origForced)

	colorsForced.Store(true)

	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Str("k", "v").Msg("hello")

	got := buf.String()

	// With colors enabled, output should contain ANSI escape codes.
	assert.Contains(t, got, "hello")
	assert.Contains(t, got, "k")
	assert.True(t, strings.HasSuffix(got, "\n"))
}

func TestLogFormattedOutputColoredWithTimestamp(t *testing.T) {
	origForced := colorsForced.Load()
	defer colorsForced.Store(origForced)

	colorsForced.Store(true)

	var buf bytes.Buffer

	l := New(&buf)
	l.SetReportTimestamp(true)
	l.Info().Msg("hello")

	got := buf.String()

	assert.Contains(t, got, "hello")
}

func TestLogFormattedOutputAllLevels(t *testing.T) {
	tests := []struct {
		name    string
		method  func(*Logger) *Event
		wantLvl string
	}{
		{"trace", (*Logger).Trace, "TRC"},
		{"debug", (*Logger).Debug, "DBG"},
		{"info", (*Logger).Info, "INF"},
		{"dry", (*Logger).Dry, "DRY"},
		{"warn", (*Logger).Warn, "WRN"},
		{"error", (*Logger).Error, "ERR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			l := New(&buf)
			l.SetLevel(TraceLevel)
			tt.method(l).Msg("test")

			got := buf.String()
			assert.True(
				t,
				strings.HasPrefix(got, tt.wantLvl+" "),
				"output = %q, expected prefix %q",
				got,
				tt.wantLvl,
			)
		})
	}
}

func TestLogEmptyMessageNoDoubleSpace(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Str("status", "ok").Send()

	got := buf.String()

	// Should not have double space between prefix and field.
	assert.NotContains(t, got, "  status")
	// Should contain the field directly after the prefix.
	assert.Contains(t, got, "status=ok")
}

func TestLogEmptyMessageNoFieldsNoTrailingSpace(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Send()

	got := buf.String()

	// Should end with prefix + newline, no trailing spaces.
	assert.True(t, strings.HasSuffix(got, "ℹ️\n"), "got %q", got)
}

func TestLogWithMessageHasSpace(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Str("k", "v").Msg("hello")

	got := buf.String()

	// Message should be separated from prefix and fields.
	assert.Contains(t, got, "ℹ️ hello k=v")
}

func TestSetParts(t *testing.T) {
	t.Run("reorder", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(&buf)
		l.SetParts(PartMessage, PartLevel, PartPrefix)
		l.Info().Msg("hello")

		assert.Equal(t, "hello INF ℹ️\n", buf.String())
	})

	t.Run("omit_parts", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(&buf)
		l.SetParts(PartMessage)
		l.Info().Msg("hello")

		assert.Equal(t, "hello\n", buf.String())
	})

	t.Run("fields_before_message", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(&buf)
		l.SetParts(PartLevel, PartFields, PartMessage)
		l.Info().Str("k", "v").Msg("hello")

		assert.Equal(t, "INF k=v hello\n", buf.String())
	})

	t.Run("all_parts_with_timestamp", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(&buf)
		l.SetReportTimestamp(true)
		l.SetParts(PartLevel, PartMessage, PartTimestamp)
		l.Info().Msg("hello")

		got := buf.String()
		assert.True(t, strings.HasPrefix(got, "INF hello "))
	})

	t.Run("empty_panics", func(t *testing.T) {
		l := New(io.Discard)
		assert.Panics(t, func() { l.SetParts() })
	})
}

func TestPackageLevelSetParts(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetParts(PartMessage, PartLevel)

	Default.mu.Lock()
	got := Default.parts
	Default.mu.Unlock()

	assert.Equal(t, []Part{PartMessage, PartLevel}, got)
}

func TestDefaultParts(t *testing.T) {
	order := DefaultParts()
	assert.Equal(t, []Part{PartTimestamp, PartLevel, PartPrefix, PartMessage, PartFields}, order)

	// Should return a new slice each time.
	order[0] = PartFields
	order2 := DefaultParts()
	assert.Equal(t, PartTimestamp, order2[0])
}

func TestPerLevelMessageStyle(t *testing.T) {
	t.Run("uses_per_level_style", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(&buf)
		l.SetParts(PartMessage)
		l.styles.Messages[ErrorLevel] = l.styles.Levels[ErrorLevel]

		l.Error().Msg("boom")

		want := l.styles.Levels[ErrorLevel].Render("boom") + "\n"
		assert.Equal(t, want, buf.String())
	})

	t.Run("default_is_unstyled", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(&buf)
		l.SetParts(PartMessage)

		l.Info().Msg("hello")

		assert.Equal(t, "hello\n", buf.String())
	})
}

func TestSubLoggerInheritsPartOrder(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetParts(PartMessage, PartLevel, PartFields)

	sub := l.With().Str("k", "v").Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "hello INF k=v\n", buf.String())
}

func TestOmitEmptyDisabledByDefault(t *testing.T) {
	l := New(io.Discard)
	assert.False(t, l.omitEmpty)
	assert.False(t, l.omitZero)
}

func TestOmitEmpty(t *testing.T) {
	var got Entry

	l := New(io.Discard)
	l.SetOmitEmpty(true)
	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().
		Str("empty", "").
		Str("present", "hello").
		Any("nilval", nil).
		Any("nilslice", ([]string)(nil)).
		Strs("emptyslice", []string{}).
		Int("zero", 0).
		Bool("falsy", false).
		Msg("test")

	// Empty string, nil, nil slice, and empty slice should be omitted.
	keys := make([]string, len(got.Fields))
	for i, f := range got.Fields {
		keys[i] = f.Key
	}

	assert.NotContains(t, keys, "empty")
	assert.NotContains(t, keys, "nilval")
	assert.NotContains(t, keys, "nilslice")
	assert.NotContains(t, keys, "emptyslice")

	// Non-empty values and zero-but-not-empty values should be kept.
	assert.Contains(t, keys, "present")
	assert.Contains(t, keys, "zero")
	assert.Contains(t, keys, "falsy")
}

func TestOmitZero(t *testing.T) {
	var got Entry

	l := New(io.Discard)
	l.SetOmitZero(true)
	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().
		Str("empty", "").
		Str("present", "hello").
		Any("nilval", nil).
		Int("zero", 0).
		Bool("falsy", false).
		Float64("zerof", 0.0).
		Strs("emptyslice", []string{}).
		Int("nonzero", 42).
		Msg("test")

	keys := make([]string, len(got.Fields))
	for i, f := range got.Fields {
		keys[i] = f.Key
	}

	// All zero/empty values should be omitted.
	assert.NotContains(t, keys, "empty")
	assert.NotContains(t, keys, "nilval")
	assert.NotContains(t, keys, "zero")
	assert.NotContains(t, keys, "falsy")
	assert.NotContains(t, keys, "zerof")
	assert.NotContains(t, keys, "emptyslice")

	// Non-zero values should be kept.
	assert.Contains(t, keys, "present")
	assert.Contains(t, keys, "nonzero")
}

func TestOmitZeroSupersedesOmitEmpty(t *testing.T) {
	var got Entry

	l := New(io.Discard)
	l.SetOmitEmpty(true)
	l.SetOmitZero(true)
	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	// When both are set, omitZero takes precedence.
	l.Info().Int("zero", 0).Int("nonzero", 1).Msg("test")

	require.Len(t, got.Fields, 1)
	assert.Equal(t, "nonzero", got.Fields[0].Key)
}

func TestOmitEmptyFormattedOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetOmitEmpty(true)
	l.Info().Str("a", "").Str("b", "keep").Msg("test")

	assert.Equal(t, "INF ℹ️ test b=keep\n", buf.String())
}

func TestOmitZeroFormattedOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetOmitZero(true)
	l.Info().Int("a", 0).Int("b", 1).Msg("test")

	assert.Equal(t, "INF ℹ️ test b=1\n", buf.String())
}

func TestSubLoggerInheritsOmitSettings(t *testing.T) {
	l := New(io.Discard)
	l.SetOmitEmpty(true)
	l.SetOmitZero(true)

	sub := l.With().Str("k", "v").Logger()

	assert.True(t, sub.omitEmpty)
	assert.True(t, sub.omitZero)
}

func TestPackageLevelSetOmitEmpty(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetOmitEmpty(true)

	assert.True(t, Default.omitEmpty)
}

func TestPackageLevelSetOmitZero(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetOmitZero(true)

	assert.True(t, Default.omitZero)
}

func TestOmitQuotesDisabledByDefault(t *testing.T) {
	l := New(io.Discard)
	assert.Equal(t, QuoteAuto, l.quoteMode)
}

func TestOmitQuotes(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetOmitQuotes(true)
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=hello world\n", buf.String())
}

func TestOmitQuotesInStringSlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetOmitQuotes(true)
	l.Info().Strs("args", []string{"hello world", "ok"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test args=[hello world, ok]\n", buf.String())
}

func TestOmitQuotesInAnySlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetOmitQuotes(true)
	l.Info().Anys("vals", []any{"hello world", 1}).Msg("test")

	assert.Equal(t, "INF ℹ️ test vals=[hello world, 1]\n", buf.String())
}

func TestQuoteChar(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteChar('\'')
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg='hello world'\n", buf.String())
}

func TestQuoteCharInStringSlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteChar('\'')
	l.Info().Strs("args", []string{"hello world", "ok"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test args=['hello world', ok]\n", buf.String())
}

func TestQuoteCharInAnySlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteChar('\'')
	l.Info().Anys("vals", []any{"hello world", 1}).Msg("test")

	assert.Equal(t, "INF ℹ️ test vals=['hello world', 1]\n", buf.String())
}

func TestQuoteCharDefaultUsesStrconvQuote(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	// Default quoteChar (0) should use strconv.Quote with escaping.
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=\"hello world\"\n", buf.String())
}

func TestSubLoggerInheritsQuoteSettings(t *testing.T) {
	l := New(io.Discard)
	l.SetOmitQuotes(true)
	l.SetQuoteChars('[', ']')

	sub := l.With().Str("k", "v").Logger()

	assert.Equal(t, QuoteNever, sub.quoteMode)
	assert.Equal(t, '[', sub.quoteOpen)
	assert.Equal(t, ']', sub.quoteClose)
}

func TestSetOmitQuotesFalse(t *testing.T) {
	l := New(io.Discard)
	// First set to QuoteNever via SetOmitQuotes(true).
	l.SetOmitQuotes(true)
	assert.Equal(t, QuoteNever, l.quoteMode)

	// SetOmitQuotes(false) should restore QuoteAuto.
	l.SetOmitQuotes(false)
	assert.Equal(t, QuoteAuto, l.quoteMode)
}

func TestPackageLevelSetOmitQuotes(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetOmitQuotes(true)

	assert.Equal(t, QuoteNever, Default.quoteMode)
}

func TestPackageLevelSetQuoteChar(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetQuoteChar('\'')

	assert.Equal(t, '\'', Default.quoteOpen)
	assert.Equal(t, '\'', Default.quoteClose)
}

func TestQuoteChars(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteChars('[', ']')
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=[hello world]\n", buf.String())
}

func TestQuoteCharsInStringSlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteChars('«', '»')
	l.Info().Strs("args", []string{"hello world", "ok"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test args=[«hello world», ok]\n", buf.String())
}

func TestPackageLevelSetQuoteChars(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetQuoteChars('[', ']')

	assert.Equal(t, '[', Default.quoteOpen)
	assert.Equal(t, ']', Default.quoteClose)
}

func TestOmitQuotesTakesPrecedenceOverQuoteChar(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetOmitQuotes(true)
	l.SetQuoteChar('\'')
	l.Info().Str("msg", "hello world").Msg("test")

	// OmitQuotes should suppress quoting entirely, regardless of quoteChar.
	assert.Equal(t, "INF ℹ️ test msg=hello world\n", buf.String())
}

func TestQuoteModeAuto(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	// QuoteAuto is the default — simple strings unquoted, spaced strings quoted.
	l.Info().Str("simple", "timeout").Str("spaced", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test simple=timeout spaced=\"hello world\"\n", buf.String())
}

func TestQuoteModeAlways(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteMode(QuoteAlways)
	l.Info().Str("reason", "timeout").Msg("test")

	assert.Equal(t, "INF ℹ️ test reason=\"timeout\"\n", buf.String())
}

func TestQuoteModeNever(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteMode(QuoteNever)
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=hello world\n", buf.String())
}

func TestQuoteModeAlwaysInStringSlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetQuoteMode(QuoteAlways)
	l.Info().Strs("tags", []string{"api", "v2"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test tags=[\"api\", \"v2\"]\n", buf.String())
}

func TestPackageLevelSetQuoteMode(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetQuoteMode(QuoteAlways)

	assert.Equal(t, QuoteAlways, Default.quoteMode)
}

func TestSetFieldStyleLevel(t *testing.T) {
	l := New(io.Discard)

	assert.Equal(t, InfoLevel, l.fieldStyleLevel)

	l.SetFieldStyleLevel(TraceLevel)
	assert.Equal(t, TraceLevel, l.fieldStyleLevel)
}

func TestPackageLevelSetFieldStyleLevel(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetFieldStyleLevel(DebugLevel)

	Default.mu.Lock()
	got := Default.fieldStyleLevel
	Default.mu.Unlock()

	assert.Equal(t, DebugLevel, got)
}

func TestSubLoggerInheritsFieldStyleLevel(t *testing.T) {
	l := New(io.Discard)
	l.SetFieldStyleLevel(TraceLevel)

	sub := l.With().Str("k", "v").Logger()

	assert.Equal(t, TraceLevel, sub.fieldStyleLevel)
}

func TestSetFieldTimeFormat(t *testing.T) {
	l := New(io.Discard)

	assert.Equal(t, time.RFC3339, l.fieldTimeFormat)

	l.SetFieldTimeFormat(time.DateTime)
	assert.Equal(t, time.DateTime, l.fieldTimeFormat)
}

func TestPackageLevelSetFieldTimeFormat(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(io.Discard)
	SetFieldTimeFormat(time.RFC3339)

	Default.mu.Lock()
	got := Default.fieldTimeFormat
	Default.mu.Unlock()

	assert.Equal(t, time.RFC3339, got)
}

func TestLogFormattedOutputWithTimeField(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	l.Info().Time("created", ts).Msg("test")

	assert.Equal(t, "INF ℹ️ test created=2025-06-15T10:30:00Z\n", buf.String())
}

func TestLogFormattedOutputWithTimeFieldCustomFormat(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.SetFieldTimeFormat(time.DateOnly)

	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	l.Info().Time("created", ts).Msg("test")

	assert.Equal(t, "INF ℹ️ test created=2025-06-15\n", buf.String())
}

func TestSubLoggerInheritsFieldTimeFormat(t *testing.T) {
	l := New(io.Discard)
	l.SetFieldTimeFormat(time.Kitchen)

	sub := l.With().Str("k", "v").Logger()

	assert.Equal(t, time.Kitchen, sub.fieldTimeFormat)
}
