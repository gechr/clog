package clog

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		wantStr  string
		wantKind valueKind
	}{
		{
			name:     "string",
			value:    "hello",
			wantStr:  "hello",
			wantKind: kindString,
		},
		{
			name:     "empty_string",
			value:    "",
			wantStr:  "",
			wantKind: kindString,
		},
		{
			name:     "string_slice",
			value:    []string{"a", "b"},
			wantStr:  "[a, b]",
			wantKind: kindSlice,
		},
		{
			name:     "string_slice_quoting",
			value:    []string{"hello world", "ok"},
			wantStr:  `["hello world", ok]`,
			wantKind: kindSlice,
		},
		{
			name:     "empty_string_slice",
			value:    []string{},
			wantStr:  "[]",
			wantKind: kindSlice,
		},
		{
			name:     "single_string_slice",
			value:    []string{"only"},
			wantStr:  "[only]",
			wantKind: kindSlice,
		},
		{
			name:     "int",
			value:    42,
			wantStr:  "42",
			wantKind: kindNumber,
		},
		{
			name:     "int_slice",
			value:    []int{1, 2, 3},
			wantStr:  "[1, 2, 3]",
			wantKind: kindSlice,
		},
		{
			name:     "empty_int_slice",
			value:    []int{},
			wantStr:  "[]",
			wantKind: kindSlice,
		},
		{
			name:     "int64",
			value:    int64(9223372036854775807),
			wantStr:  "9223372036854775807",
			wantKind: kindNumber,
		},
		{
			name:     "uint",
			value:    uint(12345),
			wantStr:  "12345",
			wantKind: kindNumber,
		},
		{
			name:     "uint64",
			value:    uint64(999),
			wantStr:  "999",
			wantKind: kindNumber,
		},
		{
			name:     "uint64_slice",
			value:    []uint64{10, 20, 30},
			wantStr:  "[10, 20, 30]",
			wantKind: kindSlice,
		},
		{
			name:     "empty_uint64_slice",
			value:    []uint64{},
			wantStr:  "[]",
			wantKind: kindSlice,
		},
		{
			name:     "float64",
			value:    3.14,
			wantStr:  "3.14",
			wantKind: kindNumber,
		},
		{
			name:     "bool_true",
			value:    true,
			wantStr:  "true",
			wantKind: kindBool,
		},
		{
			name:     "bool_false",
			value:    false,
			wantStr:  "false",
			wantKind: kindBool,
		},
		{
			name:     "bool_slice",
			value:    []bool{true, false, true},
			wantStr:  "[true, false, true]",
			wantKind: kindSlice,
		},
		{
			name:     "empty_bool_slice",
			value:    []bool{},
			wantStr:  "[]",
			wantKind: kindSlice,
		},
		{
			name:     "float64_slice",
			value:    []float64{1.5, 2.7, 3.14},
			wantStr:  "[1.5, 2.7, 3.14]",
			wantKind: kindSlice,
		},
		{
			name:     "empty_float64_slice",
			value:    []float64{},
			wantStr:  "[]",
			wantKind: kindSlice,
		},
		{
			name:     "any_slice",
			value:    []any{"hello", 42, true},
			wantStr:  "[hello, 42, true]",
			wantKind: kindSlice,
		},
		{
			name:     "empty_any_slice",
			value:    []any{},
			wantStr:  "[]",
			wantKind: kindSlice,
		},
		{
			name:     "any_slice_quoting",
			value:    []any{"hello world", 1},
			wantStr:  `["hello world", 1]`,
			wantKind: kindSlice,
		},
		{
			name:     "duration",
			value:    time.Second,
			wantStr:  "1s",
			wantKind: kindDuration,
		},
		{
			name:     "time",
			value:    time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
			wantStr:  "2025-06-15 10:30:00", // empty timeFormat falls back to time.DateTime
			wantKind: kindTime,
		},
		{
			name:     "error",
			value:    errors.New("boom"),
			wantStr:  "boom",
			wantKind: kindError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, kind := formatValue(tt.value, QuoteAuto, 0, 0, "", 0)
			assert.Equal(t, tt.wantStr, got)
			assert.Equal(t, tt.wantKind, kind)
		})
	}
}

func TestFormatValuePercent(t *testing.T) {
	got, kind := formatValue(percent(75), QuoteAuto, 0, 0, "", 0)
	assert.Equal(t, "75%", got)
	assert.Equal(t, kindPercent, kind)
}

func TestFormatValuePercentDecimal(t *testing.T) {
	got, kind := formatValue(percent(33.333), QuoteAuto, 0, 0, "", 0)
	assert.Equal(t, "33%", got)
	assert.Equal(t, kindPercent, kind)
}

func TestFormatValuePercentPrecision(t *testing.T) {
	got, kind := formatValue(percent(33.333), QuoteAuto, 0, 0, "", 1)
	assert.Equal(t, "33.3%", got)
	assert.Equal(t, kindPercent, kind)

	got, kind = formatValue(percent(33.333), QuoteAuto, 0, 0, "", 2)
	assert.Equal(t, "33.33%", got)
	assert.Equal(t, kindPercent, kind)
}

