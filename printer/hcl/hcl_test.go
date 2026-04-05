package hcl_test

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer/hcl"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
)

func TestHighlightNilStyles(t *testing.T) {
	input := `key = "value"`
	assert.Equal(t, input, hcl.Highlight(input, nil))
}

func TestHighlightEmpty(t *testing.T) {
	assert.Empty(t, hcl.Highlight("", &style.HCL{}))
}

func TestHighlightBasicAttribute(t *testing.T) {
	input := `name = "alice"`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightPreservesFormatting(t *testing.T) {
	input := `resource "aws_instance" "web" {
  ami           = "ami-12345678"
  instance_type = "t2.micro"
  count         = 3
}`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightAllValueTypes(t *testing.T) {
	input := `str     = "hello"
num     = 42
float   = 3.14
yes     = true
no      = false
nothing = null`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightComments(t *testing.T) {
	input := `# header comment
key = "value" // inline comment`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightBlock(t *testing.T) {
	input := `resource "aws_instance" "web" {
  ami = "ami-12345678"
}`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightNestedBlocks(t *testing.T) {
	input := `resource "aws_instance" "web" {
  provisioner "local-exec" {
    command = "echo hello"
  }
}`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightStyledOutput(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	numStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	styles := &style.HCL{Key: new(keyStyle), Number: new(numStyle)}
	got := hcl.Highlight("port = 8080", styles)
	assert.Equal(t, "\x1b[31mport\x1b[m = \x1b[32m8080\x1b[m", got)
}

func TestHighlightIdentContext(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	blockStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	boolStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	styles := &style.HCL{
		Key:       new(keyStyle),
		BlockType: new(blockStyle),
		BoolTrue:  new(boolStyle),
	}

	got := hcl.Highlight(`resource "type" "name" {
  enabled = true
}`, styles)

	assert.Equal(
		t,
		"\x1b[32mresource\x1b[m \"type\" \"name\" {\n  \x1b[31menabled\x1b[m = \x1b[33mtrue\x1b[m\n}",
		got,
	)
}

func TestHighlightInvalidHCL(t *testing.T) {
	input := `{{{{ not valid`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightBoolFalse(t *testing.T) {
	falseStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styles := &style.HCL{BoolFalse: new(falseStyle)}
	got := hcl.Highlight("enabled = false", styles)
	assert.Equal(t, "enabled = \x1b[35mfalse\x1b[m", got)
}

func TestHighlightNestedKey(t *testing.T) {
	nestedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styles := &style.HCL{NestedKey: new(nestedStyle)}
	input := `outer "label" {
  inner "label" {
    key = "value"
  }
}`
	got := hcl.Highlight(input, styles)
	assert.Equal(
		t,
		"outer \"label\" {\n  inner \"label\" {\n    \x1b[36mkey\x1b[m = \"value\"\n  }\n}",
		got,
	)
}

func TestHighlightIdentAtEnd(t *testing.T) {
	// A bare identifier at end of input with no following non-newline token
	// exercises nextNonNewline returning nil and resolveIdentStyle next==nil path.
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles := &style.HCL{Key: new(keyStyle)}
	input := "bareident"
	got := hcl.Highlight(input, styles)
	// "bareident" is an ident with next==nil, so it returns nil (unstyled).
	assert.Equal(t, input, got)
}

func TestHighlightIdentFollowedByNewlinesOnly(t *testing.T) {
	// Identifier followed only by newlines exercises nextNonNewline returning nil.
	input := "myident\n\n"
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}

func TestHighlightHeredoc(t *testing.T) {
	strStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styles := &style.HCL{String: new(strStyle)}
	input := `content = <<-EOT
hello world
EOT`
	got := hcl.Highlight(input, styles)
	assert.Equal(
		t,
		"content = \x1b[32m<<-EOT\x1b[m\n\x1b[32mhello world\x1b[m\n\x1b[32mEOT\x1b[m",
		got,
	)
}

func TestHighlightTemplateInterpUnstyled(t *testing.T) {
	// Template interpolation tokens should remain unstyled even with all styles set.
	strStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles := &style.HCL{
		String: new(strStyle),
		Key:    new(keyStyle),
	}
	input := `greeting = "hello ${name}"`
	got := hcl.Highlight(input, styles)
	assert.Equal(
		t,
		"\x1b[31mgreeting\x1b[m = \x1b[32m\"\x1b[m\x1b[32mhello\x1b[m ${name}\x1b[32m\"\x1b[m",
		got,
	)
}

func TestHighlightIdentCatchAll(t *testing.T) {
	// An identifier followed by a closing brace exercises the large catch-all
	// case in resolveIdentStyle which returns nil.
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	styles := &style.HCL{Key: new(keyStyle)}
	// In "a { myident }", "myident" is followed by "}" which hits the catch-all.
	input := `a {
  myident
}`
	got := hcl.Highlight(input, styles)
	assert.Equal(t, "a {\n  myident\n}", got)
}
