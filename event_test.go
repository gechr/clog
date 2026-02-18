package clog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStringer struct {
	s string
}

func (ts testStringer) String() string { return ts.s }

func TestEventStr(t *testing.T) {
	e := New(io.Discard).Info()
	e.Str("key", "val")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "key", e.fields[0].Key)
	assert.Equal(t, "val", e.fields[0].Value)
}

func TestEventStrs(t *testing.T) {
	e := New(io.Discard).Info()
	e.Strs("keys", []string{"a", "b"})

	require.Len(t, e.fields, 1)

	vals, ok := e.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "b"}, vals)
}

func TestEventInt(t *testing.T) {
	e := New(io.Discard).Info()
	e.Int("count", 42)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "count", e.fields[0].Key)
	assert.Equal(t, 42, e.fields[0].Value)
}

func TestEventInts(t *testing.T) {
	e := New(io.Discard).Info()
	e.Ints("nums", []int{1, 2, 3})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "nums", e.fields[0].Key)
}

func TestEventUint64(t *testing.T) {
	e := New(io.Discard).Info()
	e.Uint64("size", 999)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "size", e.fields[0].Key)
	assert.Equal(t, uint64(999), e.fields[0].Value)
}

func TestEventUints64(t *testing.T) {
	e := New(io.Discard).Info()
	e.Uints64("sizes", []uint64{1, 2, 3})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "sizes", e.fields[0].Key)
}

func TestEventFloat64(t *testing.T) {
	e := New(io.Discard).Info()
	e.Float64("pi", 3.14)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "pi", e.fields[0].Key)
	assert.InDelta(t, 3.14, e.fields[0].Value, 0)
}

func TestEventFloats64(t *testing.T) {
	e := New(io.Discard).Info()
	e.Floats64("vals", []float64{1.1, 2.2})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "vals", e.fields[0].Key)
	assert.Equal(t, []float64{1.1, 2.2}, e.fields[0].Value)
}

func TestEventLink(t *testing.T) {
	l := New(io.Discard)
	e := l.Info()
	e.Link("docs", "https://example.com", "docs")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "docs", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so returns plain text.
	assert.Equal(t, "docs", e.fields[0].Value)
}

func TestEventLinkColorAlways(t *testing.T) {
	l := New(io.Discard)
	l.SetColorMode(ColorAlways)

	e := l.Info()
	e.Link("docs", "https://example.com", "docs")

	require.Len(t, e.fields, 1)

	val, ok := e.fields[0].Value.(string)
	require.True(t, ok)
	assert.Contains(t, val, "\x1b]8;;https://example.com")
	assert.Contains(t, val, "docs")
}

func TestEventURL(t *testing.T) {
	l := New(io.Discard)
	e := l.Info()
	e.URL("link", "https://example.com")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "link", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so returns plain text.
	assert.Equal(t, "https://example.com", e.fields[0].Value)
}

func TestEventURLColorAlways(t *testing.T) {
	l := New(io.Discard)
	l.SetColorMode(ColorAlways)

	e := l.Info()
	e.URL("link", "https://example.com")

	require.Len(t, e.fields, 1)

	val, ok := e.fields[0].Value.(string)
	require.True(t, ok)
	assert.Equal(t, "\x1b]8;;https://example.com\x1b\\https://example.com\x1b]8;;\x1b\\", val)
}

func TestEventBool(t *testing.T) {
	e := New(io.Discard).Info()
	e.Bool("ok", true)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "ok", e.fields[0].Key)
	assert.Equal(t, true, e.fields[0].Value)
}

func TestEventBools(t *testing.T) {
	e := New(io.Discard).Info()
	e.Bools("flags", []bool{true, false})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "flags", e.fields[0].Key)
	assert.Equal(t, []bool{true, false}, e.fields[0].Value)
}

func TestEventDur(t *testing.T) {
	e := New(io.Discard).Info()
	e.Duration("elapsed", time.Second)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "elapsed", e.fields[0].Key)
	assert.Equal(t, time.Second, e.fields[0].Value)
}

func TestEventTime(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	e := New(io.Discard).Info()
	e.Time("created", ts)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "created", e.fields[0].Key)
	assert.Equal(t, ts, e.fields[0].Value)
}

func TestEventAny(t *testing.T) {
	e := New(io.Discard).Info()
	e.Any("data", 123)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "data", e.fields[0].Key)
	assert.Equal(t, 123, e.fields[0].Value)
}

