# Structured Fields

Events and contexts support typed field methods. All methods are safe to call on a nil receiver (disabled events are no-ops).

## Event Fields

| Method       | Signature                                                  | Description                                                                         |
| ------------ | ---------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| `AnErr`      | `AnErr(key string, err error)`                             | Error as keyed field (no-op if nil); unlike `Err`, no special `Send`/`Msg` behavior |
| `Any`        | `Any(key string, val any)`                                 | Arbitrary value                                                                     |
| `Anys`       | `Anys(key string, vals []any)`                             | Arbitrary value slice                                                               |
| `Base64`     | `Base64(key string, val []byte)`                           | Byte slice as base64 string                                                         |
| `Bool`       | `Bool(key string, val bool)`                               | Boolean field                                                                       |
| `Bools`      | `Bools(key string, vals []bool)`                           | Boolean slice field                                                                 |
| `Bytes`      | `Bytes(key string, val []byte)`                            | Byte slice - auto-detected as JSON with highlighting, otherwise string              |
| `Column`     | `Column(key, path string, line, column int)`               | Clickable file:line:column hyperlink                                                |
| `Discard`    | `Discard()`                                                | Disables the event; returns nil to short-circuit subsequent methods                 |
| `Dict`       | `Dict(key string, dict *Event)`                            | Nested fields with dot-notation keys                                                |
| `Duration`   | `Duration(key string, val time.Duration)`                  | Duration field                                                                      |
| `Durations`  | `Durations(key string, vals []time.Duration)`              | Duration slice field                                                                |
| `Enabled`    | `Enabled() bool`                                           | Returns true if the event is enabled (non-nil)                                      |
| `Disabled`   | `Disabled() bool`                                          | Returns true if the event is disabled (nil)                                         |
| `Err`        | `Err(err error)`                                           | Attach error; `Send` uses it as message, `Msg`/`Msgf` add `"error"` field           |
| `Errs`       | `Errs(key string, vals []error)`                           | Error slice as string slice (nil errors render as `<nil>`)                          |
| `Float32`    | `Float32(key string, val float32)`                         | 32-bit float field                                                                  |
| `Float64`    | `Float64(key string, val float64)`                         | 64-bit float field                                                                  |
| `Floats32`   | `Floats32(key string, vals []float32)`                     | 32-bit float slice field                                                            |
| `Floats64`   | `Floats64(key string, vals []float64)`                     | 64-bit float slice field                                                            |
| `Fraction`   | `Fraction(key string, current, total int, ...Option)`      | Current/total field with gradient color (e.g. `3/10`)                               |
| `Func`       | `Func(fn func(*Event))`                                    | Lazy field builder; callback skipped on nil (disabled) events                       |
| `Hex`        | `Hex(key string, val []byte)`                              | Byte slice as hex string                                                            |
| `Int`        | `Int(key string, val int)`                                 | Integer field                                                                       |
| `Int8`       | `Int8(key string, val int8)`                               | 8-bit integer field                                                                 |
| `Int16`      | `Int16(key string, val int16)`                             | 16-bit integer field                                                                |
| `Int32`      | `Int32(key string, val int32)`                             | 32-bit integer field                                                                |
| `Int64`      | `Int64(key string, val int64)`                             | 64-bit integer field                                                                |
| `Ints`       | `Ints(key string, vals []int)`                             | Integer slice field                                                                 |
| `Ints8`      | `Ints8(key string, vals []int8)`                           | 8-bit integer slice field                                                           |
| `Ints16`     | `Ints16(key string, vals []int16)`                         | 16-bit integer slice field                                                          |
| `Ints32`     | `Ints32(key string, vals []int32)`                         | 32-bit integer slice field                                                          |
| `Ints64`     | `Ints64(key string, vals []int64)`                         | 64-bit integer slice field                                                          |
| `JSON`       | `JSON(key string, val any)`                                | Marshals val to JSON with syntax highlighting                                       |
| `Line`       | `Line(key, path string, line int)`                         | Clickable file:line hyperlink                                                       |
| `Link`       | `Link(key, url, text string)`                              | Clickable URL hyperlink                                                             |
| `Links`      | `Links(key string, links []Link)`                          | Clickable URL hyperlink slice                                                       |
| `MsgFunc`    | `MsgFunc(createMsg func() string)`                         | Finalise with lazily-computed message; fn skipped on nil events                     |
| `Path`       | `Path(key, path string)`                                   | Clickable file/directory hyperlink                                                  |
| `PathText`   | `PathText(key, text, path string)`                         | Clickable hyperlink with a custom label (text) that links to path                   |
| `Percent`    | `Percent(key string, val float64, opts ...percent.Option)` | Percentage with gradient color; accepts [percent.Option] values                     |
| `Quantities` | `Quantities(key string, vals []string)`                    | Quantity slice field                                                                |
| `Quantity`   | `Quantity(key, val string)`                                | Quantity field (e.g. `"10GB"`)                                                      |
| `RawJSON`    | `RawJSON(key string, val []byte)`                          | Pre-serialized JSON bytes, emitted verbatim with syntax highlighting                |
| `Str`        | `Str(key, val string)`                                     | String field                                                                        |
| `Stringer`   | `Stringer(key string, val fmt.Stringer)`                   | Calls `String()` (nil-safe)                                                         |
| `Stringers`  | `Stringers(key string, vals []fmt.Stringer)`               | Slice of `fmt.Stringer` values                                                      |
| `Strs`       | `Strs(key string, vals []string)`                          | String slice field                                                                  |
| `Time`       | `Time(key string, val time.Time)`                          | Time field                                                                          |
| `TimeDiff`   | `TimeDiff(key string, t, start time.Time)`                 | Duration between two times (zero if t is not after start)                           |
| `Times`      | `Times(key string, vals []time.Time)`                      | Time slice field                                                                    |
| `Uint`       | `Uint(key string, val uint)`                               | Unsigned integer field                                                              |
| `Uint8`      | `Uint8(key string, val uint8)`                             | 8-bit unsigned integer field                                                        |
| `Uint16`     | `Uint16(key string, val uint16)`                           | 16-bit unsigned integer field                                                       |
| `Uint32`     | `Uint32(key string, val uint32)`                           | 32-bit unsigned integer field                                                       |
| `Uint64`     | `Uint64(key string, val uint64)`                           | 64-bit unsigned integer field                                                       |
| `Uints`      | `Uints(key string, vals []uint)`                           | Unsigned integer slice field                                                        |
| `Uints8`     | `Uints8(key string, vals []uint8)`                         | 8-bit unsigned integer slice field                                                  |
| `Uints16`    | `Uints16(key string, vals []uint16)`                       | 16-bit unsigned integer slice field                                                 |
| `Uints32`    | `Uints32(key string, vals []uint32)`                       | 32-bit unsigned integer slice field                                                 |
| `Uints64`    | `Uints64(key string, vals []uint64)`                       | 64-bit unsigned integer slice field                                                 |
| `URL`        | `URL(key, url string)`                                     | Clickable URL hyperlink (URL as text)                                               |
| `URLs`       | `URLs(key string, urls []string)`                          | Clickable URL hyperlink slice (URLs as text)                                        |
| `When`       | `When(condition bool, fn func(*Event))`                    | Conditional field builder; fn called only when condition is true                    |

