package clog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldBuilderInt64(t *testing.T) {
	b := Spinner("test").Int64("count", 42)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "count", b.fields[0].Key)
	assert.Equal(t, int64(42), b.fields[0].Value)
}

func TestFieldBuilderUint(t *testing.T) {
	b := Spinner("test").Uint("size", 100)

	require.Len(t, b.fields, 1)
	assert.Equal(t, "size", b.fields[0].Key)
	assert.Equal(t, uint(100), b.fields[0].Value)
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
