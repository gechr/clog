# Configuration

## Default Logger

The package-level functions (`Info()`, `Warn()`, etc.) use the `Default` logger which writes to `os.Stdout` at `LevelInfo`.

To configure part order, symbols, styles, and spinner defaults together in one call, see [Presets](presets.md).

```go
// Full configuration
clog.Configure(&clog.Config{
  Verbose: true,                            // enables debug level + timestamps
  Output:  clog.Stderr(clog.ColorAuto),     // custom output
  Styles:  customStyles,                    // custom visual styles
})

// Toggle verbose mode
clog.SetVerbose(true)
```

Use `Default()` when an API needs the package logger itself. To replace it, construct a logger and call `SetDefault`:

```go
logger := clog.New(clog.Stderr(clog.ColorAuto))
clog.SetDefault(logger)

clog.Default().Info().Msg("Using the replacement logger")
```

## Output

Each `Logger` writes to an `*Output`, which bundles an `io.Writer` with its terminal capabilities (TTY detection, width, color profile):

```go
// Standard constructors
out := clog.Stdout(clog.ColorAuto)                  // os.Stdout with auto-detection
out := clog.Stderr(clog.ColorAlways)                // os.Stderr with forced colors
out := clog.NewOutput(w, clog.ColorNever)           // arbitrary writer, colors disabled
out := clog.TestOutput(&buf)                        // shorthand for NewOutput(w, ColorNever)
```

`Output` methods:

| Method             | Description                                                                |
| ------------------ | -------------------------------------------------------------------------- |
| `Writer()`         | Returns the underlying `io.Writer`                                         |
| `IsTTY()`          | True if the writer is connected to a terminal                              |
| `ColorsDisabled()` | True if colors are suppressed for this output                              |
| `Width()`          | Terminal width (0 for non-TTY, lazily cached)                              |
| `RefreshWidth()`   | Re-detect terminal width on next `Width()` call                            |

## Custom Logger

```go
logger := clog.New(clog.Stderr(clog.ColorAuto))
logger.SetLevel(clog.LevelDebug)
logger.SetReportTimestamp(true)
logger.SetTimeFormat("15:04:05.000")
logger.SetFieldTimeFormat(time.Kitchen)    // format for .Time() fields (default: time.RFC3339)
logger.SetTimeLocation(time.UTC)           // timezone for timestamps (default: time.Local)
logger.SetFieldStyleLevel(clog.LevelTrace) // min level for field value styling (default: clog.LevelInfo)
logger.SetNonTTYLevel(clog.LevelWarn)      // suppress below Warn on non-TTY writers
logger.SetHandler(myHandler)
```

For simple cases where you just need a writer with default color detection:

```go
logger := clog.NewWriter(os.Stderr) // equivalent to New(NewOutput(os.Stderr, ColorAuto))
```

## Field Formats

Field formatting (durations, elapsed timers, percentages, numbers, hyperlinks, quantity units) is configured per-logger via the `FieldFormats` struct. Start from `DefaultFieldFormats()`, set the fields you want, and apply with `SetFieldFormats` - configuration is per-`Logger`, so two loggers can format fields differently:

```go
f := clog.DefaultFieldFormats()
f.PercentPrecision = 1                  // "75.0%" instead of "75%"
f.ElapsedGradientMax = 30 * time.Second // enable the elapsed gradient
f.HyperlinkLineFormat = "vscode"        // preset name, expanded on SetFieldFormats
logger.SetFieldFormats(f)

// Or configure the package-level Default logger:
clog.SetFieldFormats(f)

// Read back the current configuration:
current := logger.FieldFormats()
```

`SetFieldFormats` has replace-all semantics (like `SetParts` and `SetFieldShapes`): the struct you pass replaces the logger's entire field-format configuration, so always start from `DefaultFieldFormats()` (or `logger.FieldFormats()`) rather than a zero value.

To change a single option without the read-modify-write dance, each field has a per-field convenience setter that preserves the rest of the snapshot - e.g. `logger.SetPercentPrecision(1)`, `logger.SetElapsedGradientMax(30*time.Second)`, `logger.SetHyperlinkFileFormat("vscode")` (presets are expanded and pushed to the output automatically). These also exist as package-level functions for the `Default` logger (`clog.SetPercentPrecision(1)`, etc.).

`SetTimeGradientMax(max)` is a shorthand that sets both `DurationGradientMax` and `ElapsedGradientMax` at once, since duration and elapsed fields usually share a gradient ceiling.

Time fields select rounding and decimal precision by magnitude with `TimeScale`. The default shared scale renders values below one second in milliseconds, values below ten seconds with up to one decimal place, and larger values as whole seconds. `DurationScale == nil` inherits that shared scale. Live `Elapsed` and `Deadline` fields default to a single-step whole-second `ElapsedScale` (`clog.TimeScale{{Round: time.Second}}`) so their width stays stable instead of changing as a timer crosses scale brackets.

