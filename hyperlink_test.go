package clog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// saveFormats saves and returns a cleanup func that restores all format pointers.
func saveFormats(t *testing.T) {
	t.Helper()

	origPath := hyperlinkPathFormat.Load()
	origFile := hyperlinkFileFormat.Load()
	origDir := hyperlinkDirFormat.Load()
	origLine := hyperlinkLineFormat.Load()
	origCol := hyperlinkColumnFormat.Load()

	t.Cleanup(func() {
		hyperlinkPathFormat.Store(origPath)
		hyperlinkFileFormat.Store(origFile)
		hyperlinkDirFormat.Store(origDir)
		hyperlinkLineFormat.Store(origLine)
		hyperlinkColumnFormat.Store(origCol)
	})
}

func clearFormats(t *testing.T) {
	t.Helper()

	saveFormats(t)

	hyperlinkPathFormat.Store(nil)
	hyperlinkFileFormat.Store(nil)
	hyperlinkDirFormat.Store(nil)
	hyperlinkLineFormat.Store(nil)
	hyperlinkColumnFormat.Store(nil)
}

func TestHyperlinkColorsDisabled(t *testing.T) {
	// In test environment, ColorsDisabled() returns true (no terminal).
	got := Hyperlink("https://example.com", "click here")
	assert.Equal(t, "click here", got)
}

func TestHyperlinkEnabled(t *testing.T) {
	origForced := colorsForced.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		hyperlinksEnabled.Store(origEnabled)
	}()

	colorsForced.Store(true)
	hyperlinksEnabled.Store(true)

	got := Hyperlink("https://example.com", "click")
	want := "\x1b]8;;https://example.com\x1b\\click\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestHyperlinkDisabledViaFlag(t *testing.T) {
	origForced := colorsForced.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		hyperlinksEnabled.Store(origEnabled)
	}()

	colorsForced.Store(true)
	hyperlinksEnabled.Store(false)

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

func TestSetHyperlinksEnabled(t *testing.T) {
	origEnabled := hyperlinksEnabled.Load()
	defer hyperlinksEnabled.Store(origEnabled)

	SetHyperlinksEnabled(false)
	assert.False(t, hyperlinksEnabled.Load(), "expected hyperlinks disabled")

	SetHyperlinksEnabled(true)
	assert.True(t, hyperlinksEnabled.Load(), "expected hyperlinks enabled")
}

func TestPathLinkEnabled(t *testing.T) {
	origForced := colorsForced.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		hyperlinksEnabled.Store(origEnabled)
	}()

	clearFormats(t)

	colorsForced.Store(true)
	hyperlinksEnabled.Store(true)

	got := PathLink("/tmp/test.go", 42)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestPathLinkRelativePath(t *testing.T) {
	origForced := colorsForced.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		hyperlinksEnabled.Store(origEnabled)
	}()

	clearFormats(t)

	colorsForced.Store(true)
	hyperlinksEnabled.Store(true)

	// Relative path should be resolved to absolute.
	got := PathLink("test.go", 0)

	assert.Contains(t, got, "\x1b]8;;")
	// Display text should be relative path.
	assert.Contains(t, got, "test.go")
}

func TestPathLinkDirectory(t *testing.T) {
	origForced := colorsForced.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		hyperlinksEnabled.Store(origEnabled)
	}()

	clearFormats(t)

	colorsForced.Store(true)
	hyperlinksEnabled.Store(true)

	// Set a line format — directories should still use file://.
	fmtStr := "vscode://file{path}:{line}"
	hyperlinkLineFormat.Store(&fmtStr)

	got := PathLink("/tmp", 0)

	assert.Contains(t, got, "file:///tmp")
}

func TestPathLinkWithLineFormat(t *testing.T) {
	origForced := colorsForced.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		hyperlinksEnabled.Store(origEnabled)
	}()

	clearFormats(t)

	colorsForced.Store(true)
	hyperlinksEnabled.Store(true)

	fmtStr := "vscode://file{path}:{line}"
	hyperlinkLineFormat.Store(&fmtStr)

	got := PathLink("/tmp/test.go", 10)

	assert.Contains(t, got, "vscode://file/tmp/test.go:10")
}

