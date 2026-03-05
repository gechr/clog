package clog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/field/json"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStringer struct {
	s string
}

func (ts testStringer) String() string { return ts.s }

func TestEventStr(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Str("key", "val")
	assertSingleField(t, e.fields, "key", "val")
}

func TestEventStrs(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Strs("keys", []string{"a", "b"})
	assertSliceField(t, e.fields, []string{"a", "b"})
}

func TestEventBase64(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Base64("data", []byte("hello"))
	assertSingleField(t, e.fields, "data", "aGVsbG8=")
}

func TestEventBytes(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Bytes("data", []byte("hello"))
	assertSingleField(t, e.fields, "data", "hello")
}

func TestEventBytesJSON(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Bytes("body", []byte(`{"status":"ok"}`))

	require.Len(t, e.fields, 1)
	assert.Equal(t, "body", e.fields[0].Key)
	_, ok := e.fields[0].Value.(core.RawJSON)
	assert.True(t, ok, "valid JSON bytes should be stored as rawJSON")
}

func TestEventHex(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Hex("id", []byte{0xde, 0xad, 0xbe, 0xef})
	assertSingleField(t, e.fields, "id", "deadbeef")
}

func TestEventInt(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Int("count", 42)
	assertSingleField(t, e.fields, "count", 42)
}

func TestEventInts(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Ints("nums", []int{1, 2, 3})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "nums", e.fields[0].Key)
}

func TestEventInt64(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Int64("big", 9223372036854775807)
	assertSingleField(t, e.fields, "big", int64(9223372036854775807))
}

func TestEventUint(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Uint("count", 42)
	assertSingleField(t, e.fields, "count", uint(42))
}

func TestEventUint64(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Uint64("size", 999)
	assertSingleField(t, e.fields, "size", uint64(999))
}

func TestEventUints64(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Uints64("sizes", []uint64{1, 2, 3})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "sizes", e.fields[0].Key)
}

func TestEventFloat64(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Float64("pi", 3.14)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "pi", e.fields[0].Key)
	assert.InDelta(t, 3.14, e.fields[0].Value, 0)
}

func TestEventFloats64(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Floats64("vals", []float64{1.1, 2.2})
	assertSingleField(t, e.fields, "vals", []float64{1.1, 2.2})
}

func TestEventLink(t *testing.T) {
	l := NewWriter(io.Discard)
	e := l.Info()
	e.Link("docs", "https://example.com", "docs")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "docs", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so returns plain text.
	assert.Equal(t, "docs", e.fields[0].Value)
}

func TestEventLinkColorAlways(t *testing.T) {
	l := New(NewOutput(io.Discard, ColorAlways))

	e := l.Info()
	e.Link("docs", "https://example.com", "docs")

	require.Len(t, e.fields, 1)

	val, ok := e.fields[0].Value.(string)
	require.True(t, ok)
	assert.Contains(t, val, "\x1b]8;;https://example.com")
	assert.Contains(t, val, "docs")
}

func TestEventURL(t *testing.T) {
	l := NewWriter(io.Discard)
	e := l.Info()
	e.URL("link", "https://example.com")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "link", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so returns plain text.
	assert.Equal(t, "https://example.com", e.fields[0].Value)
}

func TestEventURLColorAlways(t *testing.T) {
	l := New(NewOutput(io.Discard, ColorAlways))

	e := l.Info()
	e.URL("link", "https://example.com")

	require.Len(t, e.fields, 1)

	val, ok := e.fields[0].Value.(string)
	require.True(t, ok)
	assert.Equal(t, "\x1b]8;;https://example.com\x1b\\https://example.com\x1b]8;;\x1b\\", val)
}

func TestEventBool(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Bool("ok", true)
	assertSingleField(t, e.fields, "ok", true)
}

func TestEventBools(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Bools("flags", []bool{true, false})
	assertSingleField(t, e.fields, "flags", []bool{true, false})
}

func TestEventDur(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Duration("elapsed", time.Second)
	assertSingleField(t, e.fields, "elapsed", time.Second)
}

func TestEventTime(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	e := NewWriter(io.Discard).Info()
	e.Time("created", ts)
	assertSingleField(t, e.fields, "created", ts)
}

func TestEventAny(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Any("data", 123)
	assertSingleField(t, e.fields, "data", 123)
}

func TestEventAnys(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	vals := []any{"hello", 42, true}
	e.Anys("mixed", vals)
	assertSliceField(t, e.fields, vals)
}

func TestEventErrs(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	errs := []error{errors.New("a"), nil, errors.New("c")}
	e.Errs("problems", errs)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "problems", e.fields[0].Key)

	vals, ok := e.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"a", "<nil>", "c"}, vals)
}

func TestEventErrsNilReceiver(t *testing.T) {
	var e *Event
	got := e.Errs("k", []error{errors.New("x")})
	assert.Nil(t, got)
}

