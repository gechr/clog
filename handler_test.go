package clog

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerFuncAdapter(t *testing.T) {
	var got Entry

	h := HandlerFunc(func(e Entry) {
		got = e
	})

	h.Log(Entry{
		Level:   WarnLevel,
		Message: "test message",
		Prefix:  "warning",
		Fields:  []Field{{Key: "k", Value: "v"}},
	})

	assert.Equal(t, WarnLevel, got.Level)
	assert.Equal(t, "test message", got.Message)
	assert.Equal(t, "warning", got.Prefix)
	require.Len(t, got.Fields, 1)
	assert.Equal(t, "k", got.Fields[0].Key)
	assert.Equal(t, "v", got.Fields[0].Value)
}

func TestEntryFieldsPopulated(t *testing.T) {
	l := NewWriter(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))
	l.SetReportTimestamp(true)

	l.Info().Str("key", "val").Msg("hello")

	assert.Equal(t, InfoLevel, got.Level)
	assert.Equal(t, "hello", got.Message)
	assert.False(t, got.Time.IsZero(), "expected non-zero Time when reportTimestamp is true")
	assert.Equal(t, defaultPrefixes[InfoLevel], got.Prefix)
	require.Len(t, got.Fields, 1)
	assert.Equal(t, "key", got.Fields[0].Key)
	assert.Equal(t, "val", got.Fields[0].Value)
}

func TestEntryTimeZeroWhenTimestampDisabled(t *testing.T) {
	l := NewWriter(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Msg("hello")

	assert.True(t, got.Time.IsZero(), "expected zero Time when reportTimestamp is false")
}
