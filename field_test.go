package clog

import (
	"bytes"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/gechr/clog/fx"
	"github.com/gechr/clog/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertSingleField asserts that fields contains exactly one entry with the given key and value.
func assertSingleField[T any](t *testing.T, fields []Field, key string, value T) {
	t.Helper()
	require.Len(t, fields, 1)
	assert.Equal(t, key, fields[0].Key)
	assert.Equal(t, value, fields[0].Value)
}

// assertSliceField asserts that fields contains exactly one entry whose value
// type-asserts to []T and equals expected.
func assertSliceField[T any](t *testing.T, fields []Field, expected []T) {
	t.Helper()
	require.Len(t, fields, 1)
	got, ok := fields[0].Value.([]T)
	require.True(t, ok, "expected %T value, got %T", expected, fields[0].Value)
	assert.Equal(t, expected, got)
}

func TestFieldBuilderInt64(t *testing.T) {
	b := Spinner("test").Int64("count", 42)
	assertSingleField(t, b.Fields, "count", int64(42))
}

func TestFieldBuilderUint(t *testing.T) {
	b := Spinner("test").Uint("size", 100)
	assertSingleField(t, b.Fields, "size", uint(100))
}

func TestFieldBuilderInt64Chaining(t *testing.T) {
	b := Spinner("test").Int64("a", 1).Int64("b", 2).Str("c", "x")

	require.Len(t, b.Fields, 3)
	assert.Equal(t, int64(1), b.Fields[0].Value)
	assert.Equal(t, int64(2), b.Fields[1].Value)
	assert.Equal(t, "x", b.Fields[2].Value)
}

func TestFieldBuilderUintChaining(t *testing.T) {
	b := Spinner("test").Uint("a", 1).Uint("b", 2).Str("c", "x")

	require.Len(t, b.Fields, 3)
	assert.Equal(t, uint(1), b.Fields[0].Value)
	assert.Equal(t, uint(2), b.Fields[1].Value)
	assert.Equal(t, "x", b.Fields[2].Value)
}

func TestFieldBuilderErrs(t *testing.T) {
	errs := []error{errors.New("a"), nil, errors.New("c")}
	b := Spinner("test").Errs("problems", errs)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "problems", b.Fields[0].Key)

	vals, ok := b.Fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "<nil>", "c"}, vals)
}

func TestFieldBuilderPercent(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"normal value", 0.50, 0.50},
		{"zero", 0.0, 0.0},
		{"one", 1.0, 1.0},
		{"negative stored as-is", -0.10, -0.10},
		{"over maximum stored as-is", 1.50, 1.50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Spinner("test").Percent("pct", tt.input)

			require.Len(t, b.Fields, 1)
			assert.Equal(t, "pct", b.Fields[0].Key)

			p, ok := b.Fields[0].Value.(core.Percent)
			require.True(t, ok, "expected percent value")
			assert.InDelta(t, tt.expected, p.Value, 0)
		})
	}
}

func TestFieldBuilderRawJSON(t *testing.T) {
	data := []byte(`{"a":1}`)
	b := Spinner("test").RawJSON("data", data)

	require.Len(t, b.Fields, 1)
	assert.Equal(t, "data", b.Fields[0].Key)

	got, ok := b.Fields[0].Value.(core.RawJSON)
	require.True(t, ok, "expected rawJSON value")
	assert.Equal(t, core.RawJSON(data), got)
}

func TestFieldBuilderJSON(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		val := struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{"alice", 30}

		b := Spinner("test").JSON("person", val)

		require.Len(t, b.Fields, 1)
		assert.Equal(t, "person", b.Fields[0].Key)

		_, ok := b.Fields[0].Value.(core.RawJSON)
		require.True(t, ok, "expected rawJSON value for valid input")
	})

	t.Run("marshal error", func(t *testing.T) {
		b := Spinner("test").JSON("bad", math.Inf(1))

		require.Len(t, b.Fields, 1)
		assert.Equal(t, "bad", b.Fields[0].Key)

		_, isRaw := b.Fields[0].Value.(core.RawJSON)
		assert.False(t, isRaw, "marshal error should not produce rawJSON")

		_, isStr := b.Fields[0].Value.(string)
		assert.True(t, isStr, "expected error string value")
	})
}

