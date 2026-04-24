package clog

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/gechr/clog/field/elapsed"
	"github.com/gechr/clog/field/percent"
	"github.com/gechr/clog/field/quantity"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	assert.Equal(t, LevelInfo, l.level)
	assert.Nil(t, l.symbol)
	assert.NotNil(t, l.mu)
	assert.Nil(t, l.handler)
	assert.Equal(t, "15:04:05.000", l.timeFormat)
	assert.False(t, l.reportTimestamp)
	assert.NotNil(t, l.styles)
}

func TestNewNilOutput(t *testing.T) {
	// New(nil) must not panic - it should default to Stderr.
	require.NotPanics(t, func() {
		l := New(nil)
		assert.NotNil(t, l)
		assert.NotNil(t, l.output)
	})
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelTrace, "TRC"},
		{LevelDebug, "DBG"},
		{LevelInfo, "INF"},
		{LevelDry, "DRY"},
		{LevelWarn, "WRN"},
		{LevelError, "ERR"},
		{LevelFatal, "FTL"},
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
		{"trace_at_trace", LevelTrace, (*Logger).Trace, false},
		{"trace_at_debug", LevelDebug, (*Logger).Trace, true},
		{"trace_at_info", LevelInfo, (*Logger).Trace, true},
		{"debug_at_info", LevelInfo, (*Logger).Debug, true},
		{"info_at_info", LevelInfo, (*Logger).Info, false},
		{"dry_at_info", LevelInfo, (*Logger).Dry, false},
		{"warn_at_info", LevelInfo, (*Logger).Warn, false},
		{"error_at_info", LevelInfo, (*Logger).Error, false},
		{"fatal_at_info", LevelInfo, (*Logger).Fatal, false},
		{"debug_at_trace", LevelTrace, (*Logger).Debug, false},
		{"debug_at_debug", LevelDebug, (*Logger).Debug, false},
		{"info_at_warn", LevelWarn, (*Logger).Info, true},
		{"dry_at_warn", LevelWarn, (*Logger).Dry, true},
		{"warn_at_warn", LevelWarn, (*Logger).Warn, false},
		{"error_at_error", LevelError, (*Logger).Error, false},
		{"warn_at_error", LevelError, (*Logger).Warn, true},
		{"error_at_fatal", LevelFatal, (*Logger).Error, true},
		{"fatal_at_fatal", LevelFatal, (*Logger).Fatal, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewWriter(io.Discard)
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
	l := NewWriter(io.Discard)

	l.SetLevel(LevelDebug)
	assert.Equal(t, LevelDebug, l.level)

	l.SetLevel(LevelError)
	assert.Equal(t, LevelError, l.level)
}

func TestLoadLogLevelFromEnv(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		wantLevel     Level
		wantTimestamp bool
	}{
		{"trace", "trace", LevelTrace, true},
		{"debug", "debug", LevelDebug, true},
		{"info", "info", LevelInfo, false},
		{"dry", "dry", LevelDry, false},
		{"warn", "warn", LevelWarn, false},
		{"warning", "warning", LevelWarn, false},
		{"error", "error", LevelError, false},
		{"fatal", "fatal", LevelFatal, false},
		{"case_insensitive", "DEBUG", LevelDebug, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origDefault := Default
			defer func() { Default = origDefault }()

			Default = NewWriter(io.Discard)
			t.Setenv("CLOG_LOG_LEVEL", tt.value)
			loadLogLevelFromEnv()

			assert.Equal(t, tt.wantLevel, Default.level)
			assert.Equal(t, tt.wantTimestamp, Default.reportTimestamp)
		})
	}
}

func TestLoadLogLevelFromEnvNotSet(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	Default.SetLevel(LevelWarn)
	t.Setenv("CLOG_LOG_LEVEL", "")

	loadLogLevelFromEnv()

	assert.Equal(t, LevelWarn, Default.level)
}

func TestGetLevel(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	Default.SetLevel(LevelWarn)

	assert.Equal(t, LevelWarn, GetLevel())
}

func TestIsVerbose(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	Default.SetLevel(LevelInfo)
	assert.False(t, IsVerbose(), "expected IsVerbose() false at LevelInfo")

	Default.SetLevel(LevelDebug)
	assert.True(t, IsVerbose(), "expected IsVerbose() true at LevelDebug")

	Default.SetLevel(LevelTrace)
	assert.True(t, IsVerbose(), "expected IsVerbose() true at LevelTrace")
}

func TestResolveSymbol(t *testing.T) {
	tests := []struct {
		name         string
		loggerSymbol *string
		eventSymbol  *string
		level        Level
		want         string
	}{
		{name: "default_info", level: LevelInfo, want: "ℹ️"},
		{name: "default_trace", level: LevelTrace, want: "🔍"},
		{name: "default_debug", level: LevelDebug, want: "🐞"},
		{name: "default_warn", level: LevelWarn, want: "⚠️"},
		{name: "default_error", level: LevelError, want: "❌"},
		{name: "default_fatal", level: LevelFatal, want: "💥"},
		{name: "default_dry", level: LevelDry, want: "🚧"},
		{
			name:         "logger_symbol",
			loggerSymbol: new("LOG"),
			level:        LevelInfo,
			want:         "LOG",
		},
		{
			name:         "event_overrides_logger",
			loggerSymbol: new("LOG"),
			eventSymbol:  new("EVT"),
			level:        LevelInfo,
			want:         "EVT",
		},
		{
			name:        "event_overrides_default",
			eventSymbol: new("EVT"),
			level:       LevelInfo,
			want:        "EVT",
		},
		{
			name:         "empty_logger_symbol",
			loggerSymbol: new(""),
			level:        LevelInfo,
			want:         "",
		},
		{
			name:        "empty_event_symbol",
			eventSymbol: new(""),
			level:       LevelInfo,
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewWriter(io.Discard)
			l.symbol = tt.loggerSymbol

			e := &Event{logger: l, level: tt.level, symbol: tt.eventSymbol}

			got := l.resolveSymbol(e)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigure(t *testing.T) {
	t.Run("verbose", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)
		Configure(&Config{Verbose: true})

		assert.Equal(t, LevelDebug, Default.level)
		assert.True(t, Default.reportTimestamp)
	})

	t.Run("output", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)

		var buf bytes.Buffer

		out := TestOutput(&buf)
		Configure(&Config{Output: out})

		Default.mu.Lock()
		got := Default.output
		Default.mu.Unlock()

		assert.Same(t, out, got)
	})

	t.Run("styles", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)
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

		Default = NewWriter(io.Discard)
		Default.SetLevel(LevelDebug)
		Default.SetReportTimestamp(true)
		t.Setenv("CLOG_LOG_LEVEL", "")

		Configure(&Config{Verbose: false})

		assert.Equal(t, LevelInfo, Default.level)
		assert.False(t, Default.reportTimestamp)
	})

	t.Run("non_verbose_with_env", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)
		Default.SetLevel(LevelDebug)
		t.Setenv("CLOG_LOG_LEVEL", "debug")

		Configure(&Config{Verbose: false})

		assert.Equal(t, LevelDebug, Default.level)
	})
}

