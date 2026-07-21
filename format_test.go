package clog

import (
	"bytes"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/internal/core"
	"github.com/gechr/clog/style"
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
		{
			name: "raw_json_object",
			value: core.RawJSON(
				`{"status":"unprocessable_entity","detail":"something went wrong"}`,
			),
			wantStr:  `{"status":"unprocessable_entity","detail":"something went wrong"}`,
			wantKind: kindJSON,
		},
		{
			name:     "raw_json_array",
			value:    core.RawJSON(`[1,2,3]`),
			wantStr:  `[1,2,3]`,
			wantKind: kindJSON,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, kind := formatValue(
				tt.value,
				sliceFormat{open: "[", close: "]", sep: ", "},
				QuoteAuto,
				0,
				0,
				nil,
				"",
				&defaultFieldFormats,
			)
			assert.Equal(t, tt.wantStr, got)
			assert.Equal(t, tt.wantKind, kind)
		})
	}
}

func TestFormatValueFraction(t *testing.T) {
	got, kind := formatValue(
		core.Fraction{Current: 7, Total: 10},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
	assert.Equal(t, "7/10", got)
	assert.Equal(t, kindFraction, kind)
}

func TestFormatValueFractionZero(t *testing.T) {
	got, kind := formatValue(
		core.Fraction{Current: 0, Total: 5},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
	assert.Equal(t, "0/5", got)
	assert.Equal(t, kindFraction, kind)
}

func TestFormatValuePercent(t *testing.T) {
	got, kind := formatValue(
		core.Percent{Value: 0.75},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
	assert.Equal(t, "75%", got)
	assert.Equal(t, kindPercent, kind)
}

func TestFormatValuePercentDecimal(t *testing.T) {
	got, kind := formatValue(
		core.Percent{Value: 0.33333},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
	assert.Equal(t, "33%", got)
	assert.Equal(t, kindPercent, kind)
}

func TestFormatValuePercentPrecision(t *testing.T) {
	f := DefaultFieldFormats()
	f.PercentPrecision = 1
	got, kind := formatValue(
		core.Percent{Value: 0.33333},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&f,
	)
	assert.Equal(t, "33.3%", got)
	assert.Equal(t, kindPercent, kind)

	f.PercentPrecision = 2
	got, kind = formatValue(
		core.Percent{Value: 0.33333},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&f,
	)
	assert.Equal(t, "33.33%", got)
	assert.Equal(t, kindPercent, kind)
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		name      string
		dur       time.Duration
		precision int
		want      string
	}{
		{"zero", 0, 1, "0s"},
		{"nanoseconds", 500 * time.Nanosecond, 1, "500.0ns"},
		{"microseconds", 1500 * time.Nanosecond, 1, "1.5µs"},
		{"milliseconds", 42 * time.Millisecond, 1, "42.0ms"},
		{"milliseconds_fractional", 1500 * time.Microsecond, 1, "1.5ms"},
		{"seconds", 3200 * time.Millisecond, 1, "3.2s"},
		{"seconds_whole", 5 * time.Second, 1, "5.0s"},
		{"minutes_composite", 90 * time.Second, 0, "1m30s"},
		{"hours_composite", 2*time.Hour + 30*time.Minute, 0, "2h30m"},
		{"precision_0", 3200 * time.Millisecond, 0, "3s"},
		{"precision_2", 3210 * time.Millisecond, 2, "3.21s"},
		{"negative", -3200 * time.Millisecond, 1, "3.2s"},
		{"no_trim", 3*time.Second + 100*time.Millisecond, 2, "3.10s"},
		{"61s", 61 * time.Second, 0, "1m1s"},
		{"60s", 60 * time.Second, 0, "1m"},
		{"3600s", 3600 * time.Second, 0, "1h"},
		{"3661s", 3661 * time.Second, 0, "1h1m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDurationValueOptions(tt.dur, tt.precision, false)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatValueElapsed(t *testing.T) {
	// Default precision 0 → no decimal places.
	got, kind := formatValue(
		core.ElapsedField{Value: 3200 * time.Millisecond},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
	assert.Equal(t, "3s", got)
	assert.Equal(t, kindElapsed, kind)

	// Precision 1 → one decimal place, no trimming.
	f := DefaultFieldFormats()
	f.ElapsedPrecision = 1
	got, kind = formatValue(
		core.ElapsedField{Value: 3200 * time.Millisecond},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&f,
	)
	assert.Equal(t, "3.2s", got)
	assert.Equal(t, kindElapsed, kind)
}

func TestFormatValueElapsedPrecision(t *testing.T) {
	got, kind := formatValue(
		core.ElapsedField{Value: 3210 * time.Millisecond},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
	assert.Equal(t, "3s", got)
	assert.Equal(t, kindElapsed, kind)

	f := DefaultFieldFormats()
	f.ElapsedPrecision = 2
	got, kind = formatValue(
		core.ElapsedField{Value: 3210 * time.Millisecond},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&f,
	)
	assert.Equal(t, "3.21s", got)
	assert.Equal(t, kindElapsed, kind)
}

func TestFormatValueTimeCustomFormat(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	got, kind := formatValue(
		ts,
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		time.RFC3339,
		&defaultFieldFormats,
	)
	assert.Equal(t, "2025-06-15T10:30:00Z", got)
	assert.Equal(t, kindTime, kind)
}

func TestFormatValueTimeEmptyFormat(t *testing.T) {
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	// Empty timeFormat should fall back to time.DateTime.
	got, kind := formatValue(
		ts,
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
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
		{
			name: "raw_json_not_quoted",
			fields: []Field{
				{
					Key: "error",
					Value: core.RawJSON(
						`{"status":"unprocessable_entity","detail":"something went wrong"}`,
					),
				},
			},
			want: ` error={"status":"unprocessable_entity","detail":"something went wrong"}`,
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
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "k",
		Value: "v",
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"k",
	) + styles.Separator.Render(
		"=",
	) + styles.FieldString.Render("v")
	assert.Equal(t, want, got)
}

func TestFormatFieldsHighlightsBacktickValue(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{Key: "k", Value: "x`y`z"}}, opts)
	want := " " + styles.KeyDefault.Render("k") +
		styles.Separator.Render("=") +
		styles.FieldString.Render("x") +
		styles.Backtick.Render("y") +
		styles.FieldString.Render("z")
	assert.Equal(t, want, got)
}

func TestFormatFieldsBacktickKeepModeRetainsDelimiters(t *testing.T) {
	styles := DefaultStyles()
	styles.BacktickMode = style.BacktickKeep
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{Key: "k", Value: "x`y`z"}}, opts)
	want := " " + styles.KeyDefault.Render("k") +
		styles.Separator.Render("=") +
		styles.FieldString.Render("x") +
		styles.Backtick.Render("`y`") +
		styles.FieldString.Render("z")
	assert.Equal(t, want, got)
}

func TestFormatFieldsErrorValueNotBacktickStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{Key: "err", Value: errors.New("bad`x`")}}, opts)
	// An error value keeps its own style verbatim: the backtick is content, so it
	// is neither restyled nor stripped.
	want := " " + styles.KeyDefault.Render("err") +
		styles.Separator.Render("=") +
		styles.FieldError.Render("bad`x`")
	assert.Equal(t, want, got)
}

func TestFormatFieldsWithKeyStyles(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles.Keys["path"] = new(keyStyle)

	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "path",
		Value: "/tmp/test",
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"path",
	) + styles.Separator.Render(
		"=",
	) + keyStyle.Render(
		"/tmp/test",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsWithValueStyles(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ok",
		Value: true,
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"ok",
	) + styles.Separator.Render(
		"=",
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
		level:   LevelInfo,
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
		"=",
	) + keyStyle.Render(
		"true",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsKeyValueMatch(t *testing.T) {
	styles := DefaultStyles()
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles.KeyValues["status"] = style.KeyValue{
		Values: style.ValueMap{"active": new(activeStyle)},
	}

	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "status",
		Value: "active",
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"status",
	) + styles.Separator.Render(
		"=",
	) + activeStyle.Render(
		"active",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsKeyValueDefault(t *testing.T) {
	styles := DefaultStyles()
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	defaultStyle := lipgloss.NewStyle().Faint(true)
	styles.KeyValues["status"] = style.KeyValue{
		Values:  style.ValueMap{"active": new(activeStyle)},
		Default: new(defaultStyle),
	}

	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "status",
		Value: "queued",
	}}, opts)

	// Unlisted value falls back to the entry's Default.
	want := " " + styles.KeyDefault.Render(
		"status",
	) + styles.Separator.Render(
		"=",
	) + defaultStyle.Render(
		"queued",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsKeyValuePlainWhenNoDefault(t *testing.T) {
	styles := DefaultStyles()
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	// String type style would normally apply; the governing entry suppresses it.
	styles.FieldString = new(lipgloss.NewStyle().Foreground(lipgloss.Color("15")))
	styles.Keys["status"] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("5")))
	styles.KeyValues["status"] = style.KeyValue{
		Values: style.ValueMap{"active": new(activeStyle)},
	}

	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "status",
		Value: "queued",
	}}, opts)

	// No match and no Default: value renders plain - no fall-through to Keys,
	// Values or the type style.
	want := " " + styles.KeyDefault.Render(
		"status",
	) + styles.Separator.Render(
		"=",
	) + "queued"
	assert.Equal(t, want, got)
}

