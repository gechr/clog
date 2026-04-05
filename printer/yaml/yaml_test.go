package yaml_test

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer/yaml"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
)

func TestHighlightNilStyles(t *testing.T) {
	input := "key: value"
	assert.Equal(t, input, yaml.Highlight(input, nil))
}

func TestHighlightEmpty(t *testing.T) {
	assert.Empty(t, yaml.Highlight("", &style.YAML{}))
}

func TestHighlightBasicMapping(t *testing.T) {
	got := yaml.Highlight("name: alice", &style.YAML{})
	assert.Equal(t, "name: alice", got)
}

func TestHighlightPreservesFormatting(t *testing.T) {
	input := `name: alice
nested:
  key: value
  list:
    - one
    - two`
	got := yaml.Highlight(input, &style.YAML{})
	assert.Equal(t, input, got)
}

func TestHighlightScalarTypes(t *testing.T) {
	input := `str: hello
num: 42
flt: 3.14
hex: 0xFF
yes: true
no: false
empty: null`
	got := yaml.Highlight(input, &style.YAML{})
	assert.Equal(t, input, got)
}

func TestHighlightComments(t *testing.T) {
	input := `# header comment
key: value`
	got := yaml.Highlight(input, &style.YAML{})
	assert.Equal(t, input, got)
}

func TestHighlightAnchorsAndAliases(t *testing.T) {
	input := `defaults: &defaults
  host: localhost
dev:
  <<: *defaults`
	got := yaml.Highlight(input, &style.YAML{})
	assert.Equal(t, input, got)
}

func TestHighlightDocumentMarkers(t *testing.T) {
	input := `---
key: value
...`
	got := yaml.Highlight(input, &style.YAML{})
	assert.Equal(t, input, got)
}

func TestHighlightQuotedStrings(t *testing.T) {
	input := `single: 'hello'
double: "world"`
	got := yaml.Highlight(input, &style.YAML{})
	assert.Equal(t, input, got)
}

func TestHighlightQuotedKeysGetKeyStyle(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	strStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	styles := &style.YAML{Key: new(keyStyle), String: new(strStyle)}

	got := yaml.Highlight("'a b': val", styles)
	assert.Equal(t, "\x1b[31m'a b'\x1b[m: \x1b[32mval\x1b[m", got)

	got = yaml.Highlight(`"a b": val`, styles)
	assert.Equal(t, "\x1b[31m\"a b\"\x1b[m: \x1b[32mval\x1b[m", got)
}

func TestHighlightStyledOutput(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	styles := &style.YAML{Key: new(keyStyle), Number: new(numStyle)}
	got := yaml.Highlight("port: 8080", styles)
	assert.Equal(t, "\x1b[31mport\x1b[m: \x1b[32m8080\x1b[m", got)
}

func TestHighlightRoundTripFallback(t *testing.T) {
	// If the lexer mangles the input, Highlight should return it unchanged.
	// Use a case known to not round-trip: explicit key syntax.
	input := "? [a, b]\n: 1"
	got := yaml.Highlight(input, &style.YAML{})
	assert.Equal(t, input, got)
}

func TestHighlightFlowStyle(t *testing.T) {
	input := "{a: 1, b: 2}"
	got := yaml.Highlight(input, &style.YAML{})
	// Should either preserve or fall back - not crash.
	assert.NotEmpty(t, got)
}

func TestHighlightInvalidYAML(t *testing.T) {
	input := ": : : not valid"
	got := yaml.Highlight(input, &style.YAML{})
	// Should not panic; returns something.
	assert.NotEmpty(t, got)
}

func TestHighlightMultilineString(t *testing.T) {
	input := `desc: |
  line one
  line two`
	got := yaml.Highlight(input, &style.YAML{})
	// Should contain the multi-line content.
	assert.True(t, strings.Contains(got, "line one") || got == input)
}

func TestHighlightMergeKeyGetsKeyStyle(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles := &style.YAML{Key: new(keyStyle)}

	input := "defaults: &defaults\n  host: localhost\ndev:\n  <<: *defaults"
	got := yaml.Highlight(input, styles)
	assert.Equal(
		t,
		"\x1b[31mdefaults\x1b[m: &defaults\n  \x1b[31mhost\x1b[m: localhost\n\x1b[31mdev\x1b[m:\n  \x1b[31m<<\x1b[m: *defaults",
		got,
	)
}