func TestSetVerbose(t *testing.T) {
	t.Run("enable", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)
		SetVerbose(true)

		assert.Equal(t, LevelDebug, Default.level)
		assert.True(t, Default.reportTimestamp)
	})

	t.Run("disable_without_env", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)
		Default.SetLevel(LevelDebug)
		t.Setenv("CLOG_LOG_LEVEL", "")

		SetVerbose(false)

		assert.Equal(t, LevelInfo, Default.level)
	})

	t.Run("disable_with_env", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)
		Default.SetLevel(LevelDebug)
		t.Setenv("CLOG_LOG_LEVEL", "debug")

		SetVerbose(false)

		assert.Equal(t, LevelDebug, Default.level)
	})
}

func TestPackageLevelConvenienceFunctions(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	Default.SetLevel(LevelTrace)

	var got Entry

	Default.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	tests := []struct {
		name  string
		fn    func() *Event
		level Level
	}{
		{"Trace", Trace, LevelTrace},
		{"Debug", Debug, LevelDebug},
		{"Info", Info, LevelInfo},
		{"Dry", Dry, LevelDry},
		{"Warn", Warn, LevelWarn},
		{"Error", Error, LevelError},
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

	Default = NewWriter(io.Discard)

	ctx := With()
	assert.NotNil(t, ctx, "expected non-nil context from With()")
}

func TestPackageLevelSetters(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)

	SetLevel(LevelWarn)
	assert.Equal(t, LevelWarn, Default.level)

	SetReportTimestamp(true)
	assert.True(t, Default.reportTimestamp)

	SetTimeFormat("2006-01-02")
	assert.Equal(t, "2006-01-02", Default.timeFormat)

	h := HandlerFunc(func(_ Entry) {})
	SetHandler(h)
	assert.NotNil(t, Default.handler)

	var buf bytes.Buffer

	SetOutputWriter(&buf)

	Default.mu.Lock()
	out := Default.output.Writer()
	Default.mu.Unlock()

	assert.Equal(t, &buf, out)

	styles := DefaultStyles()
	SetStyles(styles)

	Default.mu.Lock()
	gotStyles := Default.styles
	Default.mu.Unlock()

	assert.Equal(t, styles, gotStyles)

	var exitCode int

	SetExitFunc(func(code int) {
		exitCode = code
	})

	Default.mu.Lock()
	fn := Default.exitFunc
	Default.mu.Unlock()

	require.NotNil(t, fn)

	fn(2)

	assert.Equal(t, 2, exitCode)
}

func TestCustomHandlerReceivesEntries(t *testing.T) {
	l := NewWriter(io.Discard)

	var entries []Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		entries = append(entries, e)
	}))

	l.Info().Str("a", "1").Msg("first")
	l.Warn().Str("b", "2").Msg("second")

	require.Len(t, entries, 2)
	assert.Equal(t, LevelInfo, entries[0].Level)
	assert.Equal(t, "first", entries[0].Message)
	assert.Equal(t, LevelWarn, entries[1].Level)
	assert.Equal(t, "second", entries[1].Message)
}

func TestCustomHandlerNoBufferOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetHandler(HandlerFunc(func(_ Entry) {}))

	l.Info().Msg("intercepted")

	assert.Zero(t, buf.Len(), "expected no output to buffer when handler is set")
}

func TestSubLoggerWithWith(t *testing.T) {
	l := NewWriter(io.Discard)

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
	l := NewWriter(io.Discard)
	sub := l.With().Str("k", "v").Logger()

	assert.Same(t, l.mu, sub.mu, "sub-logger should share parent's mutex")
}

func TestWithCopiesFields(t *testing.T) {
	l := NewWriter(io.Discard)
	l.fields = []Field{{Key: "parent", Value: "yes"}}

	ctx := l.With()
	ctx.Str("child", "added")

	assert.Len(t, l.fields, 1, "parent fields should not be modified")
}

func TestEventFieldsDoNotModifyLogger(t *testing.T) {
	l := NewWriter(io.Discard)
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

	l := New(TestOutput(&buf))
	l.Info().Msg("hello")

	assert.Equal(t, "INF ℹ️ hello\n", buf.String())
}

func TestLogFormattedOutputWithFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Str("key", "val").Msg("hello")

	assert.Equal(t, "INF ℹ️ hello key=val\n", buf.String())
}

func TestLogFormattedOutputCustomSymbol(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Symbol(">>>").Msg("hello")

	assert.Equal(t, "INF >>> hello\n", buf.String())
}

func TestSymbolStyle(t *testing.T) {
	var buf bytes.Buffer

	l := New(NewOutput(&buf, ColorAlways))

	styles := DefaultStyles()
	styles.Symbols[LevelInfo] = new(
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")),
	)
	l.SetStyles(styles)

	l.Info().Symbol("warning").Msg("hello")

	got := buf.String()
	// The symbol should appear styled (contain ANSI escapes), not bare.
	assert.Equal(t, "\x1b[1;32mINF\x1b[m \x1b[1;33mwarning\x1b[m hello\n", got)
}

func TestSymbolStyleNoColor(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))

	styles := DefaultStyles()
	styles.Symbols[LevelInfo] = new(
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")),
	)
	l.SetStyles(styles)

	l.Info().Symbol("warning").Msg("hello")

	// No-color output should be plain.
	assert.Equal(t, "INF warning hello\n", buf.String())
}

func TestLogFormattedOutputEmptySymbol(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Symbol("").Msg("hello")

	assert.Equal(t, "INF hello\n", buf.String())
}

func TestLogFormattedOutputWithTimestamp(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetReportTimestamp(true)
	l.Info().Msg("hello")

	got := buf.String()

	assert.Contains(t, got, "INF")
	assert.Contains(t, got, "hello")
	assert.True(t, strings.HasSuffix(got, nl))
	// Timestamp format "HH:MM:SS.mmm" = 12 chars, plus trailing space.
	assert.GreaterOrEqual(t, len(got), 12, "output too short for timestamp")
}

func TestLogFormattedOutputQuotedFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=\"hello world\"\n", buf.String())
}

func TestLogFormattedOutputMultipleFields(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Str("a", "1").Int("b", 2).Bool("c", true).Msg("test")

	assert.Equal(t, "INF ℹ️ test a=1 b=2 c=true\n", buf.String())
}

func TestLoadLogLevelFromEnvDry(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	t.Setenv("CLOG_LOG_LEVEL", "dry")
	loadLogLevelFromEnv()

	assert.Equal(t, LevelDry, Default.level)
}

func TestLoadLogLevelFromEnvFatal(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	t.Setenv("CLOG_LOG_LEVEL", "fatal")
	loadLogLevelFromEnv()

	assert.Equal(t, LevelFatal, Default.level)
}

func TestLoadLogLevelFromEnvUnrecognised(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	t.Setenv("CLOG_LOG_LEVEL", "bogus")

	// Should not change the level, just print to stderr.
	loadLogLevelFromEnv()

	assert.Equal(t, LevelInfo, Default.level)
}

func TestSetLabels(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLabels(LabelMap{LevelWarn: "WARN"})

	assert.Equal(t, "WARN", l.labels[LevelWarn])
	// Other labels should retain defaults.
	assert.Equal(t, "INF", l.labels[LevelInfo])
}

func TestSetLevelAlign(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLevelAlign(AlignLeft)

	assert.Equal(t, AlignLeft, l.levelAlign)
}

func TestFormatLabelAlignNone(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLevelAlign(AlignNone)

	assert.Equal(t, "INF", l.formatLabel(LevelInfo))
}

func TestFormatLabelAlignLeft(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLabels(LabelMap{
		LevelInfo:  "INF",
		LevelWarn:  "WARN",
		LevelError: "ERROR",
	})
	l.SetLevelAlign(AlignLeft)

	// maxLabelWidth is 5 (ERROR), so INF should be left-padded to 5 chars.
	assert.Equal(t, "INF  ", l.formatLabel(LevelInfo))
}

