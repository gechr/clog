# Hyperlinks

Render clickable terminal hyperlinks using OSC 8 escape sequences:

```go
// Typed field methods (recommended)
clog.Info().Path("dir", "src/").Msg("Directory")
clog.Info().PathText("bin", "~/bin/foo", "/home/user/bin/foo").Msg("Label differs from target")
clog.Info().Line("file", "config.yaml", 42).Msg("File with line")
clog.Info().Column("loc", "main.go", 42, 10).Msg("File with line and column")
clog.Info().URL("docs", "https://example.com/docs").Msg("See docs")
clog.Info().Link("docs", "https://example.com", "docs").Msg("URL")

// Slice variants
clog.Info().URLs("refs", []string{"https://a.com", "https://b.com"}).Msg("References")
clog.Info().Links("repos", []clog.Link{
    {URL: "https://github.com/foo/bar", Text: "foo/bar"},
    {URL: "https://github.com/baz/qux", Text: "baz/qux"},
}).Msg("Repositories")

// Standalone functions (for use with Str)
link := clog.PathLink("config.yaml", 42)                        // file path with line number
link := clog.PathLink("src/", 0)                                // directory (no line number)
link := clog.PathLinkText("~/bin/foo", "/home/user/bin/foo", 0) // custom label, links to full path
link := clog.Hyperlink("https://example.com", "docs")           // arbitrary URL
```

## Custom link labels

`PathText` (and the standalone `PathLinkText`) render a visible label that differs from the link target, so you can show an abbreviated or home-contracted path while still linking to its full location:

```go
// Display ~/bin/foo, but the hyperlink resolves to /home/user/bin/foo.
clog.Warn().PathText("path", "~/bin/foo", "/home/user/bin/foo").Msg("Shadowing binary on $PATH")
```

## IDE Integration

Configure hyperlinks to open files directly in your editor via [`FieldFormats`](configuration.md#field-formats):

```go
f := clog.DefaultFieldFormats()

// Generic fallback for any path (file or directory)
f.HyperlinkPathFormat = "vscode://file{path}"

// File-specific (overrides path format for files)
f.HyperlinkFileFormat = "vscode://file{path}"

// Directory-specific (overrides path format for directories)
f.HyperlinkDirFormat = "finder://{path}"

// File+line hyperlinks (Line, PathLink with line > 0)
f.HyperlinkLineFormat = "vscode://file{path}:{line}"
f.HyperlinkLineFormat = "idea://open?file={path}&line={line}"

// File+line+column hyperlinks (Column)
f.HyperlinkColumnFormat = "vscode://file{path}:{line}:{column}"

clog.SetFieldFormats(f)
```

Use `{path}`, `{line}`, and `{column}` (or `{col}`) as placeholders. Default format is `file://{path}`.

Format resolution order:

| Context        | Fallback chain                                                       |
| -------------- | -------------------------------------------------------------------- |
| Directory      | `HyperlinkDirFormat`    -> `HyperlinkPathFormat` -> `file://{path}`  |
| File (no line) | `HyperlinkFileFormat`   -> `HyperlinkPathFormat` -> `file://{path}`  |
| File + line    | `HyperlinkLineFormat`   -> `file://{path}`                           |
| File + column  | `HyperlinkColumnFormat` -> `HyperlinkLineFormat` -> `file://{path}`  |

These can also be set via environment variables:

```sh
export CLOG_HYPERLINK_FORMAT="vscode"                      # named preset (sets all slots)
export CLOG_HYPERLINK_PATH_FORMAT="vscode://{path}"        # generic fallback
export CLOG_HYPERLINK_FILE_FORMAT="vscode://file{path}"    # files only
export CLOG_HYPERLINK_DIR_FORMAT="finder://{path}"         # directories only
export CLOG_HYPERLINK_LINE_FORMAT="vscode://{path}:{line}"
export CLOG_HYPERLINK_COLUMN_FORMAT="vscode://{path}:{line}:{column}"
```

`CLOG_HYPERLINK_FORMAT` accepts a preset name and configures all slots at once. Individual format vars override the preset for their specific slot. Environment variables apply to the `Default` logger's `FieldFormats`.

## Named Presets

Hyperlink format fields accept a preset name (e.g. `"vscode"`) in place of a full format string - it is expanded for that slot when `SetFieldFormats` is called:

```go
f := clog.DefaultFieldFormats()
f.HyperlinkLineFormat = "vscode"
clog.SetFieldFormats(f)
```

To apply a preset to all format slots at once, use `hyperlink.Preset` (or set `CLOG_HYPERLINK_FORMAT=vscode` in the environment):

```go
import "github.com/gechr/clog/field/hyperlink"

cfg, err := hyperlink.Preset("vscode")
if err != nil {
  // unknown preset name
}

f := clog.DefaultFieldFormats()
f.HyperlinkPathFormat = cfg.PathFormat
f.HyperlinkFileFormat = cfg.FileFormat
f.HyperlinkDirFormat = cfg.DirFormat
f.HyperlinkLineFormat = cfg.LineFormat
f.HyperlinkColumnFormat = cfg.ColumnFormat
clog.SetFieldFormats(f)
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
f := clog.DefaultFieldFormats()
f.HyperlinkEnabled = false // disable all hyperlink rendering
clog.SetFieldFormats(f)
```

Hyperlinks are automatically disabled when colors are disabled.