func TestFormatValueTimeCustomFormat(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	got, kind := formatValue(ts, QuoteAuto, 0, 0, time.RFC3339, 0)
	assert.Equal(t, "2025-06-15T10:30:00Z", got)
	assert.Equal(t, kindTime, kind)
}

func TestFormatValueTimeEmptyFormat(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	// Empty timeFormat should fall back to time.DateTime.
	got, kind := formatValue(ts, QuoteAuto, 0, 0, "", 0)
	assert.Equal(t, "2025-06-15 10:30:00", got)
	assert.Equal(t, kindTime, kind)
}

func TestNeedsQuoting(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want bool
	}{
		{
			name: "simple",
			s:    "hello",
			want: false,
		},
		{
			name: "empty",
			s:    "",
			want: false,
		},
		{
			name: "space",
			s:    "hello world",
			want: true,
		},
		{
			name: "tab",
			s:    "hello\tworld",
			want: true,
		},
		{
			name: "newline",
			s:    "hello\nworld",
			want: true,
		},
		{
			name: "double_quote",
			s:    `say "hi"`,
			want: true,
		},
		{
			name: "equals",
			s:    "a=b",
			want: false,
		},
		{
			name: "ansi_escape",
			s:    "\x1b[31mred\x1b[0m",
			want: false,
		},
		{
			name: "osc8",
			s:    "\x1b]8;;https://example.com\x1b\\text\x1b]8;;\x1b\\",
			// want: false,
		},
		{
			name: "non_printable",
			s:    "hello\x00world",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, needsQuoting(tt.s))
		})
	}
}

