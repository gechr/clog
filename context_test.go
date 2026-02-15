package clog

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextStr(t *testing.T) {
	ctx := New(io.Discard).With().Str("key", "val")

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "key", ctx.fields[0].Key)
	assert.Equal(t, "val", ctx.fields[0].Value)
}

func TestContextStrs(t *testing.T) {
	ctx := New(io.Discard).With().Strs("keys", []string{"a", "b"})

	require.Len(t, ctx.fields, 1)

	vals, ok := ctx.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "b"}, vals)
}

func TestContextInt(t *testing.T) {
	ctx := New(io.Discard).With().Int("n", 42)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "n", ctx.fields[0].Key)
	assert.Equal(t, 42, ctx.fields[0].Value)
}

func TestContextInts(t *testing.T) {
	ctx := New(io.Discard).With().Ints("nums", []int{1, 2, 3})

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "nums", ctx.fields[0].Key)
}

func TestContextUint64(t *testing.T) {
	ctx := New(io.Discard).With().Uint64("size", 999)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "size", ctx.fields[0].Key)
	assert.Equal(t, uint64(999), ctx.fields[0].Value)
}

func TestContextUints64(t *testing.T) {
	ctx := New(io.Discard).With().Uints64("sizes", []uint64{1, 2, 3})

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "sizes", ctx.fields[0].Key)
}

func TestContextFloat64(t *testing.T) {
	ctx := New(io.Discard).With().Float64("pi", 3.14)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "pi", ctx.fields[0].Key)
	assert.InDelta(t, 3.14, ctx.fields[0].Value, 0)
}

func TestContextFloats64(t *testing.T) {
	ctx := New(io.Discard).With().Floats64("vals", []float64{1.1, 2.2})

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "vals", ctx.fields[0].Key)
	assert.Equal(t, []float64{1.1, 2.2}, ctx.fields[0].Value)
}

func TestContextLink(t *testing.T) {
	ctx := New(io.Discard).With().Link("docs", "https://example.com", "docs")

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "docs", ctx.fields[0].Key)
	assert.Equal(t, "docs", ctx.fields[0].Value)
}

func TestContextBool(t *testing.T) {
	ctx := New(io.Discard).With().Bool("ok", true)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "ok", ctx.fields[0].Key)
	assert.Equal(t, true, ctx.fields[0].Value)
}

func TestContextBools(t *testing.T) {
	ctx := New(io.Discard).With().Bools("flags", []bool{true, false})

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "flags", ctx.fields[0].Key)
	assert.Equal(t, []bool{true, false}, ctx.fields[0].Value)
}

func TestContextDur(t *testing.T) {
	ctx := New(io.Discard).With().Dur("elapsed", time.Second)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "elapsed", ctx.fields[0].Key)
	assert.Equal(t, time.Second, ctx.fields[0].Value)
}

func TestContextTime(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	ctx := New(io.Discard).With().Time("created", ts)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "created", ctx.fields[0].Key)
	assert.Equal(t, ts, ctx.fields[0].Value)
}

func TestContextAny(t *testing.T) {
	ctx := New(io.Discard).With().Any("data", 123)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "data", ctx.fields[0].Key)
	assert.Equal(t, 123, ctx.fields[0].Value)
}

func TestContextAnys(t *testing.T) {
	vals := []any{"hello", 42, true}
	ctx := New(io.Discard).With().Anys("mixed", vals)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "mixed", ctx.fields[0].Key)

	got, ok := ctx.fields[0].Value.([]any)
	require.True(t, ok, "expected []any value")
	assert.Equal(t, vals, got)
}

func TestContextDict(t *testing.T) {
	ctx := New(io.Discard).With().Dict("db", Dict().Str("host", "localhost").Int("port", 5432))

	require.Len(t, ctx.fields, 2)
	assert.Equal(t, "db.host", ctx.fields[0].Key)
	assert.Equal(t, "localhost", ctx.fields[0].Value)
	assert.Equal(t, "db.port", ctx.fields[1].Key)
	assert.Equal(t, 5432, ctx.fields[1].Value)
}

func TestContextErr(t *testing.T) {
	ctx := New(io.Discard).With().Err(errors.New("boom"))

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "error", ctx.fields[0].Key)
}

func TestContextErrNil(t *testing.T) {
	ctx := New(io.Discard).With().Err(nil)

	assert.Empty(t, ctx.fields)
}

func TestContextPath(t *testing.T) {
	ctx := New(io.Discard).With().Path("dir", "/tmp")

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "dir", ctx.fields[0].Key)
	assert.Equal(t, "/tmp", ctx.fields[0].Value)
}

func TestContextLine(t *testing.T) {
	ctx := New(io.Discard).With().Line("file", "main.go", 10)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "file", ctx.fields[0].Key)
	assert.Equal(t, "main.go:10", ctx.fields[0].Value)
}

func TestContextColumn(t *testing.T) {
	ctx := New(io.Discard).With().Column("loc", "main.go", 10, 5)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "loc", ctx.fields[0].Key)
	assert.Equal(t, "main.go:10:5", ctx.fields[0].Value)
}

func TestContextColumnMinimum(t *testing.T) {
	ctx := New(io.Discard).With().Column("loc", "main.go", 0, 0)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "loc", ctx.fields[0].Key)
	// Both line and column should be clamped to 1.
	assert.Equal(t, "main.go:1:1", ctx.fields[0].Value)
}

func TestContextLineMinimum(t *testing.T) {
	ctx := New(io.Discard).With().Line("file", "main.go", 0)

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "file", ctx.fields[0].Key)
	// line < 1 is clamped to 1.
	assert.Equal(t, "main.go:1", ctx.fields[0].Value)
}

func TestContextStringer(t *testing.T) {
	ctx := New(io.Discard).With().Stringer("name", testStringer{s: "hello"})

	require.Len(t, ctx.fields, 1)
	assert.Equal(t, "name", ctx.fields[0].Key)
	assert.Equal(t, "hello", ctx.fields[0].Value)
}

func TestContextStringerNil(t *testing.T) {
	ctx := New(io.Discard).With().Stringer("key", nil)

	assert.Empty(t, ctx.fields)
}

func TestContextStringers(t *testing.T) {
	ctx := New(
		io.Discard,
	).With().
		Stringers("items", []fmt.Stringer{testStringer{s: "a"}, testStringer{s: "b"}})

	require.Len(t, ctx.fields, 1)

	vals, ok := ctx.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "b"}, vals)
}

func TestContextStringersWithNil(t *testing.T) {
	ctx := New(io.Discard).With().Stringers("items", []fmt.Stringer{testStringer{s: "a"}, nil})

	require.Len(t, ctx.fields, 1)

	vals, ok := ctx.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "<nil>"}, vals)
}

func TestContextPrefix(t *testing.T) {
	l := New(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	sub := l.With().Prefix("CTX").Logger()
	sub.Info().Msg("test")

	assert.Equal(t, "CTX", got.Prefix)
}

func TestContextLoggerInheritsSettings(t *testing.T) {
	l := New(io.Discard)
	l.SetLevel(DebugLevel)
	l.SetReportTimestamp(true)
	l.SetTimeFormat("2006-01-02")

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	sub := l.With().Str("component", "db").Logger()

	assert.Equal(t, DebugLevel, sub.level)
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
	l := New(io.Discard)
	sub := l.With().Str("k", "v").Logger()

	assert.Same(t, l.mu, sub.mu, "sub-logger should share parent's mutex")
}