func TestFormatFieldsKeyValueTakesPriorityOverKey(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles.Keys["status"] = new(keyStyle)
	styles.KeyValues["status"] = style.KeyValue{
		Values: style.ValueMap{"active": new(activeStyle)},
	}

	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "status",
		Value: "active",
	}}, opts)

	// KeyValues entry wins over the Keys style for the same key.
	want := " " + styles.KeyDefault.Render(
		"status",
	) + styles.Separator.Render(
		"=",
	) + activeStyle.Render(
		"active",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsKeyValueScopedToKey(t *testing.T) {
	styles := DefaultStyles()
	activeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	stringStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	styles.FieldString = new(stringStyle)
	styles.KeyValues["status"] = style.KeyValue{
		Values: style.ValueMap{"active": new(activeStyle)},
	}

	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "state",
		Value: "active",
	}}, opts)

	// A different key with the same value is unaffected by the entry.
	want := " " + styles.KeyDefault.Render(
		"state",
	) + styles.Separator.Render(
		"=",
	) + stringStyle.Render(
		"active",
	)
	assert.Equal(t, want, got)
}

func TestFormatFieldsNumberStyle(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "count",
		Value: 42,
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"count",
	) + styles.Separator.Render(
		"=",
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
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "count",
		Value: 42,
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"count",
	) + styles.Separator.Render(
		"=",
	) + "42"
	assert.Equal(t, want, got)
}

func TestStyleValuePriority(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles.Keys["count"] = new(keyStyle)

	// Key style should win over number style.
	assert.Equal(
		t,
		keyStyle.Render("42"),
		styleValue("42", 42, "count", kindNumber, styles, &defaultFieldFormats),
	)

	// Without key style, number style should apply.
	assert.Equal(
		t,
		styles.FieldNumber.Render("42"),
		styleValue("42", 42, "other", kindNumber, styles, &defaultFieldFormats),
	)

	// Value style should apply for matching values (typed bool key).
	assert.Equal(
		t,
		styles.Values[true].Render("true"),
		styleValue("true", true, "field", kindBool, styles, &defaultFieldFormats),
	)

	// No style for unrecognised default kind values.
	assert.Empty(
		t,
		styleValue("something", "something", "field", kindDefault, styles, &defaultFieldFormats),
	)

	// No style for slices (styledFieldValue handles slices before calling
	// styleValue, but if it does reach here the slice itself is not styled).
	assert.Empty(
		t,
		styleValue("[1, 2]", []int{1, 2}, "field", kindSlice, styles, &defaultFieldFormats),
	)
}

func TestFormatFieldsIntSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
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
		"=",
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
		level:   LevelInfo,
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
		"=",
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
		level:   LevelInfo,
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
		"=",
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
		level:   LevelInfo,
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
		"=",
	) + "[" + s("true") + ", " + s("other") + "]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsSliceKeyStylePriority(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles.Keys["ids"] = new(keyStyle)

	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
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
		"=",
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
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "ids",
		Value: []int{1, 2},
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"ids",
	) + styles.Separator.Render(
		"=",
	) + "[1, 2]"
	assert.Equal(t, want, got)
}

func TestFormatFieldsStylesSkippedBelowInfo(t *testing.T) {
	styles := DefaultStyles()
	styles.Keys["path"] = new(lipgloss.NewStyle().Foreground(lipgloss.Color("4")))

	// At LevelDebug (< LevelInfo), value styles should not be applied.
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelDebug,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "path",
		Value: "/tmp/test",
	}}, opts)

	want := " " + styles.KeyDefault.Render(
		"path",
	) + styles.Separator.Render(
		"=",
	) + "/tmp/test"
	assert.Equal(t, want, got)
}

func TestStyledSliceBool(t *testing.T) {
	styles := DefaultStyles()
	got := styledSlice(
		[]bool{true, false},
		sliceFormat{open: "[", close: "]", sep: ", "},
		styles,
		QuoteAuto,
		0,
		0,
		nil,
		&defaultFieldFormats,
	)

	trueStyled := styles.Values[true].Render("true")
	falseStyled := styles.Values[false].Render("false")
	want := "[" + trueStyled + ", " + falseStyled + "]"

	assert.Equal(t, want, got)
}

func TestStyledSliceFloat64(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldNumber = nil // disable number styling so output is plain
	got := styledSlice(
		[]float64{1.5, 2.5},
		sliceFormat{open: "[", close: "]", sep: ", "},
		styles,
		QuoteAuto,
		0,
		0,
		nil,
		&defaultFieldFormats,
	)

	assert.Equal(t, "[1.5, 2.5]", got)
}

func TestFormatFieldsAnySliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
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
		"=",
	) + "[" + styles.FieldString.Render("hello") + ", " + n(
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
		level:   LevelInfo,
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
		"=",
	) + keyStyle.Render(
		"[hello, 42]",
	)
	assert.Equal(t, want, got)
}

func TestStyledSliceAny(t *testing.T) {
	styles := DefaultStyles()
	got := styledSlice(
		[]any{true, 42, "text"},
		sliceFormat{open: "[", close: "]", sep: ", "},
		styles,
		QuoteAuto,
		0,
		0,
		nil,
		&defaultFieldFormats,
	)

	trueStyled := styles.Values[true].Render("true")
	numStyled := styles.FieldNumber.Render("42")
	want := "[" + trueStyled + ", " + numStyled + ", " + styles.FieldString.Render("text") + "]"

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
	got := styledSlice(
		[]byte{1, 2},
		sliceFormat{open: "[", close: "]", sep: ", "},
		styles,
		QuoteAuto,
		0,
		0,
		nil,
		&defaultFieldFormats,
	)

	assert.Equal(t, "[1 2]", got)
}

func TestFormatBoolSliceNoMatchingValueStyle(t *testing.T) {
	styles := DefaultStyles()
	// Remove all value styles so the bool values have no matching style.
	styles.Values = style.ValueMap{}

	got := formatBoolSlice(
		[]bool{true, false},
		sliceFormat{open: "[", close: "]", sep: ", "},
		styles,
	)

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
			got := core.MergeFields(tt.base, tt.over)
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
	got := styleValue("5s", 5*time.Second, "elapsed", kindDuration, styles, &defaultFieldFormats)

	want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
	assert.Equal(t, want, got)
}

func TestStyleValueDurationNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldDurationNumber = nil
	styles.FieldDurationUnit = nil

	got := styleValue("5s", 5*time.Second, "elapsed", kindDuration, styles, &defaultFieldFormats)
	assert.Empty(t, got)
}

func TestStyleValueTime(t *testing.T) {
	styles := DefaultStyles()
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	got := styleValue("2025-06-15 10:30:00", ts, "ts", kindTime, styles, &defaultFieldFormats)
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
		&defaultFieldFormats,
	)
	assert.Empty(t, got)
}

func TestStyleValueError(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue("boom", errors.New("boom"), "err", kindError, styles, &defaultFieldFormats)
	assert.Equal(t, styles.FieldError.Render("boom"), got)
}

func TestStyleValueErrorNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldError = nil
	got := styleValue("boom", errors.New("boom"), "err", kindError, styles, &defaultFieldFormats)
	assert.Empty(t, got)
}

func TestStyleValuePerKeyMatch(t *testing.T) {
	styles := DefaultStyles()
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles.Keys["status"] = new(keyStyle)

	got := styleValue("running", "running", "status", kindString, styles, &defaultFieldFormats)
	assert.Equal(t, keyStyle.Render("running"), got)
}

func TestStyleValuePerValueMatch(t *testing.T) {
	styles := DefaultStyles()
	valStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.Values["running"] = new(valStyle)

	// No key style set, so value style should apply.
	got := styleValue("running", "running", "status", kindString, styles, &defaultFieldFormats)
	assert.Equal(t, valStyle.Render("running"), got)
}

func TestStyleAnyElementError(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("boom", errors.New("boom"), kindError, styles, &defaultFieldFormats)
	assert.Equal(t, styles.FieldError.Render("boom"), got)
}

func TestStyleAnyElementErrorNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldError = nil
	got := styleAnyElement("boom", errors.New("boom"), kindError, styles, &defaultFieldFormats)
	assert.Empty(t, got)
}

func TestStyleAnyElementDuration(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("5s", 5*time.Second, kindDuration, styles, &defaultFieldFormats)

	want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
	assert.Equal(t, want, got)
}

func TestStyleAnyElementDurationNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldDurationNumber = nil
	styles.FieldDurationUnit = nil

	got := styleAnyElement("5s", 5*time.Second, kindDuration, styles, &defaultFieldFormats)
	assert.Empty(t, got)
}

func TestStyleAnyElementTime(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement("2025-06-15", "2025-06-15", kindTime, styles, &defaultFieldFormats)
	assert.Equal(t, styles.FieldTime.Render("2025-06-15"), got)
}

func TestStyleAnyElementTimeNil(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldTime = nil
	got := styleAnyElement("2025-06-15", "2025-06-15", kindTime, styles, &defaultFieldFormats)
	assert.Empty(t, got)
}

func TestReflectValueKindBool(t *testing.T) {
	assert.Equal(t, kindBool, reflectValueKind(true))
	assert.Equal(t, kindBool, reflectValueKind(false))
}

func TestQuoteStringOpenCharNoCloseChar(t *testing.T) {
	// When closeChar is 0, openChar should be used for both sides.
	got := quoteString("hello", '\'', 0, nil)
	assert.Equal(t, "'hello'", got)
}

func TestQuoteStringOpenAndCloseChar(t *testing.T) {
	got := quoteString("hello", '(', ')', nil)
	assert.Equal(t, "(hello)", got)
}

func TestQuoteStringDefaultQuoting(t *testing.T) {
	// When openChar is 0, strconv.Quote is used.
	got := quoteString("hello", 0, 0, nil)
	assert.Equal(t, `"hello"`, got)
}