## Field Shapes

A field normally renders as `key=value`. `SetFieldShapes` changes which tokens a named key renders - dropping the key, or wrapping the value - without touching how it is coloured:

```go
clog.SetFieldShapes(clog.FieldShapeMap{
  "region": {OmitKey: true, Prefix: "(", Suffix: ")"},
  "branch": {OmitKey: true, Prefix: "@"},
})

clog.Info().Str("region", "emea").Str("queue", "ingest").Msg("draining")
// INF ℹ️ draining (emea) queue=ingest
```

`OmitKey` drops the key and its separator; `Prefix` and `Suffix` wrap the value. Keys absent from the map render normally. Like `SetParts`, shapes replace any previously configured ones - pass `nil` to clear them.

Shapes compose with the styling tiers rather than competing with them: shape decides the tokens, [`KeyValues`](styles.md#per-key-value-styles) (or `Keys`, or `Values`) decides the colour, so a key can render as a badge and still colour each value differently.

```go
clog.SetStyles(&style.Config{
  KeyValues: style.KeyValueMap{
    "region": {Values: style.ValueMap{
      "emea": new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))),
      "apac": new(lipgloss.NewStyle().Foreground(lipgloss.Color("4"))),
    }},
  },
})
// "(emea)" renders green, brackets included; "(apac)" blue.
```

The affixes take the value's resolved style so the whole token reads as one unit. Values styled per segment or by gradient (durations, percentages, quantities, slices, highlighted JSON) have no single colour, so their affixes render plain unless a `KeyValues`, `Keys` or `Values` entry governs the key.

A few details worth knowing:

- Affixes wrap **outside** quoting and slice brackets: a quoted value renders `("emea north")`, a slice renders `⟨[a, b]⟩`.
- Shapes are structural, so they apply with colour off too - piped output shows the same tokens as a terminal.
- Sorting still uses the real key, even when it is hidden.
- Shapes apply to the built-in formatter, including animation task rows. A custom [`Handler`](handlers.md) receives the raw fields.

## Slice Formatting

Slice fields render as `[a, b, c]` by default. The brackets and separator are configurable:

```go
clog.SetSliceSeparator(" ")        // [a b c]
clog.SetSliceBrackets('(', ')')    // (a, b, c)
clog.SetSliceBrackets('«', '»')    // «a, b, c»
clog.SetSliceBrackets('|', '|')    // |a, b, c| - same char for open and close
```

## Duration Formatting

By default, `Duration` fields use the shared `TimeScale`: values below one second render in milliseconds (`450ms`), values below ten seconds render with up to one decimal place (`1.5s`, but `1s` rather than `1.0s`), and larger values round to whole seconds (`12s`). `DurationMinimum` defaults to `0`, so sub-second values remain visible. Durations >= 1m use composite format (e.g. `1m30s`, `2h15m`). See [`FieldFormats`](configuration.md#field-formats) for custom scales, scalar round/precision mode, and inheritance. Set `DurationFormat` to apply a custom formatter per logger instead:

```go
f := clog.DefaultFieldFormats()
f.DurationFormat = commonutil.FormatDuration
clog.SetFieldFormats(f)

clog.Info().Duration("took", time.Since(start)).Msg("done")
// INF ℹ️ done took=2.3s
```

`DurationFormat` also applies as a fallback for [`Elapsed`](elapsed.md) fields. Duration slices continue to use `time.Duration.String()` per element and do not apply `TimeScale`. See [Elapsed Configuration](elapsed.md#elapsed-configuration) for details.

## Duration Gradient

`Duration` fields support the same green → yellow → red gradient as `Elapsed` fields. Enable it by setting a max duration:

```go
f := clog.DefaultFieldFormats()
f.DurationGradientMax = 20 * time.Second
clog.SetFieldFormats(f)

clog.Info().Duration("duration", 2*time.Second).Msg("fast")    // green
clog.Info().Duration("duration", 10*time.Second).Msg("medium") // yellow
clog.Info().Duration("duration", 25*time.Second).Msg("slow")   // red (clamped)
```

When active, the gradient overrides the `FieldDuration` segment styles. See [Duration Gradient](styles.md#duration-gradient) in the styles reference for gradient mode and custom stop configuration.

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
clog.Info().Str("k", "v").Send()           // Log with empty message
clog.Error().Err(err).Send()               // Log with error as message (no error= field)
clog.Error().Err(err).Msg("failed")        // Log with message + error= field

// Lazy message - fn only called when event is enabled:
clog.Debug().MsgFunc(func() string { return expensiveString() })
```