func TestEventFunc(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Func(func(e *Event) {
		e.Str("lazy", "computed")
	})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "lazy", e.fields[0].Key)
	assert.Equal(t, "computed", e.fields[0].Value)
}

func TestEventFuncNilReceiver(t *testing.T) {
	var e *Event
	called := false
	got := e.Func(func(_ *Event) {
		called = true
	})
	assert.Nil(t, got)
	assert.False(t, called, "callback should not be called on nil event")
}

func TestEventWhenTrue(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.When(true, func(e *Event) {
		e.Str("key", "value")
	})

	assertSingleField(t, e.fields, "key", "value")
}

func TestEventWhenFalse(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	called := false
	e.When(false, func(_ *Event) {
		called = true
	})

	assert.False(t, called, "callback should not be called when condition is false")
	assert.Empty(t, e.fields)
}

func TestEventWhenNilReceiver(t *testing.T) {
	var e *Event
	called := false
	got := e.When(true, func(_ *Event) {
		called = true
	})
	assert.Nil(t, got)
	assert.False(t, called, "callback should not be called on nil event")
}

func TestEventWhenNilFn(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	assert.NotPanics(t, func() {
		e.When(true, nil)
	})
	assert.Empty(t, e.fields)
}

func TestEventDict(t *testing.T) {
	e := NewWriter(io.Discard).Info()
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

	l := New(TestOutput(&buf))
	l.Info().Dict("req", Dict().Str("method", "GET").Int("status", 200)).Msg("handled")

	assert.Equal(t, "INF ℹ️ handled req.method=GET req.status=200\n", buf.String())
}

func TestEventErr(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	err := errors.New("boom")
	e.Err(err)

	assert.Equal(t, err, e.err)
	assert.Empty(t, e.fields)
}

func TestEventErrNil(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	result := e.Err(nil)

	assert.Same(t, e, result, "expected same event returned")
	require.NoError(t, e.err)
	assert.Empty(t, e.fields)
}

func TestEventErrSendUsesErrorAsMessage(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.Error().Err(errors.New("connection refused")).Send()

	got := buf.String()
	assert.Contains(t, got, "connection refused")
	assert.NotContains(t, got, "error=")
}

func TestEventErrMsgAddsErrorField(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.Error().Err(errors.New("connection refused")).Msg("an error occurred")

	got := buf.String()
	assert.Contains(t, got, "an error occurred")
	assert.Contains(t, got, `error="connection refused"`)
}

func TestEventErrMsgfAddsErrorField(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.Error().Err(errors.New("connection refused")).Msgf("failed after %d retries", 3)

	got := buf.String()
	assert.Contains(t, got, "failed after 3 retries")
	assert.Contains(t, got, `error="connection refused"`)
}

func TestEventErrSendPreservesFields(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.Error().Err(errors.New("connection refused")).Str("host", "db1").Send()

	got := buf.String()
	assert.Contains(t, got, "connection refused")
	assert.Contains(t, got, `host=db1`)
	assert.NotContains(t, got, "error=")
}

func TestEventJSON(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.JSON("data", map[string]any{"status": "ok", "n": 42})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "data", e.fields[0].Key)
	_, ok := e.fields[0].Value.(core.RawJSON)
	require.True(t, ok, "expected rawJSON value")
}

func TestEventJSONAppearsUnquotedInOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.Info().JSON("resp", map[string]any{"detail": "ok"}).Msg("done")

	got := buf.String()
	assert.Contains(t, got, `resp={"detail":"ok"}`)
	assert.NotContains(t, got, `resp="{`)
}

func TestEventJSONMarshalError(t *testing.T) {
	// Channels are not JSON-serializable - error stored as field value.
	e := NewWriter(io.Discard).Info()
	e.JSON("bad", make(chan int))

	require.Len(t, e.fields, 1)
	_, isRaw := e.fields[0].Value.(core.RawJSON)
	assert.False(t, isRaw, "marshal error should not produce rawJSON")
	_, isStr := e.fields[0].Value.(string)
	assert.True(t, isStr, "expected error string value")
}

func TestEventRawJSON(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	data := []byte(`{"status":"unprocessable_entity","detail":"something went wrong"}`)
	e.RawJSON("error", data)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "error", e.fields[0].Key)

	got, ok := e.fields[0].Value.(core.RawJSON)
	require.True(t, ok, "expected rawJSON value")
	assert.Equal(t, core.RawJSON(data), got)
}

func TestEventRawJSONAppearsUnquotedInOutput(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	l.Info().RawJSON("error", []byte(`{"detail":"something went wrong"}`)).Msg("request failed")

	got := buf.String()
	assert.Contains(t, got, `error={"detail":"something went wrong"}`)
	assert.NotContains(t, got, `error="{`)
}

func TestEventRawJSONHighlighted(t *testing.T) {
	// Verify highlightJSON produces styled output when a style is provided.
	// We test the function directly since lipgloss doesn't emit ANSI to a
	// non-TTY bytes.Buffer.
	styles := &style.JSON{
		Number: new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff79c6"))),
	}
	result := json.Highlight(`{"n":1}`, styles)
	assert.Contains(t, result, styles.Number.Render("1"))
	assert.Contains(t, result, `"n"`) // key unstyled (no Key style set)
}