func TestQuoteStringSmartTakesPrecedence(t *testing.T) {
	// A non-empty smart list overrides open/close runes.
	got := quoteString(`a"b`, '(', ')', defaultSmartQuoteChars)
	assert.Equal(t, `'a"b'`, got)
}

func TestSmartQuoteEscalation(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "no_quotes_uses_double", in: "hello world", want: `"hello world"`},
		{name: "double_quote_uses_single", in: `say "hi"`, want: `'say "hi"'`},
		{name: "both_uses_backtick", in: `it's a "test"`, want: "`it's a \"test\"`"},
		{name: "all_three_escapes", in: "a\"b'c`d", want: "\"a\\\"b'c`d\""},
		{name: "backslash_escapes", in: `a\b`, want: `"a\\b"`},
		{name: "newline_escapes", in: "a\nb", want: `"a\nb"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, quoteString(tt.in, 0, 0, defaultSmartQuoteChars))
		})
	}
}

func TestSmartQuoteCustomPairsDistinctDelimiters(t *testing.T) {
	pairs := []QuotePair{{Open: '«', Close: '»'}, {Open: '[', Close: ']'}}
	// First pair fits.
	assert.Equal(t, "«hi there»", quoteString("hi there", 0, 0, pairs))
	// Value contains the close delimiter of the first pair, so fall through.
	assert.Equal(t, "[a»b]", quoteString("a»b", 0, 0, pairs))
	// Value collides with both pairs -> escaped fallback.
	assert.Equal(t, `"a»b]c"`, quoteString("a»b]c", 0, 0, pairs))
}

func TestFormatFieldsSmartQuotes(t *testing.T) {
	opts := formatFieldsOpts{noColor: true, quoteSmart: defaultSmartQuoteChars}
	got := formatFields([]Field{{Key: "k", Value: `value with "quotes"`}}, opts)
	assert.Equal(t, ` k='value with "quotes"'`, got)
}

func TestFormatFieldsSmartQuotesAllDelimitersFallBack(t *testing.T) {
	// Value contains all three default delimiters (" ' `), so no bare wrap
	// fits and it falls back to Go-style escaped quoting.
	const value = "a \"b\" 'c' `d`"
	opts := formatFieldsOpts{noColor: true, quoteSmart: defaultSmartQuoteChars}
	got := formatFields([]Field{{Key: "k", Value: value}}, opts)
	assert.Equal(t, " k=\"a \\\"b\\\" 'c' `d`\"", got)
}

func TestSetSmartQuotesEndToEnd(t *testing.T) {
	var buf strings.Builder
	l := New(NewOutput(&buf, ColorNever))
	l.SetParts(PartFields)
	l.SetSmartQuotes(true)
	l.SetSmartQuoteChars(QuotePair{Open: '«', Close: '»'})

	l.Info().Str("k", "needs quoting").Msg("")
	assert.Equal(t, "k=«needs quoting»", strings.TrimSpace(buf.String()))
}

func TestFormatFieldsQuoteDelimiterStyle(t *testing.T) {
	styles := DefaultStyles()
	quote := lipgloss.NewStyle().Bold(true)
	styles.FieldQuote = &style.QuoteStyle{Style: quote}
	styles.FieldString = nil // isolate the body so only delimiters carry style

	opts := formatFieldsOpts{level: LevelInfo, styles: styles}
	got := formatFields([]Field{{Key: "k", Value: "hello world"}}, opts)

	want := " " + styles.KeyDefault.Render("k") + styles.Separator.Render("=") +
		quote.Render(`"`) + "hello world" + quote.Render(`"`)
	assert.Equal(t, want, got)
}

func TestFormatFieldsQuoteDelimiterStyleInherit(t *testing.T) {
	styles := DefaultStyles()
	valueColor := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles.FieldString = &valueColor
	// Inherit: delimiters keep the value's color but add bold.
	styles.FieldQuote = &style.QuoteStyle{Style: lipgloss.NewStyle().Bold(true), Inherit: true}

	opts := formatFieldsOpts{level: LevelInfo, styles: styles}
	got := formatFields([]Field{{Key: "k", Value: "hello world"}}, opts)

	delim := lipgloss.NewStyle().Bold(true).Inherit(valueColor)
	want := " " + styles.KeyDefault.Render("k") + styles.Separator.Render("=") +
		delim.Render(`"`) + valueColor.Render("hello world") + delim.Render(`"`)
	assert.Equal(t, want, got)
}

func TestFormatFieldsQuoteDelimiterStyleNilIsLegacy(t *testing.T) {
	// With no FieldQuote style, the whole quoted value is styled as one unit.
	styles := DefaultStyles()
	styles.FieldQuote = nil

	opts := formatFieldsOpts{level: LevelInfo, styles: styles}
	got := formatFields([]Field{{Key: "k", Value: "hello world"}}, opts)

	want := " " + styles.KeyDefault.Render("k") + styles.Separator.Render("=") +
		styles.FieldString.Render(`"hello world"`)
	assert.Equal(t, want, got)
}

func TestFormatStringSliceQuoteDelimiterStyle(t *testing.T) {
	styles := DefaultStyles()
	quote := lipgloss.NewStyle().Bold(true)
	styles.FieldQuote = &style.QuoteStyle{Style: quote}
	styles.FieldString = nil

	got := formatStringSlice(
		[]string{"a b"},
		sliceFormat{open: "[", close: "]", sep: ", "},
		styles,
		QuoteAuto,
		0,
		0,
		nil,
	)
	want := "[" + quote.Render(`"`) + "a b" + quote.Render(`"`) + "]"
	assert.Equal(t, want, got)
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
			got := styleQuantity(tt.input, styles, true)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStyleQuantityPartialNil(t *testing.T) {
	styles := DefaultStyles()
	unit := styles.FieldQuantityUnit.Render

	styles.FieldQuantityNumber = nil

	got := styleQuantity("5s", styles, true)
	assert.Equal(t, "5"+unit("s"), got)
}

func TestFormatValueQuantity(t *testing.T) {
	got, kind := formatValue(
		core.QuantityField("5.1km"),
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
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
	got := styleValue(
		"hello",
		core.QuantityField("hello"),
		"field",
		kindQuantity,
		styles,
		&defaultFieldFormats,
	)
	assert.Equal(t, styles.FieldString.Render("hello"), got)
}

func TestStyleValueQuantityFallbackNilString(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldString = nil

	// No quantity match, no string style - should return "".
	got := styleValue(
		"hello",
		core.QuantityField("hello"),
		"field",
		kindQuantity,
		styles,
		&defaultFieldFormats,
	)
	assert.Empty(t, got)
}

func TestStyleAnyElementQuantityFallbackToString(t *testing.T) {
	styles := DefaultStyles()

	got := styleAnyElement(
		"hello",
		core.QuantityField("hello"),
		kindQuantity,
		styles,
		&defaultFieldFormats,
	)
	assert.Equal(t, styles.FieldString.Render("hello"), got)
}

func TestStyleQuantityUnitOverride(t *testing.T) {
	styles := DefaultStyles()
	kmStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityUnits["km"] = new(kmStyle)

	num := styles.FieldQuantityNumber.Render

	got := styleQuantity("5.1km", styles, true)
	assert.Equal(t, num("5.1")+kmStyle.Render("km"), got)
}

func TestStyleQuantityUnitOverrideCompound(t *testing.T) {
	styles := DefaultStyles()
	hStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityUnits["h"] = new(hStyle)

	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	// "h" gets the override, "m" gets the default.
	got := styleQuantity("2h30m", styles, true)
	assert.Equal(t, num("2")+hStyle.Render("h")+num("30")+unit("m"), got)
}

func TestStyleQuantityOnlyUnitOverrides(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldQuantityNumber = nil
	styles.FieldQuantityUnit = nil

	kmStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityUnits["km"] = new(kmStyle)

	got := styleQuantity("5km", styles, true)
	assert.Equal(t, "5"+kmStyle.Render("km"), got)
}

func TestStyleQuantityUnitIgnoreCase(t *testing.T) {
	styles := DefaultStyles()
	mbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles.QuantityUnits["mb"] = new(mbStyle)

	num := styles.FieldQuantityNumber.Render

	// "MB" should match "mb" with case-insensitive lookup (default).
	got := styleQuantity("100MB", styles, true)
	assert.Equal(t, num("100")+mbStyle.Render("MB"), got)
}

func TestStyleQuantityUnitCaseSensitive(t *testing.T) {
	styles := DefaultStyles()

	mbStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles.QuantityUnits["mb"] = new(mbStyle)

	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	// "MB" should NOT match "mb" when case-sensitive.
	got := styleQuantity("100MB", styles, false)
	assert.Equal(t, num("100")+unit("MB"), got)
}

func TestFormatDurationSlicePlain(t *testing.T) {
	vals := []time.Duration{5 * time.Second, 2*time.Minute + 30*time.Second}
	got := formatDurationSlice(
		vals,
		sliceFormat{open: "[", close: "]", sep: ", "},
		nil,
		&defaultFieldFormats,
	)
	assert.Equal(t, "[5s, 2m30s]", got)
}

func TestFormatDurationSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldDurationNumber.Render
	unit := styles.FieldDurationUnit.Render

	vals := []time.Duration{5 * time.Second, 500 * time.Millisecond}
	got := formatDurationSlice(
		vals,
		sliceFormat{open: "[", close: "]", sep: ", "},
		styles,
		&defaultFieldFormats,
	)

	want := "[" +
		num("5") + unit("s") +
		", " +
		num("500") + unit("ms") +
		"]"
	assert.Equal(t, want, got)
}

func TestFormatDurationSliceEmpty(t *testing.T) {
	got := formatDurationSlice(
		[]time.Duration{},
		sliceFormat{open: "[", close: "]", sep: ", "},
		nil,
		&defaultFieldFormats,
	)
	assert.Equal(t, "[]", got)
}

func TestFormatFieldsDurationSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
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
		"=",
	) + "[" + num("5") + unit("s") +
		", " + num("2") + unit("m") + num("0") + unit("s") + "]"
	assert.Equal(t, want, got)
}

