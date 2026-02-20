package clog

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
)

// hyperlinkPreset holds the per-slot URL format templates for a named editor preset.
// path is used for the path, file, and dir format slots; line and column for their
// respective slots.
type hyperlinkPreset struct {
	description string
	path        string
	line        string
	column      string
}

// hyperlinkPresets maps short preset names (lower-case) to their format templates.
var hyperlinkPresets = map[string]hyperlinkPreset{
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

// hyperlinkColumnFormat holds the URL format for file+line+column hyperlinks.
// Use {path}, {line}, and {column} (or {col}) as placeholders. Nil means fall back to line format.
var hyperlinkColumnFormat atomic.Pointer[string]

// hyperlinkDirFormat holds the URL format for directory hyperlinks.
// Falls back to hyperlinkPathFormat if nil.
var hyperlinkDirFormat atomic.Pointer[string]

// hyperlinkFileFormat holds the URL format for file-only hyperlinks (no line number).
// Falls back to hyperlinkPathFormat if nil.
var hyperlinkFileFormat atomic.Pointer[string]

// hyperlinkLineFormat holds the URL format for file+line hyperlinks.
// Use {path} and {line} as placeholders. Nil means use default (file://{path}).
var hyperlinkLineFormat atomic.Pointer[string]

// hyperlinkPathFormat is the generic fallback URL format for any path.
// Use {path} as placeholder. Nil means use default (file://{path}).
var hyperlinkPathFormat atomic.Pointer[string]

// hyperlinksEnabled controls whether hyperlinks are rendered at all.
var hyperlinksEnabled atomic.Bool

// SetHyperlinkColumnFormat configures the URL format for file+line+column hyperlinks
// (used by [Column]).
//
// Accepts a full format string or a preset name (e.g. "vscode"). Known presets:
// cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
//
// Use {path}, {line}, and {column} (or {col}) as placeholders. Examples:
//
//   - vscode://file{path}:{line}:{column}
//   - idea://open?file={path}&line={line}&column={column}
//
// Default (empty): falls back to the line format.
func SetHyperlinkColumnFormat(format string) {
	hyperlinkColumnFormat.Store(new(expandPreset(format, "column")))
}

// SetHyperlinkDirFormat configures the URL format for directory hyperlinks.
//
// Accepts a full format string or a preset name (e.g. "vscode"). Known presets:
// cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
//
// Falls back to [SetHyperlinkPathFormat] if not set.
//
// Use {path} as placeholder.
func SetHyperlinkDirFormat(format string) {
	hyperlinkDirFormat.Store(new(expandPreset(format, "path")))
}

// SetHyperlinkFileFormat configures the URL format for file-only hyperlinks
// (used by [Path] and [PathLink] with line 0, when the path is not a directory).
//
// Accepts a full format string or a preset name (e.g. "vscode"). Known presets:
// cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
//
// Falls back to [SetHyperlinkPathFormat] if not set.
//
// Use {path} as placeholder.
func SetHyperlinkFileFormat(format string) {
	hyperlinkFileFormat.Store(new(expandPreset(format, "path")))
}

// SetHyperlinkLineFormat configures the URL format for file+line hyperlinks
// (used by [Line] and [PathLink] with line > 0).
//
// Accepts a full format string or a preset name (e.g. "vscode"). Known presets:
// cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
//
// Use {path} and {line} as placeholders. Examples:
//
//   - vscode://file{path}:{line}
//   - idea://open?file={path}&line={line}
//   - subl://open?url=file://{path}&line={line}
//
// Default (empty): file://{path}
func SetHyperlinkLineFormat(format string) {
	hyperlinkLineFormat.Store(new(expandPreset(format, "line")))
}

// SetHyperlinkPathFormat configures the generic fallback URL format for any path.
// This is used when no file-specific or directory-specific format is configured.
//
// Accepts a full format string or a preset name (e.g. "vscode"). Known presets:
// cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
//
// Use {path} as placeholder. Examples:
//
//   - vscode://file{path}
//   - idea://open?file={path}
//
// Default (empty): file://{path}
func SetHyperlinkPathFormat(format string) {
	hyperlinkPathFormat.Store(new(expandPreset(format, "path")))
}

// SetHyperlinkPreset configures all hyperlink format slots using a named preset.
// This is a convenience wrapper around the individual SetHyperlink*Format functions.
//
// Known presets: cursor, kitty, macvim, textmate, vscode, vscode-insiders, vscodium.
//
// Individual formats set afterwards (via SetHyperlink*Format or environment variables)
// override the preset for that specific slot.
func SetHyperlinkPreset(name string) error {
	p, ok := hyperlinkPresets[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return fmt.Errorf("clog: unknown hyperlink preset %q", name)
	}
	hyperlinkPathFormat.Store(new(p.path))
	hyperlinkFileFormat.Store(new(p.path))
	hyperlinkDirFormat.Store(new(p.path))
	hyperlinkLineFormat.Store(new(p.line))
	hyperlinkColumnFormat.Store(new(p.column))
	return nil
}

// SetHyperlinksEnabled enables or disables all hyperlink rendering.
// When disabled, hyperlink functions return plain text without OSC 8 sequences.
func SetHyperlinksEnabled(enabled bool) {
	hyperlinksEnabled.Store(enabled)
}

// Hyperlink wraps text in an OSC 8 terminal hyperlink escape sequence.
// Returns plain text when colours or hyperlinks are disabled globally.
func Hyperlink(url, text string) string {
	if !hyperlinksEnabled.Load() || ColorsDisabled() {
		return text
	}
	return osc8(url, text)
}

// PathLink creates a clickable terminal hyperlink for a file path.
// The line parameter is optional — pass 0 to omit line numbers.
func PathLink(path string, line int) string {
	display := pathDisplayText(path, line, 0)

	if !hyperlinksEnabled.Load() || ColorsDisabled() {
		return display
	}
	return Hyperlink(resolvePathURL(path, line, 0), display)
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

func buildPathURL(absPath string, line, column int, isDir bool) string {
	var fmtPtr *string

	switch {
	case isDir:
		// dirFormat -> pathFormat -> file://
		fmtPtr = loadFormat(&hyperlinkDirFormat, &hyperlinkPathFormat)
	case column > 0:
		// columnFormat -> lineFormat -> file://
		fmtPtr = loadFormat(&hyperlinkColumnFormat, &hyperlinkLineFormat)
	case line > 0:
		// lineFormat -> file://
		fmtPtr = loadFormat(&hyperlinkLineFormat)
	default:
		// fileFormat -> pathFormat -> file://
		fmtPtr = loadFormat(&hyperlinkFileFormat, &hyperlinkPathFormat)
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

// hyperlink is like [Hyperlink] but uses the Output's colour settings.
func (o *Output) hyperlink(url, text string) string {
	if !hyperlinksEnabled.Load() || o.ColorsDisabled() {
		return text
	}
	return osc8(url, text)
}

// isDirectory reports whether path is an existing directory.
func isDirectory(path string) bool {
	//nolint:gosec // path comes from the caller's own code, not user input
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// expandPreset resolves a preset name to its format string for the given slot
// ("path", "line", or "column"). Returns value unchanged if it is not a known
// preset name, so full format strings pass through unmodified.
func expandPreset(value, slot string) string {
	p, ok := hyperlinkPresets[strings.ToLower(strings.TrimSpace(value))]
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

// osc8 wraps text in raw OSC 8 escape sequences unconditionally.
func osc8(url, text string) string {
	return "\x1b]8;;" + url + "\x1b\\" + text + "\x1b]8;;\x1b\\"
}

// pathDisplayText returns the display string for a path hyperlink.
func pathDisplayText(path string, line, column int) string {
	if column > 0 && line > 0 {
		return path + ":" + strconv.Itoa(line) + ":" + strconv.Itoa(column)
	}

	if line > 0 {
		return path + ":" + strconv.Itoa(line)
	}
	return path
}

// pathLink is like [PathLink] but uses the Output's colour settings.
func (o *Output) pathLink(path string, line, column int) string {
	display := pathDisplayText(path, line, column)

	if !hyperlinksEnabled.Load() || o.ColorsDisabled() {
		return display
	}
	return osc8(resolvePathURL(path, line, column), display)
}

// resolvePathURL builds the full hyperlink URL for a file path.
func resolvePathURL(path string, line, column int) string {
	abs := absPath(path)
	return buildPathURL(abs, line, column, isDirectory(abs))
}