func TestFormatFields(t *testing.T) {
	opts := formatFieldsOpts{noColor: true}

	tests := []struct {
		name   string
		fields []Field
		want   string
	}{
		{
			name: "nil", fields: nil, want: "",
		},
		{
			name: "empty", fields: []Field{}, want: "",
		},
		{
			name: "single_string", fields: []Field{{
				Key:   "k",
				Value: "v",
			}}, want: " k=v",
		},
		{
			name: "multiple", fields: []Field{
				{
					Key:   "a",
					Value: "1",
				},
				{
					Key:   "b",
					Value: "2",
				},
			}, want: " a=1 b=2",
		},
		{
			name: "quoted_value",
			fields: []Field{{
				Key:   "msg",
				Value: "hello world",
			}},
			want: ` msg="hello world"`,
		},
		{
			name: "string_slice_comma_separated",
			fields: []Field{{
				Key:   "tags",
				Value: []string{"x", "y"},
			}},
			want: " tags=[x, y]",
		},
		{
			name: "string_slice_per_element_quoting",
			fields: []Field{{
				Key:   "args",
				Value: []string{"simple", "has space", "ok"},
			}},
			want: ` args=[simple, "has space", ok]`,
		},
		{
			name: "int_slice_comma_separated",
			fields: []Field{{
				Key:   "ids",
				Value: []int{1, 2, 3},
			}},
			want: " ids=[1, 2, 3]",
		},
		{
			name: "uint64_slice_comma_separated",
			fields: []Field{{
				Key:   "sizes",
				Value: []uint64{10, 20, 30},
			}},
			want: " sizes=[10, 20, 30]",
		},
		{
			name: "float64_slice_comma_separated",
			fields: []Field{{
				Key:   "temps",
				Value: []float64{36.6, 37.2},
			}},
			want: " temps=[36.6, 37.2]",
		},
		{
			name: "any_slice_comma_separated",
			fields: []Field{{
				Key:   "mixed",
				Value: []any{"a", 1, true},
			}},
			want: " mixed=[a, 1, true]",
		},
		{
			name: "int_value", fields: []Field{{
				Key:   "n",
				Value: 42,
			}}, want: " n=42",
		},
		{
			name: "bool_value", fields: []Field{{
				Key:   "ok",
				Value: true,
			}}, want: " ok=true",
		},
		{
			name: "empty_string_value", fields: []Field{{
				Key:   "k",
				Value: "",
			}}, want: " k=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFields(tt.fields, opts)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatFieldsWithColors(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "k",
		Value: "v",
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"k",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "v"
	assert.Equal(t, want, got)
}

func TestFormatFieldsWithKeyStyles(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles.Keys["path"] = new(keyStyle)

	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "path",
		Value: "/tmp/test",
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"path",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + keyStyle.Render(
		"/tmp/test",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsWithValueStyles(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ok",
		Value: true,
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"ok",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + styles.Values[true].Render(
		"true",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsKeyStyleTakesPriority(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styles.Keys["ok"] = new(keyStyle)

	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ok",
		Value: true,
	}}, opts)

	// Key style wins over value style for "true".
	want := " " + styles.KeyDefault.Render(
		"ok",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + keyStyle.Render(
		"true",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsNumberStyle(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "count",
		Value: 42,
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"count",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + styles.FieldNumber.Render(
		"42",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsNumberStyleNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldNumber = nil

	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "count",
		Value: 42,
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"count",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "42"
	assert.Equal(t, want, got)
}

func TestStyleValuePriority(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles.Keys["count"] = new(keyStyle)

	// Key style should win over number style.
	assert.Equal(t, keyStyle.Render("42"), styleValue("42", 42, "count", kindNumber, styles))

	// Without key style, number style should apply.
	assert.Equal(
		t,
		styles.FieldNumber.Render("42"),
		styleValue("42", 42, "other", kindNumber, styles),
	)

	// Value style should apply for matching values (typed bool key).
	assert.Equal(
		t,
		styles.Values[true].Render("true"),
		styleValue("true", true, "field", kindBool, styles),
	)

	// No style for unrecognised default kind values.
	assert.Empty(t, styleValue("something", "something", "field", kindDefault, styles))

	// No style for slices (styledFieldValue handles slices before calling
	// styleValue, but if it does reach here the slice itself is not styled).
	assert.Empty(t, styleValue("[1, 2]", []int{1, 2}, "field", kindSlice, styles))
}

func TestFormatFieldsIntSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ids",
		Value: []int{1, 2},
	}}, opts)

	n := styles.FieldNumber.Render
	want := " " + styles.KeyDefault.Render(
		"ids",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[" + n(
		"1",
	) + ", " + n(
		"2",
	) + "]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsUint64SliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ids",
		Value: []uint64{10, 20},
	}}, opts)

	n := styles.FieldNumber.Render
	want := " " + styles.KeyDefault.Render(
		"ids",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[" + n(
		"10",
	) + ", " + n(
		"20",
	) + "]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsFloat64SliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "vals",
		Value: []float64{1.5, 2.5},
	}}, opts)

	n := styles.FieldNumber.Render
	want := " " + styles.KeyDefault.Render(
		"vals",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[" + n(
		"1.5",
	) + ", " + n(
		"2.5",
	) + "]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsStringSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "vals",
		Value: []string{"true", "other"},
	}}, opts)

	// String "true" does NOT match bool true in the Values map,
	// so both elements get default FieldString styling.
	s := styles.FieldString.Render
	want := " " + styles.KeyDefault.Render(
		"vals",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[" + s("true") + ", " + s("other") + "]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsSliceKeyStylePriority(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles.Keys["ids"] = new(keyStyle)

	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ids",
		Value: []int{1, 2},
	}}, opts)

	// KeyStyles should style the whole slice value, not per-element.
	want := " " + styles.KeyDefault.Render(
		"ids",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + keyStyle.Render(
		"[1, 2]",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsNumberStyleNilSlice(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldNumber = nil

	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ids",
		Value: []int{1, 2},
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"ids",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[1, 2]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsStylesSkippedBelowInfo(t *testing.T) {
	styles := DefaultStyles()
	styles.Keys["path"] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))

	// At DebugLevel (< InfoLevel), value styles should not be applied.
	opts := formatFieldsOpts{
		noColor: false,
		level:   DebugLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "path",
		Value: "/tmp/test",
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"path",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "/tmp/test"
	assert.Equal(t, want, got)
}

func TestStyledSliceBool(t *testing.T) {
	styles := DefaultStyles()
	got := styledSlice([]bool{true, false}, styles, QuoteAuto, 0, 0)

	trueStyled := styles.Values[true].Render("true")
	falseStyled := styles.Values[false].Render("false")
	want := "[" + trueStyled + ", " + falseStyled + "]"

	assert.Equal(t, want, got)
}

func TestStyledSliceFloat64(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldNumber = nil // disable number styling so output is plain
	got := styledSlice([]float64{1.5, 2.5}, styles, QuoteAuto, 0, 0)

	assert.Equal(t, "[1.5, 2.5]", got)
}

func TestFormatFieldsAnySliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "mixed",
		Value: []any{"hello", 42, true},
	}}, opts)

	n := styles.FieldNumber.Render
	trueStyled := styles.Values[true].Render("true")
	want := " " + styles.KeyDefault.Render(
		"mixed",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[hello, " + n(
		"42",
	) + ", " + trueStyled + "]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsAnySliceKeyStylePriority(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles.Keys["mixed"] = new(keyStyle)

	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "mixed",
		Value: []any{"hello", 42},
	}}, opts)

	// KeyStyles should style the whole slice value, not per-element.
	want := " " + styles.KeyDefault.Render(
		"mixed",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + keyStyle.Render(
		"[hello, 42]",
	)
	assert.Equal(t, want, got)
}

func TestStyledSliceAny(t *testing.T) {
	styles := DefaultStyles()
	got := styledSlice([]any{true, 42, "text"}, styles, QuoteAuto, 0, 0)

	trueStyled := styles.Values[true].Render("true")
	numStyled := styles.FieldNumber.Render("42")
	want := "[" + trueStyled + ", " + numStyled + ", text]"

	assert.Equal(t, want, got)
}

func TestReflectValueKind(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want valueKind
	}{
		{
			name: "nil", val: nil, want: kindDefault,
		},
		{
			name: "int", val: 42, want: kindNumber,
		},
		{
			name: "int64", val: int64(42), want: kindNumber,
		},
		{
			name: "float32", val: float32(1.5), want: kindNumber,
		},
		{
			name: "float64", val: 3.14, want: kindNumber,
		},
		{
			name: "uint", val: uint(10), want: kindNumber,
		},
		{
			name: "uint8", val: uint8(10), want: kindNumber,
		},
		{
			name: "bool", val: true, want: kindBool,
		},
		{
			name: "string", val: "hello", want: kindString,
		},
		{
			name: "error", val: errors.New("fail"), want: kindError,
		},
		{
			name: "slice", val: []int{1}, want: kindDefault,
		},
		{
			name: "struct", val: struct{}{}, want: kindDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, reflectValueKind(tt.val))
		})
	}
}