```go
f := clog.DefaultFieldFormats()
f.TimeScale = clog.TimeScale{
  {Below: time.Second, Round: time.Millisecond},
  {Below: 10 * time.Second, Precision: 1, Round: 100 * time.Millisecond, Trim: true},
  {Round: time.Second},
}
f.ElapsedScale = nil // opt live elapsed/deadline fields into the shared scale
logger.SetFieldFormats(f)
```

`DurationScale` and `ElapsedScale` have three states: `nil` inherits `TimeScale`, a non-empty scale overrides it, and a non-nil empty scale disables rounding and decimal display entirely. `SetTimeScale` installs one shared scale and clears both field-specific overrides. A scale step's `Below` is an exclusive upper bound, zero is the catch-all, and `Trim` removes trailing fractional zeroes (for example, `1.0s` becomes `1s`).

| Field                     | Type                         | Default          | Description                                                                                  |
| ------------------------- | ---------------------------- | ---------------- | -------------------------------------------------------------------------------------------- |
| `DurationFormat`          | `func(time.Duration) string` | `nil` (built-in) | Custom formatter for `Duration` fields (also used for elapsed when `ElapsedFormat` is `nil`) |
| `DurationGradientMax`     | `time.Duration`              | `0` (disabled)   | Max duration for the `Duration` field gradient                                               |
| `DurationMinimum`         | `time.Duration`              | `0`              | Hide duration fields below this duration (`0` shows all values)                              |
| `DurationScale`           | `TimeScale`                  | `nil` (inherit)  | Duration-specific scale; nil inherits `TimeScale`, empty disables rounding                   |
| `ElapsedFormat`           | `func(time.Duration) string` | `nil` (built-in) | Custom formatter for elapsed fields (takes priority over `DurationFormat`)                   |
| `ElapsedGradientMax`      | `time.Duration`              | `0` (disabled)   | Max duration for the elapsed gradient                                                        |
| `ElapsedMinimum`          | `time.Duration`              | `time.Second`    | Hide elapsed fields below this duration (`0` shows all values)                               |
| `ElapsedScale`            | `TimeScale`                  | whole seconds    | Elapsed/deadline-specific scale; nil inherits `TimeScale`, empty disables rounding           |
| `TimeScale`               | `TimeScale`                  | three brackets   | Shared magnitude-keyed rounding and precision scale                                          |
| `HyperlinkEnabled`        | `bool`                       | `true`           | Enable/disable all hyperlink rendering                                                       |
| `HyperlinkColumnFormat`   | `string`                     | `""`             | URL format for file+line+column hyperlinks                                                   |
| `HyperlinkDirFormat`      | `string`                     | `""`             | URL format for directory hyperlinks                                                          |
| `HyperlinkFileFormat`     | `string`                     | `""`             | URL format for file-only hyperlinks                                                          |
| `HyperlinkLineFormat`     | `string`                     | `""`             | URL format for file+line hyperlinks                                                          |
| `HyperlinkPathFormat`     | `string`                     | `""`             | Generic fallback URL format for any path                                                     |
| `PercentFormat`           | `func(float64) string`       | `nil` (built-in) | Custom formatter for `Percent` fields (receives the display value, already scaled to 0–100)  |
| `PercentMaximum`          | `float64`                    | `0` (= `1.0`)    | Percent input maximum (`0` means `1.0` = fractions 0–1; set `100` for 0–100 input)           |
| `PercentPrecision`        | `int`                        | `0`              | Decimal places for `Percent` display (`0` = `75%`, `1` = `75.0%`)                            |
| `PercentReverseGradient`  | `bool`                       | `false`          | Reverse the percent gradient (green=0%, red=100%)                                            |
| `NumberFormat`            | `NumberFormat`               | `NumberPlain`    | How integers and both halves of fractions render (`plain`, `grouped`, `compact`)             |
| `FractionFormat`          | `*NumberFormat`              | `nil` (inherit)  | Overrides `NumberFormat` for fraction fields only (`nil` inherits `NumberFormat`)            |
| `NumberGroupSeparator`    | `string`                     | `","`            | Digit-group separator for `NumberGrouped` (e.g. `1,234,567`)                                 |
| `NumberCompactMinimum`    | `int64`                      | `1000`           | Smallest magnitude `NumberCompact` abbreviates; values below it use the fallback             |
| `NumberCompactFallback`   | `NumberFormat`               | `NumberGrouped`  | How `NumberCompact` renders sub-minimum values (`NumberGrouped` or `NumberPlain`)            |
| `QuantityUnitsIgnoreCase` | `bool`                       | `true`           | Case-insensitive quantity unit matching                                                      |