func TestFormatQuantitySlicePlain(t *testing.T) {
	vals := []core.QuantityField{"5m", "2h30m", "100 MB"}
	got := formatQuantitySlice(vals, sliceFormat{open: "[", close: "]", sep: ", "}, nil, true)
	assert.Equal(t, "[5m, 2h30m, 100 MB]", got)
}

func TestFormatQuantitySliceStyled(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	vals := []core.QuantityField{"5m", "100MB"}
	got := formatQuantitySlice(vals, sliceFormat{open: "[", close: "]", sep: ", "}, styles, true)

	want := "[" +
		num("5") + unit("m") +
		", " +
		num("100") + unit("MB") +
		"]"
	assert.Equal(t, want, got)
}

func TestFormatQuantitySliceEmpty(t *testing.T) {
	got := formatQuantitySlice(
		[]core.QuantityField{},
		sliceFormat{open: "[", close: "]", sep: ", "},
		nil,
		true,
	)
	assert.Equal(t, "[]", got)
}

func TestFormatFieldsQuantitySliceStyled(t *testing.T) {
	styles := DefaultStyles()
	opts := formatFieldsOpts{
		noColor: false,
		level:   LevelInfo,
		styles:  styles,
	}

	got := formatFields([]Field{{
		Key:   "rates",
		Value: []core.QuantityField{"5m", "10s"},
	}}, opts)

	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render
	want := " " + styles.KeyDefault.Render(
		"rates",
	) + styles.Separator.Render(
		"=",
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

	styles.QuantityThresholds["ms"] = []style.Threshold{
		{Value: 5000, Style: style.ThresholdStyle{Number: new(redNum), Unit: new(redUnit)}},
		{Value: 1000, Style: style.ThresholdStyle{Number: new(yellowNum), Unit: new(yellowUnit)}},
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
			got := styleQuantity(tt.input, styles, true)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestStyleThresholdCompound(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render
	unit := styles.FieldQuantityUnit.Render

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles.QuantityThresholds["h"] = []style.Threshold{
		{Value: 10, Style: style.ThresholdStyle{Number: new(redNum)}},
	}

	// "12h30m" - "h" threshold fires for 12, "m" uses default.
	got := styleQuantity("12h30m", styles, true)
	assert.Equal(t, redNum.Render("12")+unit("h")+num("30")+unit("m"), got)
}

func TestStyleThresholdNilOverrides(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render

	// Threshold with only Number override (Unit = nil keeps default).
	yellowNum := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles.QuantityThresholds["s"] = []style.Threshold{
		{Value: 30, Style: style.ThresholdStyle{Number: new(yellowNum)}},
	}

	got := styleQuantity("60s", styles, true)
	assert.Equal(t, yellowNum.Render("60")+styles.FieldQuantityUnit.Render("s"), got)

	// Below threshold - uses default.
	got = styleQuantity("5s", styles, true)
	assert.Equal(t, num("5")+styles.FieldQuantityUnit.Render("s"), got)
}

func TestStyleDurationThreshold(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldDurationNumber.Render

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	redUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)

	styles.DurationThresholds["s"] = []style.Threshold{
		{Value: 30, Style: style.ThresholdStyle{Number: new(redNum), Unit: new(redUnit)}},
	}

	// 45s exceeds 30s threshold.
	got := styleDuration("45s", time.Duration(0), styles, 0)
	assert.Equal(t, redNum.Render("45")+redUnit.Render("s"), got)

	// 5s does not exceed threshold - uses default.
	got = styleDuration("5s", time.Duration(0), styles, 0)
	assert.Equal(t, num("5")+styles.FieldDurationUnit.Render("s"), got)
}

func TestStyleThresholdIgnoreCase(t *testing.T) {
	styles := DefaultStyles()
	num := styles.FieldQuantityNumber.Render

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles.QuantityThresholds["mb"] = []style.Threshold{
		{Value: 500, Style: style.ThresholdStyle{Number: new(redNum)}},
	}

	// "MB" should match "mb" threshold with case-insensitive matching (default).
	got := styleQuantity("1000MB", styles, true)
	assert.Equal(t, redNum.Render("1000")+styles.FieldQuantityUnit.Render("MB"), got)

	// Below threshold - uses default number style.
	got = styleQuantity("100MB", styles, true)
	assert.Equal(t, num("100")+styles.FieldQuantityUnit.Render("MB"), got)
}

func TestStyleThresholdOnlyOverridesEnabled(t *testing.T) {
	styles := DefaultStyles()
	styles.FieldQuantityNumber = nil
	styles.FieldQuantityUnit = nil

	redNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	redUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Faint(true)
	styles.QuantityThresholds["ms"] = []style.Threshold{
		{Value: 100, Style: style.ThresholdStyle{Number: new(redNum), Unit: new(redUnit)}},
	}

	// Above threshold - threshold styles apply even with nil defaults.
	got := styleQuantity("500ms", styles, true)
	assert.Equal(t, redNum.Render("500")+redUnit.Render("ms"), got)

	// Below threshold - no default styles, no threshold match.
	got = styleQuantity("50ms", styles, true)
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
	got := styleValue("<nil>", nil, "k", kindDefault, styles, &defaultFieldFormats)
	assert.NotEmpty(t, got, "nil value should be styled via Values[nil]")
}

func TestStyleValueBoolMatchesTyped(t *testing.T) {
	styles := DefaultStyles()

	// Use distinct styles so we can tell them apart without ANSI color codes.
	boolStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	strStyle := lipgloss.NewStyle().Italic(true)
	styles.Values[true] = new(boolStyle)
	styles.FieldString = new(strStyle)

	// Bool field true -> styled via typed Values[true].
	got := styleValue("true", true, "ok", kindBool, styles, &defaultFieldFormats)
	assert.Equal(t, boolStyle.Render("true"), got)

	// String field "true" -> NOT styled via Values (no string "true" key).
	// Should fall through to FieldString styling.
	got = styleValue("true", "true", "ok", kindString, styles, &defaultFieldFormats)
	assert.Equal(t, strStyle.Render("true"), got)
}

func TestClampPercent(t *testing.T) {
	assert.InDelta(t, 0.0, core.ClampPercent(-10, 1), 0)
	assert.InDelta(t, 0.0, core.ClampPercent(0, 1), 0)
	assert.InDelta(t, 0.5, core.ClampPercent(0.5, 1), 0)
	assert.InDelta(t, 1.0, core.ClampPercent(1, 1), 0)
	assert.InDelta(t, 1.0, core.ClampPercent(2, 1), 0)
}

func TestClampPercentMaximum100(t *testing.T) {
	assert.InDelta(t, 0.0, core.ClampPercent(-10, 100), 0)
	assert.InDelta(t, 50.0, core.ClampPercent(50, 100), 0)
	assert.InDelta(t, 100.0, core.ClampPercent(100, 100), 0)
	assert.InDelta(t, 100.0, core.ClampPercent(200, 100), 0)
}

func TestClampPercentNaN(t *testing.T) {
	assert.InDelta(t, 0.0, core.ClampPercent(math.NaN(), 1), 0)
}

func TestClampPercentPositionInf(t *testing.T) {
	assert.InDelta(t, 1.0, core.ClampPercent(math.Inf(1), 1), 0)
}

func TestClampPercentNegInf(t *testing.T) {
	assert.InDelta(t, 0.0, core.ClampPercent(math.Inf(-1), 1), 0)
}

func TestInterpolateGradientEmpty(t *testing.T) {
	c := style.InterpolateGradient(0.5, nil)
	// Empty -> white fallback.
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 1.0, c.G, 0.01)
	assert.InDelta(t, 1.0, c.B, 0.01)
}

func TestInterpolateGradientSingleStop(t *testing.T) {
	red := colorful.Color{R: 1, G: 0, B: 0}
	c := style.InterpolateGradient(0.5, []style.ColorStop{{Position: 0.5, Color: red}})
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 0.0, c.G, 0.01)
	assert.InDelta(t, 0.0, c.B, 0.01)
}

func TestInterpolateGradientEdges(t *testing.T) {
	stops := style.DefaultPercentGradient()

	// At 0.0 -> red.
	c := style.InterpolateGradient(0.0, stops)
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 0.0, c.G, 0.1)

	// At 1.0 -> green.
	c = style.InterpolateGradient(1.0, stops)
	assert.InDelta(t, 0.0, c.R, 0.1)
	assert.InDelta(t, 1.0, c.G, 0.01)

	// Below 0.0 -> clamp to red.
	c = style.InterpolateGradient(-0.5, stops)
	assert.InDelta(t, 1.0, c.R, 0.01)

	// Above 1.0 -> clamp to green.
	c = style.InterpolateGradient(1.5, stops)
	assert.InDelta(t, 0.0, c.R, 0.1)
	assert.InDelta(t, 1.0, c.G, 0.01)
}

func TestInterpolateGradientMidpoint(t *testing.T) {
	stops := style.DefaultPercentGradient()

	// At 0.5 -> yellow (R=1, G=1, B=0).
	c := style.InterpolateGradient(0.5, stops)
	assert.InDelta(t, 1.0, c.R, 0.01)
	assert.InDelta(t, 1.0, c.G, 0.01)
	assert.InDelta(t, 0.0, c.B, 0.1)
}

