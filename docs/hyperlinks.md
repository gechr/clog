# Hyperlinks

Render clickable terminal hyperlinks using OSC 8 escape sequences:

```go
// Typed field methods (recommended)
clog.Info().Path("dir", "src/").Msg("Directory")
clog.Info().Line("file", "config.yaml", 42).Msg("File with line")
clog.Info().Column("loc", "main.go", 42, 10).Msg("File with line and column")
clog.Info().URL("docs", "https://example.com/docs").Msg("See docs")
clog.Info().Link("docs", "https://example.com", "docs").Msg("URL")

// Standalone functions (for use with Str)
link := clog.PathLink("config.yaml", 42)               // file path with line number
link := clog.PathLink("src/", 0)                       // directory (no line number)
link := clog.Hyperlink("https://example.com", "docs")  // arbitrary URL
```

## IDE Integration

Configure hyperlinks to open files directly in your editor:

```go
// Generic fallback for any path (file or directory)
clog.SetHyperlinkPathFormat("vscode://file{path}")

// File-specific (overrides path format for files)
clog.SetHyperlinkFileFormat("vscode://file{path}")

// Directory-specific (overrides path format for directories)
clog.SetHyperlinkDirFormat("finder://{path}")

// File+line hyperlinks (Line, PathLink with line > 0)
clog.SetHyperlinkLineFormat("vscode://file{path}:{line}")
clog.SetHyperlinkLineFormat("idea://open?file={path}&line={line}")

// File+line+column hyperlinks (Column)
clog.SetHyperlinkColumnFormat("vscode://file{path}:{line}:{column}")
```

Use `{path}`, `{line}`, and `{column}` (or `{col}`) as placeholders. Default format is `file://{path}`.

Format resolution order:

| Context        | Fallback chain                                    |
| -------------- | ------------------------------------------------- |
| Directory      | `DirFormat`    -> `PathFormat` -> `file://{path}` |
| File (no line) | `FileFormat`   -> `PathFormat` -> `file://{path}` |
| File + line    | `LineFormat`   -> `file://{path}`                 |
| File + column  | `ColumnFormat` -> `LineFormat` -> `file://{path}` |

These can also be set via environment variables:

```sh
export CLOG_HYPERLINK_FORMAT="vscode"                      # named preset (sets all slots)
export CLOG_HYPERLINK_PATH_FORMAT="vscode://{path}"        # generic fallback
export CLOG_HYPERLINK_FILE_FORMAT="vscode://file{path}"    # files only
export CLOG_HYPERLINK_DIR_FORMAT="finder://{path}"         # directories only
export CLOG_HYPERLINK_LINE_FORMAT="vscode://{path}:{line}"
export CLOG_HYPERLINK_COLUMN_FORMAT="vscode://{path}:{line}:{column}"
```

`CLOG_HYPERLINK_FORMAT` accepts a preset name and configures all slots at once. Individual format vars override the preset for their specific slot.

## Named Presets

```go
clog.SetHyperlinkPreset("vscode") // or CLOG_HYPERLINK_FORMAT=vscode
```

| Preset            | Scheme                 |
| ----------------- | ---------------------- |
| `cursor`          | `cursor://`            |
| `kitty`           | `file://` with `#line` |
| `macvim`          | `mvim://`              |
| `subl`            | `subl://`              |
| `textmate`        | `txmt://`              |
| `vscode`          | `vscode://`            |
| `vscode-insiders` | `vscode-insiders://`   |
| `vscodium`        | `vscodium://`          |

## Enabling / Disabling

Hyperlinks are enabled by default (when colors are active). Disable them programmatically:

```go
clog.SetHyperlinkEnabled(false) // disable all hyperlink rendering
clog.SetHyperlinkEnabled(true)  // re-enable
```

Hyperlinks are automatically disabled when colors are disabled.
