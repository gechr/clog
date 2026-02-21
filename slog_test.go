package clog

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"testing/slogtest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSlogHandler returns a SlogHandler that writes to buf with a predictable format.
func newTestSlogHandler(buf *bytes.Buffer) slog.Handler {
	l := New(TestOutput(buf))
	l.SetLevel(TraceLevel)
	l.SetReportTimestamp(true)
	l.SetTimeFormat(time.RFC3339)
	l.SetTimeLocation(time.UTC)
	return NewSlogHandler(l, nil)
}

func TestSlogConformance(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	l.SetLevel(TraceLevel)

	// Use a handler to capture structured entries for slogtest verification.
	var entries []map[string]any
	l.SetHandler(HandlerFunc(func(e Entry) {
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

	h := NewSlogHandler(l, &SlogOptions{AddSource: false})

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

func TestSlogLevelMapping(t *testing.T) {
	tests := []struct {
		slogLevel slog.Level
		clogLevel Level
	}{
		{slog.Level(-8), TraceLevel}, // below debug
		{slog.LevelDebug - 1, TraceLevel},
		{slog.LevelDebug, DebugLevel},
		{slog.LevelDebug + 1, DebugLevel},
		{slog.LevelInfo - 1, DebugLevel},
		{slog.LevelInfo, InfoLevel},
		{slog.LevelInfo + 1, InfoLevel},
		{slog.LevelWarn - 1, InfoLevel},
		{slog.LevelWarn, WarnLevel},
		{slog.LevelWarn + 1, WarnLevel},
		{slog.LevelError - 1, WarnLevel},
		{slog.LevelError, ErrorLevel},
		{slog.LevelError + 1, FatalLevel},
		{slog.Level(12), FatalLevel}, // well above error
	}

	for _, tt := range tests {
		got := slogLevelToClog(tt.slogLevel)
		assert.Equal(t, tt.clogLevel, got, "slog.Level(%d)", tt.slogLevel)
	}
}

func TestSlogEnabled(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	l.SetLevel(WarnLevel)

	h := NewSlogHandler(l, nil)

	assert.False(t, h.Enabled(context.Background(), slog.LevelDebug))
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.True(t, h.Enabled(context.Background(), slog.LevelWarn))
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
}

func TestSlogEnabledWithOptions(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	l.SetLevel(TraceLevel) // logger allows everything

	h := NewSlogHandler(l, &SlogOptions{Level: slog.LevelError})

	assert.False(t, h.Enabled(context.Background(), slog.LevelDebug))
	assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
	assert.False(t, h.Enabled(context.Background(), slog.LevelWarn))
	assert.True(t, h.Enabled(context.Background(), slog.LevelError))
}

func TestSlogHandle(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

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
	assert.Contains(t, output, "hello world")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "val")
}

func TestSlogAttrConversion(t *testing.T) {
	tests := []struct {
		name string
		attr slog.Attr
		want string // substring to find in output
	}{
		{"string", slog.String("k", "v"), "k=v"},
		{"int64", slog.Int64("k", 42), "k=42"},
		{"uint64", slog.Uint64("k", 99), "k=99"},
		{"float64", slog.Float64("k", 3.14), "k=3.14"},
		{"bool", slog.Bool("k", true), "k=true"},
		{"duration", slog.Duration("k", 5*time.Second), "k=5s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			h := newTestSlogHandler(&buf)

			r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
			r.AddAttrs(tt.attr)
			err := h.Handle(context.Background(), r)
			require.NoError(t, err)

			assert.Contains(t, buf.String(), tt.want)
		})
	}
}

func TestSlogWithAttrsOrdering(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

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

func TestSlogWithGroup(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	h = h.WithGroup("a").WithGroup("b")

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("key", "val"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "a.b.key=val")
}

func TestSlogWithGroupAndAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	h = h.WithGroup("g").WithAttrs([]slog.Attr{slog.String("preset", "v1")})

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("dynamic", "v2"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "g.preset=v1")
	assert.Contains(t, output, "g.dynamic=v2")
}

func TestSlogWithGroupEmpty(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	h2 := h.WithGroup("")
	assert.Same(t, h, h2, "empty group name should return same handler")
}

func TestSlogWithAttrsEmpty(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	h2 := h.WithAttrs(nil)
	assert.Same(t, h, h2, "nil attrs should return same handler")
}

func TestSlogEmptyAttrDropped(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Attr{}) // empty attr
	r.AddAttrs(slog.String("real", "val"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "real=val")
}

func TestSlogAddSource(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	l.SetLevel(TraceLevel)
	h := NewSlogHandler(l, &SlogOptions{AddSource: true})

	logger := slog.New(h)
	logger.Info("with source")

	output := buf.String()
	assert.Contains(t, output, slog.SourceKey)
	assert.Contains(t, output, "slog_test.go")
}

type testLogValuer struct {
	val string
}

func (t testLogValuer) LogValue() slog.Value {
	return slog.StringValue(t.val)
}

func TestSlogLogValuer(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Any("resolved", testLogValuer{val: "inner"}))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "resolved=inner")
}

func TestSlogFatalLevelDoesNotExit(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	l.SetLevel(TraceLevel)

	exited := false
	l.SetExitFunc(func(int) { exited = true })

	h := NewSlogHandler(l, nil)

	// slog.Level above Error maps to FatalLevel
	r := slog.NewRecord(time.Now(), slog.LevelError+4, "should not exit", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.False(t, exited, "slog handler should not trigger exit for fatal-level records")
	assert.Contains(t, buf.String(), "should not exit")
}

func TestSlogGroupAttr(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Group("req",
		slog.String("method", "GET"),
		slog.Int("status", 200),
	))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "req.method=GET")
	assert.Contains(t, output, "req.status=200")
}

func TestSlogInlineGroupAttr(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	// Empty key = inline group
	r.AddAttrs(slog.Group("",
		slog.String("a", "1"),
		slog.String("b", "2"),
	))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "a=1")
	assert.Contains(t, output, "b=2")
}

func TestSlogTimestamp(t *testing.T) {
	var buf bytes.Buffer
	h := newTestSlogHandler(&buf)

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "timestamped", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "2024-06-15T14:30:00Z")
}

func TestSlogInterface(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	h := NewSlogHandler(l, nil)

	logger := slog.New(h)
	logger.Info("via slog", "key", "val")

	assert.Contains(t, buf.String(), "via slog")
	assert.Contains(t, buf.String(), "key=val")
}

// setNested stores val in m at a potentially dot-separated key path,
// creating nested maps as needed. This reconstructs the nested structure
// that slogtest.Run expects from dot-notation keys like "G.a".
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
