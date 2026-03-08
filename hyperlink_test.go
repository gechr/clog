package clog

import (
	"io"
	"testing"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/stretchr/testify/assert"
)

// saveFormats saves and returns a cleanup func that restores all format pointers.
func saveFormats(t *testing.T) {
	t.Helper()

	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })
}

func clearFormats(t *testing.T) {
	t.Helper()

	saveFormats(t)
	hyperlink.ClearFormats()
}

// withColorsEnabled sets Default to ColorAlways and enables hyperlinks for the
// duration of the test. Restores the original Default and hyperlinks flag on cleanup.
func withColorsEnabled(t *testing.T) {
	t.Helper()

	origDefault := Default
	origEnabled := hyperlink.Enabled()

	t.Cleanup(func() {
		Default = origDefault
		hyperlink.SetEnabled(origEnabled)
	})

	Default = New(NewOutput(io.Discard, ColorAlways))
	hyperlink.SetEnabled(true)
}

func TestHyperlinkColorsDisabled(t *testing.T) {
	// In test environment, ColorsDisabled() returns true (no terminal).
	got := Hyperlink("https://example.com", "click here")
	assert.Equal(t, "click here", got)
}

func TestHyperlinkEnabled(t *testing.T) {
	withColorsEnabled(t)

	got := Hyperlink("https://example.com", "click")
	want := "\x1b]8;;https://example.com\x1b\\click\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestHyperlinkDisabledViaFlag(t *testing.T) {
	origDefault := Default
	origEnabled := hyperlink.Enabled()

	defer func() {
		Default = origDefault
		hyperlink.SetEnabled(origEnabled)
	}()

	Default = New(NewOutput(io.Discard, ColorAlways))
	hyperlink.SetEnabled(false)

	got := Hyperlink("https://example.com", "text")
	assert.Equal(t, "text", got)
}