func TestFormatLabelAlignRight(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLabels(LabelMap{
		LevelInfo:  "INF",
		LevelWarn:  "WARN",
		LevelError: "ERROR",
	})
	l.SetLevelAlign(AlignRight)

	// maxLabelWidth is 5 (ERROR), so INF should be right-padded.
	assert.Equal(t, "  INF", l.formatLabel(LevelInfo))
}

func TestFormatLabelAlignCenter(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLabels(LabelMap{
		LevelInfo:  "INF",
		LevelWarn:  "WARN",
		LevelError: "ERROR",
	})
	l.SetLevelAlign(AlignCenter)

	// maxLabelWidth is 5 (ERROR), so INF (3) gets 1 left + 1 right padding.
	assert.Equal(t, " INF ", l.formatLabel(LevelInfo))
	// WARN (4) gets 0 left + 1 right.
	assert.Equal(t, "WARN ", l.formatLabel(LevelWarn))
	// ERROR (5) fits exactly.
	assert.Equal(t, "ERROR", l.formatLabel(LevelError))
}

func TestFormatLabelUnknownAlign(t *testing.T) {
	l := NewWriter(io.Discard)
	l.levelAlign = Align(99) // invalid value

	assert.Equal(t, "INF", l.formatLabel(LevelInfo))
}

func TestSetSymbols(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetSymbols(LabelMap{LevelInfo: ">>>"})

	assert.Equal(t, ">>>", l.symbols[LevelInfo])
	// Other symbols should retain defaults.
	assert.Equal(t, "🐞", l.symbols[LevelDebug])
}

func TestPackageLevelSetSymbols(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetSymbols(LabelMap{LevelInfo: ">>>"})

	assert.Equal(t, ">>>", Default.symbols[LevelInfo])
}

func TestSetTimeLocation(t *testing.T) {
	l := NewWriter(io.Discard)
	loc := time.UTC
	l.SetTimeLocation(loc)

	assert.Equal(t, loc, l.timeLocation)
}

func TestPackageLevelSetTimeLocation(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	loc := time.UTC
	SetTimeLocation(loc)

	Default.mu.Lock()
	got := Default.timeLocation
	Default.mu.Unlock()

	assert.Equal(t, loc, got)
}

func TestDefaultSymbols(t *testing.T) {
	p := DefaultSymbols()

	assert.Equal(t, "ℹ️", p[LevelInfo])
	assert.Equal(t, "🔍", p[LevelTrace])
	assert.Equal(t, "🐞", p[LevelDebug])

	// Modifying the returned map should not affect defaults.
	p[LevelInfo] = "CHANGED"

	p2 := DefaultSymbols()
	assert.Equal(t, "ℹ️", p2[LevelInfo], "DefaultSymbols should return a copy")
}

func TestResolveSymbolUsesCustomSymbols(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetSymbols(LabelMap{LevelInfo: "CUSTOM"})

	e := &Event{logger: l, level: LevelInfo}
	assert.Equal(t, "CUSTOM", l.resolveSymbol(e))
}

func TestPackageLevelSetLabels(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetLabels(LabelMap{LevelWarn: "WARN"})

	assert.Equal(t, "WARN", Default.labels[LevelWarn])
}

func TestPackageLevelSetLevelAlign(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetLevelAlign(AlignNone)

	assert.Equal(t, AlignNone, Default.levelAlign)
}

func TestColorsDisabledPerOutput(t *testing.T) {
	always := New(NewOutput(io.Discard, ColorAlways))
	assert.False(t, always.colorsDisabled())

	never := New(NewOutput(io.Discard, ColorNever))
	assert.True(t, never.colorsDisabled())

	auto := New(NewOutput(io.Discard, ColorAuto))
	// ColorAuto on a non-TTY writer -> colors disabled.
	assert.True(t, auto.colorsDisabled())
}

func TestPackageLevelSetColorMode(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetColorMode(ColorAlways)

	assert.False(
		t,
		Default.colorsDisabled(),
		"expected colors enabled after SetColorMode(ColorAlways)",
	)

	SetColorMode(ColorNever)

	assert.True(
		t,
		Default.colorsDisabled(),
		"expected colors disabled after SetColorMode(ColorNever)",
	)
}

func TestPackageLevelFatal(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	// Fatal() should return non-nil event (LevelFatal is always >= any level).
	e := Fatal()

	assert.NotNil(t, e, "expected non-nil event from Fatal()")
}

func TestLogFormattedOutputColored(t *testing.T) {
	var buf bytes.Buffer

	l := New(NewOutput(&buf, ColorAlways))
	l.Info().Str("k", "v").Msg("hello")

	got := buf.String()

	// With colors enabled, output should contain ANSI escape codes.
	assert.Equal(
		t,
		"\x1b[1;32mINF\x1b[m ℹ️ hello \x1b[34mk\x1b[m\x1b[2m=\x1b[m\x1b[97mv\x1b[m\n",
		got,
	)
}

func TestLogFormattedOutputColoredWithTimestamp(t *testing.T) {
	var buf bytes.Buffer

	l := New(NewOutput(&buf, ColorAlways))
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

			l := New(TestOutput(&buf))
			l.SetLevel(LevelTrace)
			tt.method(l).Msg("test")

			got := buf.String()
			assert.True(
				t,
				strings.HasPrefix(got, tt.wantLvl+" "),
				"output = %q, expected symbol %q",
				got,
				tt.wantLvl,
			)
		})
	}
}

func TestLogEmptyMessageNoDoubleSpace(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Str("status", "ok").Send()

	got := buf.String()

	// Should not have double space between symbol and field.
	assert.NotContains(t, got, "  status")
	// Should contain the field directly after the symbol.
	assert.Equal(t, "INF ℹ️ status=ok\n", got)
}

func TestLogEmptyMessageNoFieldsNoTrailingSpace(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Send()

	got := buf.String()

	// Should end with symbol + newline, no trailing spaces.
	assert.True(t, strings.HasSuffix(got, "ℹ️\n"), "got %q", got)
}

func TestLogWithMessageHasSpace(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Str("k", "v").Msg("hello")

	got := buf.String()

	// Message should be separated from symbol and fields.
	assert.Equal(t, "INF ℹ️ hello k=v\n", got)
}

func TestSetParts(t *testing.T) {
	t.Run("reorder", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartMessage, PartLevel, PartSymbol)
		l.Info().Msg("hello")

		assert.Equal(t, "hello INF ℹ️\n", buf.String())
	})

	t.Run("omit_parts", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartMessage)
		l.Info().Msg("hello")

		assert.Equal(t, "hello\n", buf.String())
	})

	t.Run("fields_before_message", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartLevel, PartFields, PartMessage)
		l.Info().Str("k", "v").Msg("hello")

		assert.Equal(t, "INF k=v hello\n", buf.String())
	})

	t.Run("all_parts_with_timestamp", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetReportTimestamp(true)
		l.SetParts(PartLevel, PartMessage, PartTimestamp)
		l.Info().Msg("hello")

		got := buf.String()
		assert.True(t, strings.HasPrefix(got, "INF hello "))
	})

	t.Run("empty_panics", func(t *testing.T) {
		l := NewWriter(io.Discard)
		assert.Panics(t, func() {
			l.SetParts()
		})
	})
}