func TestPathLinkWithPathFormat(t *testing.T) {
	origForced := colorsForced.Load()
	origEnabled := hyperlinksEnabled.Load()

	defer func() {
		colorsForced.Store(origForced)
		hyperlinksEnabled.Store(origEnabled)
	}()

	clearFormats(t)

	colorsForced.Store(true)
	hyperlinksEnabled.Store(true)

	fmtStr := "vscode://file{path}"
	hyperlinkPathFormat.Store(&fmtStr)

	got := PathLink("/tmp/test.go", 0)
	want := "\x1b]8;;vscode://file/tmp/test.go\x1b\\/tmp/test.go\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestSetHyperlinkPathFormat(t *testing.T) {
	saveFormats(t)

	SetHyperlinkPathFormat("vscode://file{path}")

	got := hyperlinkPathFormat.Load()
	if assert.NotNil(t, got) {
		assert.Equal(t, "vscode://file{path}", *got)
	}
}

func TestSetHyperlinkLineFormat(t *testing.T) {
	saveFormats(t)

	SetHyperlinkLineFormat("vscode://file{path}:{line}")

	got := hyperlinkLineFormat.Load()
	if assert.NotNil(t, got) {
		assert.Equal(t, "vscode://file{path}:{line}", *got)
	}
}

func TestSetHyperlinkFileFormat(t *testing.T) {
	saveFormats(t)

	SetHyperlinkFileFormat("vscode://file{path}")

	got := hyperlinkFileFormat.Load()
	if assert.NotNil(t, got) {
		assert.Equal(t, "vscode://file{path}", *got)
	}
}

func TestSetHyperlinkDirFormat(t *testing.T) {
	saveFormats(t)

	SetHyperlinkDirFormat("finder://{path}")

	got := hyperlinkDirFormat.Load()
	if assert.NotNil(t, got) {
		assert.Equal(t, "finder://{path}", *got)
	}
}

func TestLoadHyperlinkFileAndDirFormatsFromEnv(t *testing.T) {
	saveFormats(t)

	t.Setenv("CLOG_HYPERLINK_FILE_FORMAT", "vscode://file{path}")
	t.Setenv("CLOG_HYPERLINK_DIR_FORMAT", "finder://{path}")

	hyperlinkFileFormat.Store(nil)
	hyperlinkDirFormat.Store(nil)

	loadHyperlinkFormatsFromEnv()

	gotFile := hyperlinkFileFormat.Load()
	gotDir := hyperlinkDirFormat.Load()

	if assert.NotNil(t, gotFile) {
		assert.Equal(t, "vscode://file{path}", *gotFile)
	}

	if assert.NotNil(t, gotDir) {
		assert.Equal(t, "finder://{path}", *gotDir)
	}
}

func TestLoadHyperlinkFormatsFromEnv(t *testing.T) {
	saveFormats(t)

	t.Setenv("CLOG_HYPERLINK_PATH_FORMAT", "vscode://file{path}")
	t.Setenv("CLOG_HYPERLINK_LINE_FORMAT", "vscode://file{path}:{line}")

	hyperlinkPathFormat.Store(nil)
	hyperlinkLineFormat.Store(nil)

	loadHyperlinkFormatsFromEnv()

	gotPath := hyperlinkPathFormat.Load()
	gotLine := hyperlinkLineFormat.Load()

	if assert.NotNil(t, gotPath) {
		assert.Equal(t, "vscode://file{path}", *gotPath)
	}

	if assert.NotNil(t, gotLine) {
		assert.Equal(t, "vscode://file{path}:{line}", *gotLine)
	}
}

func TestLoadHyperlinkFormatsFromEnvEmpty(t *testing.T) {
	saveFormats(t)

	t.Setenv("CLOG_HYPERLINK_PATH_FORMAT", "")
	t.Setenv("CLOG_HYPERLINK_LINE_FORMAT", "")

	hyperlinkPathFormat.Store(nil)
	hyperlinkLineFormat.Store(nil)

	loadHyperlinkFormatsFromEnv()

	assert.Nil(t, hyperlinkPathFormat.Load())
	assert.Nil(t, hyperlinkLineFormat.Load())
}

func TestHyperlinkWithModeNever(t *testing.T) {
	got := hyperlinkWithMode("https://example.com", "text", ColorNever)
	assert.Equal(t, "text", got)
}