func TestStyledSliceDefault(t *testing.T) {
	styles := DefaultStyles()
	// Pass an unsupported slice type to exercise the default branch.
	got := styledSlice([]byte{1, 2}, styles, QuoteAuto, 0, 0)

	assert.Equal(t, "[1 2]", got)
}

func TestFormatBoolSliceNoMatchingValueStyle(t *testing.T) {
	styles := DefaultStyles()
	// Remove all value styles so the bool values have no matching style.
	styles.Values = ValueStyleMap{}

	got := formatBoolSlice([]bool{true, false}, styles)

	assert.Equal(t, "[true, false]", got)
}

func TestMergeFields(t *testing.T) {
	tests := []struct {
		name     string
		base     []Field
		over     []Field
		wantKeys []string
		wantVals []any
	}{
		{
			name: "empty_overrides",
			base: []Field{{
				Key:   "a",
				Value: "1",
			}},
			over:     nil,
			wantKeys: []string{"a"},
			wantVals: []any{"1"},
		},
		{
			name: "override_existing",
			base: []Field{{
				Key:   "a",
				Value: "1",
			}, {
				Key:   "b",
				Value: "2",
			}},
			over: []Field{{
				Key:   "a",
				Value: "new",
			}},
			wantKeys: []string{"a", "b"},
			wantVals: []any{"new", "2"},
		},
		{
			name: "add_new",
			base: []Field{{
				Key:   "a",
				Value: "1",
			}},
			over: []Field{{
				Key:   "b",
				Value: "2",
			}},
			wantKeys: []string{"a", "b"},
			wantVals: []any{"1", "2"},
		},
		{
			name: "override_and_add",
			base: []Field{{
				Key:   "a",
				Value: "1",
			}},
			over: []Field{{
				Key:   "a",
				Value: "X",
			}, {
				Key:   "b",
				Value: "Y",
			}},
			wantKeys: []string{"a", "b"},
			wantVals: []any{"X", "Y"},
		},
		{
			name: "empty_base",
			base: nil,
			over: []Field{{
				Key:   "a",
				Value: "1",
			}},
			wantKeys: []string{"a"},
			wantVals: []any{"1"},
		},
		{
			name: "preserves_order",
			base: []Field{
				{
					Key:   "c",
					Value: "3",
				},
				{
					Key:   "a",
					Value: "1",
				},
				{
					Key:   "b",
					Value: "2",
				},
			},
			over: []Field{{
				Key:   "a",
				Value: "new",
			}},
			wantKeys: []string{"c", "a", "b"},
			wantVals: []any{"3", "new", "2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeFields(tt.base, tt.over)
			require.Len(t, got, len(tt.wantKeys))

			for i := range got {
				assert.Equal(t, tt.wantKeys[i], got[i].Key, "field[%d].Key", i)
				assert.Equal(t, tt.wantVals[i], got[i].Value, "field[%d].Value", i)
			}
		})
	}
}

func TestStyleValueDuration(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue("5s", 5*time.Second, "elapsed", kindDuration, styles)

	want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
	assert.Equal(t, want, got)
}

func TestStyleValueDurationNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldDurationNumber = nil
	styles.FieldDurationUnit = nil

	got := styleValue("5s", 5*time.Second, "elapsed", kindDuration, styles)
	assert.Empty(t, got)
}

func TestStyleValueTime(t *testing.T) {
	styles := DefaultStyles()
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	got := styleValue("2025-06-15 10:30:00", ts, "ts", kindTime, styles)
	assert.Equal(t, styles.FieldTime.Render("2025-06-15 10:30:00"), got)
}

func TestStyleValueTimeNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldTime = nil
	got := styleValue(
		"2025-06-15 10:30:00",
		time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		"ts",
		kindTime,
		styles,
	)
	assert.Empty(t, got)
}

func TestStyleValueError(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue("boom", errors.New("boom"), "err", kindError, styles)
	assert.Equal(t, styles.FieldError.Render("boom"), got)
}

func TestStyleValueErrorNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldError = nil
	got := styleValue("boom", errors.New("boom"), "err", kindError, styles)
	assert.Empty(t, got)
}

func TestStyleValuePerKeyMatch(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles.Keys["status"] = new(keyStyle)

	got := styleValue("running", "running", "status", kindString, styles)
	assert.Equal(t, keyStyle.Render("running"), got)
}

func TestStyleValuePerValueMatch(t *testing.T) {
	styles := DefaultStyles()
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.Values["running"] = new(valStyle)

	// No key style set, so value style should apply.
	got := styleValue("running", "running", "status", kindString, styles)
	assert.Equal(t, valStyle.Render("running"), got)
}

func TestStyleAnyElementError(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("boom", errors.New("boom"), kindError, styles)
	assert.Equal(t, styles.FieldError.Render("boom"), got)
}

func TestStyleAnyElementErrorNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldError = nil
	got := styleAnyElement("boom", errors.New("boom"), kindError, styles)
	assert.Empty(t, got)
}

func TestStyleAnyElementDuration(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("5s", 5*time.Second, kindDuration, styles)

	want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
	assert.Equal(t, want, got)
}

func TestStyleAnyElementDurationNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldDurationNumber = nil
	styles.FieldDurationUnit = nil

	got := styleAnyElement("5s", 5*time.Second, kindDuration, styles)
	assert.Empty(t, got)
}

func TestStyleAnyElementTime(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("2025-06-15", "2025-06-15", kindTime, styles)
	assert.Equal(t, styles.FieldTime.Render("2025-06-15"), got)
}

func TestStyleAnyElementTimeNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldTime = nil
	got := styleAnyElement("2025-06-15", "2025-06-15", kindTime, styles)
	assert.Empty(t, got)
}

func TestReflectValueKindBool(t *testing.T) {
	assert.Equal(t, kindBool, reflectValueKind(true))
	assert.Equal(t, kindBool, reflectValueKind(false))
}

func TestQuoteStringOpenCharNoCloseChar(t *testing.T) {
	// When closeChar is 0, openChar should be used for both sides.
	got := quoteString("hello", '\'', 0)
	assert.Equal(t, "'hello'", got)
}

func TestQuoteStringOpenAndCloseChar(t *testing.T) {
	got := quoteString("hello", '(', ')')
	assert.Equal(t, "(hello)", got)
}

func TestQuoteStringDefaultQuoting(t *testing.T) {
	// When openChar is 0, strconv.Quote is used.
	got := quoteString("hello", 0, 0)
	assert.Equal(t, `"hello"`, got)
}

func TestStyleQuantity(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "seconds", input: "5s", want: num("5") + unit("s")},
		{
			name:  "minutes_seconds",
			input: "2m30s",
			want:  num("2") + unit("m") + num("30") + unit("s"),
		},
		{name: "hours_minutes", input: "1h30m", want: num("1") + unit("h") + num("30") + unit("m")},
		{name: "zero", input: "0s", want: num("0") + unit("s")},
		{name: "milliseconds", input: "500ms", want: num("500") + unit("ms")},
		{name: "microseconds", input: "1.5µs", want: num("1.5") + unit("µs")},
		{name: "negative", input: "-1h30m", want: num("-1") + unit("h") + num("30") + unit("m")},
		{name: "weeks_days", input: "1w2d", want: num("1") + unit("w") + num("2") + unit("d")},
		{name: "distance", input: "5.1km", want: num("5.1") + unit("km")},
		{name: "filesize", input: "100MB", want: num("100") + unit("MB")},
		{name: "spaced", input: "5.1 km", want: num("5.1") + " " + unit("km")},
		{name: "spaced_filesize", input: "100 MB", want: num("100") + " " + unit("MB")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := styleQuantity(tt.input, styles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStyleQuantityPartialNil(t *testing.T) {
	styles := DefaultStyles()
	unit := styles.FieldQuantityUnit.Render

	styles.FieldQuantityNumber = nil

	got := styleQuantity("5s", styles)
	assert.Equal(t, "5"+unit("s"), got)
}

func TestFormatValueQuantity(t *testing.T) {
	got, kind := formatValue(quantity("5.1km"), QuoteAuto, 0, 0, "", 0)
	assert.Equal(t, "5.1km", got)
	assert.Equal(t, kindQuantity, kind)
}

func TestIsQuantityString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "simple", input: "5s", want: true},
		{name: "compound", input: "2h30m", want: true},
		{name: "negative", input: "-1h30m", want: true},
		{name: "decimal", input: "1.5µs", want: true},
		{name: "weeks_days", input: "1w2d", want: true},
		{name: "milliseconds", input: "500ms", want: true},
		{name: "zero", input: "0s", want: true},
		{name: "distance", input: "5.1km", want: true},
		{name: "filesize", input: "100MB", want: true},
		{name: "spaced", input: "5 m", want: true},
		{name: "spaced_distance", input: "5.1 km", want: true},
		{name: "spaced_filesize", input: "100 MB", want: true},
		{name: "word", input: "hello", want: false},
		{name: "empty", input: "", want: false},
		{name: "bare_number", input: "42", want: false},
		{name: "bare_unit", input: "ms", want: false},
		{name: "trailing_number", input: "5m2", want: false},
		{name: "just_minus", input: "-", want: false},
		{name: "minus_unit", input: "-m", want: false},
		{name: "only_spaces", input: "   ", want: false},
		{name: "space_then_number", input: " 5m", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isQuantityString(tt.input))
		})
	}
}

func TestStyleValueQuantityFallbackToString(t *testing.T) {
	styles := DefaultStyles()

	// "hello" is not a valid quantity, so styleValue should fall back to FieldString.
	got := styleValue("hello", quantity("hello"), "field", kindQuantity, styles)
	assert.Equal(t, styles.FieldString.Render("hello"), got)
}

