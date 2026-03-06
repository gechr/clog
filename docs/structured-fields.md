# Structured Fields

Events and contexts support typed field methods. All methods are safe to call on a nil receiver (disabled events are no-ops).

## Event Fields

| Method       | Signature                                                  | Description                                                               |
| ------------ | ---------------------------------------------------------- | ------------------------------------------------------------------------- |
| `Any`        | `Any(key string, val any)`                                 | Arbitrary value                                                           |
| `Anys`       | `Anys(key string, vals []any)`                             | Arbitrary value slice                                                     |
| `Base64`     | `Base64(key string, val []byte)`                           | Byte slice as base64 string                                               |
| `Bool`       | `Bool(key string, val bool)`                               | Boolean field                                                             |
| `Bools`      | `Bools(key string, vals []bool)`                           | Boolean slice field                                                       |
| `Bytes`      | `Bytes(key string, val []byte)`                            | Byte slice - auto-detected as JSON with highlighting, otherwise string    |
| `Column`     | `Column(key, path string, line, column int)`               | Clickable file:line:column hyperlink                                      |
| `Dict`       | `Dict(key string, dict *Event)`                            | Nested fields with dot-notation keys                                      |
| `Duration`   | `Duration(key string, val time.Duration)`                  | Duration field                                                            |
| `Durations`  | `Durations(key string, vals []time.Duration)`              | Duration slice field                                                      |
| `Err`        | `Err(err error)`                                           | Attach error; `Send` uses it as message, `Msg`/`Msgf` add `"error"` field |
| `Errs`       | `Errs(key string, vals []error)`                           | Error slice as string slice (nil errors render as `<nil>`)                |
| `Float64`    | `Float64(key string, val float64)`                         | Float field                                                               |
| `Floats64`   | `Floats64(key string, vals []float64)`                     | Float slice field                                                         |
| `Func`       | `Func(fn func(*Event))`                                    | Lazy field builder; callback skipped on nil (disabled) events             |
| `Hex`        | `Hex(key string, val []byte)`                              | Byte slice as hex string                                                  |
| `Int`        | `Int(key string, val int)`                                 | Integer field                                                             |
| `Int64`      | `Int64(key string, val int64)`                             | 64-bit integer field                                                      |
| `Ints`       | `Ints(key string, vals []int)`                             | Integer slice field                                                       |
| `Ints64`     | `Ints64(key string, vals []int64)`                         | 64-bit integer slice field                                                |
| `JSON`       | `JSON(key string, val any)`                                | Marshals val to JSON with syntax highlighting                             |
| `Line`       | `Line(key, path string, line int)`                         | Clickable file:line hyperlink                                             |
| `Link`       | `Link(key, url, text string)`                              | Clickable URL hyperlink                                                   |
| `Path`       | `Path(key, path string)`                                   | Clickable file/directory hyperlink                                        |
| `Percent`    | `Percent(key string, val float64, opts ...percent.Option)` | Percentage with gradient color; accepts [percent.Option] values           |
| `Quantities` | `Quantities(key string, vals []string)`                    | Quantity slice field                                                      |
| `Quantity`   | `Quantity(key, val string)`                                | Quantity field (e.g. `"10GB"`)                                            |
| `RawJSON`    | `RawJSON(key string, val []byte)`                          | Pre-serialized JSON bytes, emitted verbatim with syntax highlighting      |
| `Str`        | `Str(key, val string)`                                     | String field                                                              |
| `Stringer`   | `Stringer(key string, val fmt.Stringer)`                   | Calls `String()` (nil-safe)                                               |
| `Stringers`  | `Stringers(key string, vals []fmt.Stringer)`               | Slice of `fmt.Stringer` values                                            |
| `Strs`       | `Strs(key string, vals []string)`                          | String slice field                                                        |
| `Time`       | `Time(key string, val time.Time)`                          | Time field                                                                |
| `Times`      | `Times(key string, vals []time.Time)`                      | Time slice field                                                          |
| `Uint`       | `Uint(key string, val uint)`                               | Unsigned integer field                                                    |
| `Uint64`     | `Uint64(key string, val uint64)`                           | 64-bit unsigned integer field                                             |
| `Uints`      | `Uints(key string, vals []uint)`                           | Unsigned integer slice field                                              |
| `Uints64`    | `Uints64(key string, vals []uint64)`                       | 64-bit unsigned integer slice field                                       |
| `URL`        | `URL(key, url string)`                                     | Clickable URL hyperlink (URL as text)                                     |
| `When`       | `When(condition bool, fn func(*Event))`                    | Conditional field builder; fn called only when condition is true          |

## Duration Formatting

By default, `Duration` fields use Go's built-in `time.Duration.String()` (e.g. `3.2s`, `1m30s`). Use `SetDurationFormatFunc` to apply a custom formatter globally:

```go
clog.SetDurationFormatFunc(commonutil.FormatDuration)

clog.Info().Duration("took", time.Since(start)).Msg("done")
// INF ℹ️ done took=2.3s
```

`SetDurationFormatFunc` also applies as a fallback for [`Elapsed`](elapsed.md) fields. See [Elapsed Configuration](elapsed.md#elapsed-configuration) for details.

## Duration Gradient

`Duration` fields support the same green → yellow → red gradient as `Elapsed` fields. Enable it by setting a max duration:

```go
clog.SetDurationGradientMax(20 * time.Second)

clog.Info().Duration("duration", 500*time.Millisecond).Msg("fast")  // green
clog.Info().Duration("duration", 10*time.Second).Msg("medium")       // yellow
clog.Info().Duration("duration", 25*time.Second).Msg("slow")         // red (clamped)
```

When active, the gradient overrides `FieldDurationNumber` / `FieldDurationUnit`. See [Duration Gradient](styles.md#duration-gradient) in the styles reference for gradient mode and custom stop configuration.

## Nested Fields (Dict)

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

## Finalising Events

```go
clog.Info().Str("k", "v").Msg("message")  // Log with message
clog.Info().Str("k", "v").Msgf("n=%d", 5) // Log with formatted message
clog.Info().Str("k", "v").Send()          // Log with empty message
clog.Error().Err(err).Send()              // Log with error as message (no error= field)
clog.Error().Err(err).Msg("failed")       // Log with message + error= field
```