func TestHighlightAnchorNameStyle(t *testing.T) {
	anchorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles := &style.YAML{Anchor: new(anchorStyle)}

	input := "defaults: &myanchor\n  host: localhost"
	got := yaml.Highlight(input, styles)
	assert.Equal(t, "defaults: \x1b[33m&\x1b[m\x1b[33mmyanchor\x1b[m\n  host: localhost", got)
}

func TestHighlightAliasNameStyle(t *testing.T) {
	aliasStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styles := &style.YAML{Alias: new(aliasStyle)}

	input := "defaults: &defaults\n  host: localhost\ndev:\n  <<: *defaults"
	got := yaml.Highlight(input, styles)
	assert.Equal(
		t,
		"defaults: &defaults\n  host: localhost\ndev:\n  <<: \x1b[35m*\x1b[m\x1b[35mdefaults\x1b[m",
		got,
	)
}

func TestHighlightTagStyle(t *testing.T) {
	tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles := &style.YAML{Tag: new(tagStyle)}

	input := "typed: !!str 42"
	got := yaml.Highlight(input, styles)
	assert.Equal(t, "typed: \x1b[36m!!str\x1b[m 42", got)
}

func TestHighlightBoolFalseVariants(t *testing.T) {
	boolFalseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles := &style.YAML{BoolFalse: new(boolFalseStyle)}

	for _, v := range []string{"false", "False", "FALSE"} {
		got := yaml.Highlight("key: "+v, styles)
		assert.Equal(t, "key: \x1b[31m"+v+"\x1b[m", got)
	}
}

func TestHighlightBoolTrueVariants(t *testing.T) {
	boolTrueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles := &style.YAML{BoolTrue: new(boolTrueStyle)}

	for _, v := range []string{"true", "True", "TRUE"} {
		got := yaml.Highlight("key: "+v, styles)
		assert.Equal(t, "key: \x1b[32m"+v+"\x1b[m", got)
	}
}

func TestHighlightNullStyle(t *testing.T) {
	nullStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	styles := &style.YAML{Null: new(nullStyle)}

	for _, v := range []string{"null", "~"} {
		got := yaml.Highlight("key: "+v, styles)
		assert.Equal(t, "key: \x1b[34m"+v+"\x1b[m", got)
	}
}

func TestHighlightPunctuationStyle(t *testing.T) {
	punctStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles := &style.YAML{Punctuation: new(punctStyle)}

	assert.Equal(t, "a\x1b[36m:\x1b[m b", yaml.Highlight("a: b", styles))
	assert.Equal(
		t,
		"items\x1b[36m:\x1b[m\n  \x1b[36m-\x1b[m one",
		yaml.Highlight("items:\n  - one", styles),
	)
	assert.Equal(
		t,
		"\x1b[36m---\x1b[m\nkey\x1b[36m:\x1b[m val\n\x1b[36m...\x1b[m",
		yaml.Highlight("---\nkey: val\n...", styles),
	)
	assert.Equal(
		t,
		"\x1b[36m{\x1b[ma\x1b[36m:\x1b[m 1\x1b[36m,\x1b[m b\x1b[36m:\x1b[m 2\x1b[36m}\x1b[m",
		yaml.Highlight("{a: 1, b: 2}", styles),
	)
	assert.Equal(
		t,
		"items\x1b[36m:\x1b[m \x1b[36m[\x1b[m1\x1b[36m,\x1b[m 2\x1b[36m]\x1b[m",
		yaml.Highlight("items: [1, 2]", styles),
	)
}

func TestHighlightNumberTypes(t *testing.T) {
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styles := &style.YAML{Number: new(numStyle)}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"integer", "val: 42", "val: \x1b[33m42\x1b[m"},
		{"hex", "val: 0xFF", "val: \x1b[33m0xFF\x1b[m"},
		{"octal", "val: 0o77", "val: \x1b[33m0o77\x1b[m"},
		{"binary", "val: 0b1010", "val: \x1b[33m0b1010\x1b[m"},
		{"float", "val: 3.14", "val: \x1b[33m3.14\x1b[m"},
		{"infinity", "val: .inf", "val: \x1b[33m.inf\x1b[m"},
		{"nan", "val: .nan", "val: \x1b[33m.nan\x1b[m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, yaml.Highlight(tt.input, styles))
		})
	}
}