func TestPathLinkWithModeAlways(t *testing.T) {
	clearFormats(t)

	got := pathLinkWithMode("/tmp/test.go", 42, 0, ColorAlways)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestPathLinkWithModeAlwaysDir(t *testing.T) {
	clearFormats(t)

	got := pathLinkWithMode("/tmp", 0, 0, ColorAlways)
	want := "\x1b]8;;file:///tmp\x1b\\/tmp\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestPathLinkWithModeNever(t *testing.T) {
	got := pathLinkWithMode("/tmp/test.go", 42, 0, ColorNever)

	assert.Equal(t, "/tmp/test.go:42", got)
}

func TestPathLinkWithModeNoLine(t *testing.T) {
	got := pathLinkWithMode("/tmp/test.go", 0, 0, ColorNever)

	assert.Equal(t, "/tmp/test.go", got)
}

func TestPathLinkWithModeColumn(t *testing.T) {
	got := pathLinkWithMode("/tmp/test.go", 42, 10, ColorNever)

	assert.Equal(t, "/tmp/test.go:42:10", got)
}

func TestPathLinkWithModeColumnAlways(t *testing.T) {
	clearFormats(t)

	got := pathLinkWithMode("/tmp/test.go", 42, 10, ColorAlways)
	want := "\x1b]8;;file:///tmp/test.go\x1b\\/tmp/test.go:42:10\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestPathLinkWithModeColumnFormat(t *testing.T) {
	clearFormats(t)

	colFmt := "vscode://file{path}:{line}:{column}"
	hyperlinkColumnFormat.Store(&colFmt)

	got := pathLinkWithMode("/tmp/test.go", 42, 10, ColorAlways)
	want := "\x1b]8;;vscode://file/tmp/test.go:42:10\x1b\\/tmp/test.go:42:10\x1b]8;;\x1b\\"

	assert.Equal(t, want, got)
}

func TestPathDisplayTextColumn(t *testing.T) {
	got := pathDisplayText("/tmp/test.go", 42, 10)
	assert.Equal(t, "/tmp/test.go:42:10", got)
}

func TestPathDisplayTextColumnNoLine(t *testing.T) {
	// Column without line — column is ignored.
	got := pathDisplayText("/tmp/test.go", 0, 10)
	assert.Equal(t, "/tmp/test.go", got)
}

func TestSetHyperlinkColumnFormat(t *testing.T) {
	saveFormats(t)

	SetHyperlinkColumnFormat("vscode://file{path}:{line}:{col}")

	got := hyperlinkColumnFormat.Load()
	if assert.NotNil(t, got) {
		assert.Equal(t, "vscode://file{path}:{line}:{col}", *got)
	}
}

func TestLoadHyperlinkColumnFormatFromEnv(t *testing.T) {
	saveFormats(t)

	t.Setenv("CLOG_HYPERLINK_COLUMN_FORMAT", "vscode://file{path}:{line}:{column}")

	hyperlinkColumnFormat.Store(nil)

	loadHyperlinkFormatsFromEnv()

	got := hyperlinkColumnFormat.Load()
	if assert.NotNil(t, got) {
		assert.Equal(t, "vscode://file{path}:{line}:{column}", *got)
	}
}

func TestAbsPathAlreadyAbsolute(t *testing.T) {
	got := absPath("/already/absolute/path.go")
	assert.Equal(t, "/already/absolute/path.go", got)
}

func TestAbsPathRelative(t *testing.T) {
	got := absPath("relative.go")
	assert.True(t, filepath.IsAbs(got), "expected absolute path for relative input")
	assert.True(t, strings.HasSuffix(got, "/relative.go"), "expected path to end with relative.go")
}

func TestAbsPathFallbackOnGetwdFailure(t *testing.T) {
	// When filepath.Abs cannot resolve a relative path (Getwd fails),
	// absPath should return the original path unchanged.
	tmp := t.TempDir()
	t.Chdir(tmp)

	// Remove the directory we just entered — Getwd will now fail on some platforms.
	if err := os.Remove(tmp); err != nil {
		t.Fatal(err)
	}

	got := absPath("relative.go")

	// On some platforms (e.g. macOS), the kernel can still resolve the cwd
	// even after the directory is removed. Accept either outcome.
	if got == "relative.go" {
		return // Getwd failed, fallback path taken
	}

	assert.True(t, filepath.IsAbs(got), "expected fallback or absolute path")
}

func TestIsDirectory(t *testing.T) {
	// Existing directory should return true.
	assert.True(t, isDirectory(os.TempDir()))

	// Non-existent path should return false.
	assert.False(t, isDirectory("/nonexistent/path/that/does/not/exist"))

	// File (not directory) should return false.
	f, err := os.CreateTemp(t.TempDir(), "clog-test-*")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	assert.False(t, isDirectory(f.Name()))
}

func TestHyperlinkWithModeAlways(t *testing.T) {
	got := hyperlinkWithMode("https://example.com", "text", ColorAlways)
	want := "\x1b]8;;https://example.com\x1b\\text\x1b]8;;\x1b\\"
	assert.Equal(t, want, got)
}

func TestBuildPathURL(t *testing.T) {
	clearFormats(t)

	tests := []struct {
		name    string
		pathFmt string
		fileFmt string
		dirFmt  string
		lineFmt string
		colFmt  string
		path    string
		line    int
		column  int
		isDir   bool
		want    string
	}{
		{
			name: "default_no_line",
			path: "/tmp/test.go",
			want: "file:///tmp/test.go",
		},
		{
			name: "default_with_line",
			path: "/tmp/test.go",
			line: 42,
			want: "file:///tmp/test.go",
		},
		{
			name:    "line_format",
			lineFmt: "vscode://file{path}:{line}",
			path:    "/tmp/test.go",
			line:    42,
			want:    "vscode://file/tmp/test.go:42",
		},
		{
			name:    "path_format_file_fallback",
			pathFmt: "vscode://file{path}",
			path:    "/tmp/test.go",
			want:    "vscode://file/tmp/test.go",
		},
		{
			name:    "path_format_dir_fallback",
			pathFmt: "custom://{path}",
			path:    "/tmp/dir",
			isDir:   true,
			want:    "custom:///tmp/dir",
		},
		{
			name:    "file_format",
			fileFmt: "vscode://file{path}",
			path:    "/tmp/test.go",
			want:    "vscode://file/tmp/test.go",
		},
		{
			name:    "file_format_overrides_path_format",
			pathFmt: "generic://{path}",
			fileFmt: "vscode://file{path}",
			path:    "/tmp/test.go",
			want:    "vscode://file/tmp/test.go",
		},
		{
			name:   "dir_format",
			dirFmt: "finder://{path}",
			path:   "/tmp/dir",
			isDir:  true,
			want:   "finder:///tmp/dir",
		},
		{
			name:    "dir_format_overrides_path_format",
			pathFmt: "generic://{path}",
			dirFmt:  "finder://{path}",
			path:    "/tmp/dir",
			isDir:   true,
			want:    "finder:///tmp/dir",
		},
		{
			name:    "idea_format",
			lineFmt: "idea://open?file={path}&line={line}",
			path:    "/tmp/test.go",
			line:    10,
			want:    "idea://open?file=/tmp/test.go&line=10",
		},
		{
			name:    "dir_ignores_line_format",
			lineFmt: "vscode://file{path}:{line}",
			path:    "/tmp/dir",
			isDir:   true,
			want:    "file:///tmp/dir",
		},
		{
			name:    "dir_ignores_file_format",
			fileFmt: "vscode://file{path}",
			path:    "/tmp/dir",
			isDir:   true,
			want:    "file:///tmp/dir",
		},
		{
			name:    "line_format_not_used_without_line",
			lineFmt: "vscode://file{path}:{line}",
			path:    "/tmp/test.go",
			line:    0,
			want:    "file:///tmp/test.go",
		},
		{
			name:    "file_format_not_used_with_line",
			fileFmt: "vscode://file{path}",
			path:    "/tmp/test.go",
			line:    42,
			want:    "file:///tmp/test.go",
		},
		{
			name:   "column_format",
			colFmt: "vscode://file{path}:{line}:{column}",
			path:   "/tmp/test.go",
			line:   42,
			column: 10,
			want:   "vscode://file/tmp/test.go:42:10",
		},
		{
			name:   "column_format_with_col_alias",
			colFmt: "vscode://file{path}:{line}:{col}",
			path:   "/tmp/test.go",
			line:   42,
			column: 5,
			want:   "vscode://file/tmp/test.go:42:5",
		},
		{
			name:    "column_falls_back_to_line_format",
			lineFmt: "vscode://file{path}:{line}",
			path:    "/tmp/test.go",
			line:    42,
			column:  10,
			want:    "vscode://file/tmp/test.go:42",
		},
		{
			name:   "column_falls_back_to_default",
			path:   "/tmp/test.go",
			line:   42,
			column: 10,
			want:   "file:///tmp/test.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hyperlinkPathFormat.Store(nil)
			hyperlinkFileFormat.Store(nil)
			hyperlinkDirFormat.Store(nil)
			hyperlinkLineFormat.Store(nil)
			hyperlinkColumnFormat.Store(nil)

			if tt.pathFmt != "" {
				hyperlinkPathFormat.Store(&tt.pathFmt)
			}

			if tt.fileFmt != "" {
				hyperlinkFileFormat.Store(&tt.fileFmt)
			}

			if tt.dirFmt != "" {
				hyperlinkDirFormat.Store(&tt.dirFmt)
			}

			if tt.lineFmt != "" {
				hyperlinkLineFormat.Store(&tt.lineFmt)
			}

			if tt.colFmt != "" {
				hyperlinkColumnFormat.Store(&tt.colFmt)
			}

			got := buildPathURL(tt.path, tt.line, tt.column, tt.isDir)
			assert.Equal(t, tt.want, got)
		})
	}
}