func TestEventRawJSONNoHighlightWhenNil(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewOutput(&buf, ColorAlways))
	styles := DefaultStyles()
	styles.FieldJSON = nil
	l.SetStyles(styles)
	l.Info().RawJSON("data", []byte(`{"n":1}`)).Msg("ok")

	got := buf.String()
	// The raw JSON value itself should appear verbatim (no per-token styling).
	assert.Contains(t, got, `{"n":1}`)
}

func TestEventRawJSONUnquoted(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	styles := DefaultStyles()
	styles.FieldJSON = style.DefaultJSON()
	l.SetStyles(styles)
	l.Info().RawJSON("data", []byte(`{"key":"val","n":1,"ok":true,"x":null}`)).Msg("ok")

	got := buf.String()
	// JSON content is present and unquoted
	assert.Contains(t, got, `"key"`)
	assert.Contains(t, got, `"val"`)
	assert.Contains(t, got, "null")
	assert.NotContains(t, got, `data="{`, "JSON should not be quoted")
}

func TestHighlightJSONNullDistinctFromBool(t *testing.T) {
	// null, true, and false each use distinct styles.
	trueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
	falseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6600"))
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))

	styles := &style.JSON{BoolTrue: &trueStyle, BoolFalse: &falseStyle, Null: &nullStyle}
	result := json.Highlight(`{"a":true,"b":false,"c":null}`, styles)

	assert.Contains(t, result, trueStyle.Render("true"))
	assert.Contains(t, result, falseStyle.Render("false"))
	assert.Contains(t, result, nullStyle.Render("null"))
}

func TestHighlightJSONNilFieldsUnstyled(t *testing.T) {
	// Tokens without a style render as plain text.
	styles := &style.JSON{
		Key: style.DefaultJSON().Key, // only keys get a style
	}
	result := json.Highlight(`{"k":42}`, styles)

	assert.Contains(t, result, `"k"`)
	assert.Contains(t, result, "42")
	assert.Contains(t, result, styles.Key.Render(`"k"`))
}

func TestHighlightJSONFlattensWhitespace(t *testing.T) {
	pretty := `{
  "a": 1,
  "b": "x"
}`
	compact := `{"a":1,"b":"x"}`
	result := json.Highlight(pretty, nil)
	// nil styles: returned unchanged
	assert.Equal(t, pretty, result)

	// with styles: whitespace stripped
	result = json.Highlight(pretty, &style.JSON{})
	assert.Equal(t, compact, result)
}

func TestHighlightJSONInvalidFallback(t *testing.T) {
	// invalid JSON: scanner emits styled output then falls back unstyled
	result := json.Highlight(`{"k":INVALID}`, &style.JSON{})
	assert.Contains(t, result, `"k"`)
	assert.Contains(t, result, "INVALID}")
}

