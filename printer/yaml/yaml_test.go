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

	// Single-quoted key should use Key style, not String.
	got := yaml.Highlight("'a b': val", styles)
	assert.Contains(t, got, "\x1b[31m", "single-quoted key should have Key ANSI color")

	// Double-quoted key
	got = yaml.Highlight(`"a b": val`, styles)
	assert.Contains(t, got, "\x1b[31m", "double-quoted key should have Key ANSI color")
}

func TestHighlightStyledOutput(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	styles := &style.YAML{Key: new(keyStyle), Number: new(numStyle)}
	got := yaml.Highlight("port: 8080", styles)

	// Should contain ANSI escape sequences.
	assert.Contains(t, got, "\x1b[")
}

func TestHighlightRoundTripFallback(t *testing.T) {
	// If the lexer mangles the input, Highlight should return it unchanged.
	// Use a case known to not round-trip: explicit key syntax.
	input := "? [a, b]\n: 1"
	got := yaml.Highlight(input, &style.YAML{})
	// Should either round-trip or fall back to original - never produce mangled output.
	if got != input {
		// If it didn't fall back, verify it at least doesn't lose content.
		assert.Contains(t, got, "a")
		assert.Contains(t, got, "b")
	}
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
