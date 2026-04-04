package toml_test

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/gechr/clog/printer/toml"
	"github.com/gechr/clog/style"
	"github.com/stretchr/testify/assert"
)

func TestHighlightNilStyles(t *testing.T) {
	input := "key = \"value\""
	assert.Equal(t, input, toml.Highlight(input, nil))
}

func TestHighlightEmpty(t *testing.T) {
	assert.Empty(t, toml.Highlight("", &style.TOML{}))
}

func TestHighlightBasicKeyValue(t *testing.T) {
	input := `name = "alice"
port = 8080
debug = true
ratio = 3.14
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightComment(t *testing.T) {
	input := `# this is a comment
key = "value"
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightInlineComment(t *testing.T) {
	input := "port = 8080 # listen port\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightTableHeader(t *testing.T) {
	input := `[server]
host = "localhost"
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightArrayTable(t *testing.T) {
	input := `[[features]]
name = "dark-mode"
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightDottedKey(t *testing.T) {
	input := `server.host = "localhost"
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightQuotedKey(t *testing.T) {
	input := `"quoted key" = "value"
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightMultilineString(t *testing.T) {
	input := `desc = """
hello
world"""
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightLiteralString(t *testing.T) {
	input := "path = 'C:\\Users\\alice'\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightBooleans(t *testing.T) {
	input := `yes = true
no = false
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightNumbers(t *testing.T) {
	input := `int = 42
hex = 0xDEAD
oct = 0o755
bin = 0b1010
float = 3.14
inf_val = inf
nan_val = nan
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightDateTime(t *testing.T) {
	input := `date = 2024-01-15
time = 10:30:00
datetime = 2024-01-15T10:30:00Z
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightArray(t *testing.T) {
	input := `tags = ["prod", "staging"]
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightInlineTable(t *testing.T) {
	input := `point = {x = 1, y = 2}
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightStyledOutput(t *testing.T) {
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	intStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

	styles := &style.TOML{Key: new(keyStyle), Integer: new(intStyle)}
	got := toml.Highlight("port = 8080\n", styles)

	assert.Contains(t, got, "\x1b[")
}

func TestHighlightPreservesFormatting(t *testing.T) {
	input := `# Config
[server]
  host = "localhost"
  port = 8080

[database]
  name = "myapp"
`
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}
