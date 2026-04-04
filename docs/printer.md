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

The `Printer` writes directly to the logger's output, bypassing any custom [Handler](handlers.md). It uses the logger's `FieldJSON` style configuration, so JSON output is highlighted identically to inline `RawJSON` fields.

## JSON

`JSON` marshals any Go value; `RawJSON` accepts pre-serialized bytes:

```go
clog.Print().JSON(userStruct)
clog.Print().RawJSON(responseBody)
```

## Print Mode

The default mode is `PrintMultiline`, which pretty-prints with indentation. Set the global default with `SetPrintMode`, or override per-call with `Mode`:

```go
// Global default: flatten to single line
clog.SetPrintMode(clog.PrintInline)

// Per-call override
clog.Print().Mode(clog.PrintMultiline).RawJSON(data)
clog.Print().Mode(clog.PrintInline).RawJSON(data)
```

| Mode             | Description                                              |
| ---------------- | -------------------------------------------------------- |
| `PrintMultiline` | Pretty-print with indentation (default)                  |
| `PrintInline`    | Flatten to a single line (matches inline log fields)     |

## Indentation

The default indent is two spaces. Customise with `SetPrintIndent`:

```go
clog.SetPrintIndent("\t")       // tabs
clog.SetPrintIndent("    ")     // 4 spaces
```

## Styling

Printer JSON inherits token colors (keys, strings, numbers, etc.) from the logger's `FieldJSON` styles. Field-specific rendering modes (`JSONModeHuman`, `JSONModeFlat`) are not applied - the Printer always uses standard JSON rendering.

```go
custom := style.DefaultJSON()
custom.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
clog.SetStyles(&style.Config{FieldJSON: custom})

clog.Print().RawJSON(data)
```

## Sub-loggers

Each logger has its own `Printer`, so sub-loggers with different styles produce different output:

```go
logger := clog.NewWriter(os.Stdout)
logger.SetStyles(&style.Config{
    FieldJSON: style.DefaultJSON().WithSpacing(style.JSONSpacingAll),
})

logger.Print().RawJSON(data)
```