func TestHighlightJSONTrailingBackslash(t *testing.T) {
	// Trailing backslash in a string must not panic.
	styles := &style.JSON{}
	assert.NotPanics(t, func() {
		json.Highlight(`"hello\`, styles)
	})
	assert.NotPanics(t, func() {
		json.Highlight(`{"key\":1}`, styles)
	})
	assert.NotPanics(t, func() {
		json.Highlight(`{"k":"val\`, styles)
	})

	// Flat mode exercises collectFlatPairs and scanJSONValueEnd.
	flat := &style.JSON{Mode: style.JSONModeFlat}
	assert.NotPanics(t, func() {
		json.Highlight(`{"k":"val\`, flat)
	})
	assert.NotPanics(t, func() {
		json.Highlight(`{"k":["\`, flat)
	})
}

func TestHighlightJSONHJSONUnquotesKeys(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeHuman}
	result := json.Highlight(`{"status":"ok","code":200}`, styles)

	assert.Contains(t, result, "status")
	assert.NotContains(t, result, `"status"`)
	assert.Contains(t, result, "code")
	assert.NotContains(t, result, `"code"`)
}

func TestHighlightJSONHJSONUnquotesSimpleValues(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeHuman}
	result := json.Highlight(`{"status":"ok","msg":"hello world"}`, styles)

	// Simple value without special chars: unquoted.
	assert.Contains(t, result, ":ok")
	assert.NotContains(t, result, `:"ok"`)
	// Value with a space: still unquoted (spaces are safe in HJSON values).
	assert.Contains(t, result, "hello world")
}

func TestHighlightJSONHumanKeepsQuotedSpecialValues(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeHuman}

	// Value starting with { stays quoted (ambiguous).
	result := json.Highlight(`{"a":"{not an object}"}`, styles)
	assert.Contains(t, result, `"{not an object}"`)

	// Value starting with [ stays quoted.
	result = json.Highlight(`{"a":"[1,2]"}`, styles)
	assert.Contains(t, result, `"[1,2]"`)

	// Value with escape sequence stays quoted.
	result = json.Highlight(`{"s":"line1\nline2"}`, styles)
	assert.Contains(t, result, `"line1\nline2"`)

	// Keyword values stay quoted (would be ambiguous as bare tokens).
	result = json.Highlight(`{"x":"true"}`, styles)
	assert.Contains(t, result, `"true"`)
	result = json.Highlight(`{"x":"null"}`, styles)
	assert.Contains(t, result, `"null"`)

	// Number-like values stay quoted.
	result = json.Highlight(`{"x":"42"}`, styles)
	assert.Contains(t, result, `"42"`)
	result = json.Highlight(`{"x":"-1.5"}`, styles)
	assert.Contains(t, result, `"-1.5"`)

	// Empty string stays quoted.
	result = json.Highlight(`{"x":""}`, styles)
	assert.Contains(t, result, `""`)
}

func TestHighlightJSONHumanKeepsQuotedSpecialKeys(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeHuman}

	// Key with space stays quoted.
	result := json.Highlight(`{"my key":1}`, styles)
	assert.Contains(t, result, `"my key"`)

	// Key with colon stays quoted.
	result = json.Highlight(`{"a:b":1}`, styles)
	assert.Contains(t, result, `"a:b"`)

	// Key with hash stays quoted.
	result = json.Highlight(`{"a#b":1}`, styles)
	assert.Contains(t, result, `"a#b"`)
}

func TestHighlightJSONHumanUnquotesNonIdentifierKeys(t *testing.T) {
	// Per HJSON spec, keys only need quoting for ,{}[]\s:#"' and ///*.
	// Digits, dots, slashes (not //) etc. are fine unquoted.
	styles := &style.JSON{Mode: style.JSONModeHuman}

	result := json.Highlight(`{"1key":1}`, styles)
	assert.NotContains(t, result, `"1key"`)
	assert.Contains(t, result, "1key")

	result = json.Highlight(`{"a.b":1}`, styles)
	assert.NotContains(t, result, `"a.b"`)
	assert.Contains(t, result, "a.b")
}

func TestHighlightJSONDefaultModeKeepsQuotes(t *testing.T) {
	// JSONModeJSON (default) preserves all quotes.
	styles := &style.JSON{}
	result := json.Highlight(`{"key":"value"}`, styles)
	assert.Contains(t, result, `"key"`)
	assert.Contains(t, result, `"value"`)
}

func TestHighlightJSONSpacingAfterColon(t *testing.T) {
	styles := &style.JSON{Spacing: style.JSONSpacingAfterColon}
	result := json.Highlight(`{"a":1,"b":"x"}`, styles)

	assert.Contains(t, result, `"a": 1`)
	assert.Contains(t, result, `"b": "x"`)
	assert.NotContains(t, result, ", ") // no space after comma
}

func TestHighlightJSONSpacingAfterComma(t *testing.T) {
	styles := &style.JSON{Spacing: style.JSONSpacingAfterComma}
	result := json.Highlight(`{"a":1,"b":"x"}`, styles)

	assert.Contains(t, result, `1, "b"`)
	assert.NotContains(t, result, `"a": `) // no space after colon
}

func TestHighlightJSONSpacingAll(t *testing.T) {
	styles := &style.JSON{Spacing: style.JSONSpacingAll}
	result := json.Highlight(`{"a":1,"b":"x"}`, styles)

	assert.Contains(t, result, `"a": 1`)
	assert.Contains(t, result, `1, "b"`)
}

func TestHighlightJSONSpacingInArray(t *testing.T) {
	styles := &style.JSON{Spacing: style.JSONSpacingAfterComma}
	result := json.Highlight(`[1,2,3]`, styles)

	assert.Contains(t, result, "1, 2")
	assert.Contains(t, result, "2, 3")
}

func TestHighlightJSONSpacingWithFlatMode(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeFlat, Spacing: style.JSONSpacingAll}
	result := json.Highlight(`{"user":{"name":"alice"},"count":3}`, styles)

	assert.Contains(t, result, "user.name: ")
	assert.Contains(t, result, ", count")
}

func TestHighlightJSONSpacingNone(t *testing.T) {
	// zero value: no spaces anywhere
	styles := &style.JSON{}
	result := json.Highlight(`{"a":1,"b":2}`, styles)

	assert.NotContains(t, result, " ")
}

func TestHighlightJSONWithSpacingMethod(t *testing.T) {
	styles := style.DefaultJSON().WithSpacing(style.JSONSpacingAll)
	result := json.Highlight(`{"n":1}`, styles)

	// With JSONSpacingAll a space is inserted after the colon.
	// Tokens are styled, so check for colon-space-number using rendered values.
	assert.Contains(t, result, styles.Colon.Render(":"))
	assert.Contains(t, result, " "+styles.Number.Render("1"))
}

func TestHighlightJSONFlatNestedObject(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeFlat}
	result := json.Highlight(`{"user":{"name":"alice","age":30}}`, styles)

	assert.Contains(t, result, "user.name")
	assert.Contains(t, result, "user.age")
	assert.NotContains(t, result, `"user"`)
	assert.NotContains(t, result, `"name"`)
}

func TestHighlightJSONFlatArrayKeptIntact(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeFlat}
	result := json.Highlight(`{"tags":["a","b","c"]}`, styles)

	// Array is kept as-is; no indexing like tags[0]
	assert.Contains(t, result, "tags")
	assert.Contains(t, result, "[")
	assert.NotContains(t, result, "tags[0]")
	assert.NotContains(t, result, "tags.0")
}

func TestHighlightJSONFlatDeeplyNested(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeFlat}
	result := json.Highlight(`{"a":{"b":{"c":1}}}`, styles)

	assert.Contains(t, result, "a.b.c")
	assert.NotContains(t, result, `"a"`)
}

func TestHighlightJSONFlatMixedTypes(t *testing.T) {
	styles := &style.JSON{Mode: style.JSONModeFlat}
	result := json.Highlight(
		`{"status":"ok","meta":{"count":3,"active":true},"tags":["x","y"]}`,
		styles,
	)

	assert.Contains(t, result, "status")
	assert.Contains(t, result, "meta.count")
	assert.Contains(t, result, "meta.active")
	assert.Contains(t, result, "tags")
	// array preserved
	assert.Contains(t, result, "[")
}

func TestHighlightJSONFlatNonObjectFallsBack(t *testing.T) {
	// A root array should not be flattened
	styles := &style.JSON{Mode: style.JSONModeFlat}
	result := json.Highlight(`[1,2,3]`, styles)

	assert.Contains(t, result, "1")
	assert.Contains(t, result, "2")
	assert.Contains(t, result, "3")
	assert.NotContains(t, result, "0.") // no index-based keys
}

func TestHighlightJSONFlatUnquotesValues(t *testing.T) {
	// Flat mode implies human-style unquoting for scalar values
	styles := &style.JSON{Mode: style.JSONModeFlat}
	result := json.Highlight(`{"status":"ok","code":200}`, styles)

	// "ok" should be unquoted (human mode for values)
	assert.NotContains(t, result, `"ok"`)
	assert.Contains(t, result, "ok")
}

func TestHighlightJSONRootBrace(t *testing.T) {
	rootStyle := lipgloss.NewStyle().Bold(true)
	nestedStyle := lipgloss.NewStyle().Faint(true)

	styles := &style.JSON{Brace: &nestedStyle, BraceRoot: &rootStyle}
	result := json.Highlight(`{"a":{"b":1}}`, styles)

	// Root braces use RootBrace style.
	assert.Contains(t, result, rootStyle.Render("{"))
	assert.Contains(t, result, rootStyle.Render("}"))
	// Nested braces use Brace style.
	assert.Contains(t, result, nestedStyle.Render("{"))
	assert.Contains(t, result, nestedStyle.Render("}"))
}

func TestHighlightJSONRootBracket(t *testing.T) {
	rootStyle := lipgloss.NewStyle().Bold(true)
	nestedStyle := lipgloss.NewStyle().Faint(true)

	styles := &style.JSON{Bracket: &nestedStyle, BracketRoot: &rootStyle}
	result := json.Highlight(`[[1,2],[3]]`, styles)

	assert.Contains(t, result, rootStyle.Render("["))
	assert.Contains(t, result, rootStyle.Render("]"))
	assert.Contains(t, result, nestedStyle.Render("["))
	assert.Contains(t, result, nestedStyle.Render("]"))
}

func TestHighlightJSONRootBraceFallsBackToBrace(t *testing.T) {
	// When RootBrace is nil, root braces use Brace style.
	braceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	styles := &style.JSON{Brace: &braceStyle}
	result := json.Highlight(`{"a":1}`, styles)

	assert.Equal(t, braceStyle.Render("{")+"\"a\""+":1"+braceStyle.Render("}"), result)
}

func TestHighlightJSONRootBracketFallsBackToBracket(t *testing.T) {
	bracketStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	styles := &style.JSON{Bracket: &bracketStyle}
	result := json.Highlight(`[1,2]`, styles)

	assert.Equal(t, bracketStyle.Render("[")+"1"+",2"+bracketStyle.Render("]"), result)
}

func TestHighlightJSONRootArray(t *testing.T) {
	// A bare array is valid JSON at the root.
	rootStyle := lipgloss.NewStyle().Bold(true)
	styles := &style.JSON{BracketRoot: &rootStyle}
	result := json.Highlight(`[1,"x",null]`, styles)

	assert.Contains(t, result, rootStyle.Render("["))
	assert.Contains(t, result, rootStyle.Render("]"))
}

func TestEventPath(t *testing.T) {
	l := NewWriter(io.Discard)
	e := l.Info()
	e.Path("dir", "/tmp")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "dir", e.fields[0].Key)
	assert.Equal(t, "/tmp", e.fields[0].Value)
}

func TestEventLine(t *testing.T) {
	l := NewWriter(io.Discard)
	e := l.Info()
	e.Line("file", "main.go", 42)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "file", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so pathLinkWithMode returns plain text.
	assert.Equal(t, "main.go:42", e.fields[0].Value)
}

func TestEventLineColorAlways(t *testing.T) {
	l := New(NewOutput(io.Discard, ColorAlways))

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
	l := New(NewOutput(io.Discard, ColorNever))

	e := l.Info()
	e.Line("file", "main.go", 10)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "main.go:10", e.fields[0].Value)
}

func TestEventLineMinimum(t *testing.T) {
	l := NewWriter(io.Discard)
	e := l.Info()
	e.Line("file", "main.go", 0)

	require.Len(t, e.fields, 1)
	// Line number 0 should be clamped to 1.
	assert.Equal(t, "main.go:1", e.fields[0].Value)
}

func TestEventColumn(t *testing.T) {
	l := NewWriter(io.Discard)
	e := l.Info()
	e.Column("loc", "main.go", 42, 10)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "loc", e.fields[0].Key)
	// Colors disabled in tests (no TTY), so returns plain text.
	assert.Equal(t, "main.go:42:10", e.fields[0].Value)
}

func TestEventColumnColorAlways(t *testing.T) {
	clearFormats(t)

	l := New(NewOutput(io.Discard, ColorAlways))

	e := l.Info()
	e.Column("loc", "/tmp/test.go", 10, 5)

	require.Len(t, e.fields, 1)

	val, ok := e.fields[0].Value.(string)
	require.True(t, ok)
	assert.Contains(t, val, "\x1b]8;;")
	assert.Contains(t, val, "/tmp/test.go:10:5")
}

func TestEventColumnMinimum(t *testing.T) {
	l := NewWriter(io.Discard)
	e := l.Info()
	e.Column("loc", "main.go", 0, 0)

	require.Len(t, e.fields, 1)
	// Both line and column should be clamped to 1.
	assert.Equal(t, "main.go:1:1", e.fields[0].Value)
}

func TestEventStringer(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Stringer("name", testStringer{s: "hello"})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "name", e.fields[0].Key)
	assert.Equal(t, "hello", e.fields[0].Value)
}

func TestEventStringerNil(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	result := e.Stringer("key", nil)

	assert.Same(t, e, result, "expected same event returned")
	assert.Empty(t, e.fields)
}

func TestEventStringers(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Stringers("items", []fmt.Stringer{testStringer{s: "x"}, testStringer{s: "y"}})

	require.Len(t, e.fields, 1)

	vals, ok := e.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"x", "y"}, vals)
}

func TestEventStringersWithNil(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Stringers("items", []fmt.Stringer{testStringer{s: "x"}, nil})

	require.Len(t, e.fields, 1)

	vals, ok := e.fields[0].Value.([]string)
	require.True(t, ok, "expected []string value")
	assert.Equal(t, []string{"x", "<nil>"}, vals)
}

func TestEventSymbol(t *testing.T) {
	l := NewWriter(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Symbol(">>>").Msg("test")

	assert.Equal(t, ">>>", got.Symbol)
}

func TestEventNilReceiverSafety(t *testing.T) {
	var e *Event

	// All field methods should return nil without panic.
	assert.Nil(t, e.Any("k", "v"))
	assert.Nil(t, e.Anys("k", []any{"v"}))
	assert.Nil(t, e.Base64("k", []byte("v")))
	assert.Nil(t, e.Bool("k", true))
	assert.Nil(t, e.Bools("k", []bool{true}))
	assert.Nil(t, e.Bytes("k", []byte("v")))
	assert.Nil(t, e.Column("k", "file.go", 1, 1))
	assert.Nil(t, e.Dict("k", Dict().Str("a", "b")))
	assert.Nil(t, e.Duration("k", time.Second))
	assert.Nil(t, e.Durations("k", []time.Duration{time.Second}))
	assert.Nil(t, e.Err(errors.New("x")))
	assert.Nil(t, e.Errs("k", []error{errors.New("x")}))
	assert.Nil(t, e.Func(func(*Event) {}))
	assert.Nil(t, e.When(true, func(*Event) {}))
	assert.Nil(t, e.Float64("k", 1.0))
	assert.Nil(t, e.Floats64("k", []float64{1.0}))
	assert.Nil(t, e.Hex("k", []byte{0xab}))
	assert.Nil(t, e.Int("k", 1))
	assert.Nil(t, e.Int64("k", 1))
	assert.Nil(t, e.Ints("k", []int{1}))
	assert.Nil(t, e.Line("k", "file.go", 1))
	assert.Nil(t, e.Link("k", "https://example.com", "text"))
	assert.Nil(t, e.Path("k", "file.go"))
	assert.Nil(t, e.Parts(PartMessage))
	assert.Nil(t, e.Percent("k", 50))
	assert.Nil(t, e.Symbol("p"))
	assert.Nil(t, e.Quantities("k", []string{"10GB"}))
	assert.Nil(t, e.Quantity("k", "10GB"))
	assert.Nil(t, e.Str("k", "v"))
	assert.Nil(t, e.Stringer("k", testStringer{s: "x"}))
	assert.Nil(t, e.Stringers("k", []fmt.Stringer{testStringer{s: "x"}}))
	assert.Nil(t, e.Strs("k", []string{"v"}))
	assert.Nil(t, e.Time("k", time.Now()))
	assert.Nil(t, e.Uint("k", 1))
	assert.Nil(t, e.Uint64("k", 1))
	assert.Nil(t, e.Uints64("k", []uint64{1}))
	assert.Nil(t, e.URL("k", "https://example.com"))
	assert.Nil(t, e.withFields([]Field{{Key: "k", Value: "v"}}))
	assert.Nil(t, e.withParts(&[]Part{PartMessage}))
	assert.Nil(t, e.withSymbol("p"))

	// Finalizers should not panic.
	e.Msg("test")
	e.Msgf("test %s", "arg")
	e.Send()
}

func TestEventMsg(t *testing.T) {
	l := NewWriter(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Str("k", "v").Msg("hello")

	assert.Equal(t, LevelInfo, got.Level)
	assert.Equal(t, "hello", got.Message)
	require.Len(t, got.Fields, 1)
	assert.Equal(t, "k", got.Fields[0].Key)
}

func TestEventMsgf(t *testing.T) {
	l := NewWriter(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Msgf("hello %s %d", "world", 42)

	assert.Equal(t, "hello world 42", got.Message)
}

func TestEventSend(t *testing.T) {
	l := NewWriter(io.Discard)

	var got Entry

	l.SetHandler(HandlerFunc(func(e Entry) {
		got = e
	}))

	l.Info().Str("k", "v").Send()

	assert.Empty(t, got.Message)
	assert.Len(t, got.Fields, 1)
}

func TestEventWithFields(t *testing.T) {
	e := NewWriter(io.Discard).Info()
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

func TestEventWithSymbol(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e = e.withSymbol("CUSTOM")

	require.NotNil(t, e.symbol)
	assert.Equal(t, "CUSTOM", *e.symbol)
}

func TestEventWithSymbolNilReceiver(t *testing.T) {
	var e *Event

	got := e.withSymbol("CUSTOM")
	assert.Nil(t, got, "expected nil from withSymbol on nil event")
}

func TestEventParts(t *testing.T) {
	t.Run("reorder", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(TestOutput(&buf))
		l.Info().Parts(PartMessage, PartLevel, PartSymbol).Msg("hello")
		assert.Equal(t, "hello INF ℹ️\n", buf.String())
	})

	t.Run("omit", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(TestOutput(&buf))
		l.Info().Parts(PartMessage).Str("k", "v").Msg("hello")
		assert.Equal(t, "hello\n", buf.String())
	})

	t.Run("does_not_mutate_logger", func(t *testing.T) {
		var buf bytes.Buffer
		l := New(TestOutput(&buf))
		l.Info().Parts(PartMessage).Msg("first")
		buf.Reset()
		l.Info().Msg("second")
		assert.Contains(t, buf.String(), "INF")
	})

	t.Run("nil_receiver", func(t *testing.T) {
		var e *Event
		assert.Nil(t, e.Parts(PartMessage))
	})
}

func TestEventWithParts(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	p := []Part{PartMessage}
	e = e.withParts(&p)

	require.NotNil(t, e.parts)
	assert.Equal(t, []Part{PartMessage}, *e.parts)
}

func TestEventWithPartsNilReceiver(t *testing.T) {
	var e *Event

	got := e.withParts(&[]Part{PartMessage})
	assert.Nil(t, got, "expected nil from withParts on nil event")
}

func TestEventChaining(t *testing.T) {
	l := NewWriter(io.Discard)

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
	e := NewWriter(io.Discard).Info()
	vals := []time.Duration{time.Second, 2 * time.Millisecond}
	e.Durations("timings", vals)
	assertSliceField(t, e.fields, vals)
}

func TestEventDurationsOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Durations("d", []time.Duration{time.Second, 500 * time.Millisecond}).Msg("test")

	assert.Equal(t, "INF ℹ️ test d=[1s, 500ms]\n", buf.String())
}

func TestEventPercent(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Percent("progress", 0.75)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "progress", e.fields[0].Key)

	p, ok := e.fields[0].Value.(core.Percent)
	require.True(t, ok, "expected percent value")
	assert.InDelta(t, 0.75, p.Value, 0)
}

func TestEventPercentClamping(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Percent("low", -0.10)
	e.Percent("high", 1.50)

	require.Len(t, e.fields, 2)

	low, ok := e.fields[0].Value.(core.Percent)
	require.True(t, ok)
	assert.InDelta(t, 0.0, low.Value, 0, "negative should clamp to 0")

	high, ok := e.fields[1].Value.(core.Percent)
	require.True(t, ok)
	assert.InDelta(t, 1.0, high.Value, 0, "over scale should clamp to scale")
}

func TestEventPercentOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Percent("progress", 0.75).Msg("done")

	assert.Equal(t, "INF ℹ️ done progress=75%\n", buf.String())
}

func TestEventQuantity(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Quantity("size", "10GB")

	require.Len(t, e.fields, 1)
	assert.Equal(t, "size", e.fields[0].Key)
}

func TestEventQuantityOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Quantity("size", "10GB").Msg("done")

	assert.Equal(t, "INF ℹ️ done size=10GB\n", buf.String())
}

func TestEventQuantities(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Quantities("sizes", []string{"10GB", "5MB"})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "sizes", e.fields[0].Key)
}

func TestEventQuantitiesOutput(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Quantities("sizes", []string{"10GB", "5MB"}).Msg("test")

	assert.Equal(t, "INF ℹ️ test sizes=[10GB, 5MB]\n", buf.String())
}

func TestEventDictPanicOnMsg(t *testing.T) {
	assert.PanicsWithValue(t,
		"clog: Msg/Msgf/Send called on a Dict() event -- pass it to Event.Dict() instead",
		func() {
			Dict().Str("k", "v").Msg("boom")
		},
	)
}

func TestEventDictPanicOnMsgf(t *testing.T) {
	assert.PanicsWithValue(t,
		"clog: Msg/Msgf/Send called on a Dict() event -- pass it to Event.Dict() instead",
		func() {
			Dict().Str("k", "v").Msgf("boom %s", "arg")
		},
	)
}

func TestEventDictPanicOnSend(t *testing.T) {
	assert.PanicsWithValue(t,
		"clog: Msg/Msgf/Send called on a Dict() event -- pass it to Event.Dict() instead",
		func() {
			Dict().Str("k", "v").Send()
		},
	)
}

func TestEventDictNilParam(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Str("before", "x")

	result := e.Dict("group", nil)

	assert.Same(t, e, result, "expected same event returned")
	require.Len(t, e.fields, 1, "nil dict should not add fields")
	assert.Equal(t, "before", e.fields[0].Key)
}

func TestEventStringerTypedNil(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	var buf *bytes.Buffer // typed nil that implements fmt.Stringer

	result := e.Stringer("key", buf)

	assert.Same(t, e, result, "expected same event returned")
	assert.Empty(t, e.fields, "typed nil stringer should not add a field")
}

func TestEventEmptyFieldKey(t *testing.T) {
	var buf bytes.Buffer

	l := New(TestOutput(&buf))
	l.Info().Str("", "value").Msg("test")

	assert.Contains(t, buf.String(), "=value")
}

func TestEventMsgFatalCallsExit(t *testing.T) {
	var exitCode int

	l := NewWriter(io.Discard)
	l.SetExitFunc(func(code int) {
		exitCode = code
	})
	l.Fatal().Msg("fatal error")

	assert.Equal(t, 1, exitCode)
}

func TestEventJSONValid(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.JSON("key", map[string]int{"a": 1})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "key", e.fields[0].Key)

	_, ok := e.fields[0].Value.(core.RawJSON)
	require.True(t, ok, "expected rawJSON value")
}

func TestEventJSONNilReceiver(t *testing.T) {
	var e *Event
	got := e.JSON("key", map[string]int{"a": 1})
	assert.Nil(t, got)
}

func TestEventJSONMarshalErrorInf(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.JSON("bad", math.Inf(1))

	require.Len(t, e.fields, 1)
	assert.Equal(t, "bad", e.fields[0].Key)

	_, isRaw := e.fields[0].Value.(core.RawJSON)
	assert.False(t, isRaw, "marshal error should not produce rawJSON")

	val, isStr := e.fields[0].Value.(string)
	require.True(t, isStr, "expected error string value")
	assert.Contains(t, val, "unsupported value")
}

func TestEventRawJSONNilReceiver(t *testing.T) {
	var e *Event
	got := e.RawJSON("k", []byte("{}"))
	assert.Nil(t, got)
}

func TestEventInts64(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Ints64("nums", []int64{1, 2, 3})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "nums", e.fields[0].Key)
	assertSliceField(t, e.fields, []int64{1, 2, 3})
}

func TestEventTimes(t *testing.T) {
	t1 := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 16, 12, 0, 0, 0, time.UTC)
	e := NewWriter(io.Discard).Info()
	e.Times("timestamps", []time.Time{t1, t2})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "timestamps", e.fields[0].Key)
	assertSliceField(t, e.fields, []time.Time{t1, t2})
}

func TestEventUints(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.Uints("counts", []uint{10, 20, 30})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "counts", e.fields[0].Key)
	assertSliceField(t, e.fields, []uint{10, 20, 30})
}
