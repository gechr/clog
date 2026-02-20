package clog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
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

	require.Len(t, e.fields, 1)
	assert.Equal(t, "error", e.fields[0].Key)

	gotErr, ok := e.fields[0].Value.(error)
	require.True(t, ok, "expected error value")
	assert.Equal(t, "boom", gotErr.Error())
}

func TestEventErrNil(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	result := e.Err(nil)

	assert.Same(t, e, result, "expected same event returned")
	assert.Empty(t, e.fields)
}

func TestEventJSON(t *testing.T) {
	e := NewWriter(io.Discard).Info()
	e.JSON("data", map[string]any{"status": "ok", "n": 42})

	require.Len(t, e.fields, 1)
	assert.Equal(t, "data", e.fields[0].Key)
	_, ok := e.fields[0].Value.(rawJSON)
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
	// Channels are not JSON-serializable — error stored as field value.
	e := NewWriter(io.Discard).Info()
	e.JSON("bad", make(chan int))

	require.Len(t, e.fields, 1)
	_, isRaw := e.fields[0].Value.(rawJSON)
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

	got, ok := e.fields[0].Value.(rawJSON)
	require.True(t, ok, "expected rawJSON value")
	assert.Equal(t, rawJSON(data), got)
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
	styles := &JSONStyles{Number: new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff79c6")))}
	result := highlightJSON(`{"n":1}`, styles)
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
	assert.Contains(t, got, `data={"n":1}`)
}

func TestEventRawJSONUnquoted(t *testing.T) {
	var buf bytes.Buffer
	l := NewWriter(&buf)
	styles := DefaultStyles()
	styles.FieldJSON = DefaultJSONStyles()
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

	styles := &JSONStyles{True: &trueStyle, False: &falseStyle, Null: &nullStyle}
	result := highlightJSON(`{"a":true,"b":false,"c":null}`, styles)

	assert.Contains(t, result, trueStyle.Render("true"))
	assert.Contains(t, result, falseStyle.Render("false"))
	assert.Contains(t, result, nullStyle.Render("null"))
}

func TestHighlightJSONNilFieldsUnstyled(t *testing.T) {
	// Tokens without a style render as plain text.
	styles := &JSONStyles{
		Key: DefaultJSONStyles().Key, // only keys get a style
	}
	result := highlightJSON(`{"k":42}`, styles)

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
	result := highlightJSON(pretty, nil)
	// nil styles: returned unchanged
	assert.Equal(t, pretty, result)

	// with styles: whitespace stripped
	result = highlightJSON(pretty, &JSONStyles{})
	assert.Equal(t, compact, result)
}

func TestHighlightJSONInvalidFallback(t *testing.T) {
	// invalid JSON: scanner emits styled prefix then falls back unstyled
	result := highlightJSON(`{"k":INVALID}`, &JSONStyles{})
	assert.Contains(t, result, `"k"`)
	assert.Contains(t, result, "INVALID}")
}

func TestHighlightJSONTrailingBackslash(t *testing.T) {
	// Trailing backslash in a string must not panic.
	styles := &JSONStyles{}
	assert.NotPanics(t, func() {
		highlightJSON(`"hello\`, styles)
	})
	assert.NotPanics(t, func() {
		highlightJSON(`{"key\":1}`, styles)
	})
	assert.NotPanics(t, func() {
		highlightJSON(`{"k":"val\`, styles)
	})

	// Flat mode exercises collectFlatPairs and scanJSONValueEnd.
	flat := &JSONStyles{Mode: JSONModeFlat}
	assert.NotPanics(t, func() {
		highlightJSON(`{"k":"val\`, flat)
	})
	assert.NotPanics(t, func() {
		highlightJSON(`{"k":["\`, flat)
	})
}

func TestHighlightJSONHJSONUnquotesKeys(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeHuman}
	result := highlightJSON(`{"status":"ok","code":200}`, styles)

	assert.Contains(t, result, "status")
	assert.NotContains(t, result, `"status"`)
	assert.Contains(t, result, "code")
	assert.NotContains(t, result, `"code"`)
}

func TestHighlightJSONHJSONUnquotesSimpleValues(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeHuman}
	result := highlightJSON(`{"status":"ok","msg":"hello world"}`, styles)

	// Simple value without special chars: unquoted.
	assert.Contains(t, result, ":ok")
	assert.NotContains(t, result, `:"ok"`)
	// Value with a space: still unquoted (spaces are safe in HJSON values).
	assert.Contains(t, result, "hello world")
}

func TestHighlightJSONHumanKeepsQuotedSpecialValues(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeHuman}

	// Value starting with { stays quoted (ambiguous).
	result := highlightJSON(`{"a":"{not an object}"}`, styles)
	assert.Contains(t, result, `"{not an object}"`)

	// Value starting with [ stays quoted.
	result = highlightJSON(`{"a":"[1,2]"}`, styles)
	assert.Contains(t, result, `"[1,2]"`)

	// Value with escape sequence stays quoted.
	result = highlightJSON(`{"s":"line1\nline2"}`, styles)
	assert.Contains(t, result, `"line1\nline2"`)

	// Keyword values stay quoted (would be ambiguous as bare tokens).
	result = highlightJSON(`{"x":"true"}`, styles)
	assert.Contains(t, result, `"true"`)
	result = highlightJSON(`{"x":"null"}`, styles)
	assert.Contains(t, result, `"null"`)

	// Number-like values stay quoted.
	result = highlightJSON(`{"x":"42"}`, styles)
	assert.Contains(t, result, `"42"`)
	result = highlightJSON(`{"x":"-1.5"}`, styles)
	assert.Contains(t, result, `"-1.5"`)

	// Empty string stays quoted.
	result = highlightJSON(`{"x":""}`, styles)
	assert.Contains(t, result, `""`)
}

func TestHighlightJSONHumanKeepsQuotedSpecialKeys(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeHuman}

	// Key with space stays quoted.
	result := highlightJSON(`{"my key":1}`, styles)
	assert.Contains(t, result, `"my key"`)

	// Key with colon stays quoted.
	result = highlightJSON(`{"a:b":1}`, styles)
	assert.Contains(t, result, `"a:b"`)

	// Key with hash stays quoted.
	result = highlightJSON(`{"a#b":1}`, styles)
	assert.Contains(t, result, `"a#b"`)
}

func TestHighlightJSONHumanUnquotesNonIdentifierKeys(t *testing.T) {
	// Per HJSON spec, keys only need quoting for ,{}[]\s:#"' and ///*.
	// Digits, dots, slashes (not //) etc. are fine unquoted.
	styles := &JSONStyles{Mode: JSONModeHuman}

	result := highlightJSON(`{"1key":1}`, styles)
	assert.NotContains(t, result, `"1key"`)
	assert.Contains(t, result, "1key")

	result = highlightJSON(`{"a.b":1}`, styles)
	assert.NotContains(t, result, `"a.b"`)
	assert.Contains(t, result, "a.b")
}

func TestHighlightJSONDefaultModeKeepsQuotes(t *testing.T) {
	// JSONModeJSON (default) preserves all quotes.
	styles := &JSONStyles{}
	result := highlightJSON(`{"key":"value"}`, styles)
	assert.Contains(t, result, `"key"`)
	assert.Contains(t, result, `"value"`)
}

func TestHighlightJSONSpacingAfterColon(t *testing.T) {
	styles := &JSONStyles{Spacing: JSONSpacingAfterColon}
	result := highlightJSON(`{"a":1,"b":"x"}`, styles)

	assert.Contains(t, result, `"a": 1`)
	assert.Contains(t, result, `"b": "x"`)
	assert.NotContains(t, result, ", ") // no space after comma
}

func TestHighlightJSONSpacingAfterComma(t *testing.T) {
	styles := &JSONStyles{Spacing: JSONSpacingAfterComma}
	result := highlightJSON(`{"a":1,"b":"x"}`, styles)

	assert.Contains(t, result, `1, "b"`)
	assert.NotContains(t, result, `"a": `) // no space after colon
}

func TestHighlightJSONSpacingAll(t *testing.T) {
	styles := &JSONStyles{Spacing: JSONSpacingAll}
	result := highlightJSON(`{"a":1,"b":"x"}`, styles)

	assert.Contains(t, result, `"a": 1`)
	assert.Contains(t, result, `1, "b"`)
}

func TestHighlightJSONSpacingInArray(t *testing.T) {
	styles := &JSONStyles{Spacing: JSONSpacingAfterComma}
	result := highlightJSON(`[1,2,3]`, styles)

	assert.Contains(t, result, "1, 2")
	assert.Contains(t, result, "2, 3")
}

func TestHighlightJSONSpacingWithFlatMode(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeFlat, Spacing: JSONSpacingAll}
	result := highlightJSON(`{"user":{"name":"alice"},"count":3}`, styles)

	assert.Contains(t, result, "user.name: ")
	assert.Contains(t, result, ", count")
}

func TestHighlightJSONSpacingNone(t *testing.T) {
	// zero value: no spaces anywhere
	styles := &JSONStyles{}
	result := highlightJSON(`{"a":1,"b":2}`, styles)

	assert.NotContains(t, result, " ")
}

func TestHighlightJSONWithSpacingMethod(t *testing.T) {
	styles := DefaultJSONStyles().WithSpacing(JSONSpacingAll)
	result := highlightJSON(`{"n":1}`, styles)

	assert.Contains(t, result, `"n": 1`)
}

func TestHighlightJSONFlatNestedObject(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeFlat}
	result := highlightJSON(`{"user":{"name":"alice","age":30}}`, styles)

	assert.Contains(t, result, "user.name")
	assert.Contains(t, result, "user.age")
	assert.NotContains(t, result, `"user"`)
	assert.NotContains(t, result, `"name"`)
}

func TestHighlightJSONFlatArrayKeptIntact(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeFlat}
	result := highlightJSON(`{"tags":["a","b","c"]}`, styles)

	// Array is kept as-is; no indexing like tags[0]
	assert.Contains(t, result, "tags")
	assert.Contains(t, result, "[")
	assert.NotContains(t, result, "tags[0]")
	assert.NotContains(t, result, "tags.0")
}

func TestHighlightJSONFlatDeeplyNested(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeFlat}
	result := highlightJSON(`{"a":{"b":{"c":1}}}`, styles)

	assert.Contains(t, result, "a.b.c")
	assert.NotContains(t, result, `"a"`)
}

func TestHighlightJSONFlatMixedTypes(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeFlat}
	result := highlightJSON(
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
	styles := &JSONStyles{Mode: JSONModeFlat}
	result := highlightJSON(`[1,2,3]`, styles)

	assert.Contains(t, result, "1")
	assert.Contains(t, result, "2")
	assert.Contains(t, result, "3")
	assert.NotContains(t, result, "0.") // no index-based keys
}

func TestHighlightJSONFlatUnquotesValues(t *testing.T) {
	// Flat mode implies human-style unquoting for scalar values
	styles := &JSONStyles{Mode: JSONModeFlat}
	result := highlightJSON(`{"status":"ok","code":200}`, styles)

	// "ok" should be unquoted (human mode for values)
	assert.NotContains(t, result, `"ok"`)
	assert.Contains(t, result, "ok")
}

func TestHighlightJSONRootBrace(t *testing.T) {
	rootStyle := lipgloss.NewStyle().Bold(true)
	nestedStyle := lipgloss.NewStyle().Faint(true)

	styles := &JSONStyles{Brace: &nestedStyle, RootBrace: &rootStyle}
	result := highlightJSON(`{"a":{"b":1}}`, styles)

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

	styles := &JSONStyles{Bracket: &nestedStyle, RootBracket: &rootStyle}
	result := highlightJSON(`[[1,2],[3]]`, styles)

	assert.Contains(t, result, rootStyle.Render("["))
	assert.Contains(t, result, rootStyle.Render("]"))
	assert.Contains(t, result, nestedStyle.Render("["))
	assert.Contains(t, result, nestedStyle.Render("]"))
}

func TestHighlightJSONRootBraceFallsBackToBrace(t *testing.T) {
	// When RootBrace is nil, root braces use Brace style.
	braceStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	styles := &JSONStyles{Brace: &braceStyle}
	result := highlightJSON(`{"a":1}`, styles)

	assert.Equal(t, braceStyle.Render("{")+"\"a\""+":1"+braceStyle.Render("}"), result)
}

func TestHighlightJSONRootBracketFallsBackToBracket(t *testing.T) {
	bracketStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff"))
	styles := &JSONStyles{Bracket: &bracketStyle}
	result := highlightJSON(`[1,2]`, styles)

	assert.Equal(t, bracketStyle.Render("[")+"1"+",2"+bracketStyle.Render("]"), result)
}

func TestHighlightJSONRootArray(t *testing.T) {
	// A bare array is valid JSON at the root.
	rootStyle := lipgloss.NewStyle().Bold(true)
	styles := &JSONStyles{RootBracket: &rootStyle}
	result := highlightJSON(`[1,"x",null]`, styles)

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

func TestEventPrefix(t *testing.T) {
	l := NewWriter(io.Discard)

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
	assert.Nil(t, e.Int64("k", 1))
	assert.Nil(t, e.Uint("k", 1))
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
	l := NewWriter(io.Discard)

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

func TestEventWithPrefix(t *testing.T) {
	e := NewWriter(io.Discard).Info()
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
	e.Percent("progress", 75)

	require.Len(t, e.fields, 1)
	assert.Equal(t, "progress", e.fields[0].Key)

	p, ok := e.fields[0].Value.(percent)
	require.True(t, ok, "expected percent value")
	assert.InDelta(t, 75.0, float64(p), 0)
}

func TestEventPercentClamping(t *testing.T) {
	e := NewWriter(io.Discard).Info()
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

	l := New(TestOutput(&buf))
	l.Info().Percent("progress", 75).Msg("done")

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

	_, ok := e.fields[0].Value.(rawJSON)
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

	_, isRaw := e.fields[0].Value.(rawJSON)
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
