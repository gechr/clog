package clog

import (
	"encoding/json"
	"io"
	"testing"
	"time"

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

func TestEntryJSONMarshal(t *testing.T) {
	t.Run("lowercase_keys_and_string_level", func(t *testing.T) {
		e := Entry{
			Level:   InfoLevel,
			Message: "Server started",
			Fields:  []Field{{Key: "port", Value: "8080"}},
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &m))

		// Keys should be lowercase.
		assert.Contains(t, m, "level")
		assert.Contains(t, m, "message")
		assert.Contains(t, m, "fields")

		// Level should be a string, not an integer.
		assert.Equal(t, `"info"`, string(m["level"]))
		assert.Equal(t, `"Server started"`, string(m["message"]))
	})

	t.Run("omit_zero_time", func(t *testing.T) {
		e := Entry{
			Level:   WarnLevel,
			Message: "test",
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &m))

		assert.NotContains(t, m, "time", "zero time should be omitted")
	})

	t.Run("include_nonzero_time", func(t *testing.T) {
		ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
		e := Entry{
			Level:   InfoLevel,
			Message: "test",
			Time:    ts,
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &m))

		assert.Contains(t, m, "time", "non-zero time should be present")
	})

	t.Run("omit_empty_fields_and_prefix", func(t *testing.T) {
		e := Entry{
			Level:   ErrorLevel,
			Message: "fail",
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &m))

		assert.NotContains(t, m, "fields", "nil fields should be omitted")
		assert.NotContains(t, m, "prefix", "empty prefix should be omitted")
	})

	t.Run("full_roundtrip", func(t *testing.T) {
		e := Entry{
			Level:   InfoLevel,
			Message: "Server started",
			Fields:  []Field{{Key: "port", Value: "8080"}},
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		want := `{"fields":[{"key":"port","value":"8080"}],"level":"info","message":"Server started"}`
		assert.JSONEq(t, want, string(data))
	})
}
