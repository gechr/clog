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

	// Should contain ANSI escape sequences.
	assert.Contains(t, got, "\x1b[")
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

	// "resource" should be styled as BlockType (color 2).
	assert.Contains(t, got, "\x1b[32m", "block type should be styled")
	// "enabled" should be styled as Key (color 1).
	assert.Contains(t, got, "\x1b[31m", "key should be styled")
	// "true" should be styled as BoolTrue (color 3).
	assert.Contains(t, got, "\x1b[33m", "bool should be styled")
}

func TestHighlightInvalidHCL(t *testing.T) {
	input := `{{{{ not valid`
	got := hcl.Highlight(input, &style.HCL{})
	assert.Equal(t, input, got)
}
