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

The `Printer` writes directly to the logger's output, bypassing any custom [Handler](handlers.md). It uses the logger's style configurations for token colors.

## JSON

`JSON` marshals any Go value; `RawJSON` accepts pre-serialized bytes:

```go
clog.Print().JSON(userStruct)
clog.Print().RawJSON(responseBody)
```

### Print Mode

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

### Styling

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

See [YAML](yaml.md) for styling options.

## TOML

`TOML` marshals any Go value; `RawTOML` accepts pre-serialized bytes:

```go
clog.Print().TOML(configStruct)
clog.Print().RawTOML(configBytes)
```

See [TOML](toml.md) for styling options.

## HCL

`RawHCL` accepts pre-serialized HCL bytes (there is no marshal method since HCL has no standard Go marshal API):

```go
clog.Print().RawHCL(terraformConfig)
```

See [HCL](hcl.md) for styling options.

## Themes

Printer styles default to terminal-aware light/dark selection: on the first colored write the logger detects the background of its own output and picks a matching theme (dark mode preserves the original Dracula-based colors). The terminal is queried at most once, and never for non-terminal or color-disabled outputs, which use the dark theme. Switch all four format styles at once with `SetPrintTheme`:

```go
clog.SetPrintTheme(theme.Light())
```

Per-token overrides still work after setting a theme:

```go
clog.SetPrintTheme(theme.Monokai())
s := clog.DefaultStyles()
s.JSON.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000")))
clog.SetStyles(s)
```

To build styles from a theme directly:

```go
custom := style.NewJSON(theme.Monokai())
```

To require an explicit light/dark pair:

```go
themes := theme.MustPair(theme.CatppuccinLatte(), theme.Dracula())
clog.SetPrintTheme(themes.Auto())
```

### Environment variables

The default logger reads its theme from the environment (highest precedence first):

1. `CLOG_THEME` - an explicit theme, applied directly with no background detection:

   ```sh
   CLOG_THEME=monokai
   ```

2. `CLOG_THEME_LIGHT` and `CLOG_THEME_DARK` - a light/dark pair; the entry matching the terminal background is selected on the first write:

   ```sh
   CLOG_THEME_LIGHT=catppuccin-latte
   CLOG_THEME_DARK=dracula
   ```

`CLOG_THEME` takes precedence over the light/dark pair when both are set. With a custom prefix (see `SetEnvPrefix`), `<PREFIX>_THEME*` is checked first, then the `CLOG_*` fallback.

The same pair can also be loaded programmatically:

```go
themes, err := theme.PairFromEnv()
if err != nil {
    return err
}
clog.SetPrintTheme(themes.Auto())
```

Available themes:

- `theme.Dark()`
- `theme.Light()`
- `theme.CatppuccinFrappe()`
- `theme.CatppuccinLatte()`
- `theme.CatppuccinMacchiato()`
- `theme.CatppuccinMocha()`
- `theme.Dracula()`
- `theme.Monokai()`

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
