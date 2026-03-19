# Configuration

## Default Logger

The package-level functions (`Info()`, `Warn()`, etc.) use the `Default` logger which writes to `os.Stdout` at `LevelInfo`.

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
clog.SetHyperlinkEnabled(false)  // disable all hyperlink rendering
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