Hyperlink format fields accept either a full format string with `{path}`/`{line}`/`{column}` placeholders, or a named preset (e.g. `"vscode"`), which is expanded when `SetFieldFormats` is called. See [Hyperlinks](hyperlinks.md) for details.

### Number formatting

By default numbers render verbatim (`1234567`, `1234567/9999999`). Three modes control how integer fields and both halves of a `Fraction` are rendered:

- `NumberPlain` - verbatim, e.g. `1234567` (the default).
- `NumberGrouped` - locale-style digit grouping, e.g. `1,234,567`. The separator is configurable.
- `NumberCompact` - abbreviated with K/M/B/T suffixes, e.g. `1.2M`. Values below `NumberCompactMinimum` (default `1000`) render with `NumberCompactFallback` (grouped by default), so a series reads `9,999` → `10K` → `11K` rather than jumping straight from plain to abbreviated. Set `NumberCompactFallback = NumberPlain` to keep small values verbatim (`9999`).

Convenience setters avoid the read-modify-write dance for the common cases:

```go
logger.SetNumberFormat(clog.NumberGrouped)   // applies to ints AND fractions
logger.SetNumberGroupSeparator(" ")          // "1 234 567"

logger.SetFractionFormat(clog.NumberCompact) // fractions only; ints keep NumberFormat
logger.SetNumberCompactMinimum(10_000)       // only abbreviate at >= 10,000
```

To combine grouping and abbreviation - grouped digits for small values, K/M/B/T suffixes for large ones - select `NumberCompact` and raise the minimum:

```go
logger.SetNumberFormat(clog.NumberCompact)
logger.SetNumberCompactMinimum(10_000) // 9,999 -> 10K -> 11K -> 1.2M
```

`SetNumberFormat` is the global knob; `SetFractionFormat` overrides it for fractions and falls back to it when unset. A single field can override both via `fraction.WithFormat`:

```go
clog.Info("progress").
    Fraction("done", 1234567, 9999999, fraction.WithFormat(clog.NumberCompact)).
    Send() // done=1.2M/10M
```

The separator is not locale-aware - pick the one that suits your output (`","`, `"."`, `" "`, `"_"`).

## Utility Functions

```go
clog.GetLevel()                  // returns the current level of the Default logger
clog.IsVerbose()                 // true if level is Debug or Trace
clog.IsTerminal()                // true if Default output is a terminal
clog.ColorsDisabled()            // true if colors are disabled on the Default logger
clog.SetOutput(out)              // change the output (accepts *Output)
clog.SetOutputWriter(w)          // change the output writer (with ColorAuto)
clog.SetExitCode(2)              // set default Fatal exit code (default: 1)
clog.SetExitFunc(fn)             // override os.Exit for Fatal (useful in tests)
logger.Output()                  // returns the Logger's *Output
```

## Environment Variables

All env vars follow the pattern `{PREFIX}_{SUFFIX}`. The default prefix is `CLOG`.

| Suffix                    | Default env var                |
| ------------------------- | ------------------------------ |
| `LOG_LEVEL`               | `CLOG_LOG_LEVEL`               |
| `HYPERLINK_FORMAT`        | `CLOG_HYPERLINK_FORMAT`        |
| `HYPERLINK_PATH_FORMAT`   | `CLOG_HYPERLINK_PATH_FORMAT`   |
| `HYPERLINK_FILE_FORMAT`   | `CLOG_HYPERLINK_FILE_FORMAT`   |
| `HYPERLINK_DIR_FORMAT`    | `CLOG_HYPERLINK_DIR_FORMAT`    |
| `HYPERLINK_LINE_FORMAT`   | `CLOG_HYPERLINK_LINE_FORMAT`   |
| `HYPERLINK_COLUMN_FORMAT` | `CLOG_HYPERLINK_COLUMN_FORMAT` |

```sh
CLOG_LOG_LEVEL=debug ./some-app  # enables debug logging + timestamps
CLOG_LOG_LEVEL=warn ./some-app   # suppresses info messages
```

## Custom Env Prefix

Use `SetEnvPrefix` to whitelabel the env var names for your application. The custom prefix is checked first, with `CLOG_` as a fallback.

```go
clog.SetEnvPrefix("MYAPP")
// Now checks MYAPP_LOG_LEVEL first, then CLOG_LOG_LEVEL
// Now checks MYAPP_HYPERLINK_PATH_FORMAT first, then CLOG_HYPERLINK_PATH_FORMAT
// etc.
```

This means `CLOG_LOG_LEVEL=debug` always works as a universal escape hatch, even when the application uses a custom prefix.

`NO_COLOR` is never prefixed - it follows the [no-color.org](https://no-color.org/) standard independently.