func TestFieldBuilderBase64(t *testing.T) {
	b := Spinner("test").Base64("data", []byte("hello"))
	assertSingleField(t, b.Fields, "data", "aGVsbG8=")
}

func TestFieldBuilderBytes(t *testing.T) {
	t.Run("plain bytes", func(t *testing.T) {
		b := Spinner("test").Bytes("data", []byte("hello"))
		assertSingleField(t, b.Fields, "data", "hello")
	})

	t.Run("valid JSON bytes", func(t *testing.T) {
		b := Spinner("test").Bytes("body", []byte(`{"status":"ok"}`))

		require.Len(t, b.Fields, 1)
		assert.Equal(t, "body", b.Fields[0].Key)
		_, ok := b.Fields[0].Value.(core.RawJSON)
		assert.True(t, ok, "valid JSON bytes should be stored as rawJSON")
	})
}

func TestFieldBuilderHex(t *testing.T) {
	b := Spinner("test").Hex("id", []byte{0xde, 0xad, 0xbe, 0xef})
	assertSingleField(t, b.Fields, "id", "deadbeef")
}

func TestFieldBuilderInts64(t *testing.T) {
	b := Spinner("test").Ints64("nums", []int64{1, 2, 3})
	assertSliceField(t, b.Fields, []int64{1, 2, 3})
}

func TestFieldBuilderUints(t *testing.T) {
	b := Spinner("test").Uints("counts", []uint{10, 20, 30})
	assertSliceField(t, b.Fields, []uint{10, 20, 30})
}

func TestFieldBuilderTimes(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC)
	vals := []time.Time{t1, t2}
	b := Spinner("test").Times("timestamps", vals)
	assertSliceField(t, b.Fields, vals)
}

func TestFieldBuilderWhenTrue(t *testing.T) {
	b := Spinner("test").When(true, func(ab *fx.Builder) {
		ab.Str("key", "value")
	})
	assertSingleField(t, b.Fields, "key", "value")
}

func TestFieldBuilderWhenFalse(t *testing.T) {
	b := Spinner("test").When(false, func(ab *fx.Builder) {
		ab.Str("key", "value")
	})
	assert.Empty(t, b.Fields)
}

func TestFieldBuilderWhenNilFn(t *testing.T) {
	assert.NotPanics(t, func() {
		b := Spinner("test").When(true, nil)
		assert.Empty(t, b.Fields)
	})
}

func TestFieldBuilderWhenChaining(t *testing.T) {
	b := Spinner("test").
		Str("before", "a").
		When(true, func(ab *fx.Builder) {
			ab.Str("conditional", "b")
		}).
		Str("after", "c")

	require.Len(t, b.Fields, 3)
	assert.Equal(t, "before", b.Fields[0].Key)
	assert.Equal(t, "conditional", b.Fields[1].Key)
	assert.Equal(t, "after", b.Fields[2].Key)
}

func TestFieldBuilderNarrowIntTypes(t *testing.T) {
	b := Spinner("test").Int8("a", 1).Int16("b", 2).Int32("c", 3)

	require.Len(t, b.Fields, 3)
	assert.Equal(t, int8(1), b.Fields[0].Value)
	assert.Equal(t, int16(2), b.Fields[1].Value)
	assert.Equal(t, int32(3), b.Fields[2].Value)
}

func TestFieldBuilderNarrowIntSliceTypes(t *testing.T) {
	b := Spinner(
		"test",
	).Ints8("a", []int8{1, 2}).
		Ints16("b", []int16{3, 4}).
		Ints32("c", []int32{5, 6})

	require.Len(t, b.Fields, 3)
	assert.Equal(t, []int8{1, 2}, b.Fields[0].Value)
	assert.Equal(t, []int16{3, 4}, b.Fields[1].Value)
	assert.Equal(t, []int32{5, 6}, b.Fields[2].Value)
}

