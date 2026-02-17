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

func TestStyleValueDuration(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue("5s", "elapsed", kindDuration, styles)

	want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
	assert.Equal(t, want, got)
}

func TestStyleValueDurationNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldDurationNumber = nil
	styles.FieldDurationUnit = nil

	got := styleValue("5s", "elapsed", kindDuration, styles)
	assert.Empty(t, got)
}

func TestStyleValueTime(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue("2025-06-15 10:30:00", "ts", kindTime, styles)
	assert.Equal(t, styles.FieldTime.Render("2025-06-15 10:30:00"), got)
}

func TestStyleValueTimeNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldTime = nil
	got := styleValue("2025-06-15 10:30:00", "ts", kindTime, styles)
	assert.Empty(t, got)
}

func TestStyleValueError(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue("boom", "err", kindError, styles)
	assert.Equal(t, styles.FieldError.Render("boom"), got)
}

func TestStyleValueErrorNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldError = nil
	got := styleValue("boom", "err", kindError, styles)
	assert.Empty(t, got)
}

func TestStyleValuePerKeyMatch(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles.Keys["status"] = new(keyStyle)

	got := styleValue("running", "status", kindString, styles)
	assert.Equal(t, keyStyle.Render("running"), got)
}

func TestStyleValuePerValueMatch(t *testing.T) {
	styles := DefaultStyles()
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.Values["running"] = new(valStyle)

	// No key style set, so value style should apply.
	got := styleValue("running", "status", kindString, styles)
	assert.Equal(t, valStyle.Render("running"), got)
}

func TestStyleAnyElementError(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("boom", kindError, styles)
	assert.Equal(t, styles.FieldError.Render("boom"), got)
}

func TestStyleAnyElementErrorNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldError = nil
	got := styleAnyElement("boom", kindError, styles)
	assert.Empty(t, got)
}

func TestStyleAnyElementDuration(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("5s", kindDuration, styles)

	want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
	assert.Equal(t, want, got)
}

func TestStyleAnyElementDurationNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldDurationNumber = nil
	styles.FieldDurationUnit = nil

	got := styleAnyElement("5s", kindDuration, styles)
	assert.Empty(t, got)
}

func TestStyleAnyElementTime(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("2025-06-15", kindTime, styles)
	assert.Equal(t, styles.FieldTime.Render("2025-06-15"), got)
}

func TestStyleAnyElementTimeNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldTime = nil
	got := styleAnyElement("2025-06-15", kindTime, styles)
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
	got, kind := formatValue(quantity("5.1km"), QuoteAuto, 0, 0, "")
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
	got := styleValue("hello", "field", kindQuantity, styles)
	assert.Equal(t, styles.FieldString.Render("hello"), got)
}

func TestStyleValueQuantityFallbackNilString(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldString = nil

	// No quantity match, no string style — should return "".
	got := styleValue("hello", "field", kindQuantity, styles)
	assert.Empty(t, got)
}

func TestStyleAnyElementQuantityFallbackToString(t *testing.T) {
	styles := DefaultStyles()

	got := styleAnyElement("hello", kindQuantity, styles)
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