func TestStyleValueQuantityFallbackNilString(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldString = nil

	// No quantity match, no string style — should return "".
	got := styleValue("hello", quantity("hello"), "field", kindQuantity, styles)
	assert.Empty(t, got)
}

func TestStyleAnyElementQuantityFallbackToString(t *testing.T) {
	styles := DefaultStyles()

	got := styleAnyElement("hello", quantity("hello"), kindQuantity, styles)
	assert.Equal(t, styles.FieldString.Render("hello"), got)
}

func TestStyleQuantityUnitOverride(t *testing.T) {
	styles := DefaultStyles()
	kmStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityUnits["km"] = new(kmStyle)

	num := styles.FieldQuantityNumber.Render

	got := styleQuantity("5.1km", styles)
	assert.Equal(t, num("5.1")+kmStyle.Render("km"), got)
}

func TestStyleQuantityUnitOverrideCompound(t *testing.T) {
	styles := DefaultStyles()
	hStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityUnits["h"] = new(hStyle)

	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	// "h" gets the override, "m" gets the default.
	got := styleQuantity("2h30m", styles)
	assert.Equal(t, num("2")+hStyle.Render("h")+num("30")+unit("m"), got)
}

func TestStyleQuantityOnlyUnitOverrides(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldQuantityNumber = nil
	styles.FieldQuantityUnit = nil

	kmStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityUnits["km"] = new(kmStyle)

	got := styleQuantity("5km", styles)
	assert.Equal(t, "5"+kmStyle.Render("km"), got)
}

func TestStyleQuantityUnitIgnoreCase(t *testing.T) {
	styles := DefaultStyles()
	mbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles.QuantityUnits["mb"] = new(mbStyle)

	num := styles.FieldQuantityNumber.Render

	// "MB" should match "mb" with case-insensitive lookup (default).
	got := styleQuantity("100MB", styles)
	assert.Equal(t, num("100")+mbStyle.Render("MB"), got)
}

func TestStyleQuantityUnitCaseSensitive(t *testing.T) {
	styles := DefaultStyles()
	styles.QuantityUnitsIgnoreCase = false

	mbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles.QuantityUnits["mb"] = new(mbStyle)

	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	// "MB" should NOT match "mb" when case-sensitive.
	got := styleQuantity("100MB", styles)
	assert.Equal(t, num("100")+unit("MB"), got)
}

func TestFormatDurationSlicePlain(t *testing.T) {
	vals := []time.Duration{5 * time.Second, 2*time.Minute + 30*time.Second}
	got := formatDurationSlice(vals, nil)
	assert.Equal(t, "[5s, 2m30s]", got)
}

func TestFormatDurationSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldDurationNumber.Render
	unit := styles.FieldDurationUnit.Render

	vals := []time.Duration{5 * time.Second, 500 * time.Millisecond}
	got := formatDurationSlice(vals, styles)

	want := "[" +
		num("5") + unit("s") +
		", " +
		num("500") + unit("ms") +
		"]"
	assert.Equal(t, want, got)
}

func TestFormatDurationSliceEmpty(t *testing.T) {
	got := formatDurationSlice([]time.Duration{}, nil)
	assert.Equal(t, "[]", got)
}

func TestFormatFieldsDurationSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "latencies",
		Value: []time.Duration{5 * time.Second, 2 * time.Minute},
	}}, opts)

	num := styles.FieldDurationNumber.Render
	unit := styles.FieldDurationUnit.Render
	want := " " + styles.KeyDefault.Render(
		"latencies",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[" + num("5") + unit("s") +
		", " + num("2") + unit("m") + num("0") + unit("s") + "]"
	assert.Equal(t, want, got)
}

func TestFormatQuantitySlicePlain(t *testing.T) {
	vals := []quantity{"5m", "2h30m", "100 MB"}
	got := formatQuantitySlice(vals, nil)
	assert.Equal(t, "[5m, 2h30m, 100 MB]", got)
}

func TestFormatQuantitySliceStyled(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	vals := []quantity{"5m", "100MB"}
	got := formatQuantitySlice(vals, styles)

	want := "[" +
		num("5") + unit("m") +
		", " +
		num("100") + unit("MB") +
		"]"
	assert.Equal(t, want, got)
}

func TestFormatQuantitySliceEmpty(t *testing.T) {
	got := formatQuantitySlice([]quantity{}, nil)
	assert.Equal(t, "[]", got)
}

func TestFormatFieldsQuantitySliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   InfoLevel,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "rates",
		Value: []quantity{"5m", "10s"},
	}}, opts)

	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render
	want := " " + styles.KeyDefault.Render(
		"rates",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[" + num("5") + unit("m") +
		", " + num("10") + unit("s") + "]"
	assert.Equal(t, want, got)
}

func TestStyleThreshold(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	redUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
	yellowNum := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	yellowUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Faint(true)

	styles.QuantityThresholds["ms"] = []Threshold{
		{Value: 5000, Style: ThresholdStyle{Number: new(redNum), Unit: new(redUnit)}},
		{Value: 1000, Style: ThresholdStyle{Number: new(yellowNum), Unit: new(yellowUnit)}},
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "below_threshold",
			input: "500ms",
			want:  num("500") + styles.FieldQuantityUnit.Render("ms"),
		},
		{
			name:  "at_yellow_threshold",
			input: "1000ms",
			want:  yellowNum.Render("1000") + yellowUnit.Render("ms"),
		},
		{
			name:  "above_yellow_below_red",
			input: "3000ms",
			want:  yellowNum.Render("3000") + yellowUnit.Render("ms"),
		},
		{
			name:  "at_red_threshold",
			input: "5000ms",
			want:  redNum.Render("5000") + redUnit.Render("ms"),
		},
		{
			name:  "above_red_threshold",
			input: "9999ms",
			want:  redNum.Render("9999") + redUnit.Render("ms"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := styleQuantity(tt.input, styles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStyleThresholdCompound(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles.QuantityThresholds["h"] = []Threshold{
		{Value: 10, Style: ThresholdStyle{Number: new(redNum)}},
	}

	// "12h30m" — "h" threshold fires for 12, "m" uses default.
	got := styleQuantity("12h30m", styles)
	assert.Equal(t, redNum.Render("12")+unit("h")+num("30")+unit("m"), got)
}

func TestStyleThresholdNilOverrides(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render

	// Threshold with only Number override (Unit = nil keeps default).
	yellowNum := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityThresholds["s"] = []Threshold{
		{Value: 30, Style: ThresholdStyle{Number: new(yellowNum)}},
	}

	got := styleQuantity("60s", styles)
	assert.Equal(t, yellowNum.Render("60")+styles.FieldQuantityUnit.Render("s"), got)

	// Below threshold — uses default.
	got = styleQuantity("5s", styles)
	assert.Equal(t, num("5")+styles.FieldQuantityUnit.Render("s"), got)
}

func TestStyleDurationThreshold(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldDurationNumber.Render

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	redUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)

	styles.DurationThresholds["s"] = []Threshold{
		{Value: 30, Style: ThresholdStyle{Number: new(redNum), Unit: new(redUnit)}},
	}

	// 45s exceeds 30s threshold.
	got := styleDuration("45s", styles)
	assert.Equal(t, redNum.Render("45")+redUnit.Render("s"), got)

	// 5s does not exceed threshold — uses default.
	got = styleDuration("5s", styles)
	assert.Equal(t, num("5")+styles.FieldDurationUnit.Render("s"), got)
}

func TestStyleThresholdIgnoreCase(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles.QuantityThresholds["mb"] = []Threshold{
		{Value: 500, Style: ThresholdStyle{Number: new(redNum)}},
	}

	// "MB" should match "mb" threshold with case-insensitive matching (default).
	got := styleQuantity("1000MB", styles)
	assert.Equal(t, redNum.Render("1000")+styles.FieldQuantityUnit.Render("MB"), got)

	// Below threshold — uses default number style.
	got = styleQuantity("100MB", styles)
	assert.Equal(t, num("100")+styles.FieldQuantityUnit.Render("MB"), got)
}

func TestStyleThresholdOnlyOverridesEnabled(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldQuantityNumber = nil
	styles.FieldQuantityUnit = nil

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	redUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
	styles.QuantityThresholds["ms"] = []Threshold{
		{Value: 100, Style: ThresholdStyle{Number: new(redNum), Unit: new(redUnit)}},
	}

	// Above threshold — threshold styles apply even with nil defaults.
	got := styleQuantity("500ms", styles)
	assert.Equal(t, redNum.Render("500")+redUnit.Render("ms"), got)

	// Below threshold — no default styles, no threshold match.
	got = styleQuantity("50ms", styles)
	assert.Equal(t, "50ms", got)
}

func TestTypedValuesBoolVsString(t *testing.T) {
	styles := DefaultStyles()

	// bool true has a style in defaults.
	assert.NotNil(t, styles.Values[true], "bool true should have a style")
	assert.NotNil(t, styles.Values[false], "bool false should have a style")

	// string "true" should NOT have a style in defaults.
	assert.Nil(t, styles.Values["true"], "string \"true\" should not have a style")
	assert.Nil(t, styles.Values["false"], "string \"false\" should not have a style")
}

func TestLookupValueStyleNil(t *testing.T) {
	styles := DefaultStyles()

	// Go nil should have a style in defaults.
	assert.NotNil(t, styles.Values[nil], "nil should have a style")

	// lookupValueStyle should find it.
	got := lookupValueStyle(nil, styles.Values)
	assert.NotNil(t, got, "lookupValueStyle should match Go nil")
}

func TestStyleValueNilViaAny(t *testing.T) {
	styles := DefaultStyles()

	// Any("k", nil) -> formatValue returns "<nil>", kindDefault.
	// styleValue should find the nil value style via lookupValueStyle.
	got := styleValue("<nil>", nil, "k", kindDefault, styles)
	assert.NotEmpty(t, got, "nil value should be styled via Values[nil]")
}

func TestStyleValueBoolMatchesTyped(t *testing.T) {
	styles := DefaultStyles()

	// Use distinct styles so we can tell them apart without ANSI colour codes.
	boolStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	strStyle := lipgloss.NewStyle().Italic(true)
	styles.Values[true] = new(boolStyle)
	styles.FieldString = new(strStyle)

	// Bool field true -> styled via typed Values[true].
	got := styleValue("true", true, "ok", kindBool, styles)
	assert.Equal(t, boolStyle.Render("true"), got)

	// String field "true" -> NOT styled via Values (no string "true" key).
	// Should fall through to FieldString styling.
	got = styleValue("true", "true", "ok", kindString, styles)
	assert.Equal(t, strStyle.Render("true"), got)
}

func TestClampPercent(t *testing.T) {
	assert.InDelta(t, 0.0, clampPercent(-10), 0)
	assert.InDelta(t, 0.0, clampPercent(0), 0)
	assert.InDelta(t, 50.0, clampPercent(50), 0)
	assert.InDelta(t, 100.0, clampPercent(100), 0)
	assert.InDelta(t, 100.0, clampPercent(200), 0)
}

func TestClampPercentNaN(t *testing.T) {
	assert.InDelta(t, 0.0, clampPercent(math.NaN()), 0)
}

func TestClampPercentPosInf(t *testing.T) {
	assert.InDelta(t, 100.0, clampPercent(math.Inf(1)), 0)
}

func TestClampPercentNegInf(t *testing.T) {
	assert.InDelta(t, 0.0, clampPercent(math.Inf(-1)), 0)
}

func TestInterpolateGradientEmpty(t *testing.T) {
	c := interpolateGradient(0.5, nil)
	// Empty -> white fallback.
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 1.0, c.G, 0.01)
	assert.InDelta(t, 1.0, c.B, 0.01)
}

func TestInterpolateGradientSingleStop(t *testing.T) {
	red := colorful.Color{R: 1, G: 0, B: 0}
	c := interpolateGradient(0.5, []ColorStop{{Position: 0.5, Color: red}})
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 0.0, c.G, 0.01)
	assert.InDelta(t, 0.0, c.B, 0.01)
}

func TestInterpolateGradientEdges(t *testing.T) {
	stops := DefaultPercentGradient()

	// At 0.0 -> red.
	c := interpolateGradient(0.0, stops)
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 0.0, c.G, 0.1)

	// At 1.0 -> green.
	c = interpolateGradient(1.0, stops)
	assert.InDelta(t, 0.0, c.R, 0.1)
	assert.InDelta(t, 1.0, c.G, 0.01)

	// Below 0.0 -> clamp to red.
	c = interpolateGradient(-0.5, stops)
	assert.InDelta(t, 1.0, c.R, 0.01)

	// Above 1.0 -> clamp to green.
	c = interpolateGradient(1.5, stops)
	assert.InDelta(t, 0.0, c.R, 0.1)
	assert.InDelta(t, 1.0, c.G, 0.01)
}

func TestInterpolateGradientMidpoint(t *testing.T) {
	stops := DefaultPercentGradient()

	// At 0.5 -> yellow (R=1, G=1, B=0).
	c := interpolateGradient(0.5, stops)
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 1.0, c.G, 0.01)
	assert.InDelta(t, 0.0, c.B, 0.1)
}