func TestFieldBuilderNarrowUintTypes(t *testing.T) {
	b := Spinner("test").Uint8("a", 1).Uint16("b", 2).Uint32("c", 3)

	require.Len(t, b.Fields, 3)
	assert.Equal(t, uint8(1), b.Fields[0].Value)
	assert.Equal(t, uint16(2), b.Fields[1].Value)
	assert.Equal(t, uint32(3), b.Fields[2].Value)
}

func TestFieldBuilderNarrowUintSliceTypes(t *testing.T) {
	b := Spinner(
		"test",
	).Uints8("a", []uint8{1, 2}).
		Uints16("b", []uint16{3, 4}).
		Uints32("c", []uint32{5, 6})

	require.Len(t, b.Fields, 3)
	assert.Equal(t, []uint8{1, 2}, b.Fields[0].Value)
	assert.Equal(t, []uint16{3, 4}, b.Fields[1].Value)
	assert.Equal(t, []uint32{5, 6}, b.Fields[2].Value)
}

func TestFieldBuilderFloat32(t *testing.T) {
	b := Spinner("test").Float32("val", 3.14)
	assertSingleField(t, b.Fields, "val", float32(3.14))
}

func TestFieldBuilderFloats32(t *testing.T) {
	b := Spinner("test").Floats32("vals", []float32{1.1, 2.2})
	assertSliceField(t, b.Fields, []float32{1.1, 2.2})
}

func TestFieldBuilderTimeDiff(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 0, 0, 5, 0, time.UTC)

	t.Run("positive diff", func(t *testing.T) {
		b := Spinner("test").TimeDiff("elapsed", end, start)
		assertSingleField(t, b.Fields, "elapsed", 5*time.Second)
	})

	t.Run("zero when end before start", func(t *testing.T) {
		b := Spinner("test").TimeDiff("elapsed", start, end)
		assertSingleField(t, b.Fields, "elapsed", time.Duration(0))
	})
}

func TestEventDiscard(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	e := l.Info().Str("a", "b").Discard()
	assert.Nil(t, e)
	e.Msg("should not panic") // nil-safe
}

func TestEventEnabledDisabled(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	e := l.Info()
	assert.True(t, e.Enabled())
	assert.False(t, e.Disabled())

	var nilEvent *Event
	assert.False(t, nilEvent.Enabled())
	assert.True(t, nilEvent.Disabled())
}

func TestEventMsgFunc(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	called := false
	l.Info().MsgFunc(func() string {
		called = true
		return "lazy"
	})
	assert.True(t, called)
}

func TestEventMsgFuncNil(t *testing.T) {
	called := false
	var e *Event
	e.MsgFunc(func() string {
		called = true
		return "lazy"
	})
	assert.False(t, called, "MsgFunc should not call fn on nil event")
}

func TestEventNarrowTypesNilSafe(t *testing.T) {
	var e *Event
	assert.Nil(t, e.Int8("a", 1))
	assert.Nil(t, e.Int16("a", 1))
	assert.Nil(t, e.Int32("a", 1))
	assert.Nil(t, e.Uint8("a", 1))
	assert.Nil(t, e.Uint16("a", 1))
	assert.Nil(t, e.Uint32("a", 1))
	assert.Nil(t, e.Float32("a", 1))
	assert.Nil(t, e.Ints8("a", nil))
	assert.Nil(t, e.Ints16("a", nil))
	assert.Nil(t, e.Ints32("a", nil))
	assert.Nil(t, e.Uints8("a", nil))
	assert.Nil(t, e.Uints16("a", nil))
	assert.Nil(t, e.Uints32("a", nil))
	assert.Nil(t, e.Floats32("a", nil))
	assert.Nil(t, e.TimeDiff("a", time.Now(), time.Now()))
}

func TestEventTimeDiff(t *testing.T) {
	var buf bytes.Buffer
	l := New(TestOutput(&buf))
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 1, 1, 0, 0, 5, 0, time.UTC)

	l.Info().TimeDiff("elapsed", end, start).Msg("done")
	assert.Contains(t, buf.String(), "elapsed=5s")
}