func TestStyleFractionOutput(t *testing.T) {
	styles := DefaultStyles()
	got := styleFraction("7/10", core.Fraction{Current: 7, Total: 10}, styles, false)
	// The numbers take the gradient color; the "/" keeps that color but adds
	// the faint attribute (SGR 2) so it renders dimmed.
	assert.Equal(t,
		"\x1b[38;2;202;255;0m7\x1b[m\x1b[2;38;2;202;255;0m/\x1b[m\x1b[38;2;202;255;0m10\x1b[m",
		got)
}

func TestStyleFractionSeparatorOverride(t *testing.T) {
	styles := DefaultStyles()
	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	styles.FieldFractionSeparator = &sep
	got := styleFraction("7/10", core.Fraction{Current: 7, Total: 10}, styles, false)
	// Explicit separator style wins over the dimmed-gradient default.
	assert.Equal(t,
		"\x1b[38;2;202;255;0m7\x1b[m\x1b[38;2;255;0;0m/\x1b[m\x1b[38;2;202;255;0m10\x1b[m",
		got)
}

func TestStyleFractionNoGradient(t *testing.T) {
	styles := DefaultStyles()
	styles.PercentGradient = nil
	got := styleFraction("3/5", core.Fraction{Current: 3, Total: 5}, styles, false)
	assert.Empty(t, got)
}

func TestStyleFractionWrongType(t *testing.T) {
	styles := DefaultStyles()
	got := styleFraction("3/5", "not a fraction", styles, false)
	assert.Empty(t, got)
}

func TestStylePercentOutput(t *testing.T) {
	styles := DefaultStyles()
	got := stylePercent("75%", core.Percent{Value: 0.75}, styles, false, 0)

	// Should contain ANSI escape codes (color applied).
	assert.Equal(t, "\x1b[38;2;185;255;0m75%\x1b[m", got)
}

func TestStylePercentNoGradient(t *testing.T) {
	styles := DefaultStyles()
	styles.PercentGradient = nil
	got := stylePercent("50%", core.Percent{Value: 0.50}, styles, false, 0)
	assert.Empty(t, got, "nil gradient should return empty")
}

func TestStylePercentWrongType(t *testing.T) {
	styles := DefaultStyles()
	got := stylePercent("50%", "not a percent", styles, false, 0)
	assert.Empty(t, got, "non-percent originalValue should return empty")
}

func TestStylePercentSingleStop(t *testing.T) {
	styles := DefaultStyles()
	blue := colorful.Color{R: 0, G: 0, B: 1}
	styles.PercentGradient = []style.ColorStop{{Position: 0.5, Color: blue}}
	got := stylePercent("50%", core.Percent{Value: 0.50}, styles, false, 0)

	// Should use the single stop's color for any value.
	assert.Equal(t, "\x1b[38;2;0;0;255m50%\x1b[m", got)
}

func TestStyleValuePercent(t *testing.T) {
	styles := DefaultStyles()
	got := styleValue(
		"75%",
		core.Percent{Value: 0.75},
		"progress",
		kindPercent,
		styles,
		&defaultFieldFormats,
	)
	assert.Equal(t, "\x1b[38;2;185;255;0m75%\x1b[m", got)
}

func TestStylePercentReverse(t *testing.T) {
	// Default gradient: 0=red, 100=green.
	// At 0% with no reverse the gradient position is 0 (red end).
	// With reverse the position flips to 1 (green end).
	// We verify that reverse=true yields a different color than reverse=false.
	styles := DefaultStyles()
	normal := stylePercent("0%", core.Percent{Value: 0}, styles, false, 0)
	reversed := stylePercent("0%", core.Percent{Value: 0}, styles, true, 0)

	assert.NotEmpty(t, normal)
	assert.NotEmpty(t, reversed)
	assert.NotEqual(t, normal, reversed, "reversed gradient should produce a different color")
}

func TestPercentReverseLogger(t *testing.T) {
	var buf bytes.Buffer
	l := New(NewOutput(&buf, ColorAlways))
	f := DefaultFieldFormats()
	f.PercentReverseGradient = true
	l.SetFieldFormats(f)
	l.Info().Percent("cpu", 0.0).Send()

	got := buf.String()
	assert.Equal(
		t,
		"\x1b[1;32mINF\x1b[m ℹ️ \x1b[34mcpu\x1b[m\x1b[2m=\x1b[m\x1b[38;2;0;255;0m0%\x1b[m\n",
		got,
	)
}

func TestPercentReverseFieldTogglesLoggerDefault(t *testing.T) {
	styles := DefaultStyles()

	// Logger default = normal (reverse=false).
	// percentValue{reverse:true} should flip to reverse=true → same as stylePercent(..., true).
	normalAt0 := stylePercent("0%", core.Percent{Value: 0}, styles, false, 0)
	fieldFlippedAt0 := stylePercent(
		"0%",
		core.Percent{Value: 0, Reverse: true},
		styles,
		false,
		0,
	)
	assert.Equal(t, stylePercent("0%", core.Percent{Value: 0}, styles, true, 0), fieldFlippedAt0,
		"WithPercentReverseGradient on a normal logger should match logger reverse=true")
	assert.NotEqual(t, normalAt0, fieldFlippedAt0,
		"flipped field should differ from non-flipped")

	// Logger default = reverse=true.
	// percentValue{reverse:true} should flip back to reverse=false → same as stylePercent(..., false).
	fieldFlippedBack := stylePercent(
		"0%",
		core.Percent{Value: 0, Reverse: true},
		styles,
		true,
		0,
	)
	assert.Equal(t, normalAt0, fieldFlippedBack,
		"WithPercentReverseGradient on a reversed logger should flip back to normal")
}

func TestStyleValuePercentNilGradient(t *testing.T) {
	styles := DefaultStyles()
	styles.PercentGradient = nil
	got := styleValue(
		"50%",
		core.Percent{Value: 0.50},
		"progress",
		kindPercent,
		styles,
		&defaultFieldFormats,
	)
	assert.Empty(t, got)
}

func TestStylePercentBaseStyle(t *testing.T) {
	styles := DefaultStyles()
	bold := lipgloss.NewStyle().Bold(true)
	styles.FieldPercent = new(bold)

	got := stylePercent("75%", core.Percent{Value: 0.75}, styles, false, 0)
	assert.Equal(t, "\x1b[1;38;2;185;255;0m75%\x1b[m", got)
}

func TestStylePercentBaseStyleOnly(t *testing.T) {
	styles := DefaultStyles()
	bold := lipgloss.NewStyle().Bold(true)
	styles.FieldPercent = new(bold)
	styles.PercentGradient = nil // no gradient, base style only

	got := stylePercent("50%", core.Percent{Value: 0.50}, styles, false, 0)
	assert.Equal(t, bold.Render("50%"), got)
}

func TestStyleAnyElementPercent(t *testing.T) {
	styles := DefaultStyles()
	got := styleAnyElement(
		"75%",
		core.Percent{Value: 0.75},
		kindPercent,
		styles,
		&defaultFieldFormats,
	)
	assert.Equal(t, "\x1b[38;2;185;255;0m75%\x1b[m", got)
}

