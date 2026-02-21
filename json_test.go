package clog

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// highlightJSON
// ---------------------------------------------------------------------------

func TestHighlightJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		styles *JSONStyles
		want   string
	}{
		{
			name:   "nil_styles_passthrough",
			input:  `{"a":1}`,
			styles: nil,
			want:   `{"a":1}`,
		},
		{
			name:   "nil_styles_passthrough_with_whitespace",
			input:  `{ "a" : 1 }`,
			styles: nil,
			want:   `{ "a" : 1 }`,
		},
		{
			name:   "basic_object_no_styles",
			input:  `{"key":"value"}`,
			styles: &JSONStyles{},
			want:   `{"key":"value"}`,
		},
		{
			name:   "all_token_types",
			input:  `{"s":"v","n":42,"t":true,"f":false,"z":null}`,
			styles: &JSONStyles{},
			want:   `{"s":"v","n":42,"t":true,"f":false,"z":null}`,
		},
		{
			name:   "nested_objects",
			input:  `{"a":{"b":{"c":1}}}`,
			styles: &JSONStyles{},
			want:   `{"a":{"b":{"c":1}}}`,
		},
		{
			name:   "arrays",
			input:  `{"tags":["a","b",1,true,null]}`,
			styles: &JSONStyles{},
			want:   `{"tags":["a","b",1,true,null]}`,
		},
		{
			name:   "escaped_strings",
			input:  `{"msg":"say \"hi\""}`,
			styles: &JSONStyles{},
			want:   `{"msg":"say \"hi\""}`,
		},
		{
			name:   "number_with_exponent",
			input:  `{"v":1.5e+10}`,
			styles: &JSONStyles{},
			want:   `{"v":1.5e+10}`,
		},
		{
			name:   "negative_number",
			input:  `{"v":-42}`,
			styles: &JSONStyles{},
			want:   `{"v":-42}`,
		},
		{
			name:   "negative_float_with_exponent",
			input:  `{"v":-1.23E-4}`,
			styles: &JSONStyles{},
			want:   `{"v":-1.23E-4}`,
		},
		{
			name:   "pretty_printed_whitespace_stripped",
			input:  "{\n  \"a\" : 1 ,\n  \"b\" : 2\n}",
			styles: &JSONStyles{},
			want:   `{"a":1,"b":2}`,
		},
		{
			name:   "malformed_truncated_true",
			input:  `{"v":tru`,
			styles: &JSONStyles{},
			want:   `{"v":tru`,
		},
		{
			name:   "malformed_truncated_false",
			input:  `{"v":fal`,
			styles: &JSONStyles{},
			want:   `{"v":fal`,
		},
		{
			name:   "malformed_truncated_null",
			input:  `{"v":nul`,
			styles: &JSONStyles{},
			want:   `{"v":nul`,
		},
		{
			name:   "unexpected_byte_fallback",
			input:  `{"v":@bad}`,
			styles: &JSONStyles{},
			want:   `{"v":@bad}`,
		},
		{
			name:   "empty_object",
			input:  `{}`,
			styles: &JSONStyles{},
			want:   `{}`,
		},
		{
			name:   "empty_array",
			input:  `[]`,
			styles: &JSONStyles{},
			want:   `[]`,
		},
		{
			name:   "nested_array_of_arrays",
			input:  `[[1,2],[3,4]]`,
			styles: &JSONStyles{},
			want:   `[[1,2],[3,4]]`,
		},
		{
			name:   "root_array",
			input:  `[1,"two",true,null]`,
			styles: &JSONStyles{},
			want:   `[1,"two",true,null]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightJSON(tt.input, tt.styles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHighlightJSONHumanMode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "unquotes_simple_key_and_value",
			input: `{"name":"alice"}`,
			want:  `{name:alice}`,
		},
		{
			name:  "preserves_quoted_key_with_special",
			input: `{"a,b":"value"}`,
			want:  `{"a,b":value}`,
		},
		{
			name:  "preserves_quoted_empty_key",
			input: `{"":"value"}`,
			want:  `{"":value}`,
		},
		{
			name:  "preserves_quoted_value_with_escape",
			input: `{"k":"line\\n"}`,
			want:  `{k:"line\\n"}`,
		},
		{
			name:  "preserves_empty_string_value",
			input: `{"k":""}`,
			want:  `{k:""}`,
		},
		{
			name:  "number_values_pass_through",
			input: `{"n":42}`,
			want:  `{n:42}`,
		},
		{
			name:  "bool_and_null_pass_through",
			input: `{"t":true,"f":false,"z":null}`,
			want:  `{t:true,f:false,z:null}`,
		},
	}

	humanStyles := &JSONStyles{Mode: JSONModeHuman}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightJSON(tt.input, humanStyles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHighlightJSONFlatMode(t *testing.T) {
	styles := &JSONStyles{Mode: JSONModeFlat}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple_nested",
			input: `{"a":{"b":1}}`,
			want:  `{a.b:1}`,
		},
		{
			name:  "arrays_preserved",
			input: `{"tags":["x","y"]}`,
			want:  `{tags:[x,y]}`,
		},
		{
			name:  "deeply_nested",
			input: `{"a":{"b":{"c":"d"}}}`,
			want:  `{a.b.c:d}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightJSON(tt.input, styles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHighlightJSONRootBraceOverride(t *testing.T) {
	rootStyle := lipgloss.NewStyle().Bold(true)
	styles := &JSONStyles{
		BraceRoot: new(rootStyle),
	}

	got := highlightJSON(`{"a":{"b":1}}`, styles)
	// Root braces should be styled, nested should not.
	assert.Contains(t, got, rootStyle.Render("{"))
	assert.Contains(t, got, rootStyle.Render("}"))
}

func TestHighlightJSONRootBracketOverride(t *testing.T) {
	rootStyle := lipgloss.NewStyle().Bold(true)
	styles := &JSONStyles{
		BracketRoot: new(rootStyle),
	}

	got := highlightJSON(`[1,2]`, styles)
	assert.Contains(t, got, rootStyle.Render("["))
	assert.Contains(t, got, rootStyle.Render("]"))
}

func TestHighlightJSONSpacing(t *testing.T) {
	tests := []struct {
		name    string
		spacing JSONSpacing
		input   string
		want    string
	}{
		{
			name:    "after_colon",
			spacing: JSONSpacingAfterColon,
			input:   `{"a":1}`,
			want:    `{"a": 1}`,
		},
		{
			name:    "after_comma",
			spacing: JSONSpacingAfterComma,
			input:   `{"a":1,"b":2}`,
			want:    `{"a":1, "b":2}`,
		},
		{
			name:    "before_object",
			spacing: JSONSpacingBeforeObject,
			input:   `{"a":{"b":1}}`,
			want:    `{"a": {"b":1}}`,
		},
		{
			name:    "before_array",
			spacing: JSONSpacingBeforeArray,
			input:   `{"a":[1,2]}`,
			want:    `{"a": [1,2]}`,
		},
		{
			name:    "all_spacing",
			spacing: JSONSpacingAll,
			input:   `{"a":1,"b":{"c":2},"d":[3]}`,
			want:    `{"a": 1, "b":  {"c": 2}, "d":  [3]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			styles := &JSONStyles{Spacing: tt.spacing}
			got := highlightJSON(tt.input, styles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHighlightJSONAfterCommaSpacing(t *testing.T) {
	styles := &JSONStyles{Spacing: JSONSpacingAfterComma}

	got := highlightJSON(`{"a":1,"b":2}`, styles)
	assert.Equal(t, `{"a":1, "b":2}`, got)
}

func TestHighlightJSONOmitCommas(t *testing.T) {
	styles := &JSONStyles{OmitCommas: true}

	got := highlightJSON(`{"a":1,"b":2}`, styles)
	assert.Equal(t, `{"a":1"b":2}`, got)
}

func TestHighlightJSONOmitCommasWithSpacing(t *testing.T) {
	styles := &JSONStyles{
		OmitCommas: true,
		Spacing:    JSONSpacingAfterComma,
	}

	got := highlightJSON(`{"a":1,"b":2}`, styles)
	assert.Equal(t, `{"a":1 "b":2}`, got)
}

func TestHighlightJSONStyled(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	strStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	trueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	falseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

	styles := &JSONStyles{
		Key:       new(keyStyle),
		Number:    new(numStyle),
		String:    new(strStyle),
		BoolTrue:  new(trueStyle),
		BoolFalse: new(falseStyle),
		Null:      new(nullStyle),
	}

	got := highlightJSON(`{"n":42,"s":"v","t":true,"f":false,"z":null}`, styles)

	assert.Contains(t, got, keyStyle.Render(`"n"`))
	assert.Contains(t, got, numStyle.Render("42"))
	assert.Contains(t, got, strStyle.Render(`"v"`))
	assert.Contains(t, got, trueStyle.Render("true"))
	assert.Contains(t, got, falseStyle.Render("false"))
	assert.Contains(t, got, nullStyle.Render("null"))
}

func TestHighlightJSONUnterminatedString(t *testing.T) {
	styles := &JSONStyles{}

	// Unterminated string should not panic; scanner emits what it has.
	got := highlightJSON(`{"key":"unterminated`, styles)
	assert.Contains(t, got, "unterminated")
}

// ---------------------------------------------------------------------------
// renderFlatJSON
// ---------------------------------------------------------------------------

func TestRenderFlatJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "nested_objects",
			input: `{"user":{"name":"alice","age":30}}`,
			want:  `{user.name:alice,user.age:30}`,
		},
		{
			name:  "arrays_preserved",
			input: `{"tags":["a","b"]}`,
			want:  `{tags:[a,b]}`,
		},
		{
			name:  "empty_nested_object",
			input: `{"a":{}}`,
			want:  `{}`,
		},
		{
			name:  "non_object_root_falls_back_to_human",
			input: `[1,2,3]`,
			want:  `[1,2,3]`,
		},
		{
			name:  "string_root_falls_back_to_human",
			input: `"hello"`,
			want:  `hello`,
		},
		{
			name:  "empty_input",
			input: ``,
			want:  ``,
		},
		{
			name:  "deeply_nested",
			input: `{"a":{"b":{"c":{"d":1}}}}`,
			want:  `{a.b.c.d:1}`,
		},
	}

	flatStyles := &JSONStyles{Mode: JSONModeFlat}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderFlatJSON(tt.input, flatStyles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRenderFlatJSONWithSpacing(t *testing.T) {
	styles := &JSONStyles{
		Mode:    JSONModeFlat,
		Spacing: JSONSpacingAfterColon | JSONSpacingAfterComma,
	}

	got := renderFlatJSON(`{"a":1,"b":2}`, styles)
	assert.Equal(t, `{a: 1, b: 2}`, got)
}

func TestRenderFlatJSONOmitCommas(t *testing.T) {
	styles := &JSONStyles{
		Mode:       JSONModeFlat,
		OmitCommas: true,
		Spacing:    JSONSpacingAfterComma,
	}

	got := renderFlatJSON(`{"a":1,"b":2}`, styles)
	assert.Equal(t, `{a:1 b:2}`, got)
}

// ---------------------------------------------------------------------------
// collectFlatPairs
// ---------------------------------------------------------------------------

func TestCollectFlatPairs(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		prefix    string
		wantKeys  []string
		wantCount int
	}{
		{
			name:      "simple_object",
			input:     `{"a":1,"b":"two"}`,
			prefix:    "",
			wantKeys:  []string{"a", "b"},
			wantCount: 2,
		},
		{
			name:      "nested_object",
			input:     `{"x":{"y":1}}`,
			prefix:    "",
			wantKeys:  []string{"x.y"},
			wantCount: 1,
		},
		{
			name:      "with_prefix",
			input:     `{"k":"v"}`,
			prefix:    "root",
			wantKeys:  []string{"root.k"},
			wantCount: 1,
		},
		{
			name:      "arrays_as_leaves",
			input:     `{"a":[1,2],"b":"c"}`,
			prefix:    "",
			wantKeys:  []string{"a", "b"},
			wantCount: 2,
		},
		{
			name:      "empty_object",
			input:     `{}`,
			prefix:    "",
			wantKeys:  nil,
			wantCount: 0,
		},
		{
			name:      "malformed_no_opening_brace",
			input:     `[1,2]`,
			prefix:    "",
			wantKeys:  nil,
			wantCount: 0,
		},
		{
			name:      "escaped_key",
			input:     `{"a\\.b":1}`,
			prefix:    "",
			wantKeys:  []string{`a\\.b`},
			wantCount: 1,
		},
		{
			name:      "deeply_nested",
			input:     `{"a":{"b":{"c":99}}}`,
			prefix:    "",
			wantKeys:  []string{"a.b.c"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pairs := collectFlatPairs([]byte(tt.input), tt.prefix)
			assert.Len(t, pairs, tt.wantCount)
			if tt.wantKeys != nil {
				for i, p := range pairs {
					assert.Equal(t, tt.wantKeys[i], p.key, "pair[%d].key", i)
				}
			}
		})
	}
}

func TestCollectFlatPairsPreservesArrayValues(t *testing.T) {
	pairs := collectFlatPairs([]byte(`{"arr":[1,"two",true]}`), "")
	require.Len(t, pairs, 1)
	assert.Equal(t, "arr", pairs[0].key)
	assert.Equal(t, `[1,"two",true]`, string(pairs[0].value))
}

func TestCollectFlatPairsMalformedKey(t *testing.T) {
	// Key that doesn't start with " — scanner should bail.
	pairs := collectFlatPairs([]byte(`{bad:1}`), "")
	assert.Empty(t, pairs)
}

// ---------------------------------------------------------------------------
// scanJSONValueEnd
// ---------------------------------------------------------------------------

func TestScanJSONValueEnd(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		startAt int
		want    int
	}{
		{
			name:    "simple_string",
			data:    `"hello"`,
			startAt: 0,
			want:    7,
		},
		{
			name:    "string_with_escapes",
			data:    `"say \"hi\""`,
			startAt: 0,
			want:    12,
		},
		{
			name:    "simple_object",
			data:    `{"a":1}`,
			startAt: 0,
			want:    7,
		},
		{
			name:    "nested_object",
			data:    `{"a":{"b":2}}`,
			startAt: 0,
			want:    13,
		},
		{
			name:    "simple_array",
			data:    `[1,2,3]`,
			startAt: 0,
			want:    7,
		},
		{
			name:    "nested_array",
			data:    `[[1],[2]]`,
			startAt: 0,
			want:    9,
		},
		{
			name:    "bare_literal_true",
			data:    `true,`,
			startAt: 0,
			want:    4,
		},
		{
			name:    "bare_literal_false",
			data:    `false}`,
			startAt: 0,
			want:    5,
		},
		{
			name:    "bare_literal_null",
			data:    `null]`,
			startAt: 0,
			want:    4,
		},
		{
			name:    "number",
			data:    `42,`,
			startAt: 0,
			want:    2,
		},
		{
			name:    "negative_number",
			data:    `-3.14}`,
			startAt: 0,
			want:    5,
		},
		{
			name:    "out_of_bounds",
			data:    `hello`,
			startAt: 10,
			want:    10,
		},
		{
			name:    "at_exact_bound",
			data:    `hello`,
			startAt: 5,
			want:    5,
		},
		{
			name:    "unterminated_string",
			data:    `"unterminated`,
			startAt: 0,
			want:    13,
		},
		{
			name:    "string_inside_object",
			data:    `{"k":"v"}`,
			startAt: 0,
			want:    9,
		},
		{
			name:    "object_with_string_containing_brace",
			data:    `{"k":"a}b"}`,
			startAt: 0,
			want:    11,
		},
		{
			name:    "value_mid_data",
			data:    `xxx"hello"yyy`,
			startAt: 3,
			want:    10,
		},
		{
			name:    "array_with_nested_strings",
			data:    `["a","b"]`,
			startAt: 0,
			want:    9,
		},
		{
			name:    "bare_literal_end_of_input",
			data:    `42`,
			startAt: 0,
			want:    2,
		},
		{
			name:    "bare_literal_with_whitespace_terminator",
			data:    "42 ",
			startAt: 0,
			want:    2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scanJSONValueEnd([]byte(tt.data), tt.startAt)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// hjsonUnquoteKey
// ---------------------------------------------------------------------------

func TestHjsonUnquoteKey(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		hjson     bool
		wantText  string
		wantUnquo bool
	}{
		{
			name:      "hjson_disabled",
			raw:       `"hello"`,
			hjson:     false,
			wantText:  `"hello"`,
			wantUnquo: false,
		},
		{
			name:      "empty_key",
			raw:       `""`,
			hjson:     true,
			wantText:  `""`,
			wantUnquo: false,
		},
		{
			name:      "normal_safe_key",
			raw:       `"hello"`,
			hjson:     true,
			wantText:  "hello",
			wantUnquo: true,
		},
		{
			name:      "key_with_comma",
			raw:       `"a,b"`,
			hjson:     true,
			wantText:  `"a,b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_open_brace",
			raw:       `"a{b"`,
			hjson:     true,
			wantText:  `"a{b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_close_brace",
			raw:       `"a}b"`,
			hjson:     true,
			wantText:  `"a}b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_open_bracket",
			raw:       `"a[b"`,
			hjson:     true,
			wantText:  `"a[b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_close_bracket",
			raw:       `"a]b"`,
			hjson:     true,
			wantText:  `"a]b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_colon",
			raw:       `"a:b"`,
			hjson:     true,
			wantText:  `"a:b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_hash",
			raw:       `"a#b"`,
			hjson:     true,
			wantText:  `"a#b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_double_quote",
			raw:       `"a\"b"`,
			hjson:     true,
			wantText:  `"a\"b"`,
			wantUnquo: false, // has backslash escape
		},
		{
			name:      "key_with_single_quote",
			raw:       `"a'b"`,
			hjson:     true,
			wantText:  `"a'b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_double_slash",
			raw:       `"a//b"`,
			hjson:     true,
			wantText:  `"a//b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_slash_star",
			raw:       `"a/*b"`,
			hjson:     true,
			wantText:  `"a/*b"`,
			wantUnquo: false,
		},
		{
			name:      "key_with_single_slash_ok",
			raw:       `"a/b"`,
			hjson:     true,
			wantText:  "a/b",
			wantUnquo: true, // single slash not followed by / or * is ok
		},
		{
			name:      "key_with_escape_sequence",
			raw:       `"a\nb"`,
			hjson:     true,
			wantText:  `"a\nb"`,
			wantUnquo: false, // has backslash
		},
		{
			name:      "short_input_single_char",
			raw:       `x`,
			hjson:     true,
			wantText:  `x`,
			wantUnquo: false, // len < 2
		},
		{
			name:      "short_input_empty",
			raw:       ``,
			hjson:     true,
			wantText:  ``,
			wantUnquo: false,
		},
		{
			name:      "key_with_control_character",
			raw:       "\"a\x01b\"",
			hjson:     true,
			wantText:  "\"a\x01b\"",
			wantUnquo: false, // control char <= ' '
		},
		{
			name:      "key_with_slash_at_end",
			raw:       `"abc/"`,
			hjson:     true,
			wantText:  "abc/",
			wantUnquo: true, // slash at end, no char after
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, unquoted := hjsonUnquoteKey(tt.raw, tt.hjson)
			assert.Equal(t, tt.wantText, text)
			assert.Equal(t, tt.wantUnquo, unquoted)
		})
	}
}

// ---------------------------------------------------------------------------
// hjsonUnquoteValue
// ---------------------------------------------------------------------------

func TestHjsonUnquoteValue(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		hjson     bool
		wantText  string
		wantUnquo bool
	}{
		{
			name:      "hjson_disabled",
			raw:       `"hello"`,
			hjson:     false,
			wantText:  `"hello"`,
			wantUnquo: false,
		},
		{
			name:      "empty_value",
			raw:       `""`,
			hjson:     true,
			wantText:  `""`,
			wantUnquo: false, // empty string must remain quoted
		},
		{
			name:      "safe_value",
			raw:       `"hello"`,
			hjson:     true,
			wantText:  "hello",
			wantUnquo: true,
		},
		{
			name:      "starts_with_space",
			raw:       `" hello"`,
			hjson:     true,
			wantText:  `" hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_tab",
			raw:       "\"\thello\"",
			hjson:     true,
			wantText:  "\"\thello\"",
			wantUnquo: false,
		},
		{
			name:      "starts_with_double_quote",
			raw:       `"\"hello"`,
			hjson:     true,
			wantText:  `"\"hello"`,
			wantUnquo: false, // has backslash escape
		},
		{
			name:      "starts_with_single_quote",
			raw:       `"'hello"`,
			hjson:     true,
			wantText:  `"'hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_hash",
			raw:       `"#hello"`,
			hjson:     true,
			wantText:  `"#hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_open_brace",
			raw:       `"{hello"`,
			hjson:     true,
			wantText:  `"{hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_close_brace",
			raw:       `"}hello"`,
			hjson:     true,
			wantText:  `"}hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_open_bracket",
			raw:       `"[hello"`,
			hjson:     true,
			wantText:  `"[hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_close_bracket",
			raw:       `"]hello"`,
			hjson:     true,
			wantText:  `"]hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_colon",
			raw:       `":hello"`,
			hjson:     true,
			wantText:  `":hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_comma",
			raw:       `",hello"`,
			hjson:     true,
			wantText:  `",hello"`,
			wantUnquo: false,
		},
		{
			name:      "trailing_space",
			raw:       `"hello "`,
			hjson:     true,
			wantText:  `"hello "`,
			wantUnquo: false,
		},
		{
			name:      "trailing_tab",
			raw:       "\"hello\t\"",
			hjson:     true,
			wantText:  "\"hello\t\"",
			wantUnquo: false,
		},
		{
			name:      "trailing_newline",
			raw:       "\"hello\n\"",
			hjson:     true,
			wantText:  "\"hello\n\"",
			wantUnquo: false,
		},
		{
			name:      "trailing_return",
			raw:       "\"hello\r\"",
			hjson:     true,
			wantText:  "\"hello\r\"",
			wantUnquo: false,
		},
		{
			name:      "control_char_in_middle",
			raw:       "\"a\x01b\"",
			hjson:     true,
			wantText:  "\"a\x01b\"",
			wantUnquo: false,
		},
		{
			name:      "keyword_ambiguous_true",
			raw:       `"true"`,
			hjson:     true,
			wantText:  `"true"`,
			wantUnquo: false,
		},
		{
			name:      "keyword_ambiguous_false",
			raw:       `"false"`,
			hjson:     true,
			wantText:  `"false"`,
			wantUnquo: false,
		},
		{
			name:      "keyword_ambiguous_null",
			raw:       `"null"`,
			hjson:     true,
			wantText:  `"null"`,
			wantUnquo: false,
		},
		{
			name:      "keyword_prefix_not_ambiguous",
			raw:       `"trueness"`,
			hjson:     true,
			wantText:  "trueness",
			wantUnquo: true, // "trueness" has rest "ness" which doesn't start with delimiter
		},
		{
			name:      "keyword_followed_by_comma",
			raw:       `"true,"`,
			hjson:     true,
			wantText:  `"true,"`,
			wantUnquo: false,
		},
		{
			name:      "keyword_followed_by_space",
			raw:       `"true stuff"`,
			hjson:     true,
			wantText:  `"true stuff"`,
			wantUnquo: false,
		},
		{
			name:      "number_ambiguous",
			raw:       `"42"`,
			hjson:     true,
			wantText:  `"42"`,
			wantUnquo: false,
		},
		{
			name:      "negative_number_ambiguous",
			raw:       `"-5"`,
			hjson:     true,
			wantText:  `"-5"`,
			wantUnquo: false,
		},
		{
			name:      "negative_not_followed_by_digit",
			raw:       `"-abc"`,
			hjson:     true,
			wantText:  "-abc",
			wantUnquo: true,
		},
		{
			name:      "starts_with_slash_not_comment",
			raw:       `"/hello"`,
			hjson:     true,
			wantText:  "/hello",
			wantUnquo: true, // single slash not followed by / or *
		},
		{
			name:      "starts_with_double_slash",
			raw:       `"//hello"`,
			hjson:     true,
			wantText:  `"//hello"`,
			wantUnquo: false,
		},
		{
			name:      "starts_with_slash_star",
			raw:       `"/*hello"`,
			hjson:     true,
			wantText:  `"/*hello"`,
			wantUnquo: false,
		},
		{
			name:      "short_input_single_char",
			raw:       `x`,
			hjson:     true,
			wantText:  `x`,
			wantUnquo: false, // len < 2
		},
		{
			name:      "escape_sequence_in_value",
			raw:       `"line\nbreak"`,
			hjson:     true,
			wantText:  `"line\nbreak"`,
			wantUnquo: false, // has backslash
		},
		{
			name:      "single_slash_value",
			raw:       `"/"`,
			hjson:     true,
			wantText:  "/",
			wantUnquo: true, // single char slash, len(s)=1 so s[1] check doesn't apply
		},
		{
			name:      "false_followed_by_hash",
			raw:       `"false#"`,
			hjson:     true,
			wantText:  `"false#"`,
			wantUnquo: false, // "false" + "#" is a delimiter
		},
		{
			name:      "null_followed_by_slash",
			raw:       `"null/"`,
			hjson:     true,
			wantText:  `"null/"`,
			wantUnquo: false, // "null" + "/" is a delimiter
		},
		{
			name:      "just_minus",
			raw:       `"-"`,
			hjson:     true,
			wantText:  "-",
			wantUnquo: true, // "-" alone: s[0]=='-', len(s)==1, so second check fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, unquoted := hjsonUnquoteValue(tt.raw, tt.hjson)
			assert.Equal(t, tt.wantText, text)
			assert.Equal(t, tt.wantUnquo, unquoted)
		})
	}
}

// ---------------------------------------------------------------------------
// isJSONSpace
// ---------------------------------------------------------------------------

func TestIsJSONSpace(t *testing.T) {
	assert.True(t, isJSONSpace(' '))
	assert.True(t, isJSONSpace('\t'))
	assert.True(t, isJSONSpace('\n'))
	assert.True(t, isJSONSpace('\r'))
	assert.False(t, isJSONSpace('a'))
	assert.False(t, isJSONSpace('0'))
	assert.False(t, isJSONSpace(0))
}

// ---------------------------------------------------------------------------
// emitStyled
// ---------------------------------------------------------------------------

func TestEmitStyled(t *testing.T) {
	t.Run("nil_style", func(t *testing.T) {
		var buf strings.Builder
		emitStyled(&buf, "text", nil)
		assert.Equal(t, "text", buf.String())
	})

	t.Run("with_style", func(t *testing.T) {
		style := lipgloss.NewStyle().Bold(true)
		var buf strings.Builder
		emitStyled(&buf, "text", &style)
		assert.Equal(t, style.Render("text"), buf.String())
	})
}

// ---------------------------------------------------------------------------
// resolveStringToken
// ---------------------------------------------------------------------------

func TestResolveStringToken(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	strStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	styles := &JSONStyles{
		Key:    new(keyStyle),
		String: new(strStyle),
	}

	t.Run("key_token_no_hjson", func(t *testing.T) {
		text, style := resolveStringToken(`"mykey"`, true, false, styles)
		assert.Equal(t, `"mykey"`, text)
		assert.Equal(t, styles.Key, style)
	})

	t.Run("key_token_hjson_unquoted", func(t *testing.T) {
		text, style := resolveStringToken(`"mykey"`, true, true, styles)
		assert.Equal(t, "mykey", text)
		assert.Equal(t, styles.Key, style)
	})

	t.Run("value_token_no_hjson", func(t *testing.T) {
		text, style := resolveStringToken(`"val"`, false, false, styles)
		assert.Equal(t, `"val"`, text)
		assert.Equal(t, styles.String, style)
	})

	t.Run("value_token_hjson_unquoted", func(t *testing.T) {
		text, style := resolveStringToken(`"val"`, false, true, styles)
		assert.Equal(t, "val", text)
		assert.Equal(t, styles.String, style)
	})
}

func TestResolveNumberStyle(t *testing.T) {
	num := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")))
	pos := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")))
	neg := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")))
	zero := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")))
	float := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff00ff")))
	integer := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#00ffff")))

	tests := []struct {
		name   string
		val    string
		styles JSONStyles
		want   Style
	}{
		// Base fallback
		{
			name:   "base_number",
			val:    "42",
			styles: JSONStyles{Number: num},
			want:   num,
		},
		// Sign-based
		{
			name:   "positive_integer",
			val:    "42",
			styles: JSONStyles{Number: num, NumberPositive: pos},
			want:   pos,
		},
		{
			name:   "negative_integer",
			val:    "-7",
			styles: JSONStyles{Number: num, NumberNegative: neg},
			want:   neg,
		},
		{
			name:   "negative_falls_back_to_number",
			val:    "-7",
			styles: JSONStyles{Number: num},
			want:   num,
		},
		// Zero fallback chain
		{
			name:   "zero_uses_NumberZero",
			val:    "0",
			styles: JSONStyles{Number: num, NumberPositive: pos, NumberZero: zero},
			want:   zero,
		},
		{
			name:   "zero_falls_back_to_NumberPositive",
			val:    "0",
			styles: JSONStyles{Number: num, NumberPositive: pos},
			want:   pos,
		},
		{
			name:   "zero_falls_back_to_Number",
			val:    "0",
			styles: JSONStyles{Number: num},
			want:   num,
		},
		{
			name:   "negative_zero",
			val:    "-0",
			styles: JSONStyles{Number: num, NumberZero: zero},
			want:   zero,
		},
		{
			name:   "zero_float",
			val:    "0.0",
			styles: JSONStyles{Number: num, NumberZero: zero},
			want:   zero,
		},
		// Type-based (sign style absent)
		{
			name:   "float",
			val:    "3.14",
			styles: JSONStyles{Number: num, NumberFloat: float},
			want:   float,
		},
		{
			name:   "integer_type",
			val:    "99",
			styles: JSONStyles{Number: num, NumberInteger: integer},
			want:   integer,
		},
		{
			name:   "exponent_is_float",
			val:    "1e10",
			styles: JSONStyles{Number: num, NumberFloat: float},
			want:   float,
		},
		// Sign takes priority over type
		{
			name:   "positive_float_sign_wins",
			val:    "3.14",
			styles: JSONStyles{Number: num, NumberPositive: pos, NumberFloat: float},
			want:   pos,
		},
		{
			name:   "negative_float_sign_wins",
			val:    "-1.5",
			styles: JSONStyles{Number: num, NumberNegative: neg, NumberFloat: float},
			want:   neg,
		},
		// Nil styles fall through to Number
		{
			name:   "no_sub_styles",
			val:    "42",
			styles: JSONStyles{Number: num},
			want:   num,
		},
		{
			name:   "all_nil_returns_nil_number",
			val:    "42",
			styles: JSONStyles{},
			want:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveNumberStyle(tt.val, &tt.styles)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHighlightJSONNumberSubStyles(t *testing.T) {
	pos := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")))
	neg := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")))
	zero := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")))
	float := new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff00ff")))

	styles := &JSONStyles{
		NumberPositive: pos,
		NumberNegative: neg,
		NumberZero:     zero,
		NumberFloat:    float,
	}

	result := highlightJSON(`{"a":42,"b":-7,"c":0,"d":3.14}`, styles)

	assert.Contains(t, result, pos.Render("42"))
	assert.Contains(t, result, neg.Render("-7"))
	assert.Contains(t, result, zero.Render("0"))
	assert.Contains(t, result, float.Render("3.14"))
}
