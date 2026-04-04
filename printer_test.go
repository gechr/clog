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

func TestPrinterRawJSONInlineMode(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().Mode(PrintInline).RawJSON([]byte(`{
  "name": "alice",
  "age": 30
}`))

	// Printer uses its own formatting, not FieldJSON spacing.
	want := `{"name":"alice","age":30}`
	assert.Equal(t, want+"\n", buf.String())
}

func TestPrinterRawJSONEmpty(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().RawJSON([]byte{})
	assert.Equal(t, "\n", buf.String())
}

func TestPrinterRawJSONNil(t *testing.T) {
	l, buf := newTestPrinter()
	l.Print().RawJSON(nil)
	assert.Equal(t, "\n", buf.String())
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

func TestPrinterJSONInline(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().Mode(PrintInline).JSON(map[string]int{"a": 1})

	want := `{"a":1}`
	assert.Equal(t, want+"\n", buf.String())
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
	l.SetPrintMode(PrintInline)

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := `{"a":1}`
	assert.Equal(t, want+"\n", buf.String())
}

func TestPrinterGlobalModeOverrideToInline(t *testing.T) {
	l, buf := newTestPrinter()

	l.Print().Mode(PrintInline).RawJSON([]byte(`{"a":1}`))

	want := `{"a":1}`
	assert.Equal(t, want+"\n", buf.String())
}

func TestPrinterGlobalModeOverrideToMultiline(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetPrintMode(PrintInline)

	l.Print().Mode(PrintMultiline).RawJSON([]byte(`{"a":1}`))

	want := `{
  "a": 1
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterCustomIndent(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetPrintIndent("\t")

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := "{\n\t\"a\": 1\n}\n"
	assert.Equal(t, want, buf.String())
}

func TestPrinterSubLoggerInheritsSettings(t *testing.T) {
	l, _ := newTestPrinter()
	l.SetPrintMode(PrintInline)
	l.SetPrintIndent("\t")

	var buf2 bytes.Buffer
	sub := l.With().Str("ctx", "val").Logger()
	sub.SetOutput(TestOutput(&buf2))

	sub.Print().RawJSON([]byte(`{"a":1}`))

	// Sub-logger should inherit PrintInline from parent.
	want := `{"a":1}`
	assert.Equal(t, want+"\n", buf2.String())
}

func TestPrinterIgnoresFieldJSONMode(t *testing.T) {
	l, buf := newTestPrinter()

	// Set FieldJSON to flat mode - Printer should ignore it and use standard JSON.
	s := DefaultStyles()
	s.FieldJSON.Mode = style.JSONModeFlat
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

func TestPrinterIgnoresFieldJSONSpacing(t *testing.T) {
	l, buf := newTestPrinter()
	l.SetPrintMode(PrintInline)

	// Set OmitCommas and custom spacing on FieldJSON - Printer should ignore them.
	s := DefaultStyles()
	s.FieldJSON.OmitCommas = true
	s.FieldJSON.Spacing = style.JSONSpacingAll
	l.SetStyles(s)

	l.Print().RawJSON([]byte(`{"a":1,"b":2}`))

	// Printer uses default rendering: commas present, no extra spacing.
	want := `{"a":1,"b":2}`
	assert.Equal(t, want+"\n", buf.String())
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
