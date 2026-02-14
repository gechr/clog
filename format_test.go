package clog

import (
	"errors"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
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
			got, kind := formatValue(tt.value, QuoteAuto, 0, 0, "")
			assert.Equal(t, tt.wantStr, got)
			assert.Equal(t, tt.wantKind, kind)
		})
	}
}

func TestFormatValueTimeCustomFormat(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	got, kind := formatValue(ts, QuoteAuto, 0, 0, time.RFC3339)
	assert.Equal(t, "2025-06-15T10:30:00Z", got)
	assert.Equal(t, kindTime, kind)
}

func TestFormatValueTimeEmptyFormat(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	// Empty timeFormat should fall back to time.DateTime.
	got, kind := formatValue(ts, QuoteAuto, 0, 0, "")
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
	) + styles.Values["true"].Render(
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
	assert.Equal(t, keyStyle.Render("42"), styleValue("42", "count", kindNumber, styles))

	// Without key style, number style should apply.
	assert.Equal(t, styles.FieldNumber.Render("42"), styleValue("42", "other", kindNumber, styles))

	// Value style should apply for matching values.
	assert.Equal(
		t,
		styles.Values["true"].Render("true"),
		styleValue("true", "field", kindBool, styles),
	)

	// No style for unrecognised default kind values.
	assert.Empty(t, styleValue("something", "field", kindDefault, styles))

	// No style for slices.
	assert.Empty(t, styleValue("[1, 2]", "field", kindSlice, styles))
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

	want := " " + styles.KeyDefault.Render(
		"vals",
	) + styles.Separator.Render(
		styles.SeparatorText,
	) + "[" + styles.Values["true"].Render(
		"true",
	) + ", other]"
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

	trueStyled := styles.Values["true"].Render("true")
	falseStyled := styles.Values["false"].Render("false")
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
	trueStyled := styles.Values["true"].Render("true")
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

	trueStyled := styles.Values["true"].Render("true")
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
	styles.Values = map[string]*lipgloss.Style{}

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
