package clog

import (
	"bytes"
	"testing"

	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
)

func newTestPrinter() (*Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return New(TestOutput(&buf)), &buf
}

func TestPrinterRawJSON(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().RawJSON([]byte(`{"status":"ok","count":42}`))

	want := `{
  "status": "ok",
  "count": 42
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterRawJSONFlatMode(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().Mode(JSONFlat).RawJSON([]byte(`{
  "name": "alice",
  "age": 30
}`))

	// Printer uses its own formatting, not JSON spacing.
	want := `{"name":"alice","age":30}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterRawJSONEmpty(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().RawJSON([]byte{})
	assert.Equal(t, nl, buf.String())
}

func TestPrinterRawJSONNil(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().RawJSON(nil)
	assert.Equal(t, nl, buf.String())
}

func TestPrinterJSON(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().JSON(map[string]int{"a": 1})

	want := `{
  "a": 1
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterJSONFlat(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().Mode(JSONFlat).JSON(map[string]int{"a": 1})

	want := `{"a":1}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterJSONMarshalError(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().JSON(make(chan int))
	assert.Contains(t, buf.String(), "unsupported type")
}

func TestPrinterRawJSONHooks(t *testing.T) {
	l, buf := newTestPrinter()
	_ = buf

	var hookOrder []string
	l.AddHook(HookBeforeWrite, func() { hookOrder = append(hookOrder, "before") })
	l.AddHook(HookAfterWrite, func() { hookOrder = append(hookOrder, "after") })

	l.Print().RawJSON([]byte(`{"a":1}`))

	assert.Equal(t, []string{"before", "after"}, hookOrder)
}

func TestPrinterGlobalModeDefault(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := `{
  "a": 1
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterGlobalModeInline(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetJSONPrintMode(JSONFlat)

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := `{"a":1}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterGlobalModeOverrideToInline(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().Mode(JSONFlat).RawJSON([]byte(`{"a":1}`))

	want := `{"a":1}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterGlobalModeOverrideToMultiline(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetJSONPrintMode(JSONFlat)

	l.Print().Mode(JSONPretty).RawJSON([]byte(`{"a":1}`))

	want := `{
  "a": 1
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterJSONPreserve(t *testing.T) {
	l, buf := newTestPrinter()

	// Badly indented but valid JSON — preserve mode keeps it as-is.
	input := "{\n\"a\": 1,\n\"b\": 2\n}"
	l.Print().Mode(JSONPreserve).RawJSON([]byte(input))

	assert.Equal(t, input+nl, buf.String())
}

func TestPrinterJSONPreservePrettyPrinted(t *testing.T) {
	l, buf := newTestPrinter()

	input := `{
    "name": "alice",
    "scores": [
        1,
        2
    ]
}`
	l.Print().Mode(JSONPreserve).RawJSON([]byte(input))

	assert.Equal(t, input+nl, buf.String())
}

func TestPrinterCustomIndent(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetPrintIndent("\t")

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := "{\n\t\"a\": 1\n}\n"
	assert.Equal(t, want, buf.String())
}

func TestPrinterJSONIndentOverridesPrintIndent(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetPrintIndent("    ")
	l.SetJSONIndent("\t")

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := "{\n\t\"a\": 1\n}\n"
	assert.Equal(t, want, buf.String())
}

func TestPrinterYAMLIndentOverridesPrintIndent(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetPrintIndent("    ")
	l.SetYAMLIndent("  ")

	l.Print().YAML(map[string]int{"a": 1})

	assert.Equal(t, "a: 1\n", buf.String())
}

func TestPrinterYAMLIndentSequenceDisabled(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetYAMLIndentSequence(false)

	l.Print().YAML(map[string]any{
		"tags": []string{"a", "b"},
	})

	want := `tags:
- a
- b
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterSubLoggerInheritsSettings(t *testing.T) {
	l, _ := newTestPrinter()
	l.SetJSONPrintMode(JSONFlat)
	l.SetPrintIndent("\t")

	var buf2 bytes.Buffer
	sub := l.With().Str("ctx", "val").Logger()
	sub.SetOutput(TestOutput(&buf2))

	sub.Print().RawJSON([]byte(`{"a":1}`))

	// Sub-logger should inherit JSONFlat from parent.
	want := `{"a":1}`
	assert.Equal(t, want+nl, buf2.String())
}

func TestPrinterIgnoresJSONMode(t *testing.T) {
	l, buf := newTestPrinter()

	// Set JSON to flat mode - Printer should ignore it and use standard JSON.
	s := DefaultStyles()
	s.JSON.Mode = style.JSONModeFlat
	l.SetStyles(s)

	l.Print().RawJSON([]byte(`{"user":{"name":"alice"}}`))

	// Should render nested JSON, not flat dot-notation.
	want := `{
  "user": {
    "name": "alice"
  }
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterIgnoresJSONSpacing(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetJSONPrintMode(JSONFlat)

	// Set OmitCommas and custom spacing on JSON - Printer should ignore them.
	s := DefaultStyles()
	s.JSON.OmitCommas = true
	s.JSON.Spacing = style.JSONSpacingAll
	l.SetStyles(s)

	l.Print().RawJSON([]byte(`{"a":1,"b":2}`))

	// Printer uses default rendering: commas present, no extra spacing.
	want := `{"a":1,"b":2}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterYAML(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().YAML(map[string]int{"a": 1})

	assert.Equal(t, "a: 1\n", buf.String())
}

func TestPrinterYAMLIndentedSequence(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().YAML(map[string]any{
		"tags": []string{"a", "b"},
	})

	want := `tags:
  - a
  - b
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterRawYAML(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().RawYAML([]byte("name: alice\nage: 30\n"))

	assert.Equal(t, "name: alice\nage: 30\n", buf.String())
}

func TestPrinterRawYAMLAddsTrailingNewline(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().RawYAML([]byte("key: value"))

	assert.Equal(t, "key: value\n", buf.String())
}

func TestPrinterRawYAMLNil(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().RawYAML(nil)
	assert.Equal(t, nl, buf.String())
}

func TestPrinterRawYAMLEmpty(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().RawYAML([]byte{})
	assert.Equal(t, nl, buf.String())
}

func TestPrinterYAMLMarshalError(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().YAML(make(chan int))
	assert.Contains(t, buf.String(), "unknown value type")
}

func TestPrinterPackageLevel(t *testing.T) {
	old := Default
	l, buf := newTestPrinter()
	Default = l

	defer func() { Default = old }()

	Print().RawJSON([]byte(`{"ok":true}`))

	want := `{
  "ok": true
}
`
	assert.Equal(t, want, buf.String())
}
