# Styles

Customise the visual appearance using [lipgloss](https://charm.land/lipgloss/v2) styles:

![Styled output](assets/styles.png)

```go
clog.SetStyles(&style.Config{
  // Customise level colors
  Levels: style.LevelMap{
    clog.LevelError: new(
      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9")), // bright red
    ),
  },
  // Customise field key appearance
  KeyDefault: new(
    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")), // bright blue
  ),
})
```

`SetStyles` merges non-zero fields into the existing configuration - only the fields you set are changed, all others keep their current values.

## Value Coloring

Values are styled with a three-tier priority system:

1. **Key styles** - style all values of a specific field key
1. **Value styles** - style values matching a typed key (bool `true` != string `"true"`)
1. **Type styles** - style values by their Go type

```go
clog.SetStyles(&style.Config{
  // 1. Key styles: all values of the "status" field are green
  Keys: style.Map{
    "status": new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))),
  },
  // 2. Value styles: typed key matches (bool `true` != string "true")
  Values: style.ValueMap{
    "PASS": new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))),
    "FAIL": new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))),
  },
  // 3. Type styles
  FieldString: new(lipgloss.NewStyle().Foreground(lipgloss.Color("15"))),
  FieldNumber: new(lipgloss.NewStyle().Foreground(lipgloss.Color("5"))),
  FieldError:  new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))),
})
```

## Styles Reference

| Field                 | Type                     | Alias           | Default                   |
| --------------------- | ------------------------ | --------------- | ------------------------- |
| `DurationGradient`    | `[]style.ColorStop`      |                 | green → yellow → red      |
| `DurationGradientMode`| `style.GradientMode`     |                 | `style.GradientFade`      |
| `DurationThresholds`  | `map[string][]Threshold` | `ThresholdMap`  | `{}`                      |
| `DurationUnits`       | `map[string]Style`       | `StyleMap`      | `{}`                      |
| `ElapsedGradient`     | `[]style.ColorStop`      |                 | green → yellow → red      |
| `ElapsedGradientMode` | `style.GradientMode`     |                 | `style.GradientFade`      |
| `FieldDurationNumber` | `Style`                  |                 | magenta                   |
| `FieldDurationUnit`   | `Style`                  |                 | magenta faint             |
| `FieldElapsedNumber`  | `Style`                  |                 | `nil` (→ DurationNumber)  |
| `FieldElapsedUnit`    | `Style`                  |                 | `nil` (→ DurationUnit)    |
| `FieldError`          | `Style`                  |                 | red                       |
| `FieldJSON`           | `*style.JSON`            |                 | `style.DefaultJSON()`     |
| `FieldNumber`         | `Style`                  |                 | magenta                   |
| `FieldPercent`        | `Style`                  |                 | `nil`                     |
| `FieldQuantityNumber` | `Style`                  |                 | magenta                   |
| `FieldQuantityUnit`   | `Style`                  |                 | magenta faint             |
| `FieldString`         | `Style`                  |                 | white                     |
| `FieldTime`           | `Style`                  |                 | magenta                   |
| `KeyDefault`          | `Style`                  |                 | blue                      |
| `Keys`                | `map[string]Style`       | `StyleMap`      | `{}`                      |
| `Levels`              | `map[Level]Style`        | `LevelStyleMap` | per-level bold colors     |
| `Messages`            | `map[Level]Style`        | `LevelStyleMap` | `style.DefaultMessages()` |
| `PercentGradient`     | `[]style.ColorStop`      |                 | red → yellow → green      |
| `QuantityThresholds`  | `map[string][]Threshold` | `ThresholdMap`  | `{}`                      |
| `QuantityUnits`       | `map[string]Style`       | `StyleMap`      | `{}`                      |
| `Separator`           | `Style`                  |                 | faint                     |
| `Symbols`             | `map[Level]Style`        | `LevelStyleMap` | `{}`                      |
| `Timestamp`           | `Style`                  |                 | faint                     |
| `Values`              | `map[any]Style`          | `ValueStyleMap` | `style.DefaultValues()`   |

| Field                 | Description                                                                                |
| --------------------- | ------------------------------------------------------------------------------------------ |
| `DurationGradient`    | Gradient color stops for `Duration` fields; active when `SetDurationGradientMax` > 0       |
| `DurationGradientMode`| Gradient transition mode: `GradientFade` (smooth) or `GradientStep` (discrete)             |
| `DurationThresholds`  | Duration unit -> magnitude-based style thresholds                                          |
| `DurationUnits`       | Duration unit string -> style override                                                     |
| `ElapsedGradient`     | Gradient color stops for `Elapsed` fields; active when `SetElapsedGradientMax` > 0         |
| `ElapsedGradientMode` | Gradient transition mode: `GradientFade` (smooth) or `GradientStep` (discrete)             |
| `FieldDurationNumber` | Style for numeric segments of duration values (e.g. "1" in "1m30s"), nil to disable        |
| `FieldDurationUnit`   | Style for unit segments of duration values (e.g. "m" in "1m30s"), nil to disable           |
| `FieldElapsedNumber`  | Style for numeric segments of elapsed-time values; nil falls back to `FieldDurationNumber` |
| `FieldElapsedUnit`    | Style for unit segments of elapsed-time values; nil falls back to `FieldDurationUnit`      |
| `FieldError`          | Style for error field values, nil to disable                                               |
| `FieldJSON`           | Per-token styles for JSON syntax highlighting; nil disables highlighting                   |
| `FieldNumber`         | Style for int/float field values, nil to disable                                           |
| `FieldPercent`        | Base style for `Percent` fields (foreground overridden by gradient), nil to disable        |
| `FieldQuantityNumber` | Style for numeric part of quantity values (e.g. "5" in "5km"), nil to disable              |
| `FieldQuantityUnit`   | Style for unit part of quantity values (e.g. "km" in "5km"), nil to disable                |
| `FieldString`         | Style for string field values, nil to disable                                              |
| `FieldTime`           | Style for `time.Time` field values, nil to disable                                         |
| `KeyDefault`          | Style for field key names without a per-key override, nil to disable                       |
| `Keys`                | Field key name -> value style override                                                     |
| `Levels`              | Per-level label style (e.g. "INF", "ERR"), nil to disable                                  |
| `Messages`            | Per-level message text style, nil to disable                                               |
| `PercentGradient`     | Gradient color stops for `Percent` fields                                                  |
| `QuantityThresholds`  | Quantity unit -> magnitude-based style thresholds                                          |
| `QuantityUnits`       | Quantity unit string -> style override                                                     |
| `Separator`           | Style for the separator between key and value                                              |
| `Symbols`             | Per-level symbol style                                                                     |
| `Timestamp`           | Style for the timestamp, nil to disable                                                    |
| `Values`              | Typed value -> style (uses Go equality, so bool `true` != string `"true"`)                 |

## Configuration

Behavioural settings are configured via setter methods on `Logger` (or package-level convenience functions for the `Default` logger). Settings marked **pkg** are package-level functions only (not Logger methods):

| Setter                       | Type                         | Default       | Scope  | Description                                                          |
| ---------------------------- | ---------------------------- | ------------- | ------ | -------------------------------------------------------------------- |
| `SetAnimationInterval`       | `time.Duration`              | `67ms`        | Logger | Minimum refresh interval for all animations (0 = use built-in rates) |
| `SetDurationGradientMax`     | `time.Duration`              | `0`           | pkg    | Max duration for `Duration` field gradient (0 = disabled)            |
| `SetElapsedFormatFunc`       | `func(time.Duration) string` | `nil`         | pkg    | Custom format function for `Elapsed` fields                          |
| `SetElapsedGradientMax`      | `time.Duration`              | `0`           | pkg    | Max duration for elapsed gradient (0 = disabled)                     |
| `SetElapsedMinimum`          | `time.Duration`              | `time.Second` | pkg    | Minimum duration for `Elapsed` fields to be displayed                |
| `SetElapsedPrecision`        | `int`                        | `0`           | pkg    | Decimal places for `Elapsed` display (0 = "3s", 1 = "3.2s")          |
| `SetElapsedRound`            | `time.Duration`              | `time.Second` | pkg    | Rounding granularity for `Elapsed` values (0 to disable)             |
| `SetFieldSort`               | `Sort`                       | `SortNone`    | Logger | Sort order: `SortNone`, `SortAscending`, `SortDescending`            |
| `SetHyperlinkColumnFormat`   | `string`                     | `nil`         | pkg    | URL format for file+line+column hyperlinks                           |
| `SetHyperlinkDirFormat`      | `string`                     | `nil`         | pkg    | URL format for directory hyperlinks                                  |
| `SetHyperlinkEnabled`        | `bool`                       | `true`        | pkg    | Enable/disable all hyperlink rendering                               |
| `SetHyperlinkFileFormat`     | `string`                     | `nil`         | pkg    | URL format for file-only hyperlinks                                  |
| `SetHyperlinkLineFormat`     | `string`                     | `nil`         | pkg    | URL format for file+line hyperlinks                                  |
| `SetHyperlinkPathFormat`     | `string`                     | `nil`         | pkg    | Generic fallback URL format for any path                             |
| `SetHyperlinkPreset`         | `string`                     | `""`          | pkg    | Configure all format slots via named preset (returns `error`)        |
| `SetPercentFormatFunc`       | `func(float64) string`       | `nil`         | pkg    | Custom format function for `Percent` fields                          |
| `SetPercentReverseGradient`  | `bool`                       | `false`       | pkg    | Reverse the percent gradient (green=0%, red=100%)                    |
| `SetPercentPrecision`        | `int`                        | `0`           | pkg    | Decimal places for `Percent` display (0 = "75%", 1 = "75.0%")        |
| `SetPercentMaximum`          | `float64`                    | `1.0`         | pkg    | Percent input maximum (1 = fractions 0–1, 100 = 0–100)               |
| `SetQuantityUnitsIgnoreCase` | `bool`                       | `true`        | pkg    | Case-insensitive quantity unit matching                              |
| `SetSeparatorText`           | `string`                     | `"="`         | Logger | Key/value separator string                                           |

Each `Threshold` pairs a minimum value with style overrides:

```go
type ThresholdStyle struct {
  Number Style // Override for the number segment (nil = keep default).
  Unit   Style // Override for the unit segment (nil = keep default).
}

type Threshold struct {
  Value float64        // Minimum numeric value (inclusive) to trigger this style.
  Style ThresholdStyle // Style overrides for number and unit segments.
}
```

Thresholds are evaluated in descending order - the first match wins:

```go
styles.QuantityThresholds["ms"] = style.Thresholds{
  {Value: 5000, Style: style.ThresholdStyle{Number: redStyle, Unit: redStyle}},
  {Value: 1000, Style: style.ThresholdStyle{Number: yellowStyle, Unit: yellowStyle}},
}
```

Value styles only apply at `Info` level and above by default. Use `SetFieldStyleLevel` to change the threshold.

## Per-Level Message Styles

Style the log message text differently for each level:

```go
clog.SetStyles(&style.Config{
  Messages: style.LevelMap{
    clog.LevelError: new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))),
    clog.LevelWarn:  new(lipgloss.NewStyle().Foreground(lipgloss.Color("3"))),
  },
})
```

Use `style.DefaultMessages()` to get the defaults (unstyled for all levels).

Use `style.DefaultValues()` to get the default value styles (`true`=green, `false`=red, `nil`=grey, `""`=grey).

Use `style.DefaultPercentGradient()` to get the default red → yellow → green gradient stops used for `Percent` fields.

## Percent Gradient Direction

By default the `Percent` gradient runs red (0%) → yellow (50%) → green (100%) - useful when a higher value is better (e.g. battery, health score). For metrics where a lower value is better (CPU usage, disk usage, error rate), reverse the gradient:

```go
// Global: all Percent fields
clog.SetPercentReverseGradient(true)

// Per-field: just this Percent field, regardless of the global setting
clog.Info().
  Percent("cpu", 0.92, percent.WithReverseGradient()).
  Percent("battery", 0.85).
  Msg("System status")
// "cpu" renders red at 92%, "battery" renders green at 85%
```

`percent.WithReverseGradient()` is a `percent.Option` passed directly to `Event.Percent`. It **toggles** the package-level setting for that field - so if the gradient is already reversed, `percent.WithReverseGradient()` flips it back to normal. This makes it easy to mix metrics with different semantics on the same log line regardless of the global default.

## Duration Gradient

Color `Duration` fields on a gradient that transitions from green (fast) through yellow to red (slow). Set a max duration to enable the gradient - duration values are mapped onto 0→max, clamping at max:

```go
clog.SetDurationGradientMax(20 * time.Second)
```

When active, the gradient overrides `FieldDurationNumber` / `FieldDurationUnit` and colors the entire formatted string. When the max is 0 (the default) or `DurationGradient` stops are nil, the existing number/unit split styling is used.

Use `style.DefaultElapsedGradient()` to get the default green → yellow → red gradient stops (shared with `Elapsed` by default).

The `DurationGradientMode` field controls transition style - see [Gradient Mode](#gradient-mode) below.

## Elapsed Gradient

Color elapsed-time fields on a gradient that transitions from green (fast) through yellow to red (slow). Set a max duration to enable the gradient - elapsed values are mapped onto 0→max, clamping at max:

```go
clog.SetElapsedGradientMax(30 * time.Second)
```

When active, the gradient overrides `FieldElapsedNumber` / `FieldElapsedUnit` and colors the entire formatted string. When the max is 0 (the default) or `ElapsedGradient` stops are nil, the existing number/unit split styling is used.

Use `style.DefaultElapsedGradient()` to get the default green → yellow → red gradient stops used for `Elapsed` fields (same stops as `DurationGradient` by default).

### Gradient Mode

Both `DurationGradientMode` and `ElapsedGradientMode` control how colors transition between stops:

| Mode                 | Description                                  |
| -------------------- | -------------------------------------------- |
| `style.GradientFade` | Smooth interpolation between stops (default) |
| `style.GradientStep` | Discrete color jumps at stop boundaries      |

**Fade** blends smoothly between adjacent color stops using perceptually uniform CIE-LCh interpolation. **Step** uses the color of the last stop whose position is ≤ the current value - no blending.

```go
clog.SetStyles(&style.Config{ElapsedGradientMode: style.GradientStep})
```

### Custom Stops

```go
clog.SetStyles(&style.Config{
  ElapsedGradient: []style.ColorStop{
    {Position: 0, Color: colorful.Color{R: 0, G: 1, B: 0}},   // green
    {Position: 1, Color: colorful.Color{R: 1, G: 0, B: 0}},   // red
  },
})
```

## Format Hooks

Override the default formatting for `Elapsed` and `Percent` fields:

```go
// Custom elapsed format: truncate to whole seconds
clog.SetElapsedFormatFunc(func(d time.Duration) string {
  return d.Truncate(time.Second).String()
})

// Custom percent format: "75/100" instead of "75%"
clog.SetPercentFormatFunc(func(v float64) string {
  return fmt.Sprintf("%.0f/100", v)
})
```

When set to `nil` (the default), the built-in formatters are used (`formatElapsed` with `SetElapsedPrecision` for elapsed, `strconv.FormatFloat` with `SetPercentPrecision` + "%" for percent). Custom percent format functions receive the **display** percentage (0–100), not the raw stored value.

## Field Sort Order

Control the order fields appear in log output. By default fields preserve insertion order.

```go
// Sort fields alphabetically by key
clog.SetFieldSort(clog.SortAscending)

// Or reverse alphabetical
clog.SetFieldSort(clog.SortDescending)
```

| Constant         | Description                        |
| ---------------- | ---------------------------------- |
| `SortNone`       | Preserve insertion order (default) |
| `SortAscending`  | Sort fields by key A→Z             |
| `SortDescending` | Sort fields by key Z→A             |

```go
clog.Info().
  Str("zoo", "animals").
  Str("alpha", "first").
  Int("count", 42).
  Msg("Sorted")
// SortNone:       INF ℹ️ Sorted zoo=animals alpha=first count=42
// SortAscending:  INF ℹ️ Sorted alpha=first count=42 zoo=animals
// SortDescending: INF ℹ️ Sorted zoo=animals count=42 alpha=first
```