func TestEventAnys(t *testing.T) {
	e := New(io.Discard).Info()
	vals := []any{"hello", 42, true}
	e.Anys("mixed", vals)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "mixed", e.fields[0].Key)

	got, ok := e.fields[0].Value.([]any)
	require.True(t, ok, "expected []any value")
	assert.Equal(t, vals, got)
}

func TestEventDict(t *testing.T) {
	e := New(io.Discard).Info()
	e.Dict("request", Dict().Str("method", "GET").Int("status", 200))

	require.Len(t, e.fields, 2)
	assert.Equal(t, "request.method", e.fields[0].Key)
	assert.Equal(t, "GET", e.fields[0].Value)
	assert.Equal(t, "request.status", e.fields[1].Key)
	assert.Equal(t, 200, e.fields[1].Value)
}

func TestEventDictNilReceiver(t *testing.T) {
	var e *Event
	got := e.Dict("k", Dict().Str("a", "b"))

	assert.Nil(t, got)
}

func TestEventDictOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Dict("req", Dict().Str("method", "GET").Int("status", 200)).Msg("handled")

	assert.Equal(t, "INF ℹ️ handled req.method=GET req.status=200\n", buf.String())
}

func TestEventErr(t *testing.T) {
	e := New(io.Discard).Info()
	err := errors.New("boom")
	e.Err(err)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "error", e.fields[0].Key)

	gotErr, ok := e.fields[0].Value.(error)
	require.True(t, ok, "expected error value")
	assert.Equal(t, "boom", gotErr.Error())
}

func TestEventErrNil(t *testing.T) {
	e := New(io.Discard).Info()
	result := e.Err(nil)

	assert.Same(t, e, result, "expected same event returned")
	assert.Empty(t, e.fields)
}

func TestEventPath(t *testing.T) {
	l := New(io.Discard)
	e := l.Info()
	e.Path("dir", "/tmp")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "dir", e.fields[0].Key)
	assert.Equal(t, "/tmp", e.fields[0].Value)
}

func TestEventLine(t *testing.T) {
	l := New(io.Discard)
	e := l.Info()
	e.Line("file", "main.go", 42)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "file", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so pathLinkWithMode returns plain text.
	assert.Equal(t, "main.go:42", e.fields[0].Value)
}

func TestEventLineColorAlways(t *testing.T) {
	l := New(io.Discard)
	l.SetColorMode(ColorAlways)

	e := l.Info()
	e.Line("file", "main.go", 10)

	require.Len(t, e.fields, 1)

	val, ok := e.fields[0].Value.(string)
	require.True(t, ok)
	// ColorAlways produces OSC 8 hyperlink sequences.
	assert.Equal(t, "file", e.fields[0].Key)
	assert.Contains(t, val, "\x1b]8;;")
	assert.Contains(t, val, "main.go:10")
}

func TestEventLineColorNever(t *testing.T) {
	l := New(io.Discard)
	l.SetColorMode(ColorNever)

	e := l.Info()
	e.Line("file", "main.go", 10)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "main.go:10", e.fields[0].Value)
}

func TestEventLineMinimum(t *testing.T) {
	l := New(io.Discard)
	e := l.Info()
	e.Line("file", "main.go", 0)

	require.Len(t, e.fields, 1)
	// Line number 0 should be clamped to 1.
	assert.Equal(t, "main.go:1", e.fields[0].Value)
}

func TestEventColumn(t *testing.T) {
	l := New(io.Discard)
	e := l.Info()
	e.Column("loc", "main.go", 42, 10)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "loc", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so returns plain text.
	assert.Equal(t, "main.go:42:10", e.fields[0].Value)
}

func TestEventColumnColorAlways(t *testing.T) {
	clearFormats(t)

	l := New(io.Discard)
	l.SetColorMode(ColorAlways)

	e := l.Info()
	e.Column("loc", "/tmp/test.go", 10, 5)

	require.Len(t, e.fields, 1)

	val, ok := e.fields[0].Value.(string)
	require.True(t, ok)
	assert.Contains(t, val, "\x1b]8;;")
	assert.Contains(t, val, "/tmp/test.go:10:5")
}

func TestEventColumnMinimum(t *testing.T) {
	l := New(io.Discard)
	e := l.Info()
	e.Column("loc", "main.go", 0, 0)

	require.Len(t, e.fields, 1)
	// Both line and column should be clamped to 1.
	assert.Equal(t, "main.go:1:1", e.fields[0].Value)
}

