# Printer

Output styled data directly without a log level, key, or message:

```go
clog.Print().RawJSON([]byte(`{"status":"ok","count":42,"active":true}`))
// {
//   "status": "ok",
//   "count": 42,
//   "active": true
// }
```

The `Printer` writes directly to the logger's output, bypassing any custom [Handler](handlers.md). It uses the logger's `JSON` and `YAML` style configurations for token colors.

## JSON

`JSON` marshals any Go value; `RawJSON` accepts pre-serialized bytes:

```go
clog.Print().JSON(userStruct)
clog.Print().RawJSON(responseBody)
```

### JSON Print Mode

The default mode is `JSONPretty`, which pretty-prints with indentation. Set the global default with `SetJSONPrintMode`, or override per-call with `Mode`:

```go
// Global default: flatten to single line
clog.SetJSONPrintMode(clog.JSONFlat)

// Per-call override
clog.Print().Mode(clog.JSONPretty).RawJSON(data)
clog.Print().Mode(clog.JSONFlat).RawJSON(data)
```

| Mode           | Description                                            |
| -------------- | ------------------------------------------------------ |
| `JSONPretty`   | Pretty-print with normalized indentation (default)     |
| `JSONFlat`     | Flatten to a single line (matches inline log fields)   |
| `JSONPreserve` | Keep original whitespace, only add syntax highlighting |

### JSON Styling

Printer JSON inherits token colors (keys, strings, numbers, etc.) from the logger's `JSON` styles. Field-specific rendering modes (`JSONModeHuman`, `JSONModeFlat`) are not applied -- the Printer always uses standard JSON rendering.

```go
custom := style.DefaultJSON()
custom.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
clog.SetStyles(&style.Config{JSON: custom})
```

## YAML

`YAML` marshals any Go value; `RawYAML` accepts pre-serialized bytes:

```go
clog.Print().YAML(configStruct)
clog.Print().RawYAML(responseBody)
```

### YAML Styling

Token colors are configured via `YAML` in styles:

```go
custom := style.DefaultYAML()
custom.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
clog.SetStyles(&style.Config{YAML: custom})
```

| Style field | Tokens                                            |
| ----------- | ------------------------------------------------- |
| `Anchor`    | `&name`                                           |
| `Alias`     | `*name`                                           |
| `Bool`      | `true`, `false`, `yes`, `no`, `on`, `off`         |
| `Comment`   | `# text`                                          |
| `Key`       | mapping keys                                      |
| `Null`      | `null`, `~`                                       |
| `Number`    | integers, floats, hex, octal, binary, inf, nan    |
| `String`    | plain, single-quoted, double-quoted string values |
| `Tag`       | `!!str`, `!!int`, `!custom`                       |

## Indentation

The default indent is two spaces. `SetPrintIndent` sets the baseline for all formats; `SetJSONIndent` and `SetYAMLIndent` override it for a specific format:

```go
clog.SetPrintIndent("\t")       // tabs for both JSON and YAML
clog.SetJSONIndent("    ")      // 4 spaces for JSON only
clog.SetYAMLIndent("  ")        // 2 spaces for YAML only
```

YAML sequences are indented under their parent key by default. Disable with `SetYAMLIndentSequence(false)`.

## Sub-loggers

Each logger has its own `Printer`, so sub-loggers with different styles produce different output:

```go
logger := clog.NewWriter(os.Stdout)
logger.SetStyles(&style.Config{
    JSON: style.DefaultJSON().WithSpacing(style.JSONSpacingAll),
})

logger.Print().RawJSON(data)
```
