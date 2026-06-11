package hyperlink_test

import (
	"path/filepath"
	"testing"

	"github.com/gechr/clog/field/hyperlink"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	got := hyperlink.DefaultConfig()
	assert.Equal(t, hyperlink.Config{Enabled: true}, got)
}

func TestPreset(t *testing.T) {
	tests := []struct {
		name string
		want hyperlink.Config
	}{
		{
			name: "vscode",
			want: hyperlink.Config{
				Enabled:      true,
				PathFormat:   "vscode://file{path}",
				FileFormat:   "vscode://file{path}",
				DirFormat:    "vscode://file{path}",
				LineFormat:   "vscode://file{path}:{line}",
				ColumnFormat: "vscode://file{path}:{line}:{column}",
			},
		},
		{
			name: "kitty",
			want: hyperlink.Config{
				Enabled:      true,
				PathFormat:   "file://{path}",
				FileFormat:   "file://{path}",
				DirFormat:    "file://{path}",
				LineFormat:   "file://{path}#{line}",
				ColumnFormat: "file://{path}#{line}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := hyperlink.Preset(tt.name)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPresetCaseInsensitive(t *testing.T) {
	got, err := hyperlink.Preset("  VSCode ")
	require.NoError(t, err)
	assert.Equal(t, "vscode://file{path}", got.PathFormat)
}

func TestPresetUnknown(t *testing.T) {
	_, err := hyperlink.Preset("notepad")
	assert.ErrorContains(t, err, `unknown hyperlink preset "notepad"`)
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name  string
		value string
		slot  string
		want  string
	}{
		{name: "preset_path", value: "vscode", slot: "path", want: "vscode://file{path}"},
		{name: "preset_line", value: "vscode", slot: "line", want: "vscode://file{path}:{line}"},
		{
			name:  "preset_column",
			value: "vscode",
			slot:  "column",
			want:  "vscode://file{path}:{line}:{column}",
		},
		{
			name:  "preset_case_insensitive",
			value: " VSCode ",
			slot:  "line",
			want:  "vscode://file{path}:{line}",
		},
		{
			name:  "format_passthrough",
			value: "myeditor://{path}:{line}",
			slot:  "line",
			want:  "myeditor://{path}:{line}",
		},
		{name: "empty_passthrough", value: "", slot: "path", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hyperlink.Expand(tt.value, tt.slot)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolvePathURLFile(t *testing.T) {
	tests := []struct {
		name   string
		config hyperlink.Config
		line   int
		column int
		want   string
	}{
		{
			name: "default_file_url",
			want: "file:///nonexistent/file.go",
		},
		{
			name: "default_file_url_with_line",
			line: 42,
			want: "file:///nonexistent/file.go",
		},
		{
			name:   "file_format",
			config: hyperlink.Config{FileFormat: "editor://open{path}"},
			want:   "editor://open/nonexistent/file.go",
		},
		{
			name:   "file_falls_back_to_path_format",
			config: hyperlink.Config{PathFormat: "editor://open{path}"},
			want:   "editor://open/nonexistent/file.go",
		},
		{
			name: "file_format_wins_over_path_format",
			config: hyperlink.Config{
				FileFormat: "file://{path}",
				PathFormat: "editor://open{path}",
			},
			want: "file:///nonexistent/file.go",
		},
		{
			name:   "line_format",
			config: hyperlink.Config{LineFormat: "editor://open{path}:{line}"},
			line:   42,
			want:   "editor://open/nonexistent/file.go:42",
		},
		{
			name:   "line_ignores_path_format",
			config: hyperlink.Config{PathFormat: "editor://open{path}"},
			line:   42,
			want:   "file:///nonexistent/file.go",
		},
		{
			name:   "column_format",
			config: hyperlink.Config{ColumnFormat: "editor://open{path}:{line}:{column}"},
			line:   42,
			column: 10,
			want:   "editor://open/nonexistent/file.go:42:10",
		},
		{
			name:   "column_falls_back_to_line_format",
			config: hyperlink.Config{LineFormat: "editor://open{path}:{line}"},
			line:   42,
			column: 10,
			want:   "editor://open/nonexistent/file.go:42",
		},
		{
			name:   "col_placeholder_alias",
			config: hyperlink.Config{ColumnFormat: "editor://open{path}:{line}:{col}"},
			line:   42,
			column: 10,
			want:   "editor://open/nonexistent/file.go:42:10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ResolvePathURL("/nonexistent/file.go", tt.line, tt.column)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolvePathURLDirectory(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name   string
		config hyperlink.Config
		want   string
	}{
		{
			name: "default_file_url",
			want: "file://" + dir,
		},
		{
			name:   "dir_format",
			config: hyperlink.Config{DirFormat: "finder://{path}"},
			want:   "finder://" + dir,
		},
		{
			name:   "dir_falls_back_to_path_format",
			config: hyperlink.Config{PathFormat: "editor://open{path}"},
			want:   "editor://open" + dir,
		},
		{
			name: "dir_ignores_file_and_line_formats",
			config: hyperlink.Config{
				FileFormat: "editor://open{path}",
				LineFormat: "editor://open{path}:{line}",
			},
			want: "file://" + dir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ResolvePathURL(dir, 0, 0)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolvePathURLRelativePath(t *testing.T) {
	abs, err := filepath.Abs("nonexistent.go")
	require.NoError(t, err)

	got := hyperlink.Config{}.ResolvePathURL("nonexistent.go", 0, 0)
	assert.Equal(t, "file://"+abs, got)
}

func TestPathDisplayText(t *testing.T) {
	tests := []struct {
		name   string
		line   int
		column int
		want   string
	}{
		{name: "path_only", want: "/some/file.go"},
		{name: "with_line", line: 42, want: "/some/file.go:42"},
		{name: "with_line_and_column", line: 42, column: 10, want: "/some/file.go:42:10"},
		{name: "column_without_line", column: 10, want: "/some/file.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hyperlink.PathDisplayText("/some/file.go", tt.line, tt.column)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOSC8(t *testing.T) {
	got := hyperlink.OSC8("https://example.com", "click")
	assert.Equal(t, "\x1b]8;;https://example.com\x1b\\click\x1b]8;;\x1b\\", got)
}
