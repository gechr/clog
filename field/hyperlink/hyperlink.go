// Package hyperlink provides terminal hyperlink rendering for clog.
// All configuration is explicit: callers pass a [Config] (usually derived
// from the logger's FieldFormats) - the package holds no global state.
package hyperlink

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds resolved hyperlink rendering configuration.
// The format strings use {path}, {line}, and {column} (or {col}) as
// placeholders; an empty string means the documented fallback.
type Config struct {
	// Enabled controls whether hyperlinks are rendered at all. When false,
	// hyperlink helpers return plain text without OSC 8 sequences.
	Enabled bool

	// ColumnFormat is the URL format for file+line+column hyperlinks.
	// Falls back to LineFormat when empty.
	ColumnFormat string
	// DirFormat is the URL format for directory hyperlinks.
	// Falls back to PathFormat when empty.
	DirFormat string
	// FileFormat is the URL format for file-only hyperlinks (no line
	// number). Falls back to PathFormat when empty.
	FileFormat string
	// LineFormat is the URL format for file+line hyperlinks.
	// Falls back to "file://{path}" when empty.
	LineFormat string
	// PathFormat is the generic fallback URL format for any path.
	// Falls back to "file://{path}" when empty.
	PathFormat string
}

// DefaultConfig returns the default hyperlink configuration: enabled, with
// all format slots empty (plain file:// URLs).
func DefaultConfig() Config {
	return Config{Enabled: true}
}

// preset holds the per-slot URL format templates for a named editor preset.
type preset struct {
	description string
	path        string
	line        string
	column      string
}

// presets maps short preset names (lower-case) to their format templates.
var presets = map[string]preset{
	"cursor": {
		description: "Cursor (cursor://)",
		path:        "cursor://file{path}",
		line:        "cursor://file{path}:{line}",
		column:      "cursor://file{path}:{line}:{column}",
	},
	"kitty": {
		description: "kitty terminal (file:// with fragment line number)",
		path:        "file://{path}",
		line:        "file://{path}#{line}",
		column:      "file://{path}#{line}",
	},
	"macvim": {
		description: "MacVim (mvim://)",
		path:        "mvim://open?url=file://{path}",
		line:        "mvim://open?url=file://{path}&line={line}",
		column:      "mvim://open?url=file://{path}&line={line}&column={column}",
	},
	"subl": {
		description: "Sublime Text (subl://)",
		path:        "subl://open?url=file://{path}",
		line:        "subl://open?url=file://{path}&line={line}",
		column:      "subl://open?url=file://{path}&line={line}&column={column}",
	},
	"textmate": {
		description: "TextMate (txmt://)",
		path:        "txmt://open?url=file://{path}",
		line:        "txmt://open?url=file://{path}&line={line}",
		column:      "txmt://open?url=file://{path}&line={line}&column={column}",
	},
	"vscode": {
		description: "VS Code (vscode://)",
		path:        "vscode://file{path}",
		line:        "vscode://file{path}:{line}",
		column:      "vscode://file{path}:{line}:{column}",
	},
	"vscode-insiders": {
		description: "VS Code Insiders (vscode-insiders://)",
		path:        "vscode-insiders://file{path}",
		line:        "vscode-insiders://file{path}:{line}",
		column:      "vscode-insiders://file{path}:{line}:{column}",
	},
	"vscodium": {
		description: "VSCodium (vscodium://)",
		path:        "vscodium://file{path}",
		line:        "vscodium://file{path}:{line}",
		column:      "vscodium://file{path}:{line}:{column}",
	},
}

// Preset returns the [Config] for a named editor preset.
// Known presets: cursor, kitty, macvim, subl, textmate, vscode,
// vscode-insiders, vscodium.
func Preset(name string) (Config, error) {
	p, ok := presets[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return Config{}, fmt.Errorf("clog: unknown hyperlink preset %q", name)
	}
	return Config{
		Enabled:      true,
		PathFormat:   p.path,
		FileFormat:   p.path,
		DirFormat:    p.path,
		LineFormat:   p.line,
		ColumnFormat: p.column,
	}, nil
}

// Expand resolves a preset name to its format string for the given slot
// ("path", "line", or "column"). Returns value unchanged if it is not a
// known preset name, so full format strings pass through unmodified.
func Expand(value, slot string) string {
	p, ok := presets[strings.ToLower(strings.TrimSpace(value))]
	if !ok {
		return value
	}
	switch slot {
	case "line":
		return p.line
	case "column":
		return p.column
	default:
		return p.path
	}
}

// OSC8 wraps text in raw OSC 8 escape sequences unconditionally.
func OSC8(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// PathDisplayText returns the display string for a path hyperlink.
func PathDisplayText(path string, line, column int) string {
	if column > 0 && line > 0 {
		return path + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column)
	}

	if line > 0 {
		return path + ":" + strconv.Itoa(line)
	}
	return path
}

// ResolvePathURL builds the full hyperlink URL for a file path using the
// given configuration.
func (c Config) ResolvePathURL(path string, line, column int) string {
	abs := absPath(path)
	return c.buildPathURL(abs, line, column, isDirectory(abs))
}

// absPath resolves a path to its absolute form.
// Returns the original path if resolution fails.
func absPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}

// buildPathURL constructs the hyperlink URL using the configured formats.
func (c Config) buildPathURL(absPath string, line, column int, isDir bool) string {
	var format string

	switch {
	case isDir:
		format = firstFormat(c.DirFormat, c.PathFormat)
	case column > 0:
		format = firstFormat(c.ColumnFormat, c.LineFormat)
	case line > 0:
		format = firstFormat(c.LineFormat)
	default:
		format = firstFormat(c.FileFormat, c.PathFormat)
	}

	if format == "" {
		return "file://" + absPath
	}

	u := format
	u = strings.ReplaceAll(u, "{path}", absPath)
	u = strings.ReplaceAll(u, "{line}", strconv.Itoa(line))
	u = strings.ReplaceAll(u, "{column}", strconv.Itoa(column))
	u = strings.ReplaceAll(u, "{col}", strconv.Itoa(column))
	return u
}

// isDirectory reports whether path is an existing directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// firstFormat returns the first non-empty format string.
func firstFormat(formats ...string) string {
	for _, f := range formats {
		if f != "" {
			return f
		}
	}
	return ""
}
