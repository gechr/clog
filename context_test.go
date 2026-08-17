package clog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/gechr/clog/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextStr(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Str("key", "val")
	assertSingleField(t, ctx.Fields, "key", "val")
}

func TestContextStrs(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Strs("keys", []string{"a", "b"})
	assertSliceField(t, ctx.Fields, []string{"a", "b"})
}

func TestContextInt(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Int("n", 42)
	assertSingleField(t, ctx.Fields, "n", 42)
}

func TestContextInts(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Ints("nums", []int{1, 2, 3})

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "nums", ctx.Fields[0].Key)
}

func TestContextUint64(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Uint64("size", 999)
	assertSingleField(t, ctx.Fields, "size", uint64(999))
}

func TestContextUints64(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Uints64("sizes", []uint64{1, 2, 3})

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "sizes", ctx.Fields[0].Key)
}

func TestContextFloat64(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Float64("pi", 3.14)

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "pi", ctx.Fields[0].Key)
	assert.InDelta(t, 3.14, ctx.Fields[0].Value, 0)
}

func TestContextFloats64(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Floats64("vals", []float64{1.1, 2.2})
	assertSingleField(t, ctx.Fields, "vals", []float64{1.1, 2.2})
}

func TestContextLink(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Link("docs", "https://example.com", "docs")

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "docs", ctx.Fields[0].Key)
	// Colors disabled in tests (no TTY), so the fallback renders the URL.
	assert.Equal(t, "https://example.com", ctx.Fields[0].Value)
}

func TestContextURL(t *testing.T) {
	ctx := NewWriter(io.Discard).With().URL("link", "https://example.com")

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "link", ctx.Fields[0].Key)
	assert.Equal(t, "https://example.com", ctx.Fields[0].Value)
}

func TestContextLinks(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Links("repos", []Link{
		{URL: "https://github.com/foo/bar", Text: "foo/bar"},
		{URL: "https://github.com/baz/qux", Text: "baz/qux"},
	})

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "repos", ctx.Fields[0].Key)
	assert.Equal(
		t,
		[]string{"https://github.com/foo/bar", "https://github.com/baz/qux"},
		ctx.Fields[0].Value,
	)
}

func TestContextURLs(t *testing.T) {
	ctx := NewWriter(io.Discard).With().URLs("refs", []string{"https://a.com", "https://b.com"})

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "refs", ctx.Fields[0].Key)
	assert.Equal(t, []string{"https://a.com", "https://b.com"}, ctx.Fields[0].Value)
}

func TestContextBool(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Bool("ok", true)
	assertSingleField(t, ctx.Fields, "ok", true)
}

func TestContextBools(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Bools("flags", []bool{true, false})
	assertSingleField(t, ctx.Fields, "flags", []bool{true, false})
}

func TestContextDur(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Duration("elapsed", time.Second)
	assertSingleField(t, ctx.Fields, "elapsed", core.DurationField{Value: time.Second})
}

func TestContextTime(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := NewWriter(io.Discard).With().Time("created", ts)
	assertSingleField(t, ctx.Fields, "created", ts)
}

func TestContextAny(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Any("data", 123)
	assertSingleField(t, ctx.Fields, "data", 123)
}

func TestContextAnys(t *testing.T) {
	vals := []any{"hello", 42, true}
	ctx := NewWriter(io.Discard).With().Anys("mixed", vals)
	assertSliceField(t, ctx.Fields, vals)
}

func TestContextDict(t *testing.T) {
	ctx := NewWriter(
		io.Discard,
	).With().
		Dict("db", Dict().Str("host", "localhost").Int("port", 5432))

	require.Len(t, ctx.Fields, 2)
	assert.Equal(t, "db.host", ctx.Fields[0].Key)
	assert.Equal(t, "localhost", ctx.Fields[0].Value)
	assert.Equal(t, "db.port", ctx.Fields[1].Key)
	assert.Equal(t, 5432, ctx.Fields[1].Value)
}

func TestContextErr(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Err(errors.New("boom"))

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "error", ctx.Fields[0].Key)
}

