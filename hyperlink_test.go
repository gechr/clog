package clog

import (
	"io"
	"testing"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/stretchr/testify/assert"
)

// linkOutput returns a ColorAlways output configured with the given
// hyperlink config.
func linkOutput(c hyperlink.Config) *Output {
	out := NewOutput(io.Discard, ColorAlways)
	out.setHyperlinks(c)
	return out
}

func TestHyperlinkDefault(t *testing.T) {
	// Swap the whole Default logger: init() may have applied ambient
	// CLOG_HYPERLINK_* env vars to its FieldFormats.
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(NewOutput(io.Discard, ColorAlways))

	got := Hyperlink("https://example.com", "click")
	want := "\x1b]8;;https://example.com\x1b\\click\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestPathLinkDefault(t *testing.T) {
	// Swap the whole Default logger: init() may have applied ambient
	// CLOG_HYPERLINK_* env vars to its FieldFormats.
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(NewOutput(io.Discard, ColorAlways))

	got := PathLink("/tmp/test.go", 42)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputHyperlinkAlways(t *testing.T) {
	output := NewOutput(io.Discard, ColorAlways)
	got := output.Hyperlink("https://example.com", "text")
	want := "\x1b]8;;https://example.com\x1b\\text\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputHyperlinkNever(t *testing.T) {
	output := NewOutput(io.Discard, ColorNever)
	got := output.Hyperlink("https://example.com", "text")

	assert.Equal(t, "text", got)
}

func TestOutputHyperlinkDisabled(t *testing.T) {
	output := linkOutput(hyperlink.Config{Enabled: false})
	got := output.Hyperlink("https://example.com", "text")

	assert.Equal(t, "text", got)
}

func TestOutputPathLinkNever(t *testing.T) {
	output := NewOutput(io.Discard, ColorNever)

	tests := []struct {
		name   string
		line   int
		column int
		want   string
	}{
		{name: "with_line", line: 42, want: "/some/file.go:42"},
		{name: "no_line", want: "/some/file.go"},
		{name: "with_column", line: 42, column: 10, want: "/some/file.go:42:10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := output.PathLink("/some/file.go", tt.line, tt.column)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOutputPathLinkDisabled(t *testing.T) {
	output := linkOutput(hyperlink.Config{Enabled: false})
	got := output.PathLink("/tmp/test.go", 42, 0)

	assert.Equal(t, "/tmp/test.go:42", got)
}

func TestOutputPathLinkDefault(t *testing.T) {
	// No config pushed: default is enabled with plain file:// URLs.
	output := NewOutput(io.Discard, ColorAlways)
	got := output.PathLink("/tmp/test.go", 42, 0)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkRelativePath(t *testing.T) {
	output := NewOutput(io.Discard, ColorAlways)

	// Relative path should be resolved to absolute in the URL.
	got := output.PathLink("test.go", 0, 0)

	assert.Contains(t, got, "\x1b]8;;file:///")
	// Display text should be the relative path.
	assert.Contains(t, got, "\x1b\\test.go\x1b]8;;\x1b\\")
}

func TestOutputPathLinkDirectory(t *testing.T) {
	// Line and file formats do not apply to directories, which fall back
	// to file:// when no dir/path format is set.
	output := linkOutput(hyperlink.Config{
		Enabled:    true,
		FileFormat: "vscode://file{path}",
		LineFormat: "vscode://file{path}:{line}",
	})

	dir := t.TempDir()
	got := output.PathLink(dir, 0, 0)
	want := "\x1b]8;;file://" + dir + "\x1b\\" + dir + "\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkLineFormat(t *testing.T) {
	output := linkOutput(hyperlink.Config{
		Enabled:    true,
		LineFormat: "vscode://file{path}:{line}",
	})

	got := output.PathLink("/tmp/test.go", 10, 0)
	want := "\x1b]8;;vscode://file/tmp/test.go:10\x1b\\/tmp/test.go:10\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkPathFormat(t *testing.T) {
	output := linkOutput(hyperlink.Config{
		Enabled:    true,
		PathFormat: "vscode://file{path}",
	})

	got := output.PathLink("/tmp/test.go", 0, 0)
	want := "\x1b]8;;vscode://file/tmp/test.go\x1b\\/tmp/test.go\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkColumnFormat(t *testing.T) {
	output := linkOutput(hyperlink.Config{
		Enabled:      true,
		ColumnFormat: "vscode://file{path}:{line}:{column}",
	})

	got := output.PathLink("/tmp/test.go", 42, 10)
	want := "\x1b]8;;vscode://file/tmp/test.go:42:10\x1b\\/tmp/test.go:42:10\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkColumnDefault(t *testing.T) {
	// No column/line format configured: column links fall back to file://.
	output := NewOutput(io.Discard, ColorAlways)
	got := output.PathLink("/tmp/test.go", 42, 10)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42:10\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestSetFieldFormatsHyperlinkPreset(t *testing.T) {
	logger := New(NewOutput(io.Discard, ColorAlways))

	f := DefaultFieldFormats()
	f.HyperlinkLineFormat = "vscode"
	logger.SetFieldFormats(f)

	out := logger.Output()
	got := out.PathLink("/tmp/test.go", 42, 0)
	want := "\x1b]8;;vscode://file/tmp/test.go:42\x1b\\/tmp/test.go:42\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)

	// Preset names are expanded per slot on store.
	assert.Equal(t, "vscode://file{path}:{line}", logger.FieldFormats().HyperlinkLineFormat)
}

func TestSetFieldFormatsHyperlinkDisabled(t *testing.T) {
	logger := New(NewOutput(io.Discard, ColorAlways))

	f := DefaultFieldFormats()
	f.HyperlinkEnabled = false
	logger.SetFieldFormats(f)

	out := logger.Output()

	assert.Equal(t, "text", out.Hyperlink("https://example.com", "text"))
	assert.Equal(t, "/tmp/test.go:42", out.PathLink("/tmp/test.go", 42, 0))
}
