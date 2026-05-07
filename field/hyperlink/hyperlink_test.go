package hyperlink_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetHyperlinksEnabled(t *testing.T) {
	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })

	hyperlink.SetEnabled(false)
	assert.False(t, hyperlink.Enabled(), "expected hyperlinks disabled")

	hyperlink.SetEnabled(true)
	assert.True(t, hyperlink.Enabled(), "expected hyperlinks enabled")
}

func TestSetHyperlinkPathFormat(t *testing.T) {
	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })
	hyperlink.ClearFormats()

	hyperlink.SetPathFormat("vscode://file{path}")

	got := hyperlink.BuildPathURL("/tmp/test.go", 0, 0, false)
	assert.Equal(t, "vscode://file/tmp/test.go", got)
}

func TestSetHyperlinkLineFormat(t *testing.T) {
	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })
	hyperlink.ClearFormats()

	hyperlink.SetLineFormat("vscode://file{path}:{line}")

	got := hyperlink.BuildPathURL("/tmp/test.go", 42, 0, false)
	assert.Equal(t, "vscode://file/tmp/test.go:42", got)
}

func TestSetHyperlinkFileFormat(t *testing.T) {
	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })
	hyperlink.ClearFormats()

	hyperlink.SetFileFormat("vscode://file{path}")

	got := hyperlink.BuildPathURL("/tmp/test.go", 0, 0, false)
	assert.Equal(t, "vscode://file/tmp/test.go", got)
}

func TestSetHyperlinkDirFormat(t *testing.T) {
	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })
	hyperlink.ClearFormats()

	hyperlink.SetDirFormat("finder://{path}")

	got := hyperlink.BuildPathURL("/tmp/dir", 0, 0, true)
	assert.Equal(t, "finder:///tmp/dir", got)
}

func TestSetHyperlinkColumnFormat(t *testing.T) {
	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })
	hyperlink.ClearFormats()

	hyperlink.SetColumnFormat("vscode://file{path}:{line}:{col}")

	got := hyperlink.BuildPathURL("/tmp/test.go", 42, 5, false)
	assert.Equal(t, "vscode://file/tmp/test.go:42:5", got)
}

func TestExpandPreset(t *testing.T) {
	tests := []struct {
		name  string
		value string
		slot  string
		want  string
	}{
		{
			name:  "vscode_path_slot",
			value: "vscode",
			slot:  "path",
			want:  "vscode://file{path}",
		},
		{
			name:  "vscode_line_slot",
			value: "vscode",
			slot:  "line",
			want:  "vscode://file{path}:{line}",
		},
		{
			name:  "vscode_column_slot",
			value: "vscode",
			slot:  "column",
			want:  "vscode://file{path}:{line}:{column}",
		},
		{
			name:  "vscode_default_slot",
			value: "vscode",
			slot:  "",
			want:  "vscode://file{path}",
		},
		{
			name:  "cursor_line_slot",
			value: "cursor",
			slot:  "line",
			want:  "cursor://file{path}:{line}",
		},
		{
			name:  "unknown_preset_returns_unchanged",
			value: "unknown-editor",
			slot:  "path",
			want:  "unknown-editor",
		},
		{
			name:  "full_format_string_passes_through",
			value: "vscode://file{path}:{line}",
			slot:  "line",
			want:  "vscode://file{path}:{line}",
		},
		{
			name:  "case_insensitive_lookup",
			value: "  VSCode  ",
			slot:  "path",
			want:  "vscode://file{path}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hyperlink.ExpandPreset(tt.value, tt.slot)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildPathURL(t *testing.T) {
	snap := hyperlink.Save()
	t.Cleanup(func() { hyperlink.Restore(snap) })
	hyperlink.ClearFormats()

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
			hyperlink.ClearFormats()

			if tt.pathFmt != "" {
				hyperlink.SetPathFormat(tt.pathFmt)
			}

			if tt.fileFmt != "" {
				hyperlink.SetFileFormat(tt.fileFmt)
			}

			if tt.dirFmt != "" {
				hyperlink.SetDirFormat(tt.dirFmt)
			}

			if tt.lineFmt != "" {
				hyperlink.SetLineFormat(tt.lineFmt)
			}

			if tt.colFmt != "" {
				hyperlink.SetColumnFormat(tt.colFmt)
			}

			got := hyperlink.BuildPathURL(tt.path, tt.line, tt.column, tt.isDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPathDisplayTextColumn(t *testing.T) {
	got := hyperlink.PathDisplayText("/tmp/test.go", 42, 10)
	assert.Equal(t, "/tmp/test.go:42:10", got)
}

func TestPathDisplayTextColumnNoLine(t *testing.T) {
	// Column without line - column is ignored.
	got := hyperlink.PathDisplayText("/tmp/test.go", 0, 10)
	assert.Equal(t, "/tmp/test.go", got)
}

func TestAbsPathAlreadyAbsolute(t *testing.T) {
	got := hyperlink.AbsPath("/already/absolute/path.go")
	assert.Equal(t, "/already/absolute/path.go", got)
}

func TestAbsPathRelative(t *testing.T) {
	got := hyperlink.AbsPath("relative.go")
	assert.True(t, filepath.IsAbs(got), "expected absolute path for relative input")
	assert.True(t, strings.HasSuffix(got, "/relative.go"), "expected path to end with relative.go")
}

func TestAbsPathFallbackOnGetwdFailure(t *testing.T) {
	// When filepath.Abs cannot resolve a relative path (Getwd fails),
	// absPath should return the original path unchanged.
	tmp := t.TempDir()
	t.Chdir(tmp)

	// Remove the directory we just entered - Getwd will now fail on some platforms.
	if err := os.Remove(tmp); err != nil {
		t.Fatal(err)
	}

	got := hyperlink.AbsPath("relative.go")

	// On some platforms (e.g. macOS), the kernel can still resolve the cwd
	// even after the directory is removed. Accept either outcome.
	if got == "relative.go" {
		return // Getwd failed, fallback path taken
	}

	assert.True(t, filepath.IsAbs(got), "expected fallback or absolute path")
}

func TestIsDirectory(t *testing.T) {
	// Existing directory should return true.
	assert.True(t, hyperlink.IsDirectory(os.TempDir()))

	// Non-existent path should return false.
	assert.False(t, hyperlink.IsDirectory("/nonexistent/path/that/does/not/exist"))

	// File (not directory) should return false.
	f, err := os.CreateTemp(t.TempDir(), "clog-test-*")
	if err != nil {
		t.Fatal(err)
	}
	require.NoError(t, f.Close())

	assert.False(t, hyperlink.IsDirectory(f.Name()))
}