func TestContextErrNil(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Err(nil)

	assert.Empty(t, ctx.Fields)
}

func TestContextPath(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Path("dir", "/tmp")

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "dir", ctx.Fields[0].Key)
	assert.Equal(t, "/tmp", ctx.Fields[0].Value)
}

func TestContextLine(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Line("file", "main.go", 10)

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "file", ctx.Fields[0].Key)
	assert.Equal(t, "main.go:10", ctx.Fields[0].Value)
}

func TestContextColumn(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Column("loc", "main.go", 10, 5)

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "loc", ctx.Fields[0].Key)
	assert.Equal(t, "main.go:10:5", ctx.Fields[0].Value)
}

func TestContextColumnMinimum(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Column("loc", "main.go", 0, 0)

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "loc", ctx.Fields[0].Key)
	// Both line and column should be clamped to 1.
	assert.Equal(t, "main.go:1:1", ctx.Fields[0].Value)
}

func TestContextLineOmittedWhenZero(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Line("file", "main.go", 0)

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "file", ctx.Fields[0].Key)
	// Line number 0 omits the line suffix - equivalent to Path().
	assert.Equal(t, "main.go", ctx.Fields[0].Value)
}

func TestContextStringer(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Stringer("name", testStringer{s: "hello"})

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "name", ctx.Fields[0].Key)
	assert.Equal(t, "hello", ctx.Fields[0].Value)
}

func TestContextStringerNil(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Stringer("key", nil)

	assert.Empty(t, ctx.Fields)
}

func TestContextStringers(t *testing.T) {
	ctx := NewWriter(
		io.Discard,
	).With().
		Stringers("items", []fmt.Stringer{testStringer{s: "a"}, testStringer{s: "b"}})

	require.Len(t, ctx.Fields, 1)

	vals, ok := ctx.Fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "b"}, vals)
}

func TestContextStringersWithNil(t *testing.T) {
	ctx := NewWriter(
		io.Discard,
	).With().
		Stringers("items", []fmt.Stringer{testStringer{s: "a"}, nil})

	require.Len(t, ctx.Fields, 1)

	vals, ok := ctx.Fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "<nil>"}, vals)
}

func TestContextDurations(t *testing.T) {
	vals := []time.Duration{time.Second, 2 * time.Millisecond}
	ctx := NewWriter(io.Discard).With().Durations("timings", vals)
	assertSliceField(t, ctx.Fields, vals)
}

func TestContextQuantity(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Quantity("size", "10GB")

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "size", ctx.Fields[0].Key)
}

func TestContextQuantities(t *testing.T) {
	ctx := NewWriter(io.Discard).With().Quantities("sizes", []string{"10GB", "5MB"})

	require.Len(t, ctx.Fields, 1)
	assert.Equal(t, "sizes", ctx.Fields[0].Key)
}

