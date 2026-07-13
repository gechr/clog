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
		Level:   LevelWarn,
		Message: "test message",
		Symbol:  "warning",
		Fields:  []Field{{Key: "k", Value: "v"}},
	})

	assert.Equal(t, LevelWarn, got.Level)
	assert.Equal(t, "test message", got.Message)
	assert.Equal(t, "warning", got.Symbol)
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

	assert.Equal(t, LevelInfo, got.Level)
	assert.Equal(t, "hello", got.Message)
	assert.False(t, got.Time.IsZero(), "expected non-zero Time when reportTimestamp is true")
	assert.Equal(t, defaultSymbols[LevelInfo], got.Symbol)
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

func TestEntryExplicitTimestampRespectsReporting(t *testing.T) {
	ts := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)

	var got Entry

	l := NewWriter(io.Discard)
	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))
	l.SetTimeLocation(time.UTC)

	// Reporting disabled: an explicit timestamp must not force one through.
	l.LogFields(LevelInfo, ts, "hello", nil)
	assert.True(t, got.Time.IsZero(), "expected zero Time when reportTimestamp is false")

	// Reporting enabled: the explicit timestamp overrides time.Now().
	l.SetReportTimestamp(true)
	l.LogFields(LevelInfo, ts, "hello", nil)
	assert.Equal(t, ts, got.Time)

	// Reporting enabled, zero timestamp: the record carries no time, so none
	// is rendered (slog semantics) -- not substituted with time.Now().
	l.LogFields(LevelInfo, time.Time{}, "hello", nil)
	assert.True(t, got.Time.IsZero(), "expected zero Time for a zero adapter timestamp")
}

func TestEntryJSONMarshal(t *testing.T) {
	t.Run("lowercase_keys_and_string_level", func(t *testing.T) {
		e := Entry{
			Level:   LevelInfo,
			Message: "Server started",
			Fields:  []Field{{Key: "port", Value: "8080"}},
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &m))

		// Keys should be lowercase and JSON should match exactly.
		//nolint:testifylint // byte-exact serialization (key casing, order) under test
		assert.Equal(
			t,
			`{"level":"info","message":"Server started","fields":[{"key":"port","value":"8080"}]}`,
			string(data),
		)

		// Level should be a string, not an integer.
		assert.Equal(t, `"info"`, string(m["level"]))
		assert.Equal(t, `"Server started"`, string(m["message"]))
	})

	t.Run("omit_zero_time", func(t *testing.T) {
		e := Entry{
			Level:   LevelWarn,
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
			Level:   LevelInfo,
			Message: "test",
			Time:    ts,
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		// Fixed timestamp makes the whole payload deterministic.
		//nolint:testifylint // byte-exact serialization (time present, field order) under test
		assert.Equal(
			t,
			`{"time":"2025-06-15T10:30:00Z","level":"info","message":"test"}`,
			string(data),
		)
	})

	t.Run("omit_empty_fields_and_symbol", func(t *testing.T) {
		e := Entry{
			Level:   LevelError,
			Message: "fail",
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(data, &m))

		assert.NotContains(t, m, "fields", "nil fields should be omitted")
		assert.NotContains(t, m, "symbol", "empty symbol should be omitted")
	})

	t.Run("full_roundtrip", func(t *testing.T) {
		e := Entry{
			Level:   LevelInfo,
			Message: "Server started",
			Fields:  []Field{{Key: "port", Value: "8080"}},
		}

		data, err := json.Marshal(e)
		require.NoError(t, err)

		want := `{"fields":[{"key":"port","value":"8080"}],"level":"info","message":"Server started"}`
		assert.JSONEq(t, want, string(data))
	})
}