func TestEventPartsNilOnDisabledLevel(t *testing.T) {
	l := New(TestOutput(io.Discard))
	l.SetLevel(LevelError)
	e := l.Info() // returns nil
	assert.Nil(t, e.Parts(PartMessage))
}

func TestPackageLevelSetParts(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetParts(PartMessage, PartLevel)

	Default.mu.Lock()
	got := Default.parts
	Default.mu.Unlock()

	assert.Equal(t, []Part{PartMessage, PartLevel}, got)
}

func TestDefaultParts(t *testing.T) {
	order := DefaultParts()
	assert.Equal(t, []Part{PartTimestamp, PartLevel, PartSymbol, PartMessage, PartFields}, order)

	// Should return a new slice each time.
	order[0] = PartFields
	order2 := DefaultParts()
	assert.Equal(t, PartTimestamp, order2[0])
}

func TestPerLevelMessageStyle(t *testing.T) {
	t.Run("uses_per_level_style", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(NewOutput(&buf, ColorAlways))
		l.SetParts(PartMessage)
		l.styles.Messages[LevelError] = l.styles.Levels[LevelError]

		l.Error().Msg("boom")

		want := l.styles.Levels[LevelError].Render("boom") + nl
		assert.Equal(t, want, buf.String())
	})

	t.Run("default_is_unstyled", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartMessage)

		l.Info().Msg("hello")

		assert.Equal(t, "hello\n", buf.String())
	})
}

func TestEventPartsOverride(t *testing.T) {
	t.Run("overrides_logger_parts", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartLevel, PartSymbol, PartMessage, PartFields)

		// Override to only show message on this one event.
		l.Info().Parts(PartSymbol, PartMessage).Msg("hello")

		assert.Equal(t, "ℹ️ hello\n", buf.String())
	})

	t.Run("nil_parts_uses_logger_default", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartMessage)

		// No Parts() call - should use logger's parts.
		l.Info().Msg("hello")

		assert.Equal(t, "hello\n", buf.String())
	})

	t.Run("reorder_parts", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartLevel, PartMessage)

		// Override order on a single event.
		l.Info().Parts(PartMessage, PartLevel).Msg("hello")

		assert.Equal(t, "hello INF\n", buf.String())
	})

	t.Run("does_not_affect_next_event", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetParts(PartLevel, PartMessage)

		l.Info().Parts(PartMessage).Msg("first")
		l.Info().Msg("second")

		lines := strings.Split(strings.TrimRight(buf.String(), nl), nl)
		require.Len(t, lines, 2)
		assert.Equal(t, "first", lines[0])
		assert.Equal(t, "INF second", lines[1])
	})
}

func TestSubLoggerInheritsPartOrder(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetParts(PartMessage, PartLevel, PartFields)

	sub := l.With().Str("k", "v").Logger()
	sub.Info().Msg("hello")

	assert.Equal(t, "hello INF k=v\n", buf.String())
}

func TestOmitEmptyDisabledByDefault(t *testing.T) {
	l := NewWriter(io.Discard)
	assert.False(t, l.omitEmpty)
	assert.False(t, l.omitZero)
}

func TestOmitEmpty(t *testing.T) {
	var got Entry

	l := NewWriter(io.Discard)
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

	l := NewWriter(io.Discard)
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

	l := NewWriter(io.Discard)
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

	l := New(TestOutput(&buf))
	l.SetOmitEmpty(true)
	l.Info().Str("a", "").Str("b", "keep").Msg("test")

	assert.Equal(t, "INF ℹ️ test b=keep\n", buf.String())
}

func TestOmitZeroFormattedOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetOmitZero(true)
	l.Info().Int("a", 0).Int("b", 1).Msg("test")

	assert.Equal(t, "INF ℹ️ test b=1\n", buf.String())
}

func TestSubLoggerInheritsOmitSettings(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetOmitEmpty(true)
	l.SetOmitZero(true)

	sub := l.With().Str("k", "v").Logger()

	assert.True(t, sub.omitEmpty)
	assert.True(t, sub.omitZero)
}

func TestPackageLevelSetOmitEmpty(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetOmitEmpty(true)

	assert.True(t, Default.omitEmpty)
}

func TestPackageLevelSetOmitZero(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetOmitZero(true)

	assert.True(t, Default.omitZero)
}

func TestOmitQuotesDisabledByDefault(t *testing.T) {
	l := NewWriter(io.Discard)
	assert.Equal(t, QuoteAuto, l.quoteMode)
}

func TestQuoteChar(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuoteChar('\'')
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg='hello world'\n", buf.String())
}

func TestQuoteCharInStringSlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuoteChar('\'')
	l.Info().Strs("args", []string{"hello world", "ok"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test args=['hello world', ok]\n", buf.String())
}

func TestQuoteCharInAnySlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuoteChar('\'')
	l.Info().Anys("vals", []any{"hello world", 1}).Msg("test")

	assert.Equal(t, "INF ℹ️ test vals=['hello world', 1]\n", buf.String())
}

func TestQuoteCharDefaultUsesStrconvQuote(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	// Default quoteChar (0) should use strconv.Quote with escaping.
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=\"hello world\"\n", buf.String())
}

func TestPackageLevelSetQuoteChar(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetQuoteChar('\'')

	assert.Equal(t, '\'', Default.quoteOpen)
	assert.Equal(t, '\'', Default.quoteClose)
}

func TestQuoteChars(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuoteChars('[', ']')
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=[hello world]\n", buf.String())
}

func TestQuoteCharsInStringSlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuoteChars('«', '»')
	l.Info().Strs("args", []string{"hello world", "ok"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test args=[«hello world», ok]\n", buf.String())
}

func TestPackageLevelSetQuoteChars(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetQuoteChars('[', ']')

	assert.Equal(t, '[', Default.quoteOpen)
	assert.Equal(t, ']', Default.quoteClose)
}

func TestSliceBracket(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetSliceBracket('|')
	l.Info().Ints("vals", []int{1, 2, 3}).Msg("test")

	assert.Equal(t, "INF ℹ️ test vals=|1, 2, 3|\n", buf.String())
}

func TestSliceBrackets(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetSliceBrackets('«', '»')
	l.Info().Ints("vals", []int{1, 2, 3}).Msg("test")

	assert.Equal(t, "INF ℹ️ test vals=«1, 2, 3»\n", buf.String())
}

func TestSliceSeparator(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetSliceSeparator(" ")
	l.Info().Ints("vals", []int{1, 2, 3}).Msg("test")

	assert.Equal(t, "INF ℹ️ test vals=[1 2 3]\n", buf.String())
}

func TestPackageLevelSetSliceBracket(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetSliceBracket('|')

	assert.Equal(t, '|', Default.sliceOpen)
	assert.Equal(t, '|', Default.sliceClose)
}

func TestPackageLevelSetSliceBrackets(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetSliceBrackets('«', '»')

	assert.Equal(t, '«', Default.sliceOpen)
	assert.Equal(t, '»', Default.sliceClose)
}

func TestPackageLevelSetSliceSeparator(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetSliceSeparator(" ")

	assert.Equal(t, " ", Default.sliceSep)
}

func TestQuoteAuto(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	// QuoteAuto is the default - simple strings unquoted, spaced strings quoted.
	l.Info().Str("simple", "timeout").Str("spaced", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test simple=timeout spaced=\"hello world\"\n", buf.String())
}

func TestQuoteAlways(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuote(QuoteAlways)
	l.Info().Str("reason", "timeout").Msg("test")

	assert.Equal(t, "INF ℹ️ test reason=\"timeout\"\n", buf.String())
}

func TestQuoteNever(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuote(QuoteNever)
	l.Info().Str("msg", "hello world").Msg("test")

	assert.Equal(t, "INF ℹ️ test msg=hello world\n", buf.String())
}

func TestQuoteAlwaysInStringSlice(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetQuote(QuoteAlways)
	l.Info().Strs("tags", []string{"api", "v2"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test tags=[\"api\", \"v2\"]\n", buf.String())
}

func TestPackageLevelSetQuote(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetQuote(QuoteAlways)

	assert.Equal(t, QuoteAlways, Default.quoteMode)
}

func TestWrapNone(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Str("reason", "timeout").Msg("test")

	assert.Equal(t, "INF ℹ️ test reason=timeout\n", buf.String())
}

func TestWrapHard(t *testing.T) {
	line := "abc repositories=[alpha bravo charlie delta echo foxtrot golf hotel]"
	got := wrapLine(line, 30, WrapHard)

	// Hard wrap should break at exactly the column limit, even mid-word.
	assert.Contains(t, got, "\n")

	for l := range strings.SplitSeq(got, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(l), 30)
	}
}

func TestWrapSoft(t *testing.T) {
	line := "abc repositories=[alpha bravo charlie delta echo foxtrot golf hotel]"
	got := wrapLine(line, 30, WrapSoft)

	// Soft wrap should break at word boundaries.
	assert.Contains(t, got, "\n")

	for l := range strings.SplitSeq(got, "\n") {
		assert.LessOrEqual(t, ansi.StringWidth(l), 30)
	}
}

func TestWrapSoftBreaksOnSpace(t *testing.T) {
	line := "hello world foo bar"
	got := wrapLine(line, 12, WrapSoft)

	// Should break at space boundaries, not mid-word.
	lines := strings.SplitSeq(got, "\n")
	for l := range lines {
		// No line should start with a partial word from the previous line.
		assert.False(t, strings.HasPrefix(l, "ld"), "soft wrap broke mid-word")
	}
}

func TestWrapNonePassthrough(t *testing.T) {
	line := "a very long line that exceeds the width"
	got := wrapLine(line, 10, WrapNone)

	assert.Equal(t, line, got)
}

func TestPackageLevelSetWrap(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetWrap(WrapSoft)

	assert.Equal(t, WrapSoft, Default.wrap)
}

func TestSetFieldStyleLevel(t *testing.T) {
	l := NewWriter(io.Discard)

	assert.Equal(t, LevelInfo, l.fieldStyleLevel)

	l.SetFieldStyleLevel(LevelTrace)
	assert.Equal(t, LevelTrace, l.fieldStyleLevel)
}

func TestPackageLevelSetFieldStyleLevel(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetFieldStyleLevel(LevelDebug)

	Default.mu.Lock()
	got := Default.fieldStyleLevel
	Default.mu.Unlock()

	assert.Equal(t, LevelDebug, got)
}

func TestSubLoggerInheritsFieldStyleLevel(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetFieldStyleLevel(LevelTrace)

	sub := l.With().Str("k", "v").Logger()

	assert.Equal(t, LevelTrace, sub.fieldStyleLevel)
}

func TestSetFieldTimeFormat(t *testing.T) {
	l := NewWriter(io.Discard)

	assert.Equal(t, time.RFC3339, l.fieldTimeFormat)

	l.SetFieldTimeFormat(time.DateTime)
	assert.Equal(t, time.DateTime, l.fieldTimeFormat)
}

func TestPackageLevelSetFieldTimeFormat(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetFieldTimeFormat(time.RFC3339)

	Default.mu.Lock()
	got := Default.fieldTimeFormat
	Default.mu.Unlock()

	assert.Equal(t, time.RFC3339, got)
}

func TestLogFormattedOutputWithTimeField(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	l.Info().Time("created", ts).Msg("test")

	assert.Equal(t, "INF ℹ️ test created=2025-06-15T10:30:00Z\n", buf.String())
}

func TestLogFormattedOutputWithTimeFieldCustomFormat(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetFieldTimeFormat(time.DateOnly)

	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	l.Info().Time("created", ts).Msg("test")

	assert.Equal(t, "INF ℹ️ test created=2025-06-15\n", buf.String())
}

func TestSubLoggerInheritsFieldTimeFormat(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetFieldTimeFormat(time.Kitchen)

	sub := l.With().Str("k", "v").Logger()

	assert.Equal(t, time.Kitchen, sub.fieldTimeFormat)
}

func TestConcurrentLogging(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetLevel(LevelTrace)

	const goroutines = 10
	const iterations = 50

	done := make(chan struct{})

	for i := range goroutines {
		go func(id int) {
			defer func() { done <- struct{}{} }()
			for j := range iterations {
				l.Info().
					Int("goroutine", id).
					Int("iter", j).
					Str("msg", "concurrent").
					Msg("test")
			}
		}(i)
	}

	for range goroutines {
		<-done
	}

	got := buf.String()
	lines := strings.Split(strings.TrimSpace(got), nl)
	assert.Len(t, lines, goroutines*iterations)
}

func TestDefaultLabels(t *testing.T) {
	labels := DefaultLabels()

	assert.Equal(t, "TRC", labels[LevelTrace])
	assert.Equal(t, "DBG", labels[LevelDebug])
	assert.Equal(t, "INF", labels[LevelInfo])
	assert.Equal(t, "DRY", labels[LevelDry])
	assert.Equal(t, "WRN", labels[LevelWarn])
	assert.Equal(t, "ERR", labels[LevelError])
	assert.Equal(t, "FTL", labels[LevelFatal])

	// Modifying the returned map should not affect defaults.
	labels[LevelInfo] = "CHANGED"

	labels2 := DefaultLabels()
	assert.Equal(t, "INF", labels2[LevelInfo], "DefaultLabels should return a copy")
}

func TestSetStylesNilDefaultsToDefaultStyles(t *testing.T) {
	l := NewWriter(io.Discard)
	original := l.styles

	// Set to nil - should fall back to DefaultStyles().
	l.SetStyles(nil)

	l.mu.Lock()
	got := l.styles
	l.mu.Unlock()

	assert.NotNil(t, got, "styles should not be nil after SetStyles(nil)")
	assert.Equal(t, DefaultStyles(), got)
	// Should be a new instance, not the original pointer.
	assert.NotSame(t, original, got)
}

func TestSetTimeLocationNilDefaultsToLocal(t *testing.T) {
	l := NewWriter(io.Discard)

	// Set to UTC first.
	l.SetTimeLocation(time.UTC)
	assert.Equal(t, time.UTC, l.timeLocation)

	// Set to nil - should fall back to time.Local.
	l.SetTimeLocation(nil)

	l.mu.Lock()
	got := l.timeLocation
	l.mu.Unlock()

	assert.Equal(t, time.Local, got)
}

func TestSetExitFuncNilDefaultsToOsExit(t *testing.T) {
	l := NewWriter(io.Discard)

	// Set a custom exit func first.
	called := false
	l.SetExitFunc(func(_ int) {
		called = true
	})
	l.mu.Lock()
	fn := l.exitFunc
	l.mu.Unlock()
	fn(0)
	assert.True(t, called)

	// Set to nil - should fall back to os.Exit.
	l.SetExitFunc(nil)

	l.mu.Lock()
	got := l.exitFunc
	l.mu.Unlock()

	// We can't compare function pointers directly in Go, but we can verify
	// it is not nil and it's the same function by checking its behaviour
	// through the Fatal path. Use a sub-logger with a handler so Fatal
	// still triggers exitFunc.
	assert.NotNil(t, got, "exitFunc should not be nil after SetExitFunc(nil)")

	// Verify it's os.Exit by comparing pointer values via fmt.
	// A simpler check: ensure Fatal still invokes an exit function.
	var buf bytes.Buffer
	l2 := New(TestOutput(&buf))
	var exitCode int
	l2.SetExitFunc(nil) // should default to os.Exit
	// Override again to intercept - just verify nil didn't leave it broken.
	l2.SetExitFunc(func(code int) {
		exitCode = code
	})
	l2.Fatal().Msg("boom")
	assert.Equal(t, 1, exitCode)
}

func TestSetExitFuncNilFatalStillWorks(t *testing.T) {
	// Verify that setting nil and then overriding works correctly
	// (the nil guard should have set os.Exit, not left it nil).
	l := NewWriter(io.Discard)
	l.SetExitFunc(nil)

	// Now override with a test function to verify the logger is still functional.
	var exitCode int
	l.SetExitFunc(func(code int) {
		exitCode = code
	})
	l.Fatal().Msg("test fatal")
	assert.Equal(t, 1, exitCode)
}

func TestSetExitCode(t *testing.T) {
	var exitCode int

	l := NewWriter(io.Discard)
	l.SetExitFunc(func(code int) {
		exitCode = code
	})
	l.SetExitCode(3)
	l.Fatal().Msg("fatal error")

	assert.Equal(t, 3, exitCode)
}

func TestSetExitCodeOverriddenByEvent(t *testing.T) {
	var exitCode int

	l := NewWriter(io.Discard)
	l.SetExitFunc(func(code int) {
		exitCode = code
	})
	l.SetExitCode(3)
	l.Fatal().ExitCode(5).Msg("fatal error")

	assert.Equal(t, 5, exitCode)
}

func TestPackageLevelSetExitCode(t *testing.T) {
	saved := Default
	defer func() { Default = saved }()

	Default = NewWriter(io.Discard)

	var exitCode int
	SetExitFunc(func(code int) {
		exitCode = code
	})
	SetExitCode(42)
	Fatal().Msg("fatal")

	assert.Equal(t, 42, exitCode)
}

func TestAtomicLevelFastPath(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLevel(LevelWarn)

	// Events below the level should return nil without acquiring the mutex.
	assert.Nil(t, l.Trace(), "Trace should be nil at LevelWarn")
	assert.Nil(t, l.Debug(), "Debug should be nil at LevelWarn")
	assert.Nil(t, l.Info(), "Info should be nil at LevelWarn")

	// Events at or above the level should return non-nil.
	assert.NotNil(t, l.Warn(), "Warn should not be nil at LevelWarn")
	assert.NotNil(t, l.Error(), "Error should not be nil at LevelWarn")
}

func TestAtomicLevelConcurrent(t *testing.T) {
	t.Parallel()
	l := NewWriter(io.Discard)
	l.SetLevel(LevelError)

	var wg sync.WaitGroup

	// Concurrently create events and change levels.
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 1000 {
			l.Info()
			l.Error()
		}
	}()
	go func() {
		defer wg.Done()
		for range 1000 {
			l.SetLevel(LevelInfo)
			l.SetLevel(LevelError)
		}
	}()

	wg.Wait()
}

func TestNewLoggerAtomicLevelInitialized(t *testing.T) {
	l := NewWriter(io.Discard)
	assert.Equal(t, int32(LevelInfo), l.atomicLevel.Load(),
		"atomicLevel should be initialized to LevelInfo")
}

func TestSetLevelUpdatesAtomicLevel(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLevel(LevelDebug)
	assert.Equal(t, int32(LevelDebug), l.atomicLevel.Load())

	l.SetLevel(LevelFatal)
	assert.Equal(t, int32(LevelFatal), l.atomicLevel.Load())
}

func TestSetOutput(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = NewWriter(io.Discard)
	SetOutput(TestOutput(&buf))

	Default.Info().Msg("test")

	assert.Equal(t, "INF ℹ️ test\n", buf.String())
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"trace", LevelTrace},
		{"debug", LevelDebug},
		{"info", LevelInfo},
		{"dry", LevelDry},
		{"warn", LevelWarn},
		{"warning", LevelWarn},
		{"error", LevelError},
		{"fatal", LevelFatal},
		{"critical", LevelFatal},
		{"TRACE", LevelTrace},
		{"Debug", LevelDebug},
		{"INFO", LevelInfo},
		{"WARNING", LevelWarn},
		{"CRITICAL", LevelFatal},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseLevel(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseLevelUnknown(t *testing.T) {
	_, err := ParseLevel("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
}

func TestLevelMarshalText(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelTrace, LevelTraceValue},
		{LevelDebug, LevelDebugValue},
		{LevelInfo, LevelInfoValue},
		{LevelDry, LevelDryValue},
		{LevelWarn, LevelWarnValue},
		{LevelError, LevelErrorValue},
		{LevelFatal, LevelFatalValue},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got, err := tt.level.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestLevelMarshalTextUnknown(t *testing.T) {
	_, err := Level(99).MarshalText()
	assert.Error(t, err)
}

func TestLevelUnmarshalText(t *testing.T) {
	tests := []struct {
		input string
		want  Level
	}{
		{"trace", LevelTrace},
		{"info", LevelInfo},
		{"warning", LevelWarn},
		{"FATAL", LevelFatal},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			var l Level
			err := l.UnmarshalText([]byte(tt.input))
			require.NoError(t, err)
			assert.Equal(t, tt.want, l)
		})
	}
}

func TestLevelUnmarshalTextUnknown(t *testing.T) {
	var l Level
	err := l.UnmarshalText([]byte("bogus"))
	assert.Error(t, err)
}

func TestLevelMarshalRoundTrip(t *testing.T) {
	for _, level := range Levels() {
		text, err := level.MarshalText()
		require.NoError(t, err)

		var got Level
		err = got.UnmarshalText(text)
		require.NoError(t, err)
		assert.Equal(t, level, got)
	}
}

func TestSetElapsedFormatFunc(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	elapsed.SetFormatFunc(func(d time.Duration) string {
		return "custom:" + d.String()
	})
	t.Cleanup(func() { elapsed.SetFormatFunc(nil) })

	// Disable minimum so elapsed is always shown.
	elapsed.SetMinimum(0)
	elapsed.SetRound(0)
	t.Cleanup(func() {
		elapsed.SetMinimum(time.Second)
		elapsed.SetRound(time.Second)
	})

	// Directly inject an elapsed field via the logger's fields.
	l.mu.Lock()
	l.fields = []Field{{Key: "took", Value: core.ElapsedField(3 * time.Second)}}
	l.mu.Unlock()

	l.Info().Msg("test")

	assert.Equal(t, "INF ℹ️ test took=custom:3s\n", buf.String())
}

func TestSetElapsedMinimum(t *testing.T) {
	t.Cleanup(func() {
		elapsed.SetMinimum(time.Second)
		elapsed.SetRound(time.Second)
	})

	t.Run("below_threshold_hidden", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		elapsed.SetMinimum(2 * time.Second)
		elapsed.SetRound(0)

		l.mu.Lock()
		l.fields = []Field{{Key: "took", Value: core.ElapsedField(1 * time.Second)}}
		l.mu.Unlock()

		l.Info().Msg("test")

		assert.NotContains(t, buf.String(), "took=")
	})

	t.Run("above_threshold_shown", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		elapsed.SetMinimum(1 * time.Second)
		elapsed.SetRound(0)

		l.mu.Lock()
		l.fields = []Field{{Key: "took", Value: core.ElapsedField(2 * time.Second)}}
		l.mu.Unlock()

		l.Info().Msg("test")

		assert.Equal(t, "INF ℹ️ test took=2s\n", buf.String())
	})

	t.Run("zero_shows_all", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		elapsed.SetMinimum(0)
		elapsed.SetRound(0)

		l.mu.Lock()
		l.fields = []Field{{Key: "took", Value: core.ElapsedField(100 * time.Millisecond)}}
		l.mu.Unlock()

		l.Info().Msg("test")

		assert.Equal(t, "INF ℹ️ test took=100ms\n", buf.String())
	})
}

