# `NO_COLOR`

clog respects the [`NO_COLOR`](https://no-color.org/) convention. When the `NO_COLOR` environment variable is set (any value, including empty), all colors and hyperlinks are disabled.

## Color Control

Color behaviour is set per-`Output` via `ColorMode`:

```go
// Package-level (recreates Default logger's Output)
clog.SetColorMode(clog.ColorAlways) // force colors (overrides NO_COLOR)
clog.SetColorMode(clog.ColorNever)  // disable all colors and hyperlinks
clog.SetColorMode(clog.ColorAuto)   // detect terminal capabilities (default)

// Per-logger via Output
logger := clog.New(clog.NewOutput(os.Stdout, clog.ColorAlways))
```

This is useful in tests to verify hyperlink output without mutating global state:

```go
l := clog.New(clog.NewOutput(&buf, clog.ColorAlways))
l.Info().Line("file", "main.go", 42).Msg("Loaded")
// buf contains OSC 8 hyperlink escape sequences
```

`ColorMode` implements `encoding.TextMarshaler` and `encoding.TextUnmarshaler`, so it works directly with `flag.TextVar` and most flag libraries.

## Piped stdout

When stdout is piped (non-TTY) but stderr is a terminal, clog still renders colors correctly. In lipgloss v2, styles are plain value types that always emit full-fidelity ANSI -- color downsampling happens at the output layer. This means styles work correctly on stderr even when stdout is redirected:

```go
// Colors work on stderr even when stdout is piped.
logger := clog.New(clog.Stderr(clog.ColorAuto))
logger.Info().Str("status", "ok").Msg("Ready")
```
