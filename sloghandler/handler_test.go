package sloghandler_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/gechr/clog"
	"github.com/gechr/clog/sloghandler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestHandler returns a handler that writes to buf with a predictable format.
func newTestHandler(buf *bytes.Buffer) slog.Handler {
	l := clog.New(clog.TestOutput(buf))
	l.SetLevel(clog.LevelTrace)
	l.SetReportTimestamp(true)
	l.SetTimeFormat(time.RFC3339)
	l.SetTimeLocation(time.UTC)
	return sloghandler.New(l, nil)
}

func TestConformance(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelTrace)
	// slog conformance requires the record time to propagate; visibility is
	// governed by the logger, so reporting must be on for it to surface.
	l.SetReportTimestamp(true)

	// Use a clog handler to capture structured entries for slogtest verification.
	var entries []map[string]any
	l.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
		m := map[string]any{
			slog.MessageKey: e.Message,
			slog.LevelKey:   e.Level.String(),
		}
		if !e.Time.IsZero() {
			m[slog.TimeKey] = e.Time
		}
		for _, f := range e.Fields {
			setNested(m, f.Key, f.Value)
		}
		entries = append(entries, m)
	}))

	h := sloghandler.New(l, &sloghandler.Options{AddSource: false})

	slogtest.Run(t, func(*testing.T) slog.Handler {
		entries = nil
		return h
	}, func(*testing.T) map[string]any {
		if len(entries) == 0 {
			return nil
		}
		return entries[len(entries)-1]
	})
}

func TestEnabled(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelWarn)

	h := sloghandler.New(l, nil)

	assert.False(t, h.Enabled(context.Background(), slog.LevelDebug))
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.True(t, h.Enabled(context.Background(), slog.LevelWarn))
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
}

func TestEnabledWithOptions(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelTrace) // logger allows everything

	h := sloghandler.New(l, &sloghandler.Options{Level: slog.LevelError})

	assert.False(t, h.Enabled(context.Background(), slog.LevelDebug))
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.False(t, h.Enabled(context.Background(), slog.LevelWarn))
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
}

func TestHandle(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	r := slog.NewRecord(
		time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		slog.LevelInfo,
		"hello world",
		0,
	)
	r.AddAttrs(slog.String("key", "val"))

	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ hello world key=val\n", output)
}

