# Levels

| Level   | Value | Label | Symbol | Description                                          |
| ------- | ----: | ----- | ------ | ---------------------------------------------------- |
| `Trace` |   -10 | `TRC` | 🔍     | Finest-grained output, hidden by default             |
| `Debug` |    -5 | `DBG` | 🐞     | Verbose output, hidden by default                    |
| `Info`  |     0 | `INF` | ℹ️     | General operational messages (default minimum level) |
| `Dry`   |     2 | `DRY` | 🚧     | Dry-run indicators                                   |
| `Warn`  |     5 | `WRN` | ⚠️     | Warnings that don't prevent operation                |
| `Error` |    10 | `ERR` | ❌     | Errors that need attention                           |
| `Fatal` |    15 | `FTL` | 💥     | Fatal errors - calls `os.Exit(1)` after logging      |

Built-in levels use gaps of 5 between them (except around `Dry`, which sits at 2), leaving room for custom levels at any position (see [Custom Levels](#custom-levels)).

## Setting the Level

```go
// Programmatically
clog.SetLevel(clog.LevelDebug)

// From environment variable (CLOG_LOG_LEVEL is checked automatically on init)
// export CLOG_LOG_LEVEL=debug
```

Recognised `CLOG_LOG_LEVEL` values: `trace`, `debug`, `info`, `dry`, `warn`, `warning`, `error`, `fatal`, `critical`.

Setting `trace` or `debug` also enables timestamps.

## Parsing Levels

`ParseLevel` converts a string to a `Level` value (case-insensitive):

```go
level, err := clog.ParseLevel("debug")
```

`Level` implements `encoding.TextMarshaler` and `encoding.TextUnmarshaler`, so it works directly with `flag.TextVar` and most flag libraries.

## Non-TTY Level

When output is piped or running in CI (non-TTY), you may want to suppress lower-severity messages while keeping them visible during interactive use. `SetNonTTYLevel` sets a separate minimum level that only applies to non-TTY writers:

```go
clog.SetNonTTYLevel(clog.LevelWarn)
```

This suppresses `Trace`, `Debug`, and `Info` events when stdout is not a terminal, but leaves them visible during interactive use. The setting also applies to animation progress lines (spinners, bars, etc.).

Pass `UnsetLevel` to remove the filter and restore default behaviour:

```go
clog.SetNonTTYLevel(clog.UnsetLevel)
```

## Custom Levels

Define custom levels at any numeric value between the built-in levels. Use `RegisterLevel` to configure the label, symbol, style, and canonical name.

```go
const SuccessLevel clog.Level = clog.LevelInfo + 1

func init() {
    clog.RegisterLevel(SuccessLevel, clog.LevelConfig{
        Name:   "success",
        Label:  "SCS",
        Symbol: "✅",
        Style:  new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))),
    })
}
```

Log with `clog.Log(level)`:

```go
clog.Log(SuccessLevel).Msg("Build completed")
// SCS ✅ Build completed
```

Custom levels respect level filtering based on their numeric value. `ParseLevel`, `MarshalText`, and `UnmarshalText` all work with registered custom levels.

Use `clog.Levels()` to iterate all registered levels (built-in and custom) in ascending severity order.