func TestStylePercentOutput(t *testing.T) {
	styles := DefaultStyles()
	got := stylePercent("75%", percent(75), styles)

	// Should contain ANSI escape codes (color applied).
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "75%")
}

func TestStylePercentNoGradient(t *testing.T) {
	styles := DefaultStyles()
	styles.PercentGradient = nil
	got := stylePercent("50%", percent(50), styles)
	assert.Empty(t, got, "nil gradient should return empty")
}

func TestStylePercentWrongType(t *testing.T) {
	styles := DefaultStyles()
	got := stylePercent("50%", "not a percent", styles)
	assert.Empty(t, got, "non-percent originalValue should return empty")
}

func TestStylePercentSingleStop(t *testing.T) {
	styles := DefaultStyles()
	blue := colorful.Color{R: 0, G: 0, B: 1}
	styles.PercentGradient = []ColorStop{{Position: 0.5, Color: blue}}
	got := stylePercent("50%", percent(50), styles)

	// Should use the single stop's color for any value.
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "50%")
}

func TestStyleValuePercent(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue("75%", percent(75), "progress", kindPercent, styles)
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "75%")
}

func TestStyleValuePercentNilGradient(t *testing.T) {
	styles := DefaultStyles()
	styles.PercentGradient = nil
	got := styleValue("50%", percent(50), "progress", kindPercent, styles)
	assert.Empty(t, got)
}

func TestStylePercentBaseStyle(t *testing.T) {
	styles := DefaultStyles()
	bold := lipgloss.NewStyle().Bold(true)
	styles.FieldPercent = new(bold)

	got := stylePercent("75%", percent(75), styles)
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "75%")
}

func TestStylePercentBaseStyleOnly(t *testing.T) {
	styles := DefaultStyles()
	bold := lipgloss.NewStyle().Bold(true)
	styles.FieldPercent = new(bold)
	styles.PercentGradient = nil // no gradient, base style only

	got := stylePercent("50%", percent(50), styles)
	assert.Equal(t, bold.Render("50%"), got)
}

func TestStyleAnyElementPercent(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("75%", percent(75), kindPercent, styles)
	assert.NotEmpty(t, got)
	assert.Contains(t, got, "75%")
}