func TestSetElapsedPrecision(t *testing.T) {
	t.Cleanup(func() {
		elapsed.SetPrecision(0)
		elapsed.SetMinimum(time.Second)
		elapsed.SetRound(time.Second)
	})

	t.Run("precision_0", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		elapsed.SetPrecision(0)
		elapsed.SetMinimum(0)
		elapsed.SetRound(0)

		l.mu.Lock()
		l.fields = []Field{{Key: "took", Value: core.ElapsedField(3200 * time.Millisecond)}}
		l.mu.Unlock()

		l.Info().Msg("test")

		assert.Equal(t, "INF ℹ️ test took=3s\n", buf.String())
	})

	t.Run("precision_1", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		elapsed.SetPrecision(1)
		elapsed.SetMinimum(0)
		elapsed.SetRound(0)

		l.mu.Lock()
		l.fields = []Field{{Key: "took", Value: core.ElapsedField(3200 * time.Millisecond)}}
		l.mu.Unlock()

		l.Info().Msg("test")

		assert.Equal(t, "INF ℹ️ test took=3.2s\n", buf.String())
	})
}

func TestSetElapsedRound(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	elapsed.SetRound(time.Second)
	elapsed.SetMinimum(0)
	elapsed.SetPrecision(0)
	t.Cleanup(func() {
		elapsed.SetRound(time.Second)
		elapsed.SetMinimum(time.Second)
		elapsed.SetPrecision(0)
	})

	l.mu.Lock()
	l.fields = []Field{{Key: "took", Value: core.ElapsedField(2600 * time.Millisecond)}}
	l.mu.Unlock()

	l.Info().Msg("test")

	// 2600ms rounds to 3s.
	assert.Equal(t, "INF ℹ️ test took=3s\n", buf.String())
}

