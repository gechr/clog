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
	assert.Equal(t, "\x1b[31mport\x1b[m = \x1b[32m8080\x1b[m\n", got)
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

func TestHighlightWhitespaceOnly(t *testing.T) {
	input := "   \t  "
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightNewlineOnly(t *testing.T) {
	input := "\n\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightTableHeaderWithComment(t *testing.T) {
	input := "[server] # main server\nhost = \"localhost\"\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightArrayTableWithComment(t *testing.T) {
	input := "[[items]] # list of items\nname = \"widget\"\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightValueAtEndOfInput(t *testing.T) {
	// Value line with no trailing newline.
	input := "key ="
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightUnrecognizedValue(t *testing.T) {
	// A bare value that doesn't match any known type falls to default.
	input := "key = @unknown\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightInlineCommentAfterValue(t *testing.T) {
	input := "name = \"alice\" # user name\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightArrayWithComments(t *testing.T) {
	input := "tags = [\n  # first\n  \"a\",\n  # second\n  \"b\"\n]\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightUnterminatedArray(t *testing.T) {
	input := "tags = [\"a\", \"b\""
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightNestedArrays(t *testing.T) {
	input := "matrix = [[1, 2], [3, 4]]\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightUnterminatedInlineTable(t *testing.T) {
	input := "point = {x = 1, y = 2"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightMultilineBasicStringWithEscape(t *testing.T) {
	input := "desc = \"\"\"hello\\nworld\"\"\"\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightMultilineLiteralString(t *testing.T) {
	input := "regex = '''\\d+\\.\\d+'''\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightUnterminatedMultilineString(t *testing.T) {
	input := "desc = \"\"\"hello\nworld"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightUnterminatedSingleLineString(t *testing.T) {
	input := "name = \"hello\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightEscapeAtEndOfString(t *testing.T) {
	input := "name = \"hello\\"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightSignedNumbers(t *testing.T) {
	input := "pos = +42\nneg = -3.14\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightSpecialFloats(t *testing.T) {
	input := "a = +inf\nb = -inf\nc = +nan\nd = -nan\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightHexOctBin(t *testing.T) {
	input := "h = 0xCAFE\no = 0o777\nb = 0b11001\n"
	got := toml.Highlight(input, &style.TOML{})
	assert.Equal(t, input, got)
}

func TestHighlightSignedNumbersStyled(t *testing.T) {
	floatStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	intStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("4"))

	styles := &style.TOML{Float: new(floatStyle), Integer: new(intStyle)}

	got := toml.Highlight("a = +inf\nb = -nan\nc = +42\nd = -3.14\n", styles)
	assert.Equal(
		t,
		"a = \x1b[33m+inf\x1b[m\nb = \x1b[33m-nan\x1b[m\nc = \x1b[34m+42\x1b[m\nd = \x1b[33m-3.14\x1b[m\n",
		got,
	)
}

func TestHighlightHexOctBinStyled(t *testing.T) {
	intStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styles := &style.TOML{Integer: new(intStyle)}

	got := toml.Highlight("h = 0xDEAD\no = 0o755\nb = 0b1010\n", styles)
	assert.Equal(
		t,
		"h = \x1b[35m0xDEAD\x1b[m\no = \x1b[35m0o755\x1b[m\nb = \x1b[35m0b1010\x1b[m\n",
		got,
	)
}