func TestRenderPendingNum(t *testing.T) {
	tests := []struct {
		name   string
		num    string
		spaces string
		style  *lipgloss.Style
		want   string
	}{
		{
			name: "empty_num_noop",
			num:  "",
			want: "",
		},
		{
			name: "non_empty_nil_style",
			num:  "42",
			want: "42",
		},
		{
			name:  "non_empty_with_style",
			num:   "42",
			style: new(lipgloss.NewStyle()),
			want:  lipgloss.NewStyle().Render("42"),
		},
		{
			name:   "non_empty_with_spaces",
			num:    "42",
			spaces: "  ",
			want:   "42  ",
		},
		{
			name:   "styled_with_spaces",
			num:    "7",
			spaces: " ",
			style:  new(lipgloss.NewStyle()),
			want:   lipgloss.NewStyle().Render("7") + " ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf strings.Builder
			renderPendingNum(&buf, tt.num, tt.spaces, tt.style)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestStyleNumberUnit(t *testing.T) {
	noop := new(lipgloss.NewStyle())

	tests := []struct {
		name   string
		input  string
		num    *lipgloss.Style
		unit   *lipgloss.Style
		overr  style.Map
		thresh style.ThresholdMap
		ignore bool
		want   string
	}{
		{
			name:  "all_nil_styles_returns_empty",
			input: "5m",
			want:  "",
		},
		{
			name:  "non_quantity_returns_empty",
			input: "hello",
			num:   noop,
			want:  "",
		},
		{
			name:  "space_separated_quantity",
			input: "5 MB",
			num:   noop,
			unit:  noop,
			want:  lipgloss.NewStyle().Render("5") + " " + lipgloss.NewStyle().Render("MB"),
		},
		{
			name:  "non_quantity_trailing_number",
			input: "5m2",
			num:   noop,
			want:  "",
		},
		{
			name:   "unit_override_applied",
			input:  "100MB",
			num:    noop,
			unit:   noop,
			overr:  style.Map{"MB": noop},
			ignore: false,
			want:   lipgloss.NewStyle().Render("100") + lipgloss.NewStyle().Render("MB"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := styleNumberUnit(tt.input, tt.num, tt.unit, tt.overr, tt.thresh, tt.ignore)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInterpolateGradientThreeStops(t *testing.T) {
	red := colorful.Color{R: 1, G: 0, B: 0}
	yellow := colorful.Color{R: 1, G: 1, B: 0}
	green := colorful.Color{R: 0, G: 1, B: 0}

	stops := []style.ColorStop{
		{Position: 0.0, Color: red},
		{Position: 0.5, Color: yellow},
		{Position: 1.0, Color: green},
	}

	// At 0.25 (between red and yellow), R should still be high.
	c := style.InterpolateGradient(0.25, stops)
	assert.Greater(t, c.R, 0.8, "R at 0.25 should be high")

	// At 0.75 (between yellow and green), G should be high and R should be dropping.
	c = style.InterpolateGradient(0.75, stops)
	assert.Greater(t, c.G, 0.7, "G at 0.75 should be high")
}

func TestFormatFieldsSortAscending(t *testing.T) {
	opts := formatFieldsOpts{
		fieldSort: SortAscending,
		noColor:   true,
	}

	fields := []Field{
		{Key: "c", Value: "3"},
		{Key: "a", Value: "1"},
		{Key: "b", Value: "2"},
	}

	got := formatFields(fields, opts)
	assert.Equal(t, " a=1 b=2 c=3", got)

	// Original slice must not be mutated.
	assert.Equal(t, "c", fields[0].Key)
}

func TestFormatFieldsSortDescending(t *testing.T) {
	opts := formatFieldsOpts{
		fieldSort: SortDescending,
		noColor:   true,
	}

	got := formatFields([]Field{
		{Key: "a", Value: "1"},
		{Key: "c", Value: "3"},
		{Key: "b", Value: "2"},
	}, opts)
	assert.Equal(t, " c=3 b=2 a=1", got)
}

func TestFormatFieldsSortNone(t *testing.T) {
	styles := DefaultStyles()
	// SortNone is the default zero value.

	opts := formatFieldsOpts{
		noColor: true,
		styles:  styles,
	}

	got := formatFields([]Field{
		{Key: "c", Value: "3"},
		{Key: "a", Value: "1"},
	}, opts)
	assert.Equal(t, " c=3 a=1", got)
}

func TestDurationFormatFunc(t *testing.T) {
	f := DefaultFieldFormats()
	f.DurationFormat = func(d time.Duration) string {
		return "~" + d.Truncate(time.Second).String()
	}

	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields([]Field{
		{Key: "took", Value: 3456 * time.Millisecond},
	}, opts)
	assert.Equal(t, " took=~3s", got)
}

func TestDurationFormatFuncAppliesSlice(t *testing.T) {
	f := DefaultFieldFormats()
	f.DurationFormat = func(d time.Duration) string {
		return d.Truncate(time.Second).String() + "!"
	}

	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields([]Field{
		{Key: "times", Value: []time.Duration{time.Second, 2 * time.Second}},
	}, opts)
	assert.Equal(t, " times=[1s!, 2s!]", got)
}

func TestDurationFormatFuncFallbackForElapsed(t *testing.T) {
	f := DefaultFieldFormats()
	f.DurationFormat = func(d time.Duration) string {
		return "dur:" + d.Truncate(time.Second).String()
	}
	f.ElapsedMinimum = 0
	f.ElapsedRound = 0

	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{Value: 3456 * time.Millisecond}},
	}, opts)
	assert.Equal(t, " took=dur:3s", got)
}

func TestDurationFormatFuncElapsedSpecificOverrides(t *testing.T) {
	// ElapsedFormat takes priority over DurationFormat for elapsed fields.
	f := DefaultFieldFormats()
	f.DurationFormat = func(d time.Duration) string { return "dur:" + d.String() }
	f.ElapsedFormat = func(d time.Duration) string { return "ela:" + d.String() }
	f.ElapsedMinimum = 0
	f.ElapsedRound = 0

	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{Value: 3 * time.Second}},
	}, opts)
	assert.Equal(t, " took=ela:3s", got)
}

func TestDurationFormatFuncNilFallsBack(t *testing.T) {
	// DurationFormat is nil - plain Duration fields should use val.String().
	opts := formatFieldsOpts{noColor: true}

	got := formatFields([]Field{
		{Key: "took", Value: 3200 * time.Millisecond},
	}, opts)
	assert.Equal(t, " took=3.2s", got)
}

func TestElapsedFormatFunc(t *testing.T) {
	f := DefaultFieldFormats()
	f.ElapsedFormat = func(d time.Duration) string {
		return d.Truncate(time.Second).String()
	}
	f.ElapsedMinimum = 0
	f.ElapsedRound = 0

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{Value: 3456 * time.Millisecond}},
	}, opts)
	assert.Equal(t, " took=3s", got)
}

func TestPercentFormatFunc(t *testing.T) {
	f := DefaultFieldFormats()
	f.PercentFormat = func(v float64) string {
		return strconv.FormatFloat(v, 'f', 0, 64) + " pct"
	}

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "done", Value: core.Percent{Value: 0.75}},
	}, opts)
	assert.Equal(t, " done=75 pct", got)
}

func TestElapsedFormatFuncNilFallsBack(t *testing.T) {
	// ElapsedFormatFunc is nil - should use built-in formatElapsed with default precision 0.
	opts := formatFieldsOpts{
		noColor: true,
	}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{Value: 3200 * time.Millisecond}},
	}, opts)
	assert.Equal(t, " took=3s", got)
}

func TestPercentFormatFuncNilFallsBack(t *testing.T) {
	styles := DefaultStyles()
	// PercentFormatFunc is nil - should use built-in format.

	opts := formatFieldsOpts{
		noColor: true,
		styles:  styles,
	}

	got := formatFields([]Field{
		{Key: "done", Value: core.Percent{Value: 0.75}},
	}, opts)
	assert.Equal(t, " done=75%", got)
}

func TestLookupValueStyle(t *testing.T) {
	s := new(lipgloss.NewStyle())

	tests := []struct {
		name   string
		value  any
		values style.ValueMap
		want   *lipgloss.Style
	}{
		{
			name:   "empty_map_returns_nil",
			value:  "anything",
			values: style.ValueMap{},
			want:   nil,
		},
		{
			name:   "nil_map_returns_nil",
			value:  "anything",
			values: nil,
			want:   nil,
		},
		{
			name:   "hashable_value_present",
			value:  "ok",
			values: style.ValueMap{"ok": s},
			want:   s,
		},
		{
			name:   "hashable_value_missing",
			value:  "missing",
			values: style.ValueMap{"ok": s},
			want:   nil,
		},
		{
			name:   "unhashable_value_slice_no_panic",
			value:  []int{1, 2, 3},
			values: style.ValueMap{"ok": s},
			want:   nil,
		},
		{
			name:   "nil_value",
			value:  nil,
			values: style.ValueMap{nil: s},
			want:   s,
		},
		{
			name:   "nil_value_not_in_map",
			value:  nil,
			values: style.ValueMap{"ok": s},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotPanics(t, func() {
				got := lookupValueStyle(tt.value, tt.values)
				assert.Equal(t, tt.want, got)
			})
		})
	}
}

func TestElapsedMinimum(t *testing.T) {
	f := DefaultFieldFormats()
	f.ElapsedMinimum = time.Second
	f.ElapsedRound = 0
	f.ElapsedPrecision = 0

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{Value: 500 * time.Millisecond}},
		{Key: "name", Value: "test"},
	}, opts)
	// Elapsed below minimum is hidden; other fields remain.
	assert.Equal(t, " name=test", got)
}

func TestElapsedMinimumZeroDisabled(t *testing.T) {
	f := DefaultFieldFormats()
	f.ElapsedMinimum = 0
	f.ElapsedRound = 0

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{Value: 500 * time.Millisecond}},
	}, opts)
	// All values shown when minimum is 0.
	assert.Equal(t, " took=500ms", got)
}

func TestElapsedRound(t *testing.T) {
	f := DefaultFieldFormats()
	f.ElapsedRound = time.Second
	f.ElapsedMinimum = 0
	f.ElapsedPrecision = 0

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{Value: 2600 * time.Millisecond}},
	}, opts)
	// 2.6s rounds to 3s.
	assert.Equal(t, " took=3s", got)
}

func TestElapsedRoundPerFieldOverride(t *testing.T) {
	f := DefaultFieldFormats()
	f.ElapsedRound = time.Second
	f.ElapsedMinimum = 0
	f.ElapsedPrecision = 0

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "took", Value: core.ElapsedField{
			Value: 450 * time.Millisecond,
			Round: new(time.Millisecond),
		}},
	}, opts)
	// The per-field millisecond granularity overrides the logger's 1s rounding,
	// so a sub-second value renders instead of collapsing to 0s.
	assert.Equal(t, " took=450ms", got)
}

func TestDurationRoundPerFieldOverride(t *testing.T) {
	f := DefaultFieldFormats()
	f.DurationRound = time.Second
	f.DurationMinimum = 0
	f.DurationPrecision = 0

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "took", Value: core.DurationField{
			Value: 450 * time.Millisecond,
			Round: new(time.Millisecond),
		}},
	}, opts)
	// The per-field millisecond granularity overrides the logger's 1s rounding,
	// so a sub-second value renders instead of collapsing to 0s.
	assert.Equal(t, " took=450ms", got)
}

func TestDeadlineRoundPerFieldOverride(t *testing.T) {
	f := DefaultFieldFormats()
	f.ElapsedRound = time.Second
	f.ElapsedPrecision = 0

	opts := formatFieldsOpts{
		noColor: true,
		formats: &f,
	}

	got := formatFields([]Field{
		{Key: "left", Value: core.DeadlineField{
			Remaining: 450 * time.Millisecond,
			From:      10 * time.Second,
			Round:     new(time.Millisecond),
		}},
	}, opts)
	// The per-field millisecond granularity overrides the logger's 1s ceiling.
	assert.Equal(t, " left=450ms", got)
}

