// Package hyperlink provides terminal hyperlink configuration for clog.
package hyperlink

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
)

// columnFormat holds the URL format for file+line+column hyperlinks.
// Use {path}, {line}, and {column} (or {col}) as placeholders. Nil means fall back to line format.
var columnFormat atomic.Pointer[string]

// dirFormat holds the URL format for directory hyperlinks.
// Falls back to pathFormat if nil.
var dirFormat atomic.Pointer[string]

// fileFormat holds the URL format for file-only hyperlinks (no line number).
// Falls back to pathFormat if nil.
var fileFormat atomic.Pointer[string]

// lineFormat holds the URL format for file+line hyperlinks.
// Use {path} and {line} as placeholders. Nil means use default (file://{path}).
var lineFormat atomic.Pointer[string]

// pathFormat is the generic fallback URL format for any path.
// Use {path} as placeholder. Nil means use default (file://{path}).
var pathFormat atomic.Pointer[string]

// enabled controls whether hyperlinks are rendered at all.
var enabled atomic.Bool

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

// SetColumnFormat configures the URL format for file+line+column hyperlinks.
//
// Accepts a full format string or a preset name (e.g. "vscode"). Known presets:
// cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
//
// Use {path}, {line}, and {column} (or {col}) as placeholders.
func SetColumnFormat(format string) {
	columnFormat.Store(new(expandPreset(format, "column")))
}

// SetDirFormat configures the URL format for directory hyperlinks.
//
// Accepts a full format string or a preset name. Falls back to [SetPathFormat] if not set.
func SetDirFormat(format string) {
	dirFormat.Store(new(expandPreset(format, "path")))
}

// SetFileFormat configures the URL format for file-only hyperlinks
// (used by Path and PathLink with line 0, when the path is not a directory).
//
// Accepts a full format string or a preset name. Falls back to [SetPathFormat] if not set.
func SetFileFormat(format string) {
	fileFormat.Store(new(expandPreset(format, "path")))
}

// SetLineFormat configures the URL format for file+line hyperlinks.
//
// Accepts a full format string or a preset name (e.g. "vscode").
func SetLineFormat(format string) {
	lineFormat.Store(new(expandPreset(format, "line")))
}

// SetPathFormat configures the generic fallback URL format for any path.
//
// Accepts a full format string or a preset name.
func SetPathFormat(format string) {
	pathFormat.Store(new(expandPreset(format, "path")))
}

// SetPreset configures all hyperlink format slots using a named preset.
// This is a convenience wrapper around the individual Set*Format functions.
//
// Known presets: cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
func SetPreset(name string) error {
	p, ok := presets[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return fmt.Errorf("clog: unknown hyperlink preset %q", name)
	}
	pathFormat.Store(new(p.path))
	fileFormat.Store(new(p.path))
	dirFormat.Store(new(p.path))
	lineFormat.Store(new(p.line))
	columnFormat.Store(new(p.column))
	return nil
}

// SetEnabled enables or disables all hyperlink rendering.
// When disabled, hyperlink functions return plain text without OSC 8 sequences.
func SetEnabled(e bool) {
	enabled.Store(e)
}

// Enabled reports whether hyperlinks are enabled.
func Enabled() bool {
	return enabled.Load()
}

// Snapshot captures the current state of all format pointers and the
// enabled flag. Use [Restore] to reset the state in test cleanup.
type Snapshot struct {
	path, file, dir, line, column *string
	enabled                       bool
}

// Save captures the current hyperlink configuration so it can be
// restored later with [Restore]. Typical usage in tests:
//
//	snap := hyperlink.Save()
//	t.Cleanup(func() { hyperlink.Restore(snap) })
func Save() Snapshot {
	return Snapshot{
		path:    pathFormat.Load(),
		file:    fileFormat.Load(),
		dir:     dirFormat.Load(),
		line:    lineFormat.Load(),
		column:  columnFormat.Load(),
		enabled: enabled.Load(),
	}
}

// Restore resets the hyperlink configuration to a previously saved [Snapshot].
func Restore(s Snapshot) {
	pathFormat.Store(s.path)
	fileFormat.Store(s.file)
	dirFormat.Store(s.dir)
	lineFormat.Store(s.line)
	columnFormat.Store(s.column)
	enabled.Store(s.enabled)
}

// ClearFormats resets all format pointers to nil (unset).
func ClearFormats() {
	pathFormat.Store(nil)
	fileFormat.Store(nil)
	dirFormat.Store(nil)
	lineFormat.Store(nil)
	columnFormat.Store(nil)
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

// ResolvePathURL builds the full hyperlink URL for a file path.
func ResolvePathURL(path string, line, column int) string {
	abs := absPath(path)
	return buildPathURL(abs, line, column, isDirectory(abs))
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
func buildPathURL(absPath string, line, column int, isDir bool) string {
	var fmtPtr *string

	switch {
	case isDir:
		fmtPtr = loadFormat(&dirFormat, &pathFormat)
	case column > 0:
		fmtPtr = loadFormat(&columnFormat, &lineFormat)
	case line > 0:
		fmtPtr = loadFormat(&lineFormat)
	default:
		fmtPtr = loadFormat(&fileFormat, &pathFormat)
	}

	if fmtPtr == nil {
		return "file://" + absPath
	}

	u := *fmtPtr
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

// expandPreset resolves a preset name to its format string for the given slot
// ("path", "line", or "column"). Returns value unchanged if it is not a known
// preset name, so full format strings pass through unmodified.
func expandPreset(value, slot string) string {
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

// loadFormat returns the first non-nil, non-empty format from the given pointers.
func loadFormat(ptrs ...*atomic.Pointer[string]) *string {
	for _, p := range ptrs {
		if f := p.Load(); f != nil && *f != "" {
			return f
		}
	}
	return nil
}
