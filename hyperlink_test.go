package clog

import (
	"io"
	"path/filepath"
	"testing"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestPathLinkTextDefault(t *testing.T) {
	// Swap the whole Default logger: init() may have applied ambient
	// CLOG_HYPERLINK_* env vars to its FieldFormats.
	origDefault := Default
	defer func() { Default = origDefault }()

	Default = New(NewOutput(io.Discard, ColorAlways))

	// Visible label differs from the linked path: the URL targets the full
	// path while the label shows an abbreviated form.
	got := PathLinkText("~/bin/foo", "/home/user/bin/foo", 0)
	want := "\x1b]8;;file:///home/user/bin/foo\x1b\\~/bin/foo\x1b]8;;\x1b\\"

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

func TestOutputHyperlinkEmptyURL(t *testing.T) {
	output := NewOutput(io.Discard, ColorAlways)
	got := output.Hyperlink("", "text")

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

func TestOutputPathLinkText(t *testing.T) {
	output := NewOutput(io.Discard, ColorAlways)

	// Label is used verbatim; the URL still resolves from the full path.
	got := output.PathLinkText("~/bin/foo", "/home/user/bin/foo", 0, 0)
	want := "\x1b]8;;file:///home/user/bin/foo\x1b\\~/bin/foo\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkTextNever(t *testing.T) {
	output := NewOutput(io.Discard, ColorNever)

	// With colors off, only the label is emitted - no path, no OSC 8.
	got := output.PathLinkText("~/bin/foo", "/home/user/bin/foo", 42, 0)

	assert.Equal(t, "~/bin/foo", got)
}

func TestOutputPathLinkTextDisabled(t *testing.T) {
	output := linkOutput(hyperlink.Config{Enabled: false})
	got := output.PathLinkText("~/bin/foo", "/home/user/bin/foo", 42, 0)

	assert.Equal(t, "~/bin/foo", got)
}

func TestOutputPathLinkRelativePath(t *testing.T) {
	output := NewOutput(io.Discard, ColorAlways)

	// Relative path should be resolved to absolute in the URL.
	got := output.PathLink("test.go", 0, 0)

	abs, err := filepath.Abs("test.go")
	require.NoError(t, err)
	// The URL targets the resolved absolute path; the visible label stays
	// the relative path.
	assert.Equal(t, "\x1b]8;;file://"+abs+"\x1b\\test.go\x1b]8;;\x1b\\", got)
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