func TestSetFieldSort(t *testing.T) {
	t.Run("ascending", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetFieldSort(SortAscending)
		l.Info().Str("zoo", "last").Str("alpha", "first").Msg("test")

		got := buf.String()
		alphaIdx := strings.Index(got, "alpha=")
		zooIdx := strings.Index(got, "zoo=")
		assert.Greater(t, zooIdx, alphaIdx, "expected alpha before zoo in ascending sort")
	})

	t.Run("descending", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		l.SetFieldSort(SortDescending)
		l.Info().Str("alpha", "first").Str("zoo", "last").Msg("test")

		got := buf.String()
		alphaIdx := strings.Index(got, "alpha=")
		zooIdx := strings.Index(got, "zoo=")
		assert.Greater(t, alphaIdx, zooIdx, "expected zoo before alpha in descending sort")
	})
}

func TestSetPercentFormatFunc(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	percent.SetFormatFunc(func(f float64) string {
		return "pct:" + strings.TrimRight(strings.TrimRight(
			strconv.FormatFloat(f, 'f', 1, 64), "0"), ".") + "%"
	})
	t.Cleanup(func() { percent.SetFormatFunc(nil) })

	l.Info().Percent("progress", 0.75).Msg("test")

	assert.Equal(t, "INF ℹ️ test progress=pct:75%\n", buf.String())
}

func TestSetPercentPrecision(t *testing.T) {
	t.Cleanup(func() { percent.SetPrecision(0) })

	t.Run("precision_0", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		percent.SetPrecision(0)
		l.Info().Percent("progress", 0.75).Msg("test")

		assert.Equal(t, "INF ℹ️ test progress=75%\n", buf.String())
	})

	t.Run("precision_1", func(t *testing.T) {
		var buf bytes.Buffer

		l := New(TestOutput(&buf))
		percent.SetPrecision(1)
		l.Info().Percent("progress", 0.75).Msg("test")

		assert.Equal(t, "INF ℹ️ test progress=75.0%\n", buf.String())
	})
}

func TestSetQuantityUnitsIgnoreCase(t *testing.T) {
	// Default is true.
	assert.True(t, quantity.UnitsIgnoreCase())

	quantity.SetUnitsIgnoreCase(false)
	assert.False(t, quantity.UnitsIgnoreCase())

	quantity.SetUnitsIgnoreCase(true)
	assert.True(t, quantity.UnitsIgnoreCase())
}

func TestSetSeparatorText(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.SetSeparatorText(": ")
	l.Info().Str("key", "val").Msg("test")

	assert.Equal(t, "INF ℹ️ test key: val\n", buf.String())
}

func TestPackageLevelSetElapsedFormatFunc(t *testing.T) {
	SetElapsedFormatFunc(func(d time.Duration) string {
		return d.String()
	})
	assert.NotNil(t, elapsed.FormatFunc())

	// Reset to nil.
	SetElapsedFormatFunc(nil)
	assert.Nil(t, elapsed.FormatFunc())
}

func TestPackageLevelSetElapsedMinimum(t *testing.T) {
	t.Cleanup(func() { elapsed.SetMinimum(time.Second) })

	SetElapsedMinimum(5 * time.Second)
	assert.Equal(t, 5*time.Second, elapsed.Minimum())
}

func TestPackageLevelSetElapsedPrecision(t *testing.T) {
	t.Cleanup(func() { elapsed.SetPrecision(0) })

	SetElapsedPrecision(2)
	assert.Equal(t, 2, elapsed.Precision())
}

func TestPackageLevelSetElapsedRound(t *testing.T) {
	t.Cleanup(func() { elapsed.SetRound(time.Second) })

	SetElapsedRound(time.Minute)
	assert.Equal(t, time.Minute, elapsed.Round())
}

func TestPackageLevelSetFieldSort(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetFieldSort(SortAscending)

	Default.mu.Lock()
	got := Default.fieldSort
	Default.mu.Unlock()

	assert.Equal(t, SortAscending, got)
}

func TestPackageLevelSetPercentFormatFunc(t *testing.T) {
	SetPercentFormatFunc(func(f float64) string {
		return strconv.FormatFloat(f, 'f', 0, 64) + "%"
	})
	assert.NotNil(t, percent.FormatFunc())

	// Reset to nil.
	SetPercentFormatFunc(nil)
	assert.Nil(t, percent.FormatFunc())
}

func TestPackageLevelSetPercentPrecision(t *testing.T) {
	t.Cleanup(func() { percent.SetPrecision(0) })

	SetPercentPrecision(3)
	assert.Equal(t, 3, percent.Precision())
}