func TestEventStringer(t *testing.T) {
	e := New(io.Discard).Info()
	e.Stringer("name", testStringer{s: "hello"})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "name", e.fields[0].Key)
	assert.Equal(t, "hello", e.fields[0].Value)
}

func TestEventStringerNil(t *testing.T) {
	e := New(io.Discard).Info()
	result := e.Stringer("key", nil)

	assert.Same(t, e, result, "expected same event returned")
	assert.Empty(t, e.fields)
}

func TestEventStringers(t *testing.T) {
	e := New(io.Discard).Info()
	e.Stringers("items", []fmt.Stringer{testStringer{s: "x"}, testStringer{s: "y"}})

	require.Len(t, e.fields, 1)

	vals, ok := e.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"x", "y"}, vals)
}

func TestEventStringersWithNil(t *testing.T) {
	e := New(io.Discard).Info()
	e.Stringers("items", []fmt.Stringer{testStringer{s: "x"}, nil})

	require.Len(t, e.fields, 1)

	vals, ok := e.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"x", "<nil>"}, vals)
}

func TestEventPrefix(t *testing.T) {
	l := New(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Prefix(">>>").Msg("test")

	assert.Equal(t, ">>>", got.Prefix)
}

func TestEventNilReceiverSafety(t *testing.T) {
	var e *Event

	// All field methods should return nil without panic.
	assert.Nil(t, e.Str("k", "v"))
	assert.Nil(t, e.Strs("k", []string{"v"}))
	assert.Nil(t, e.Int("k", 1))
	assert.Nil(t, e.Ints("k", []int{1}))
	assert.Nil(t, e.Uint64("k", 1))
	assert.Nil(t, e.Uints64("k", []uint64{1}))
	assert.Nil(t, e.Float64("k", 1.0))
	assert.Nil(t, e.Floats64("k", []float64{1.0}))
	assert.Nil(t, e.Bool("k", true))
	assert.Nil(t, e.Bools("k", []bool{true}))
	assert.Nil(t, e.Duration("k", time.Second))
	assert.Nil(t, e.Time("k", time.Now()))
	assert.Nil(t, e.Err(errors.New("x")))
	assert.Nil(t, e.Any("k", "v"))
	assert.Nil(t, e.Anys("k", []any{"v"}))
	assert.Nil(t, e.Dict("k", Dict().Str("a", "b")))
	assert.Nil(t, e.Path("k", "file.go"))
	assert.Nil(t, e.Line("k", "file.go", 1))
	assert.Nil(t, e.Column("k", "file.go", 1, 1))
	assert.Nil(t, e.Link("k", "https://example.com", "text"))
	assert.Nil(t, e.URL("k", "https://example.com"))
	assert.Nil(t, e.Durations("k", []time.Duration{time.Second}))
	assert.Nil(t, e.Percent("k", 50))
	assert.Nil(t, e.Quantity("k", "10GB"))
	assert.Nil(t, e.Quantities("k", []string{"10GB"}))
	assert.Nil(t, e.Stringer("k", testStringer{s: "x"}))
	assert.Nil(t, e.Stringers("k", []fmt.Stringer{testStringer{s: "x"}}))
	assert.Nil(t, e.Prefix("p"))
	assert.Nil(t, e.withFields([]Field{{Key: "k", Value: "v"}}))
	assert.Nil(t, e.withPrefix("p"))

	// Finalizers should not panic.
	e.Msg("test")
	e.Msgf("test %s", "arg")
	e.Send()
}

func TestEventMsg(t *testing.T) {
	l := New(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Str("k", "v").Msg("hello")

	assert.Equal(t, InfoLevel, got.Level)
	assert.Equal(t, "hello", got.Message)
	require.Len(t, got.Fields, 1)
	assert.Equal(t, "k", got.Fields[0].Key)
}

func TestEventMsgf(t *testing.T) {
	l := New(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Msgf("hello %s %d", "world", 42)

	assert.Equal(t, "hello world 42", got.Message)
}

func TestEventSend(t *testing.T) {
	l := New(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Str("k", "v").Send()

	assert.Empty(t, got.Message)
	assert.Len(t, got.Fields, 1)
}

func TestEventWithFields(t *testing.T) {
	e := New(io.Discard).Info()
	e = e.withFields([]Field{{Key: "a", Value: "1"}, {Key: "b", Value: "2"}})

	require.Len(t, e.fields, 2)
	assert.Equal(t, "a", e.fields[0].Key)
	assert.Equal(t, "b", e.fields[1].Key)
}

func TestEventWithFieldsNilReceiver(t *testing.T) {
	var e *Event

	got := e.withFields([]Field{{Key: "a", Value: "1"}})
	assert.Nil(t, got, "expected nil from withFields on nil event")
}

func TestEventWithPrefix(t *testing.T) {
	e := New(io.Discard).Info()
	e = e.withPrefix("CUSTOM")

	require.NotNil(t, e.prefix)
	assert.Equal(t, "CUSTOM", *e.prefix)
}

func TestEventWithPrefixNilReceiver(t *testing.T) {
	var e *Event

	got := e.withPrefix("CUSTOM")
	assert.Nil(t, got, "expected nil from withPrefix on nil event")
}

func TestEventChaining(t *testing.T) {
	l := New(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().
		Str("s", "val").
		Int("i", 42).
		Bool("b", true).
		Msg("chained")

	assert.Equal(t, "chained", got.Message)
	require.Len(t, got.Fields, 3)
}

func TestEventDurations(t *testing.T) {
	e := New(io.Discard).Info()
	vals := []time.Duration{time.Second, 2 * time.Millisecond}
	e.Durations("timings", vals)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "timings", e.fields[0].Key)

	got, ok := e.fields[0].Value.([]time.Duration)
	require.True(t, ok, "expected []time.Duration value")
	assert.Equal(t, vals, got)
}

func TestEventDurationsOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Durations("d", []time.Duration{time.Second, 500 * time.Millisecond}).Msg("test")

	assert.Equal(t, "INF ℹ️ test d=[1s, 500ms]\n", buf.String())
}

func TestEventPercent(t *testing.T) {
	e := New(io.Discard).Info()
	e.Percent("progress", 75)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "progress", e.fields[0].Key)

	p, ok := e.fields[0].Value.(percent)
	require.True(t, ok, "expected percent value")
	assert.InDelta(t, 75.0, float64(p), 0)
}

func TestEventPercentClamping(t *testing.T) {
	e := New(io.Discard).Info()
	e.Percent("low", -10)
	e.Percent("high", 150)

	require.Len(t, e.fields, 2)

	low, ok := e.fields[0].Value.(percent)
	require.True(t, ok)
	assert.InDelta(t, 0.0, float64(low), 0, "negative should clamp to 0")

	high, ok := e.fields[1].Value.(percent)
	require.True(t, ok)
	assert.InDelta(t, 100.0, float64(high), 0, "over 100 should clamp to 100")
}

func TestEventPercentOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Percent("progress", 75).Msg("done")

	assert.Equal(t, "INF ℹ️ done progress=75%\n", buf.String())
}

func TestEventQuantity(t *testing.T) {
	e := New(io.Discard).Info()
	e.Quantity("size", "10GB")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "size", e.fields[0].Key)
}

func TestEventQuantityOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Quantity("size", "10GB").Msg("done")

	assert.Equal(t, "INF ℹ️ done size=10GB\n", buf.String())
}

func TestEventQuantities(t *testing.T) {
	e := New(io.Discard).Info()
	e.Quantities("sizes", []string{"10GB", "5MB"})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "sizes", e.fields[0].Key)
}

func TestEventQuantitiesOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Quantities("sizes", []string{"10GB", "5MB"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test sizes=[10GB, 5MB]\n", buf.String())
}

func TestEventDictNilParam(t *testing.T) {
	e := New(io.Discard).Info()
	e.Str("before", "x")

	result := e.Dict("group", nil)

	assert.Same(t, e, result, "expected same event returned")
	require.Len(t, e.fields, 1, "nil dict should not add fields")
	assert.Equal(t, "before", e.fields[0].Key)
}

func TestEventStringerTypedNil(t *testing.T) {
	e := New(io.Discard).Info()
	var buf *bytes.Buffer // typed nil that implements fmt.Stringer

	result := e.Stringer("key", buf)

	assert.Same(t, e, result, "expected same event returned")
	assert.Empty(t, e.fields, "typed nil stringer should not add a field")
}

func TestEventEmptyFieldKey(t *testing.T) {
	var buf bytes.Buffer

	l := New(&buf)
	l.Info().Str("", "value").Msg("test")

	assert.Contains(t, buf.String(), "=value")
}

func TestEventMsgFatalCallsExit(t *testing.T) {
	var exitCode int

	l := New(io.Discard)
	l.SetExitFunc(func(code int) { exitCode = code })
	l.Fatal().Msg("fatal error")

	assert.Equal(t, 1, exitCode)
}
