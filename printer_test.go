package clog

import (
	"bytes"
	"testing"

	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
)

func TestPrinterRawJSON(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().RawJSON([]byte(`{"status":"ok","count":42}`))

	want := `{
  "status": "ok",
  "count": 42
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterRawJSONFlatMode(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().Mode(JSONFlat).RawJSON([]byte(`{
  "name": "alice",
  "age": 30
}`))

	// Printer uses its own formatting, not JSON spacing.
	want := `{"name":"alice","age":30}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterRawJSONEmpty(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawJSON([]byte{})
	assert.Equal(t, nl, buf.String())
}

func TestPrinterRawJSONNil(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawJSON(nil)
	assert.Equal(t, nl, buf.String())
}

func TestPrinterJSON(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().JSON(map[string]int{"a": 1})

	want := `{
  "a": 1
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterJSONFlat(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().Mode(JSONFlat).JSON(map[string]int{"a": 1})

	want := `{"a":1}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterJSONMarshalError(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().JSON(make(chan int))
	assert.Equal(t, "json: unsupported type: chan int\n", buf.String())
}

func TestPrinterRawJSONHooks(t *testing.T) {
	l, buf := newTestLogger()
	_ = buf

	var hookOrder []string
	l.AddHook(HookBeforeWrite, func() { hookOrder = append(hookOrder, "before") })
	l.AddHook(HookAfterWrite, func() { hookOrder = append(hookOrder, "after") })

	l.Print().RawJSON([]byte(`{"a":1}`))

	assert.Equal(t, []string{"before", "after"}, hookOrder)
}

func TestPrinterGlobalModeDefault(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := `{
  "a": 1
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterGlobalModeInline(t *testing.T) {
	l, buf := newTestLogger()
	l.SetJSONPrintMode(JSONFlat)

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := `{"a":1}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterGlobalModeOverrideToInline(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().Mode(JSONFlat).RawJSON([]byte(`{"a":1}`))

	want := `{"a":1}`
	assert.Equal(t, want+nl, buf.String())
}

func TestPrinterGlobalModeOverrideToMultiline(t *testing.T) {
	l, buf := newTestLogger()
	l.SetJSONPrintMode(JSONFlat)

	l.Print().Mode(JSONPretty).RawJSON([]byte(`{"a":1}`))

	want := `{
  "a": 1
}
`
	assert.Equal(t, want, buf.String())
}

func TestPrinterJSONPreserve(t *testing.T) {
	l, buf := newTestLogger()

	// Badly indented but valid JSON - preserve mode keeps it as-is.
	input := "{\n\"a\": 1,\n\"b\": 2\n}"
	l.Print().Mode(JSONPreserve).RawJSON([]byte(input))

	assert.Equal(t, input+nl, buf.String())
}

func TestPrinterJSONPreservePrettyPrinted(t *testing.T) {
	l, buf := newTestLogger()

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
	l, buf := newTestLogger()
	l.SetPrintIndent("\t")

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := "{\n\t\"a\": 1\n}\n"
	assert.Equal(t, want, buf.String())
}

func TestPrinterJSONIndentOverridesPrintIndent(t *testing.T) {
	l, buf := newTestLogger()
	l.SetPrintIndent("    ")
	l.SetJSONIndent("\t")

	l.Print().RawJSON([]byte(`{"a":1}`))

	want := "{\n\t\"a\": 1\n}\n"
	assert.Equal(t, want, buf.String())
}

func TestPrinterYAMLIndentOverridesPrintIndent(t *testing.T) {
	l, buf := newTestLogger()
	l.SetPrintIndent("    ")
	l.SetYAMLIndent("  ")

	l.Print().YAML(map[string]int{"a": 1})

	assert.Equal(t, "a: 1\n", buf.String())
}

func TestPrinterYAMLIndentSequenceDisabled(t *testing.T) {
	l, buf := newTestLogger()
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
	l, _ := newTestLogger()
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
	l, buf := newTestLogger()

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
	l, buf := newTestLogger()
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
	l, buf := newTestLogger()

	l.Print().YAML(map[string]int{"a": 1})

	assert.Equal(t, "a: 1\n", buf.String())
}

func TestPrinterYAMLIndentedSequence(t *testing.T) {
	l, buf := newTestLogger()

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
	l, buf := newTestLogger()

	l.Print().RawYAML([]byte("name: alice\nage: 30\n"))

	assert.Equal(t, "name: alice\nage: 30\n", buf.String())
}

func TestPrinterRawYAMLAddsTrailingNewline(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().RawYAML([]byte("key: value"))

	assert.Equal(t, "key: value\n", buf.String())
}

func TestPrinterRawYAMLNil(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawYAML(nil)
	assert.Equal(t, nl, buf.String())
}

func TestPrinterRawYAMLEmpty(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawYAML([]byte{})
	assert.Equal(t, nl, buf.String())
}

func TestPrinterYAMLMarshalError(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().YAML(make(chan int))
	assert.Equal(t, "unknown value type chan int\n", buf.String())
}

func TestPrinterTOML(t *testing.T) {
	l, buf := newTestLogger()

	type Config struct {
		Port int `toml:"port"`
	}
	l.Print().TOML(Config{Port: 8080})

	assert.Equal(t, "port = 8080\n", buf.String())
}

func TestPrinterRawTOML(t *testing.T) {
	l, buf := newTestLogger()

	l.Print().RawTOML([]byte("[server]\nhost = \"localhost\"\n"))

	assert.Equal(t, "[server]\nhost = \"localhost\"\n", buf.String())
}

func TestPrinterRawTOMLNil(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawTOML(nil)
	assert.Equal(t, nl, buf.String())
}

func TestPrinterRawTOMLEmpty(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawTOML([]byte{})
	assert.Equal(t, nl, buf.String())
}

func TestPrinterRawHCL(t *testing.T) {
	l, buf := newTestLogger()

	input := `resource "aws_instance" "web" {
  ami = "ami-12345678"
}
`
	l.Print().RawHCL([]byte(input))

	assert.Equal(t, input, buf.String())
}

func TestPrinterRawHCLNil(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawHCL(nil)
	assert.Equal(t, nl, buf.String())
}

func TestPrinterRawHCLEmpty(t *testing.T) {
	l, buf := newTestLogger()
	l.Print().RawHCL([]byte{})
	assert.Equal(t, nl, buf.String())
}

func TestPrinterPackageLevel(t *testing.T) {
	old := Default()
	l, buf := newTestLogger()
	SetDefault(l)

	defer func() { SetDefault(old) }()

	Print().RawJSON([]byte(`{"ok":true}`))

	want := `{
  "ok": true
}
`
	assert.Equal(t, want, buf.String())
}