func TestPackageLevelSetQuantityUnitsIgnoreCase(t *testing.T) {
	t.Cleanup(func() { quantity.SetUnitsIgnoreCase(true) })

	SetQuantityUnitsIgnoreCase(false)
	assert.False(t, quantity.UnitsIgnoreCase())
}

func TestPackageLevelSetSeparatorText(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = NewWriter(io.Discard)
	SetSeparatorText(": ")

	Default.mu.Lock()
	got := Default.separatorText
	Default.mu.Unlock()

	assert.Equal(t, ": ", got)
}

func TestIsTerminal(t *testing.T) {
	origDefault := Default
	defer func() { Default = origDefault }()

	var buf bytes.Buffer

	Default = New(TestOutput(&buf))

	// In a test environment, output is not a terminal.
	assert.False(t, IsTerminal())
}

func TestColorModeStringBoundary(t *testing.T) {
	// Valid values.
	assert.Equal(t, "auto", ColorAuto.String())
	assert.Equal(t, "always", ColorAlways.String())
	assert.Equal(t, "never", ColorNever.String())

	// Out-of-range negative value.
	assert.Equal(t, "ColorMode(-1)", ColorMode(-1).String())

	// Out-of-range positive value.
	assert.Equal(t, "ColorMode(99)", ColorMode(99).String())
}

func TestWithContextAndCtx(t *testing.T) {
	t.Run("store_and_retrieve", func(t *testing.T) {
		l := NewWriter(io.Discard)
		ctx := l.WithContext(context.Background())

		got := Ctx(ctx)
		assert.Same(t, l, got)
	})

	t.Run("nil_ctx_returns_default", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)

		got := Ctx(nil) //nolint:staticcheck // intentionally testing nil context
		assert.Same(t, Default, got)
	})

	t.Run("no_logger_in_ctx_returns_default", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)

		got := Ctx(context.Background())
		assert.Same(t, Default, got)
	})

	t.Run("retrieved_logger_retains_fields", func(t *testing.T) {
		var got Entry

		l := NewWriter(io.Discard)
		l.SetHandler(HandlerFunc(func(e Entry) {
			got = e
		}))

		sub := l.With().Str("component", "auth").Logger()
		ctx := sub.WithContext(context.Background())

		Ctx(ctx).Info().Msg("test")

		assert.Equal(t, "test", got.Message)
		require.Len(t, got.Fields, 1)
		assert.Equal(t, "component", got.Fields[0].Key)
		assert.Equal(t, "auth", got.Fields[0].Value)
	})

	t.Run("overwrite_logger_in_ctx", func(t *testing.T) {
		l1 := NewWriter(io.Discard)
		l2 := NewWriter(io.Discard)

		ctx := l1.WithContext(context.Background())
		assert.Same(t, l1, Ctx(ctx))

		ctx = l2.WithContext(ctx)
		assert.Same(t, l2, Ctx(ctx))
	})

	t.Run("package_level_WithContext_stores_default", func(t *testing.T) {
		origDefault := Default
		defer func() { Default = origDefault }()

		Default = NewWriter(io.Discard)
		ctx := WithContext(context.Background())

		got := Ctx(ctx)
		assert.Same(t, Default, got)
	})
}

func TestRegisterLevel(t *testing.T) {
	const testLevel Level = 3 // between Dry (2) and Warn (5)

	cleanup := registerTestLevel(t, testLevel, LevelConfig{
		Name:  "success",
		Label: "SCS", Symbol: "✅",
	})
	defer cleanup()

	assert.Equal(t, "SCS", testLevel.String())

	text, err := testLevel.MarshalText()
	require.NoError(t, err)
	assert.Equal(t, "success", string(text))

	parsed, err := ParseLevel("success")
	require.NoError(t, err)
	assert.Equal(t, testLevel, parsed)

	parsed, err = ParseLevel("SUCCESS")
	require.NoError(t, err)
	assert.Equal(t, testLevel, parsed)
}

func TestLogCustomLevel(t *testing.T) {
	const testLevel Level = 3

	cleanup := registerTestLevel(t, testLevel, LevelConfig{
		Name:  "success",
		Label: "SCS", Symbol: "✅",
	})
	defer cleanup()

	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	l.SetLevel(LevelInfo)

	// Register label/symbol on the test logger.
	l.mu.Lock()
	l.labels[testLevel] = "SCS"
	l.symbols[testLevel] = "✅"
	l.labelWidth = computeLabelWidth(l.labels)
	l.recomputePaddedLabels()
	l.mu.Unlock()

	l.Log(testLevel).Msg("Build completed")
	assert.Equal(t, "SCS ✅ Build completed\n", buf.String())
}

func TestRegisterLevelFiltering(t *testing.T) {
	const testLevel Level = 3

	cleanup := registerTestLevel(t, testLevel, LevelConfig{
		Name:  "success",
		Label: "SCS",
	})
	defer cleanup()

	var buf bytes.Buffer
	l := New(TestOutput(&buf))

	// Register label on the test logger.
	l.mu.Lock()
	l.labels[testLevel] = "SCS"
	l.labelWidth = computeLabelWidth(l.labels)
	l.recomputePaddedLabels()
	l.mu.Unlock()

	// Custom level (3) should be visible at LevelInfo (0).
	l.SetLevel(LevelInfo)
	l.Log(testLevel).Msg("visible")
	assert.Equal(t, "SCS visible\n", buf.String())

	// Custom level (3) should be hidden at LevelWarn (5).
	buf.Reset()
	l.SetLevel(LevelWarn)
	e := l.Log(testLevel)
	assert.Nil(t, e, "custom level should be filtered when below minimum")
}

func TestRegisterLevelPanics(t *testing.T) {
	t.Run("empty_name", func(t *testing.T) {
		assert.PanicsWithValue(t, "clog: RegisterLevel requires a non-empty Name", func() {
			RegisterLevel(99, LevelConfig{})
		})
	})

	t.Run("builtin_conflict", func(t *testing.T) {
		assert.PanicsWithValue(t, "level: cannot override built-in level 0", func() {
			RegisterLevel(LevelInfo, LevelConfig{Name: "custom"})
		})
	})
}

func TestRegisterLevelDefaultLabel(t *testing.T) {
	const testLevel Level = 3

	cleanup := registerTestLevel(t, testLevel, LevelConfig{
		Name: "success",
	})
	defer cleanup()

	assert.Equal(
		t,
		"SUC",
		testLevel.String(),
		"default label should be uppercase name truncated to 3 chars",
	)
}

func TestLevelsIncludesCustom(t *testing.T) {
	const testLevel Level = 7

	cleanup := registerTestLevel(t, testLevel, LevelConfig{
		Name: "success",
	})
	defer cleanup()

	levels := Levels()
	assert.Contains(t, levels, testLevel)

	// Verify sorted order.
	for i := 1; i < len(levels); i++ {
		assert.LessOrEqual(t, levels[i-1], levels[i], "Levels() should return sorted levels")
	}
}

// registerTestLevel registers a custom level and returns a cleanup function
// that removes the registration. It is intended for use in tests.
func registerTestLevel(t *testing.T, lvl Level, cfg LevelConfig) func() {
	t.Helper()

	origDefault := Default
	Default = New(NewOutput(io.Discard, ColorNever))

	RegisterLevel(lvl, cfg)

	return func() {
		Default = origDefault
		customLevelsMu.Lock()
		delete(customLevels, lvl)
		delete(defaultSymbols, lvl)
		customLevelsMu.Unlock()
		level.Unregister(lvl)
	}
}