func TestPathLinkColorsDisabled(t *testing.T) {
	tests := []struct {
		name string
		path string
		line int
		want string
	}{
		{name: "with_line", path: "/some/file.go", line: 42, want: "/some/file.go:42"},
		{name: "no_line", path: "/some/file.go", line: 0, want: "/some/file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PathLink(tt.path, tt.line)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPathLinkEnabled(t *testing.T) {
	withColorsEnabled(t)
	clearFormats(t)

	got := PathLink("/tmp/test.go", 42)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestPathLinkRelativePath(t *testing.T) {
	withColorsEnabled(t)
	clearFormats(t)

	// Relative path should be resolved to absolute.
	got := PathLink("test.go", 0)

	assert.Contains(t, got, "\x1b]8;;")
	// Display text should be relative path.
	assert.Contains(t, got, "test.go")
}

func TestPathLinkDirectory(t *testing.T) {
	withColorsEnabled(t)
	clearFormats(t)

	// Set a line format - directories should still use file://.
	hyperlink.SetLineFormat("vscode://file{path}:{line}")

	got := PathLink("/tmp", 0)

	assert.Equal(t, "\x1b]8;;file:///tmp\x1b\\/tmp\x1b]8;;\x1b\\", got)
}

func TestPathLinkWithLineFormat(t *testing.T) {
	withColorsEnabled(t)
	clearFormats(t)

	hyperlink.SetLineFormat("vscode://file{path}:{line}")

	got := PathLink("/tmp/test.go", 10)

	assert.Equal(t, "\x1b]8;;vscode://file/tmp/test.go:10\x1b\\/tmp/test.go:10\x1b]8;;\x1b\\", got)
}

func TestPathLinkWithPathFormat(t *testing.T) {
	withColorsEnabled(t)
	clearFormats(t)

	hyperlink.SetPathFormat("vscode://file{path}")

	got := PathLink("/tmp/test.go", 0)
	want := "\x1b]8;;vscode://file/tmp/test.go\x1b\\/tmp/test.go\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestLoadHyperlinkFileAndDirFormatsFromEnv(t *testing.T) {
	clearFormats(t)

	t.Setenv("CLOG_HYPERLINK_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_FILE_FORMAT", "vscode://file{path}")
	t.Setenv("CLOG_HYPERLINK_DIR_FORMAT", "finder://{path}")

	loadHyperlinkFormatsFromEnv()

	// File format applied to non-directory path.
	got := hyperlink.ResolvePathURL("/test/file.go", 0, 0)
	assert.Equal(t, "vscode://file/test/file.go", got)

	// Dir format applied to directory path.
	dir := t.TempDir()
	got = hyperlink.ResolvePathURL(dir, 0, 0)
	assert.Equal(t, "finder://"+dir, got)
}

func TestLoadHyperlinkFormatsFromEnv(t *testing.T) {
	clearFormats(t)

	t.Setenv("CLOG_HYPERLINK_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_PATH_FORMAT", "vscode://file{path}")
	t.Setenv("CLOG_HYPERLINK_LINE_FORMAT", "vscode://file{path}:{line}")

	loadHyperlinkFormatsFromEnv()

	// Path format applied to file-only URL (falls back from file → path).
	got := hyperlink.ResolvePathURL("/test/file.go", 0, 0)
	assert.Equal(t, "vscode://file/test/file.go", got)

	// Line format applied to file+line URL.
	got = hyperlink.ResolvePathURL("/test/file.go", 42, 0)
	assert.Equal(t, "vscode://file/test/file.go:42", got)
}

func TestLoadHyperlinkFormatsFromEnvEmpty(t *testing.T) {
	clearFormats(t)

	t.Setenv("CLOG_HYPERLINK_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_PATH_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_LINE_FORMAT", "")

	loadHyperlinkFormatsFromEnv()

	// Default file:// format used when no formats configured.
	got := hyperlink.ResolvePathURL("/test/file.go", 0, 0)
	assert.Equal(t, "file:///test/file.go", got)
}

func TestOutputHyperlinkNever(t *testing.T) {
	output := NewOutput(io.Discard, ColorNever)
	got := output.hyperlink("https://example.com", "text")
	assert.Equal(t, "text", got)
}

func TestOutputPathLinkAlways(t *testing.T) {
	clearFormats(t)

	output := NewOutput(io.Discard, ColorAlways)
	hyperlink.SetEnabled(true)
	defer hyperlink.SetEnabled(false)

	got := output.pathLink("/tmp/test.go", 42, 0)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkAlwaysDir(t *testing.T) {
	clearFormats(t)

	output := NewOutput(io.Discard, ColorAlways)
	hyperlink.SetEnabled(true)
	defer hyperlink.SetEnabled(false)

	got := output.pathLink("/tmp", 0, 0)
	want := "\x1b]8;;file:///tmp\x1b\\/tmp\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkNever(t *testing.T) {
	output := NewOutput(io.Discard, ColorNever)
	got := output.pathLink("/tmp/test.go", 42, 0)

	assert.Equal(t, "/tmp/test.go:42", got)
}

func TestOutputPathLinkNoLine(t *testing.T) {
	output := NewOutput(io.Discard, ColorNever)
	got := output.pathLink("/tmp/test.go", 0, 0)

	assert.Equal(t, "/tmp/test.go", got)
}

func TestOutputPathLinkColumn(t *testing.T) {
	output := NewOutput(io.Discard, ColorNever)
	got := output.pathLink("/tmp/test.go", 42, 10)

	assert.Equal(t, "/tmp/test.go:42:10", got)
}

func TestOutputPathLinkColumnAlways(t *testing.T) {
	clearFormats(t)

	output := NewOutput(io.Discard, ColorAlways)
	hyperlink.SetEnabled(true)
	defer hyperlink.SetEnabled(false)

	got := output.pathLink("/tmp/test.go", 42, 10)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42:10\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputPathLinkColumnFormat(t *testing.T) {
	clearFormats(t)

	output := NewOutput(io.Discard, ColorAlways)
	hyperlink.SetEnabled(true)
	defer hyperlink.SetEnabled(false)

	hyperlink.SetColumnFormat("vscode://file{path}:{line}:{column}")

	got := output.pathLink("/tmp/test.go", 42, 10)
	want := "\x1b]8;;vscode://file/tmp/test.go:42:10\x1b\\/tmp/test.go:42:10\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestOutputHyperlinkAlways(t *testing.T) {
	output := NewOutput(io.Discard, ColorAlways)
	hyperlink.SetEnabled(true)
	defer hyperlink.SetEnabled(false)

	got := output.hyperlink("https://example.com", "text")
	want := "\x1b]8;;https://example.com\x1b\\text\x1b]8;;\x1b\\"
	assert.Equal(t, want, got)
}

func TestLoadHyperlinkColumnFormatFromEnv(t *testing.T) {
	clearFormats(t)

	t.Setenv("CLOG_HYPERLINK_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_COLUMN_FORMAT", "vscode://file{path}:{line}:{column}")

	loadHyperlinkFormatsFromEnv()

	got := hyperlink.ResolvePathURL("/test/file.go", 42, 10)
	assert.Equal(t, "vscode://file/test/file.go:42:10", got)
}
