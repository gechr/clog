package kloghandler_test

import (
	"bytes"
	"errors"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/gechr/clog"
	"github.com/gechr/clog/kloghandler"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

// newTestLogger returns a logger writing to buf with colors disabled and every
// level enabled.
func newTestLogger(buf *bytes.Buffer) *clog.Logger {
	l := clog.New(clog.TestOutput(buf))
	l.SetLevel(clog.LevelTrace)
	return l
}

// marshaler exercises the [logr.Marshaler] escape hatch.
type marshaler struct{ v string }

func (m marshaler) MarshalLog() any { return "marshaled:" + m.v }

func TestVerbosityLevel(t *testing.T) {
	tests := []struct {
		verbosity int
		want      clog.Level
	}{
		{-1, clog.LevelInfo},
		{0, clog.LevelInfo},
		{1, clog.LevelDebug},
		{2, clog.LevelTrace},
		{9, clog.LevelTrace},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, kloghandler.VerbosityLevel(tt.verbosity), "V(%d)", tt.verbosity)
	}
}

func TestEnabledFollowsLoggerLevel(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelDebug)

	sink := kloghandler.New(l, nil)

	assert.True(t, sink.Enabled(0))  // info
	assert.True(t, sink.Enabled(1))  // debug
	assert.False(t, sink.Enabled(2)) // trace
}

func TestEnabledVerbosityCap(t *testing.T) {
	var buf bytes.Buffer
	verbosity := 1
	sink := kloghandler.New(newTestLogger(&buf), &kloghandler.Options{Verbosity: &verbosity})

	assert.True(t, sink.Enabled(0))
	assert.True(t, sink.Enabled(1))
	assert.False(t, sink.Enabled(2), "levels above the cap are disabled even on a trace logger")
}

func TestEnabledCustomLevelFor(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelDebug)

	// The Kubernetes convention: V(4) is debug, V(5) and above is trace.
	sink := kloghandler.New(l, &kloghandler.Options{
		LevelFor: func(verbosity int) clog.Level {
			switch {
			case verbosity < 4:
				return clog.LevelInfo
			case verbosity == 4:
				return clog.LevelDebug
			default:
				return clog.LevelTrace
			}
		},
	})

	assert.True(t, sink.Enabled(3))
	assert.True(t, sink.Enabled(4))
	assert.False(t, sink.Enabled(5))
}

func TestInfoLevelMapping(t *testing.T) {
	tests := []struct {
		verbosity int
		want      string
	}{
		{0, "INF ℹ️ hello k=v\n"},
		{1, "DBG 🐞 hello k=v\n"},
		{2, "TRC 🔍 hello k=v\n"},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.verbosity), func(t *testing.T) {
			var buf bytes.Buffer
			logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

			logger.V(tt.verbosity).Info("hello", "k", "v")

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestInfoDisabled(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelWarn)

	kloghandler.NewLogger(l, nil).Info("hello")

	assert.Empty(t, buf.String())
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

	logger.Error(errBoom, "failed", "attempt", 3)

	assert.Equal(t, "ERR ❌ failed error=boom attempt=3\n", buf.String())
}

func TestErrorNil(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

	logger.Error(nil, "failed")

	assert.Equal(t, "ERR ❌ failed\n", buf.String(), "a nil error contributes no field")
}

func TestWithValues(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

	logger.WithValues("preset", "v1").Info("test", "dynamic", "v2")

	assert.Equal(t, "INF ℹ️ test preset=v1 dynamic=v2\n", buf.String())
}

func TestWithValuesDoesNotLeakIntoSibling(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

	base := logger.WithValues("shared", "yes")
	base.WithValues("only", "a").Info("first")
	base.WithValues("only", "b").Info("second")

	assert.Equal(
		t,
		"INF ℹ️ first shared=yes only=a\nINF ℹ️ second shared=yes only=b\n",
		buf.String(),
	)
}

func TestWithName(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

	logger.WithName("controller").WithName("pod").Info("test")

	assert.Equal(t, "INF ℹ️ test logger=controller.pod\n", buf.String())
}

func TestWithNameCustomKey(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), &kloghandler.Options{NameKey: "component"})

	logger.WithName("reconciler").Info("test")

	assert.Equal(t, "INF ℹ️ test component=reconciler\n", buf.String())
}

func TestWithNameOrdering(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

	logger.WithName("svc").WithValues("preset", "v1").Info("test", "dynamic", "v2")

	assert.Equal(t, "INF ℹ️ test logger=svc preset=v1 dynamic=v2\n", buf.String())
}

func TestWithNameEmpty(t *testing.T) {
	var buf bytes.Buffer
	sink := kloghandler.New(newTestLogger(&buf), nil)

	assert.Same(t, sink, sink.WithName(""), "an empty name should return the same sink")
	assert.Same(t, sink, sink.WithValues(), "no values should return the same sink")
}

func TestKeysAndValues(t *testing.T) {
	tests := []struct {
		name          string
		keysAndValues []any
		want          string
	}{
		{"string", []any{"k", "v"}, "INF ℹ️ test k=v\n"},
		{"int", []any{"k", 42}, "INF ℹ️ test k=42\n"},
		{"bool", []any{"k", true}, "INF ℹ️ test k=true\n"},
		{"multiple", []any{"a", 1, "b", 2}, "INF ℹ️ test a=1 b=2\n"},
		{"non-string key", []any{42, "v"}, "INF ℹ️ test 42=v\n"},
		{"dangling value", []any{"a", 1, "orphan"}, "INF ℹ️ test a=1 !BADKEY=orphan\n"},
		{"marshaler", []any{"k", marshaler{v: "x"}}, "INF ℹ️ test k=marshaled:x\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := kloghandler.NewLogger(newTestLogger(&buf), nil)

			logger.Info("test", tt.keysAndValues...)

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestAddSource(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), &kloghandler.Options{AddSource: true})

	_, wantFile, wantLine, ok := runtime.Caller(0)
	logger.Info("test") // must stay on the line directly below the Caller call
	require.True(t, ok)

	source := sourceField(t, buf.String())
	assert.Equal(t, wantFile+":"+strconv.Itoa(wantLine+1), source)
}

func TestAddSourceWithCallDepth(t *testing.T) {
	var buf bytes.Buffer
	logger := kloghandler.NewLogger(newTestLogger(&buf), &kloghandler.Options{AddSource: true})

	_, _, wantLine, ok := runtime.Caller(0)
	logViaHelper(logger, "test") // the helper frame is skipped, so this line wins
	require.True(t, ok)

	source := sourceField(t, buf.String())
	_, line := splitSource(t, source)
	assert.Equal(t, wantLine+1, line)
}

// logViaHelper logs on the caller's behalf, the pattern [logr.Logger.WithCallDepth]
// exists for.
func logViaHelper(logger logr.Logger, msg string) {
	logger.WithCallDepth(1).Info(msg)
}

// sourceField extracts the value of the source field from a rendered line.
func sourceField(t *testing.T, output string) string {
	t.Helper()

	_, value, found := strings.Cut(strings.TrimRight(output, "\n"), kloghandler.SourceKey+"=")
	require.True(t, found, "no source field in %q", output)
	return value
}

// splitSource splits a "file:line" source value back into its parts.
func splitSource(t *testing.T, source string) (string, int) {
	t.Helper()

	file, digits, found := strings.Cut(filepath.Base(source), ":")
	require.True(t, found, "malformed source %q", source)

	line, err := strconv.Atoi(digits)
	require.NoError(t, err)
	return file, line
}