func TestContextSymbol(t *testing.T) {
	l := NewWriter(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	sub := l.With().Symbol("CTX").Logger()
	sub.Info().Msg("test")

	assert.Equal(t, "CTX", got.Symbol)
}

func TestContextLoggerInheritsAtomicLevel(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLevel(LevelWarn)

	sub := l.With().Str("component", "db").Logger()

	// Sub-logger must filter events below the parent's level.
	assert.Nil(t, sub.Trace(), "Trace should be nil at LevelWarn")
	assert.Nil(t, sub.Debug(), "Debug should be nil at LevelWarn")
	assert.Nil(t, sub.Info(), "Info should be nil at LevelWarn")
	assert.NotNil(t, sub.Warn(), "Warn should not be nil at LevelWarn")
	assert.NotNil(t, sub.Error(), "Error should not be nil at LevelWarn")

	// Verify atomicLevel matches the level field.
	assert.Equal(t, int32(LevelWarn), sub.atomicLevel.Load(),
		"sub-logger atomicLevel should match parent's level")
}

func TestContextLoggerInheritsSettings(t *testing.T) {
	l := NewWriter(io.Discard)
	l.SetLevel(LevelDebug)
	l.SetReportTimestamp(true)
	l.SetTimeFormat("2006-01-02")

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	sub := l.With().Str("component", "db").Logger()

	assert.Equal(t, LevelDebug, sub.Level())
	assert.True(t, sub.reportTimestamp, "expected reportTimestamp inherited")
	assert.Equal(t, "2006-01-02", sub.timeFormat)
	assert.NotNil(t, sub.handler, "expected handler inherited")

	// Verify context fields appear in log output.
	sub.Info().Str("user", "john").Msg("login")

	require.Len(t, got.Fields, 2)
	assert.Equal(t, "component", got.Fields[0].Key)
	assert.Equal(t, "db", got.Fields[0].Value)
	assert.Equal(t, "user", got.Fields[1].Key)
	assert.Equal(t, "john", got.Fields[1].Value)
}

func TestContextLoggerSharesMutex(t *testing.T) {
	l := NewWriter(io.Discard)
	sub := l.With().Str("k", "v").Logger()

	assert.Same(t, l.mu, sub.mu, "sub-logger should share parent's mutex")
}

// Nilable types that implement fmt.Stringer for typed-nil tests.
type stringerMap map[string]string

func (m stringerMap) String() string { return "map" }

type stringerSlice []string

func (s stringerSlice) String() string { return "slice" }

type stringerChan chan struct{}

func (c stringerChan) String() string { return "chan" }

type stringerFunc func()

func (f stringerFunc) String() string { return "func" }

func TestContextStringerTypedNilPointer(t *testing.T) {
	ctx := NewWriter(io.Discard).With()
	var buf *bytes.Buffer // typed nil that implements fmt.Stringer

	result := ctx.Stringer("key", buf)

	assert.Same(t, ctx, result, "expected same context returned")
	assert.Empty(t, ctx.Fields, "typed nil pointer should not add a field")
}

func TestContextStringerTypedNilMap(t *testing.T) {
	ctx := NewWriter(io.Discard).With()
	var m stringerMap // typed nil map

	result := ctx.Stringer("key", m)

	assert.Same(t, ctx, result, "expected same context returned")
	assert.Empty(t, ctx.Fields, "typed nil map should not add a field")
}

func TestContextStringerTypedNilSlice(t *testing.T) {
	ctx := NewWriter(io.Discard).With()
	var s stringerSlice // typed nil slice

	result := ctx.Stringer("key", s)

	assert.Same(t, ctx, result, "expected same context returned")
	assert.Empty(t, ctx.Fields, "typed nil slice should not add a field")
}

func TestContextStringerTypedNilChan(t *testing.T) {
	ctx := NewWriter(io.Discard).With()
	var ch stringerChan // typed nil chan

	result := ctx.Stringer("key", ch)

	assert.Same(t, ctx, result, "expected same context returned")
	assert.Empty(t, ctx.Fields, "typed nil chan should not add a field")
}

func TestContextStringerTypedNilFunc(t *testing.T) {
	ctx := NewWriter(io.Discard).With()
	var fn stringerFunc // typed nil func

	result := ctx.Stringer("key", fn)

	assert.Same(t, ctx, result, "expected same context returned")
	assert.Empty(t, ctx.Fields, "typed nil func should not add a field")
}

func TestContextStringersTypedNils(t *testing.T) {
	var m stringerMap
	var s stringerSlice
	var ch stringerChan
	var fn stringerFunc

	ctx := NewWriter(io.Discard).With().
		Stringers("items", []fmt.Stringer{testStringer{s: "a"}, m, s, ch, fn, nil})

	require.Len(t, ctx.Fields, 1)

	vals, ok := ctx.Fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "<nil>", "<nil>", "<nil>", "<nil>", "<nil>"}, vals)
}

func TestContextDictNil(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	sub := l.With().Dict("key", nil).Logger()

	// Should not panic and should produce output without extra fields.
	sub.Info().Msg("test")

	got := buf.String()
	assert.Equal(t, "INF ℹ️ test\n", got)
}
