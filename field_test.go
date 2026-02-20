package clog

import (
	"math"
	"testing"

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
	assertSingleField(t, b.fields, "count", int64(42))
}

func TestFieldBuilderUint(t *testing.T) {
	b := Spinner("test").Uint("size", 100)
	assertSingleField(t, b.fields, "size", uint(100))
}

func TestFieldBuilderInt64Chaining(t *testing.T) {
	b := Spinner("test").Int64("a", 1).Int64("b", 2).Str("c", "x")

	require.Len(t, b.fields, 3)
	assert.Equal(t, int64(1), b.fields[0].Value)
	assert.Equal(t, int64(2), b.fields[1].Value)
	assert.Equal(t, "x", b.fields[2].Value)
}

func TestFieldBuilderUintChaining(t *testing.T) {
	b := Spinner("test").Uint("a", 1).Uint("b", 2).Str("c", "x")

	require.Len(t, b.fields, 3)
	assert.Equal(t, uint(1), b.fields[0].Value)
	assert.Equal(t, uint(2), b.fields[1].Value)
	assert.Equal(t, "x", b.fields[2].Value)
}

func TestFieldBuilderPercent(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"normal value", 50.0, 50.0},
		{"zero", 0.0, 0.0},
		{"hundred", 100.0, 100.0},
		{"negative clamped to zero", -10.0, 0.0},
		{"over 100 clamped", 150.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Spinner("test").Percent("pct", tt.input)

			require.Len(t, b.fields, 1)
			assert.Equal(t, "pct", b.fields[0].Key)

			p, ok := b.fields[0].Value.(percent)
			require.True(t, ok, "expected percent value")
			assert.InDelta(t, tt.expected, float64(p), 0)
		})
	}
}

func TestFieldBuilderRawJSON(t *testing.T) {
	data := []byte(`{"a":1}`)
	b := Spinner("test").RawJSON("data", data)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "data", b.fields[0].Key)

	got, ok := b.fields[0].Value.(rawJSON)
	require.True(t, ok, "expected rawJSON value")
	assert.Equal(t, rawJSON(data), got)
}

func TestFieldBuilderJSON(t *testing.T) {
	t.Run("valid struct", func(t *testing.T) {
		val := struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}{"alice", 30}

		b := Spinner("test").JSON("person", val)

		require.Len(t, b.fields, 1)
		assert.Equal(t, "person", b.fields[0].Key)

		_, ok := b.fields[0].Value.(rawJSON)
		require.True(t, ok, "expected rawJSON value for valid input")
	})

	t.Run("marshal error", func(t *testing.T) {
		b := Spinner("test").JSON("bad", math.Inf(1))

		require.Len(t, b.fields, 1)
		assert.Equal(t, "bad", b.fields[0].Key)

		_, isRaw := b.fields[0].Value.(rawJSON)
		assert.False(t, isRaw, "marshal error should not produce rawJSON")

		_, isStr := b.fields[0].Value.(string)
		assert.True(t, isStr, "expected error string value")
	})
}
