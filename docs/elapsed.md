# Elapsed

Measure and log the elapsed duration of an operation without animations:

```go
e := clog.Info().Elapsed("elapsed")
runMigrations()
e.Msg("database migration")
// INF ℹ️ database migration elapsed=2s
```

Use `Elapsed` when you want elapsed-time logging but don't need a spinner, progress bar, or any visual animation. The elapsed field uses the same formatting, styling, and field type as animation [`Elapsed`](spinner.md#elapsed-timer) fields.

## Finalising

Elapsed events can be finalised with `Send` (no message), `Msg`, or `Msgf`:

```go
e := clog.Info().Elapsed("elapsed")
runMigrations()
e.Send()
// INF ℹ️ elapsed=2s

e = clog.Info().Elapsed("elapsed")
runMigrations()
e.Msg("migrations complete")
// INF ℹ️ migrations complete elapsed=2s
```

## Field Positioning

`Elapsed` can appear anywhere in the field chain to control where the elapsed field is shown:

```go
// Elapsed before other fields
e := clog.Info().Elapsed("elapsed").Str("env", "prod")
deploy()
e.Msg("deploy")
// INF ℹ️ deploy elapsed=12s env=prod

// Elapsed after other fields
e = clog.Info().Str("env", "prod").Elapsed("elapsed")
deploy()
e.Msg("deploy")
// INF ℹ️ deploy env=prod elapsed=12s
```

## Custom Key

The key parameter controls the field name:

```go
e := clog.Info().Elapsed("duration")
compile()
e.Msg("compile")
// INF ℹ️ compile duration=3s
```

## Log Levels

Since `Elapsed` is a method on events, you control the level directly:

```go
clog.Warn().Elapsed("elapsed").Msg("slow query")
// WRN ⚠️ slow query elapsed=5s

clog.Error().Elapsed("elapsed").Err(err).Msg("compile")
// ERR ❌ compile elapsed=3s error="syntax error"
```

## Sub-loggers

`Elapsed` works on any logger instance:

```go
logger := clog.With().Str("component", "db").Logger()
e := logger.Info().Elapsed("elapsed")
runQuery()
e.Msg("query")
// INF ℹ️ query component=db elapsed=1s
```

## Elapsed Configuration

The elapsed field respects the same configuration as animation elapsed fields:

| Function                | Default          | Description                                                                                |
| ----------------------- | ---------------- | ------------------------------------------------------------------------------------------ |
| `SetDurationFormatFunc` | `nil` (built-in) | Custom format function for both `Duration` and `Elapsed` fields                            |
| `SetDurationGradientMax`| `0` (disabled)   | Max duration for `Duration` field gradient (see [Styles](styles.md#duration-gradient))     |
| `SetElapsedFormatFunc`  | `nil` (built-in) | Custom format function for elapsed durations (takes priority over `SetDurationFormatFunc`) |
| `SetElapsedGradientMax` | `0` (disabled)   | Max duration for gradient coloring (see [Styles](styles.md#elapsed-gradient))              |
| `SetElapsedMinimum`     | `time.Second`    | Hide elapsed field below this threshold                                                    |
| `SetElapsedPrecision`   | `0`              | Decimal places (`0` = `3s`, `1` = `3.2s`)                                                  |
| `SetElapsedRound`       | `time.Second`    | Rounding granularity (`0` disables rounding)                                               |

### Duration Format Function

`SetDurationFormatFunc` configures a single format function that applies to both [`Duration`](structured-fields.md) fields and `Elapsed` fields. This is useful when you have a shared helper (e.g. from a utility package) that you want applied consistently across all duration logging:

```go
clog.SetDurationFormatFunc(commonutil.FormatDuration)

// Both of these now use commonutil.FormatDuration:
clog.Info().Duration("took", time.Since(start)).Msg("done")
// INF ℹ️ done took=2.3s

e := clog.Info().Elapsed("elapsed")
doWork()
e.Msg("done")
// INF ℹ️ done elapsed=2.3s
```

When `SetElapsedFormatFunc` is also set, it takes priority over `SetDurationFormatFunc` for `Elapsed` fields only. `Duration` fields always use `SetDurationFormatFunc`:

```go
clog.SetDurationFormatFunc(func(d time.Duration) string { return "dur:" + d.String() })
clog.SetElapsedFormatFunc(func(d time.Duration) string { return "ela:" + d.String() })

clog.Info().Duration("latency", 3*time.Second).Msg("request")
// INF ℹ️ request latency=dur:3s  ← uses SetDurationFormatFunc

e := clog.Info().Elapsed("elapsed")
e.Msg("done")
// INF ℹ️ done elapsed=ela:3s    ← SetElapsedFormatFunc takes priority
```
