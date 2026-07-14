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

On animated builders (`fx.Builder`), fields added at runtime via a live update always render after the builder's base fields - use `elapsed.Trailing()` to pin the elapsed field to the end of the row regardless (see [Spinner](spinner.md#elapsed-timer)).

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

The elapsed field respects the same per-logger [`FieldFormats`](configuration.md#field-formats) configuration as animation elapsed fields:

| Field                 | Default          | Description                                                                            |
| --------------------- | ---------------- | -------------------------------------------------------------------------------------- |
| `DurationFormat`      | `nil` (built-in) | Custom formatter for both `Duration` and `Elapsed` fields                              |
| `DurationGradientMax` | `0` (disabled)   | Max duration for `Duration` field gradient (see [Styles](styles.md#duration-gradient)) |
| `DurationMinimum`     | `0`              | Hide `Duration` fields below this threshold (`0` shows all values)                     |
| `DurationPrecision`   | `0`              | Scalar decimal places for `Duration` fields when no scale applies                      |
| `DurationRound`       | `time.Second`    | Scalar rounding for `Duration` fields when no scale applies                            |
| `DurationScale`       | `nil` (inherit)  | Duration-specific override of `TimeScale`                                              |
| `ElapsedFormat`       | `nil` (built-in) | Custom formatter for elapsed durations (takes priority over `DurationFormat`)          |
| `ElapsedGradientMax`  | `0` (disabled)   | Max duration for gradient coloring (see [Styles](styles.md#elapsed-gradient))          |
| `ElapsedMinimum`      | `time.Second`    | Hide elapsed field below this threshold (`0` shows all values)                         |
| `ElapsedPrecision`    | `0`              | Scalar decimal places when no elapsed scale applies                                    |
| `ElapsedRound`        | `time.Second`    | Scalar rounding when no elapsed scale applies                                          |
| `ElapsedScale`        | empty (scalars)  | Elapsed/deadline override; nil inherits `TimeScale`                                    |
| `TimeScale`           | three brackets   | Shared magnitude-keyed rounding and precision scale                                    |

The default duration scale renders `450ms`, `1.5s`, and `12s`, while live elapsed/deadline fields stay at whole-second width. Set `ElapsedScale = nil` to inherit `TimeScale`, provide a non-empty scale to override it, or keep a non-nil empty scale to use `ElapsedRound` and `ElapsedPrecision`. The corresponding setters handle these states automatically.

`ElapsedGradientMax`, `ElapsedMinimum`, `ElapsedRound`, `ElapsedScale`, and the `ElapsedGradient`/`ElapsedGradientMode` style settings can also be overridden per field with options from the `elapsed` package - see [Per-Field Overrides](styles.md#per-field-overrides). The `duration` package mirrors this for `Duration` fields, including `duration.WithMinimum`, `duration.WithRound`, and `duration.WithScale`:

```go
import "github.com/gechr/clog/field/elapsed"

e := clog.Info().Elapsed("elapsed",
  elapsed.WithGradientMax(5*time.Second),
  elapsed.WithMinimum(0), // always show this field, regardless of the logger default
)
runMigrations()
e.Msg("database migration")
```

### Duration Format Function

`DurationFormat` configures a single format function that applies to both [`Duration`](structured-fields.md) fields and `Elapsed` fields. This is useful when you have a shared helper (e.g. from a utility package) that you want applied consistently across all duration logging:

```go
f := clog.DefaultFieldFormats()
f.DurationFormat = commonutil.FormatDuration
clog.SetFieldFormats(f)

// Both of these now use commonutil.FormatDuration:
clog.Info().Duration("took", time.Since(start)).Msg("done")
// INF ℹ️ done took=2.3s

e := clog.Info().Elapsed("elapsed")
doWork()
e.Msg("done")
// INF ℹ️ done elapsed=2.3s
```

When `ElapsedFormat` is also set, it takes priority over `DurationFormat` for `Elapsed` fields only. `Duration` fields always use `DurationFormat`:

```go
f := clog.DefaultFieldFormats()
f.DurationFormat = func(d time.Duration) string { return "dur:" + d.String() }
f.ElapsedFormat = func(d time.Duration) string { return "ela:" + d.String() }
clog.SetFieldFormats(f)

clog.Info().Duration("latency", 3*time.Second).Msg("request")
// INF ℹ️ request latency=dur:3s  ← uses DurationFormat

e := clog.Info().Elapsed("elapsed")
e.Msg("done")
// INF ℹ️ done elapsed=ela:3s    ← ElapsedFormat takes priority
```
