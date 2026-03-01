package sloghandler_test

import (
	"bytes"
	"context"
	"log/slog"
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
	assert.Contains(t, output, "hello world")
	assert.Contains(t, output, "key")
	assert.Contains(t, output, "val")
}

func TestAttrConversion(t *testing.T) {
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
			h := newTestHandler(&buf)

			r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
			r.AddAttrs(tt.attr)
			err := h.Handle(context.Background(), r)
			require.NoError(t, err)

			assert.Contains(t, buf.String(), tt.want)
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

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("key", "val"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "a.b.key=val")
}

func TestWithGroupAndAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	h = h.WithGroup("g").WithAttrs([]slog.Attr{slog.String("preset", "v1")})

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.String("dynamic", "v2"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "g.preset=v1")
	assert.Contains(t, output, "g.dynamic=v2")
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

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Attr{}) // empty attr
	r.AddAttrs(slog.String("real", "val"))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "real=val")
}

func TestAddSource(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelTrace)
	h := sloghandler.New(l, &sloghandler.Options{AddSource: true})

	logger := slog.New(h)
	logger.Info("with source")

	output := buf.String()
	assert.Contains(t, output, slog.SourceKey)
	assert.Contains(t, output, "handler_test.go")
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

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test", 0)
	r.AddAttrs(slog.Any("resolved", testLogValuer{val: "inner"}))
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "resolved=inner")
}

func TestFatalLevelDoesNotExit(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	l.SetLevel(clog.LevelTrace)

	exited := false
	l.SetExitFunc(func(int) { exited = true })

	h := sloghandler.New(l, nil)

	// slog.Level above Error maps to LevelFatal
	r := slog.NewRecord(time.Now(), slog.LevelError+4, "should not exit", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.False(t, exited, "slog handler should not trigger exit for fatal-level records")
	assert.Contains(t, buf.String(), "should not exit")
}

func TestGroupAttr(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

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

func TestInlineGroupAttr(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

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

func TestTimestamp(t *testing.T) {
	var buf bytes.Buffer
	h := newTestHandler(&buf)

	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	r := slog.NewRecord(ts, slog.LevelInfo, "timestamped", 0)
	err := h.Handle(context.Background(), r)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), "2024-06-15T14:30:00Z")
}

func TestInterface(t *testing.T) {
	var buf bytes.Buffer
	l := clog.New(clog.TestOutput(&buf))
	h := sloghandler.New(l, nil)

	logger := slog.New(h)
	logger.Info("via slog", "key", "val")

	assert.Contains(t, buf.String(), "via slog")
	assert.Contains(t, buf.String(), "key=val")
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