func TestAttrConversion(t *testing.T) {
	tests := []struct {
		name string
		attr slog.Attr
		want string
	}{
		{"string", slog.String("k", "v"), "2024-01-15T10:30:00Z INF ℹ️ test k=v\n"},
		{"int64", slog.Int64("k", 42), "2024-01-15T10:30:00Z INF ℹ️ test k=42\n"},
		{"uint64", slog.Uint64("k", 99), "2024-01-15T10:30:00Z INF ℹ️ test k=99\n"},
		{"float64", slog.Float64("k", 3.14), "2024-01-15T10:30:00Z INF ℹ️ test k=3.14\n"},
		{"bool", slog.Bool("k", true), "2024-01-15T10:30:00Z INF ℹ️ test k=true\n"},
		{"duration", slog.Duration("k", 5*time.Second), "2024-01-15T10:30:00Z INF ℹ️ test k=5s\n"},
	}

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := newTestHandler(&buf)

			r := slog.NewRecord(ts, slog.LevelInfo, "test", 0)
			r.AddAttrs(tt.attr)
			err := h.Handle(context.Background(), r)
			require.NoError(t, err)

			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestWithAttrsOrdering(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	h = h.WithAttrs([]slog.Attr{slog.String("handler", "first")})

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("record", "second"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	handlerIdx := strings.Index(output, "handler=first")
	recordIdx := strings.Index(output, "record=second")

	require.NotEqual(t, -1, handlerIdx, "handler attr not found")
	require.NotEqual(t, -1, recordIdx, "record attr not found")
	assert.Less(t, handlerIdx, recordIdx, "handler attrs should appear before record attrs")
}

func TestWithGroup(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	h = h.WithGroup("a").WithGroup("b")

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("key", "val"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ test a.b.key=val\n", buf.String())
}

func TestWithGroupAndAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	h = h.WithGroup("g").WithAttrs([]slog.Attr{slog.String("preset", "v1")})

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("dynamic", "v2"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ test g.preset=v1 g.dynamic=v2\n", output)
}

func TestWithGroupEmpty(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	h2 := h.WithGroup("")
	assert.Same(t, h, h2, "empty group name should return same handler")
}

func TestWithAttrsEmpty(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	h2 := h.WithAttrs(nil)
	assert.Same(t, h, h2, "nil attrs should return same handler")
}

func TestEmptyAttrDropped(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Attr{}) // empty attr
	r.AddAttrs(slog.String("real", "val"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ test real=val\n", output)
}

func TestAddSource(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelTrace)
	l.SetReportTimestamp(true)
	l.SetTimeFormat(time.RFC3339)
	l.SetTimeLocation(time.UTC)
	h := sloghandler.New(l, &sloghandler.Options{AddSource: true})

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	var pcs [1]uintptr
	runtime.Callers(1, pcs[:])
	r := slog.NewRecord(ts, slog.LevelInfo, "with source", pcs[0])
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	frames := runtime.CallersFrames(pcs[:])
	frame, _ := frames.Next()
	output := buf.String()
	assert.Equal(
		t,
		fmt.Sprintf(
			"2024-01-15T10:30:00Z INF ℹ️ with source source=%s:%d\n",
			frame.File,
			frame.Line,
		),
		output,
	)
}

type testLogValuer struct {
	val string
}

func (v testLogValuer) LogValue() slog.Value {
	return slog.StringValue(v.val)
}

func TestLogValuer(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Any("resolved", testLogValuer{val: "inner"}))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ test resolved=inner\n", buf.String())
}

func TestFatalLevelDoesNotExit(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelTrace)
	l.SetReportTimestamp(true)
	l.SetTimeFormat(time.RFC3339)
	l.SetTimeLocation(time.UTC)

	exited := false
	l.SetExitFunc(func(int) { exited = true })

	h := sloghandler.New(l, nil)

	// slog.Level above Error maps to LevelFatal
	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelError+4, "should not exit", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.False(t, exited, "slog handler should not trigger exit for fatal-level records")
	assert.Equal(t, "2024-01-15T10:30:00Z FTL 💥 should not exit\n", buf.String())
}

func TestGroupAttr(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Group("req",
		slog.String("method", "GET"),
		slog.Int("status", 200),
	))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ test req.method=GET req.status=200\n", output)
}

func TestInlineGroupAttr(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "test", 0)
	// Empty key = inline group
	r.AddAttrs(slog.Group("",
		slog.String("a", "1"),
		slog.String("b", "2"),
	))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ test a=1 b=2\n", output)
}

func TestTimestamp(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "timestamped", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Equal(t, "2024-06-15T14:30:00Z INF ℹ️ timestamped\n", buf.String())
}

func TestTimestampHiddenWhenLoggerDisablesReporting(t *testing.T) {
	var buf bytes.Buffer
	// Reporting left off: the record's (always non-zero) time must not force a
	// timestamp through - the bridge inherits the logger's visibility.
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelTrace)
	h := sloghandler.New(l, nil)

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "timestamped", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Equal(t, "INF ℹ️ timestamped\n", buf.String())
}

func TestInterface(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetReportTimestamp(true)
	l.SetTimeFormat(time.RFC3339)
	l.SetTimeLocation(time.UTC)
	h := sloghandler.New(l, nil)

	ts := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "via slog", 0)
	r.AddAttrs(slog.String("key", "val"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Equal(t, "2024-01-15T10:30:00Z INF ℹ️ via slog key=val\n", buf.String())
}

// setNested stores val in m at a potentially dot-separated key path,
// creating nested maps as needed. This reconstructs the nested structure
// that slogtest expects from dot-notation keys like "G.a".
func setNested(m map[string]any, key string, val any) {
	parts := strings.Split(key, ".")
	for _, p := range parts[:len(parts)-1] {
		if sub, ok := m[p]; ok {
			if subMap, ok := sub.(map[string]any); ok {
				m = subMap
				continue
			}
		}
		// Key collision or first time: create intermediate map.
		sub := map[string]any{}
		m[p] = sub
		m = sub
	}
	m[parts[len(parts)-1]] = val
}
