<h1 align="center"><code>clog</code></h1>

Structured CLI logging for Go with terminal-aware colours, hyperlinks, and spinners. A [zerolog](https://github.com/rs/zerolog)-style fluent API designed for command-line tools.

## Demo

<p align="center">
  <img src="./assets/demo.gif" alt="clog demo" />
</p>

## Installation

```sh
go get github.com/gechr/clog
```

## Quick Start

```go
package main

import "github.com/gechr/clog"

func main() {
  clog.Info().Str("port", "8080").Msg("Server started")
  clog.Warn().Str("path", "/old").Msg("Deprecated endpoint")
  clog.Error().Err(err).Msg("Connection failed")
}
```

Output:

```text
INF ℹ️ Server started port=8080
WRN ⚠️ Deprecated endpoint path=/old
ERR ❌ Connection failed error=connection refused
```

## Levels

| Level   | Label | Prefix | Description                                          |
| ------- | ----- | ------ | ---------------------------------------------------- |
| `Trace` | `TRC` | 🔬     | Finest-grained output, hidden by default             |
| `Debug` | `DBG` | 🔍     | Verbose output, hidden by default                    |
| `Info`  | `INF` | ℹ️     | General operational messages (default minimum level) |
| `Dry`   | `DRY` | 🚧     | Dry-run indicators                                   |
| `Warn`  | `WRN` | ⚠️     | Warnings that don't prevent operation                |
| `Error` | `ERR` | ❌     | Errors that need attention                           |
| `Fatal` | `FTL` | 💥     | Fatal errors - calls `os.Exit(1)` after logging      |

### Setting the Level

```go
// Programmatically
clog.SetLevel(clog.DebugLevel)

// From environment variable (checked automatically on init)
// export CLOG_LEVEL=debug
clog.SetLevelFromEnv("CLOG_LEVEL")
```

Recognised `CLOG_LEVEL` values: `trace`, `debug`, `info`, `dry`, `warn`, `warning`, `error`, `fatal`.

Setting `trace` or `debug` also enables timestamps.

## Structured Fields

Events and contexts support typed field methods. All methods are safe to call on a nil receiver (disabled events are no-ops).

### Event Fields

| Method      | Signature                                    | Description                           |
| ----------- | -------------------------------------------- | ------------------------------------- |
| `Str`       | `Str(key, val string)`                       | String field                          |
| `Strs`      | `Strs(key string, vals []string)`            | String slice field                    |
| `Int`       | `Int(key string, val int)`                   | Integer field                         |
| `Ints`      | `Ints(key string, vals []int)`               | Integer slice field                   |
| `Uint64`    | `Uint64(key string, val uint64)`             | Unsigned integer field                |
| `Uints64`   | `Uints64(key string, vals []uint64)`         | Unsigned integer slice field          |
| `Float64`   | `Float64(key string, val float64)`           | Float field                           |
| `Floats64`  | `Floats64(key string, vals []float64)`       | Float slice field                     |
| `Bool`      | `Bool(key string, val bool)`                 | Boolean field                         |
| `Bools`     | `Bools(key string, vals []bool)`             | Boolean slice field                   |
| `Dur`       | `Dur(key string, val time.Duration)`         | Duration field                        |
| `Time`      | `Time(key string, val time.Time)`            | Time field                            |
| `Err`       | `Err(err error)`                             | Error field (key `"error"`, nil-safe) |
| `Any`       | `Any(key string, val any)`                   | Arbitrary value                       |
| `Anys`      | `Anys(key string, vals []any)`               | Arbitrary value slice                 |
| `Dict`      | `Dict(key string, dict *Event)`              | Nested fields with dot-notation keys  |
| `Path`      | `Path(key, path string)`                     | Clickable file/directory hyperlink    |
| `Line`      | `Line(key, path string, line int)`           | Clickable file:line hyperlink         |
| `Column`    | `Column(key, path string, line, column int)` | Clickable file:line:column hyperlink  |
| `Link`      | `Link(key, url, text string)`                | Clickable URL hyperlink               |
| `Stringer`  | `Stringer(key string, val fmt.Stringer)`     | Calls `String()` (nil-safe)           |
| `Stringers` | `Stringers(key string, vals []fmt.Stringer)` | Slice of `fmt.Stringer` values        |

### Finalising Events

```go
clog.Info().Str("k", "v").Msg("message")  // Log with message
clog.Info().Str("k", "v").Msgf("n=%d", 5) // Log with formatted message
clog.Info().Str("k", "v").Send()          // Log with empty message
```

## Omitting Empty / Zero Fields

Suppress fields with empty or zero values to reduce noise in log output.

**OmitEmpty** omits fields that are semantically "nothing": `nil`, empty strings `""`, and nil or empty slices and maps.

```go
clog.SetOmitEmpty(true)
clog.Info().
  Str("name", "alice").
  Str("nickname", "").   // omitted
  Any("role", nil).      // omitted
  Int("age", 0).         // kept (zero but not empty)
  Bool("admin", false).  // kept (zero but not empty)
  Msg("User")
// INF ℹ️ User name=alice age=0 admin=false
```

**OmitZero** is a superset of OmitEmpty - it additionally omits `0`, `false`, `0.0`, zero durations, and any other typed zero value.

```go
clog.SetOmitZero(true)
clog.Info().
  Str("name", "alice").
  Str("nickname", "").   // omitted
  Any("role", nil).      // omitted
  Int("age", 0).         // omitted
  Bool("admin", false).  // omitted
  Msg("User")
// INF ℹ️ User name=alice
```

Both settings are inherited by sub-loggers created with `With()`. When both are enabled, `OmitZero` takes precedence.

## Quoting

By default, field values containing spaces or special characters are wrapped in Go-style double quotes (`"hello world"`). This behaviour can be customised with `SetQuoteMode`.

### Quote Modes

| Mode          | Description                                                                   |
| ------------- | ----------------------------------------------------------------------------- |
| `QuoteAuto`   | Quote only when needed - spaces, unprintable chars, embedded quotes (default) |
| `QuoteAlways` | Always quote string, error, and default-kind values                           |
| `QuoteNever`  | Never quote                                                                   |

```go
// Default: only quote when needed
clog.Info().Str("reason", "timeout").Str("msg", "hello world").Msg("test")
// INF ℹ️ test reason=timeout msg="hello world"

// Always quote string values
clog.SetQuoteMode(clog.QuoteAlways)
clog.Info().Str("reason", "timeout").Msg("test")
// INF ℹ️ test reason="timeout"

// Never quote
clog.SetQuoteMode(clog.QuoteNever)
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg=hello world
```

### Custom Quote Character

Use a different character for both sides:

```go
clog.SetQuoteChar('\'')
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg='hello world'
```

### Asymmetric Quote Characters

Use different opening and closing characters:

```go
clog.SetQuoteChars('«', '»')
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg=«hello world»

clog.SetQuoteChars('[', ']')
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg=[hello world]
```

Quoting applies to individual field values and to elements within string and `[]any` slices. All quoting settings are inherited by sub-loggers. Pass `0` to reset to the default (`strconv.Quote`).

> **Deprecated:** `SetOmitQuotes(true/false)` still works but delegates to `SetQuoteMode(QuoteNever)` / `SetQuoteMode(QuoteAuto)`. Prefer `SetQuoteMode` for new code.

## Sub-loggers

Create sub-loggers with preset fields using the `With()` context builder:

```go
logger := clog.With().Str("component", "auth").Logger()
logger.Info().Str("user", "john").Msg("Authenticated")
// INF ℹ️ Authenticated component=auth user=john
```

Context fields support the same typed methods as events.

## Dict (Nested Fields)

Group related fields under a common key prefix using dot notation:

```go
clog.Info().Dict("request", clog.Dict().
  Str("method", "GET").
  Int("status", 200),
).Msg("Handled")
// INF ℹ️ Handled request.method=GET request.status=200
```

Works with sub-loggers too:

```go
logger := clog.With().Dict("db", clog.Dict().
  Str("host", "localhost").
  Int("port", 5432),
).Logger()
```

## Custom Prefix

Override the default emoji prefix per-event or per-logger:

```go
// Per-event
clog.Info().Prefix("📦").Str("pkg", "clog").Msg("Installed")

// Per-logger (via sub-logger)
logger := clog.With().Prefix("🔒").Str("component", "auth").Logger()
logger.Info().Msg("Ready")
```

Prefix resolution order: event override > logger preset > default emoji for level.

## Level Alignment

Control how level labels are aligned when they have different widths:

```go
clog.SetLevelAlign(clog.AlignRight)   // default: "  INF", " WARN", "ERROR"
clog.SetLevelAlign(clog.AlignLeft)    //          "INF  ", "WARN ", "ERROR"
clog.SetLevelAlign(clog.AlignCenter)  //          " INF ", "WARN ", "ERROR"
clog.SetLevelAlign(clog.AlignNone)    //          "INF",   "WARN",  "ERROR"
```

## Part Order

Control which parts appear in log output and in what order. The default order is: timestamp, level, prefix, message, fields.

```go
// Reorder: show message before level
clog.SetParts(clog.PartMessage, clog.PartLevel, clog.PartPrefix, clog.PartFields)

// Hide parts by omitting them
clog.SetParts(clog.PartLevel, clog.PartMessage, clog.PartFields) // no prefix or timestamp

// Fields before message
clog.SetParts(clog.PartLevel, clog.PartFields, clog.PartMessage)
```

Available parts: `PartTimestamp`, `PartLevel`, `PartPrefix`, `PartMessage`, `PartFields`.

Use `DefaultParts()` to get the default ordering. Parts omitted from the list are hidden.

## Spinners

Display animated spinners during long-running operations:

```go
err := clog.Spinner("Downloading").
  Str("url", fileURL).
  Wait(ctx, func(ctx context.Context) error {
    return download(ctx, fileURL)
  }).
  Msg("Downloaded")
```

The spinner animates with moon phase emojis (🌔🌓🌒🌑🌘🌗🌖🌕) while the action runs, then logs the result.

### Dynamic Status Updates

Use `Progress` to update the spinner title and fields during execution:

```go
err := clog.Spinner("Processing").
  Progress(ctx, func(ctx context.Context, update *clog.ProgressUpdate) error {
    for i, item := range items {
      update.Title("Processing").Str("progress", fmt.Sprintf("%d/%d", i+1, len(items))).Send()
      if err := process(ctx, item); err != nil {
        return err
      }
    }
    return nil
  }).
  Msg("Processed all items")
```

### WaitResult Finalisers

| Method      | Success behaviour          | Failure behaviour               |
| ----------- | -------------------------- | ------------------------------- |
| `.Msg(s)`   | Logs at `INF` with message | Logs at `ERR` with error string |
| `.Err()`    | Logs at `INF` with title   | Logs at `ERR` with error string |
| `.Send()`   | Logs at configured level   | Logs at configured level        |
| `.Silent()` | Returns error, no logging  | Returns error, no logging       |

All finalisers return the `error` from the action. You can chain any field method (`.Str()`, `.Int()`, `.Bool()`, `.Dur()`, etc.) and `.Prefix()` on a `WaitResult` before finalising.

### Custom Success/Error Behaviour

Use `OnSuccessLevel`, `OnSuccessMessage`, `OnErrorLevel`, and `OnErrorMessage` to customise how the result is logged, then call `.Send()`:

```go
// Fatal on error instead of the default error level
err := clog.Spinner("Connecting to database").
  Str("host", "db.internal").
  Wait(ctx, connectToDB).
  OnErrorLevel(clog.FatalLevel).
  Send()
```

Spinners gracefully degrade: when colours are disabled (CI, piped output), the animation is skipped and a static status line is printed instead.

### Custom Spinner Type

```go
clog.Spinner("Loading").
  Type(spinner.Dot).
  Wait(ctx, action).
  Msg("Done")
```

## Hyperlinks

Render clickable terminal hyperlinks using OSC 8 escape sequences:

```go
// Typed field methods (recommended)
clog.Info().Path("dir", "src/").Msg("Directory")
clog.Info().Line("file", "config.yaml", 42).Msg("File with line")
clog.Info().Column("loc", "main.go", 42, 10).Msg("File with line and column")
clog.Info().Link("docs", "https://example.com", "docs").Msg("URL")

// Standalone functions (for use with Str)
link := clog.PathLink("config.yaml", 42)               // file path with line number
link := clog.PathLink("src/", 0)                       // directory (no line number)
link := clog.Hyperlink("https://example.com", "docs")  // arbitrary URL
```

### IDE Integration

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

| Context        | Fallback chain                                  |
| -------------- | ----------------------------------------------- |
| Directory      | `DirFormat`    → `PathFormat` → `file://{path}` |
| File (no line) | `FileFormat`   → `PathFormat` → `file://{path}` |
| File + line    | `LineFormat`   → `file://{path}`                |
| File + column  | `ColumnFormat` → `LineFormat` → `file://{path}` |

These can also be set via environment variables:

```sh
export CLOG_HYPERLINK_PATH_FORMAT="vscode://{path}"      # generic fallback
export CLOG_HYPERLINK_FILE_FORMAT="vscode://file{path}"  # files only
export CLOG_HYPERLINK_DIR_FORMAT="finder://{path}"       # directories only
export CLOG_HYPERLINK_LINE_FORMAT="vscode://{path}:{line}"
export CLOG_HYPERLINK_COLUMN_FORMAT="vscode://{path}:{line}:{column}"
```

Hyperlinks are automatically disabled when colours are disabled.

## Handlers

Implement the `Handler` interface for custom output formats:

```go
type Handler interface {
  Log(Entry)
}
```

The `Entry` struct provides `Level`, `Time`, `Message`, `Prefix`, and `Fields`. The logger handles level filtering, field accumulation, timestamps, and locking - the handler only formats and writes.

```go
// Using HandlerFunc adapter
clog.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
  data, _ := json.Marshal(e)
  fmt.Println(string(data))
}))
```

## Configuration

### Default Logger

The package-level functions (`Info()`, `Warn()`, etc.) use the `Default` logger which writes to `os.Stdout` at `InfoLevel`.

```go
// Full configuration
clog.Configure(&clog.Config{
  Verbose: true,           // enables debug level + timestamps
  Output:  os.Stderr,      // custom writer
  Styles:  customStyles,   // custom visual styles
})

// Toggle verbose mode
clog.ConfigureVerbose(true)
```

### Custom Logger

```go
logger := clog.New(os.Stderr)
logger.SetLevel(clog.DebugLevel)
logger.SetReportTimestamp(true)
logger.SetTimeFormat("15:04:05.000")
logger.SetFieldTimeFormat(time.Kitchen)    // format for .Time() fields (default: time.RFC3339)
logger.SetFieldStyleLevel(clog.TraceLevel) // min level for field value styling (default: "info")
logger.SetHandler(myHandler)
```

### Utility Functions

```go
clog.GetLevel()                  // returns the current level of the Default logger
clog.IsVerbose()                 // true if level is Debug or Trace
clog.IsTerminal()                // true if stdout is a terminal
clog.ColorsDisabled()            // true if colours are globally disabled
clog.SetExitFunc(fn)             // override os.Exit for Fatal (useful in tests)
clog.SetHyperlinksEnabled(false) // disable all hyperlink rendering
```

### Environment Variables

`CLOG_LEVEL` and `CLOG_SEPARATOR` are checked automatically at init.

```sh
CLOG_LEVEL=debug ./some-app  # enables debug logging + timestamps
CLOG_LEVEL=warn ./some-app   # suppresses info messages
CLOG_SEPARATOR=: ./some-app  # use ":" instead of "=" between keys and values
```

## `NO_COLOR`

clog respects the [`NO_COLOR`](https://no-color.org/) convention. When the `NO_COLOR` environment variable is set (any value, including empty), all colours and hyperlinks are disabled.

### Global Colour Control

```go
clog.ConfigureColorOutput("auto")   // detect terminal capabilities (default)
clog.ConfigureColorOutput("always") // force colours (overrides NO_COLOR)
clog.ConfigureColorOutput("never")  // disable all colours and hyperlinks
```

### Per-Logger Colour Mode

Each logger can override the global colour detection with `SetColorMode`:

```go
logger := clog.New(os.Stdout)
logger.SetColorMode(clog.ColorAlways) // force colours for this logger
logger.SetColorMode(clog.ColorNever)  // disable colours for this logger
logger.SetColorMode(clog.ColorAuto)   // use global detection (default)

// Package-level (sets Default logger)
clog.SetColorMode(clog.ColorAlways)
```

This is useful in tests to verify hyperlink output without mutating global state:

```go
l := clog.New(&buf)
l.SetColorMode(clog.ColorAlways)
l.Info().Line("file", "main.go", 42).Msg("Loaded")
// buf contains OSC 8 hyperlink escape sequences
```

## Styles

Customise the visual appearance using [lipgloss](https://github.com/charmbracelet/lipgloss) styles:

```go
styles := clog.DefaultStyles()

// Customise level colours
styles.Levels[clog.ErrorLevel] = new(
  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")), // bright red
)

// Customise field key appearance
styles.KeyDefault = new(
  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // bright blue
)

clog.SetStyles(styles)
```

### Value Colouring

Values are styled with a three-tier priority system:

1. **Key styles** - style all values of a specific field key
1. **Value styles** - style values matching an exact string
1. **Type styles** - style values by their Go type

```go
styles := clog.DefaultStyles()

// 1. Key styles: all values of the "status" field are green
styles.Keys["status"] = new(lipgloss.NewStyle().
  Foreground(lipgloss.Color("2"))) // green

// 2. Value styles: exact string matches
styles.Values["PASS"] = new(
  lipgloss.NewStyle().
  Foreground(lipgloss.Color("2")), // green
)

styles.Values["FAIL"] = new(lipgloss.NewStyle().
  Foreground(lipgloss.Color("1")), // red
)

// 3. Type styles: string values → white, numeric values → magenta, errors → red by default
styles.FieldString = new(lipgloss.NewStyle().Foreground(lipgloss.Color("15")))
styles.FieldNumber = new(lipgloss.NewStyle().Foreground(lipgloss.Color("5")))
styles.FieldError  = new(lipgloss.NewStyle().Foreground(lipgloss.Color("1")))
styles.FieldString = nil  // set to nil to disable
styles.FieldNumber = nil  // set to nil to disable

clog.SetStyles(styles)
```

### Styles Reference

| Field           | Type                         | Description                                   |
| --------------- | ---------------------------- | --------------------------------------------- |
| `FieldDuration` | `*lipgloss.Style`            | Duration value style (nil to disable)         |
| `FieldError`    | `*lipgloss.Style`            | Error value style (nil to disable)            |
| `FieldNumber`   | `*lipgloss.Style`            | Numeric value style (nil to disable)          |
| `FieldString`   | `*lipgloss.Style`            | String value style (nil to disable)           |
| `FieldTime`     | `*lipgloss.Style`            | Time value style (nil to disable)             |
| `KeyDefault`    | `*lipgloss.Style`            | Field key style (nil to disable)              |
| `Keys`          | `map[string]*lipgloss.Style` | Field key name → value style                  |
| `Levels`        | `map[Level]*lipgloss.Style`  | Per-level label style (nil to disable)        |
| `Messages`      | `map[Level]*lipgloss.Style`  | Per-level message style (nil to disable)      |
| `Separator`     | `*lipgloss.Style`            | Style for the separator between key and value |
| `SeparatorText` | `string`                     | Key/value separator (default `"="`)           |
| `Timestamp`     | `*lipgloss.Style`            | Timestamp style (nil to disable)              |
| `Values`        | `map[string]*lipgloss.Style` | Formatted value string → style                |

Value styles only apply at `Info` level and above (not `Trace` or `Debug`).

### Per-Level Message Styles

Style the log message text differently for each level:

```go
styles := clog.DefaultStyles()

styles.Messages[clog.ErrorLevel] = new(
  lipgloss.NewStyle().Foreground(lipgloss.Color("1")), // red
)

styles.Messages[clog.WarnLevel] = new(
  lipgloss.NewStyle().Foreground(lipgloss.Color("3")), // yellow
)

clog.SetStyles(styles)
```

Use `DefaultMessageStyles()` to get the defaults (unstyled for all levels).