func TestDefaultDurationScale(t *testing.T) {
	tests := []struct {
		name  string
		value time.Duration
		want  string
	}{
		{name: "milliseconds", value: 450 * time.Millisecond, want: " took=450ms"},
		{name: "negative_milliseconds", value: -450 * time.Millisecond, want: " took=450ms"},
		{name: "one_second_trims_zero", value: time.Second, want: " took=1s"},
		{name: "fractional_seconds", value: 1540 * time.Millisecond, want: " took=1.5s"},
		{name: "below_ten_seconds", value: 9940 * time.Millisecond, want: " took=9.9s"},
		{name: "rounds_across_boundary", value: 9960 * time.Millisecond, want: " took=10s"},
		{name: "whole_seconds", value: 12400 * time.Millisecond, want: " took=12s"},
	}

	f := DefaultFieldFormats()
	opts := formatFieldsOpts{noColor: true, formats: &f}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatFields(
				[]Field{{Key: "took", Value: core.DurationField{Value: tt.value}}},
				opts,
			)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDefaultLiveTimeScaleUsesStableWholeSeconds(t *testing.T) {
	f := DefaultFieldFormats()
	f.ElapsedMinimum = 0
	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields([]Field{
		{Key: "elapsed", Value: core.ElapsedField{Value: 1600 * time.Millisecond}},
		{
			Key:   "left",
			Value: core.DeadlineField{Remaining: 1100 * time.Millisecond, From: 10 * time.Second},
		},
	}, opts)
	assert.Equal(t, " elapsed=2s left=2s", got)
}

func TestTimeScaleInheritance(t *testing.T) {
	f := DefaultFieldFormats()
	f.DurationScale = nil
	f.ElapsedMinimum = 0
	f.ElapsedScale = nil
	f.TimeScale = TimeScale{{Precision: 1, Round: 100 * time.Millisecond, Trim: true}}
	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields([]Field{
		{Key: "duration", Value: core.DurationField{Value: 1240 * time.Millisecond}},
		{Key: "elapsed", Value: core.ElapsedField{Value: 1240 * time.Millisecond}},
		{
			Key:   "left",
			Value: core.DeadlineField{Remaining: 1210 * time.Millisecond, From: 10 * time.Second},
		},
	}, opts)
	assert.Equal(t, " duration=1.2s elapsed=1.2s left=1.3s", got)
}

func TestTimeScaleFieldOverridePrecedence(t *testing.T) {
	f := DefaultFieldFormats()
	f.DurationScale = TimeScale{{Round: time.Second}}
	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields([]Field{
		{Key: "scaled", Value: core.DurationField{
			Value: 450 * time.Millisecond,
			Scale: TimeScale{{Round: time.Millisecond}},
		}},
		{Key: "rounded", Value: core.DurationField{
			Value: 450 * time.Millisecond,
			Round: new(time.Second),
			Scale: TimeScale{{Round: time.Millisecond}},
		}},
	}, opts)
	assert.Equal(t, " scaled=450ms rounded=0s", got)
}

func TestTimeScaleWithoutCatchAllUsesLastStep(t *testing.T) {
	f := DefaultFieldFormats()
	f.DurationScale = TimeScale{{Below: time.Second, Precision: 1, Round: 100 * time.Millisecond}}
	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields(
		[]Field{{Key: "took", Value: core.DurationField{Value: 2200 * time.Millisecond}}},
		opts,
	)
	assert.Equal(t, " took=2.2s", got)
}

func TestDurationSliceIgnoresTimeScale(t *testing.T) {
	f := DefaultFieldFormats()
	f.TimeScale = TimeScale{{Round: time.Second}}
	opts := formatFieldsOpts{noColor: true, formats: &f}

	got := formatFields(
		[]Field{
			{Key: "took", Value: []time.Duration{450 * time.Millisecond, 1500 * time.Millisecond}},
		},
		opts,
	)
	assert.Equal(t, " took=[450ms, 1.5s]", got)
}

func TestFormatInt64SlicePlain(t *testing.T) {
	tests := []struct {
		name string
		vals []int64
		want string
	}{
		{name: "multiple", vals: []int64{10, 20, 30}, want: "[10, 20, 30]"},
		{name: "single", vals: []int64{42}, want: "[42]"},
		{name: "empty", vals: []int64{}, want: "[]"},
		{
			name: "large_values",
			vals: []int64{9223372036854775807, -9223372036854775808},
			want: "[9223372036854775807, -9223372036854775808]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumberSlice(tt.vals, sliceFormat{open: "[", close: "]", sep: ", "}, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatInt64SliceStyled(t *testing.T) {
	styles := DefaultStyles()
	n := styles.FieldNumber.Render

	got := formatNumberSlice([]int64{10, 20}, sliceFormat{open: "[", close: "]", sep: ", "}, styles)
	want := "[" + n("10") + ", " + n("20") + "]"
	assert.Equal(t, want, got)
}

func TestFormatUintSlicePlain(t *testing.T) {
	tests := []struct {
		name string
		vals []uint
		want string
	}{
		{name: "multiple", vals: []uint{10, 20, 30}, want: "[10, 20, 30]"},
		{name: "single", vals: []uint{42}, want: "[42]"},
		{name: "empty", vals: []uint{}, want: "[]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatNumberSlice(tt.vals, sliceFormat{open: "[", close: "]", sep: ", "}, nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatUintSliceStyled(t *testing.T) {
	styles := DefaultStyles()
	n := styles.FieldNumber.Render

	got := formatNumberSlice([]uint{10, 20}, sliceFormat{open: "[", close: "]", sep: ", "}, styles)
	want := "[" + n("10") + ", " + n("20") + "]"
	assert.Equal(t, want, got)
}

func TestStyleElapsed(t *testing.T) {
	t.Run("elapsed_styles_applied", func(t *testing.T) {
		styles := DefaultStyles()
		elapsedNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		elapsedUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		styles.FieldElapsedNumber = new(elapsedNum)
		styles.FieldElapsedUnit = new(elapsedUnit)

		got := styleElapsed("5s", nil, styles, 0)
		want := elapsedNum.Render("5") + elapsedUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("fallback_to_duration_styles", func(t *testing.T) {
		styles := DefaultStyles()
		// FieldElapsedNumber and FieldElapsedUnit are nil by default,
		// so it should fall back to FieldDurationNumber and FieldDurationUnit.
		styles.FieldElapsedNumber = nil
		styles.FieldElapsedUnit = nil

		got := styleElapsed("5s", nil, styles, 0)
		want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("partial_fallback_number_only", func(t *testing.T) {
		styles := DefaultStyles()
		elapsedNum := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
		styles.FieldElapsedNumber = new(elapsedNum)
		styles.FieldElapsedUnit = nil // falls back to FieldDurationUnit

		got := styleElapsed("5s", nil, styles, 0)
		want := elapsedNum.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("partial_fallback_unit_only", func(t *testing.T) {
		styles := DefaultStyles()
		elapsedUnit := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		styles.FieldElapsedNumber = nil // falls back to FieldDurationNumber
		styles.FieldElapsedUnit = new(elapsedUnit)

		got := styleElapsed("5s", nil, styles, 0)
		want := styles.FieldDurationNumber.Render("5") + elapsedUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("all_nil_returns_empty", func(t *testing.T) {
		styles := DefaultStyles()
		styles.FieldElapsedNumber = nil
		styles.FieldElapsedUnit = nil
		styles.FieldDurationNumber = nil
		styles.FieldDurationUnit = nil

		got := styleElapsed("5s", nil, styles, 0)
		assert.Empty(t, got)
	})
}

func TestStyleElapsedGradient(t *testing.T) {
	t.Run("active_gradient", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.ElapsedField{Value: 15 * time.Second} // t=0.5 → yellow
		got := styleElapsed("15s", val, styles, 30*time.Second)

		assert.Equal(t, "\x1b[38;2;255;255;0m15s\x1b[m", got)
	})

	t.Run("gradient_at_zero", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.ElapsedField{Value: 0}
		got := styleElapsed("0s", val, styles, 30*time.Second)

		assert.Equal(t, "\x1b[38;2;0;255;0m0s\x1b[m", got)
	})

	t.Run("gradient_clamped_beyond_max", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.ElapsedField{Value: 60 * time.Second} // way beyond max
		got := styleElapsed("60s", val, styles, 10*time.Second)

		// Should use the t=1.0 end color (red), not crash.
		assert.Equal(t, "\x1b[38;2;255;0;0m60s\x1b[m", got)

		// Should produce the same result as exactly at max.
		atMax := core.ElapsedField{Value: 10 * time.Second}
		gotAtMax := styleElapsed("10s", atMax, styles, 10*time.Second)
		assert.NotEmpty(t, gotAtMax)
	})

	t.Run("inactive_zero_max", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.ElapsedField{Value: 5 * time.Second}
		got := styleElapsed("5s", val, styles, 0) // disabled

		// Should fall through to number/unit path (non-empty with default styles).
		want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("inactive_nil_stops", func(t *testing.T) {
		styles := DefaultStyles()
		styles.ElapsedGradient = nil

		val := core.ElapsedField{Value: 5 * time.Second}
		got := styleElapsed("5s", val, styles, 30*time.Second)

		// Should fall through to number/unit path.
		want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("inactive_wrong_type", func(t *testing.T) {
		styles := DefaultStyles()

		// Pass a non-ElapsedField value - gradient path should not apply.
		got := styleElapsed("5s", "not elapsed", styles, 30*time.Second)

		// Falls through to number/unit path.
		want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("single_stop", func(t *testing.T) {
		styles := DefaultStyles()
		blue := colorful.Color{R: 0, G: 0, B: 1}
		styles.ElapsedGradient = []style.ColorStop{{Position: 0.5, Color: blue}}

		val := core.ElapsedField{Value: 5 * time.Second}
		got := styleElapsed("5s", val, styles, 10*time.Second)

		assert.Equal(t, "\x1b[38;2;0;0;255m5s\x1b[m", got)
	})

	t.Run("different_positions_different_colors", func(t *testing.T) {
		styles := DefaultStyles()

		earlyVal := core.ElapsedField{Value: 1 * time.Second} // t≈0.03 → green
		lateVal := core.ElapsedField{Value: 29 * time.Second} // t≈0.97 → red

		early := styleElapsed("1s", earlyVal, styles, 30*time.Second)
		late := styleElapsed("29s", lateVal, styles, 30*time.Second)

		assert.NotEqual(t, early, late, "different elapsed values should produce different colors")
	})

	t.Run("step_mode", func(t *testing.T) {
		styles := DefaultStyles()
		styles.ElapsedGradientMode = style.GradientStep

		// Two values in the same step region should produce the same color.
		val1 := core.ElapsedField{Value: 1 * time.Second}  // t≈0.03 → first stop (green)
		val2 := core.ElapsedField{Value: 10 * time.Second} // t≈0.33 → still first stop (green)

		got1 := styleElapsed("1s", val1, styles, 30*time.Second)
		got2 := styleElapsed("10s", val2, styles, 30*time.Second)

		// Both should be styled (non-empty).
		assert.NotEmpty(t, got1)
		assert.NotEmpty(t, got2)

		// Values crossing a step boundary should differ.
		val3 := core.ElapsedField{Value: 20 * time.Second} // t≈0.67 → second stop (yellow)
		got3 := styleElapsed("20s", val3, styles, 30*time.Second)
		assert.NotEqual(
			t,
			got1,
			got3,
			"values in different step regions should have different colors",
		)
	})

	t.Run("step_mode_vs_fade_mode", func(t *testing.T) {
		fadeStyles := DefaultStyles()
		fadeStyles.ElapsedGradientMode = style.GradientFade

		stepStyles := DefaultStyles()
		stepStyles.ElapsedGradientMode = style.GradientStep

		// At a midpoint, fade and step should produce different colors.
		val := core.ElapsedField{Value: 10 * time.Second} // t≈0.33
		fade := styleElapsed("10s", val, fadeStyles, 30*time.Second)
		step := styleElapsed("10s", val, stepStyles, 30*time.Second)

		assert.NotEqual(
			t,
			fade,
			step,
			"fade and step modes should produce different colors at non-boundary positions",
		)
	})
}

func TestFormatValueDeadline(t *testing.T) {
	// Default precision 0 → no decimal places.
	got, kind := formatValue(
		core.DeadlineField{Remaining: 3200 * time.Millisecond, From: 15 * time.Second},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&defaultFieldFormats,
	)
	assert.Equal(t, "3s", got)
	assert.Equal(t, kindDeadline, kind)

	// Precision 1 → one decimal place, no trimming.
	f := DefaultFieldFormats()
	f.ElapsedPrecision = 1
	got, kind = formatValue(
		core.DeadlineField{Remaining: 3200 * time.Millisecond, From: 15 * time.Second},
		sliceFormat{open: "[", close: "]", sep: ", "},
		QuoteAuto,
		0,
		0,
		nil,
		"",
		&f,
	)
	assert.Equal(t, "3.2s", got)
	assert.Equal(t, kindDeadline, kind)
}

func TestCeilDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		r    time.Duration
		want time.Duration
	}{
		{"zero", 0, time.Second, 0},
		{"negative", -500 * time.Millisecond, time.Second, -500 * time.Millisecond},
		{"exact_multiple", 3 * time.Second, time.Second, 3 * time.Second},
		{
			"rounds_up_below_half",
			14*time.Second + 200*time.Millisecond,
			time.Second,
			15 * time.Second,
		},
		{
			"rounds_up_above_half",
			14*time.Second + 800*time.Millisecond,
			time.Second,
			15 * time.Second,
		},
		{"sub_step_never_zero", 1 * time.Millisecond, time.Second, time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ceilDuration(tt.d, tt.r))
		})
	}
}

func TestStyleDeadlineGradient(t *testing.T) {
	t.Run("fresh_uses_first_stop", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.DeadlineField{
			Remaining: 30 * time.Second,
			From:      30 * time.Second,
		} // t=0 → green
		got := styleDeadline("30s", val, styles)

		assert.Equal(t, "\x1b[38;2;0;255;0m30s\x1b[m", got)
	})

	t.Run("midpoint", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.DeadlineField{
			Remaining: 15 * time.Second,
			From:      30 * time.Second,
		} // t=0.5 → yellow
		got := styleDeadline("15s", val, styles)

		assert.Equal(t, "\x1b[38;2;255;255;0m15s\x1b[m", got)
	})

	t.Run("expired_uses_last_stop", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.DeadlineField{Remaining: 0, From: 30 * time.Second} // t=1 → red
		got := styleDeadline("0s", val, styles)

		assert.Equal(t, "\x1b[38;2;255;0;0m0s\x1b[m", got)
	})

	t.Run("clamped_beyond_from", func(t *testing.T) {
		styles := DefaultStyles()

		// Remaining beyond From (negative consumed) clamps to the t=0 start
		// color (green), not crash.
		val := core.DeadlineField{Remaining: 60 * time.Second, From: 30 * time.Second}
		got := styleDeadline("60s", val, styles)

		assert.Equal(t, "\x1b[38;2;0;255;0m60s\x1b[m", got)
	})

	t.Run("inactive_zero_from", func(t *testing.T) {
		styles := DefaultStyles()

		val := core.DeadlineField{Remaining: 5 * time.Second, From: 0} // disabled

		// Should fall through to number/unit path (non-empty with default styles).
		got := styleDeadline("5s", val, styles)
		want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("inactive_nil_stops", func(t *testing.T) {
		styles := DefaultStyles()
		styles.DeadlineGradient = nil

		val := core.DeadlineField{Remaining: 5 * time.Second, From: 30 * time.Second}
		got := styleDeadline("5s", val, styles)

		// Should fall through to number/unit path.
		want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("inactive_wrong_type", func(t *testing.T) {
		styles := DefaultStyles()

		// Pass a non-DeadlineField value - gradient path should not apply.
		got := styleDeadline("5s", "not deadline", styles)

		// Falls through to number/unit path.
		want := styles.FieldDurationNumber.Render("5") + styles.FieldDurationUnit.Render("s")
		assert.Equal(t, want, got)
	})

	t.Run("per_field_gradient_override", func(t *testing.T) {
		styles := DefaultStyles()
		blue := colorful.Color{R: 0, G: 0, B: 1}

		val := core.DeadlineField{
			Remaining: 5 * time.Second,
			From:      10 * time.Second,
			Gradient:  []style.ColorStop{{Position: 0.5, Color: blue}},
		}
		got := styleDeadline("5s", val, styles)

		assert.Equal(t, "\x1b[38;2;0;0;255m5s\x1b[m", got)
	})

	t.Run("step_mode", func(t *testing.T) {
		styles := DefaultStyles()
		mode := style.GradientStep

		// Two values in the same step region should produce the same color.
		val1 := core.DeadlineField{
			Remaining:    29 * time.Second,
			From:         30 * time.Second,
			GradientMode: &mode,
		} // t≈0.03 → first stop (green)
		val2 := core.DeadlineField{
			Remaining:    20 * time.Second,
			From:         30 * time.Second,
			GradientMode: &mode,
		} // t≈0.33 → still first stop (green)

		got1 := styleDeadline("29s", val1, styles)
		got2 := styleDeadline("20s", val2, styles)

		assert.NotEmpty(t, got1)
		assert.NotEmpty(t, got2)

		// Values crossing a step boundary should differ.
		val3 := core.DeadlineField{
			Remaining:    10 * time.Second,
			From:         30 * time.Second,
			GradientMode: &mode,
		} // t≈0.67 → second stop (yellow)
		got3 := styleDeadline("10s", val3, styles)
		assert.NotEqual(
			t,
			got1,
			got3,
			"values in different step regions should have different colors",
		)
	})
}

func TestFormatETA(t *testing.T) {
	tests := []struct {
		name string
		dur  time.Duration
		want string
	}{
		{"zero", 0, "1s"},
		{"sub_second", 500 * time.Millisecond, "1s"},
		{"one_second", time.Second, "1s"},
		{"five_seconds", 5 * time.Second, "5s"},
		{"thirty_seconds", 30 * time.Second, "30s"},
		{"fifty_nine_seconds", 59 * time.Second, "59s"},
		{"one_minute", time.Minute, "1m"},
		{"one_minute_thirty", time.Minute + 30*time.Second, "1m30s"},
		{"two_minutes", 2 * time.Minute, "2m"},
		{"two_minutes_thirty", 2*time.Minute + 30*time.Second, "2m30s"},
		{"one_hour", time.Hour, "1h"},
		{"one_hour_two_min", time.Hour + 2*time.Minute, "1h2m"},
		{"two_hours_thirty_min", 2*time.Hour + 30*time.Minute, "2h30m"},
		{"negative", -5 * time.Second, "5s"},
		{"rounds_to_nearest_second", 4*time.Second + 600*time.Millisecond, "5s"},
		{"rounds_down", 4*time.Second + 400*time.Millisecond, "4s"},
		{"rounds_up_to_one", 200 * time.Millisecond, "1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, formatETA(tt.dur))
		})
	}
}
