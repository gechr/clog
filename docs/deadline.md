# Deadline

Log the time remaining until a deadline - the countdown mirror of [`Elapsed`](elapsed.md):

```go
e := clog.Info().Deadline("timeout", 15*time.Second)
waitForConfirmation()
e.Msg("confirmed")
// INF ℹ️ confirmed timeout=9s
```

The field displays the remaining time (counting down from `from` to `0s`), while its gradient colors by the time *consumed* - so a fresh deadline renders at the gradient's first stop (green) and an expiring one at its last (red), with zero configuration. The deadline field uses the same field type as animation [`Deadline`](spinner.md#deadline-countdown) fields.

## Finalising

Deadline events can be finalised with `Send` (no message), `Msg`, or `Msgf`:

```go
e := clog.Info().Deadline("timeout", 30*time.Second)
downloadArtifact()
e.Send()
// INF ℹ️ timeout=21s

e = clog.Info().Deadline("timeout", 30*time.Second)
downloadArtifact()
e.Msg("download complete")
// INF ℹ️ download complete timeout=21s
```

## Field Positioning

`Deadline` can appear anywhere in the field chain to control where the field is shown:

```go
e := clog.Info().Deadline("timeout", 10*time.Second).Str("job", "upload")
upload()
e.Msg("uploaded")
// INF ℹ️ uploaded timeout=4s job=upload
```

On animated builders (`fx.Builder`), fields added at runtime via a live update always render after the builder's base fields - use `deadline.WithTrailing()` to pin the deadline field to the end of the row regardless (see [Spinner](spinner.md#deadline-countdown)).

## Custom Key

The key parameter controls the field name:

```go
e := clog.Info().Deadline("remaining", time.Minute)
runChecks()
e.Msg("checks passed")
// INF ℹ️ checks passed remaining=42s
```

## Log Levels

Since `Deadline` is a method on events, you control the level directly:

```go
clog.Warn().Deadline("timeout", 5*time.Second).Msg("still waiting")
// WRN ⚠️ still waiting timeout=2s
```

## Sub-loggers

`Deadline` works on any logger instance:

```go
logger := clog.With().Str("component", "queue").Logger()
e := logger.Info().Deadline("timeout", 20*time.Second)
drainQueue()
e.Msg("drained")
// INF ℹ️ drained component=queue timeout=13s
```

## Deadline Configuration

Deadline fields inherit the elapsed display settings from [`FieldFormats`](configuration.md#field-formats): `ElapsedFormat` (custom formatter, falling back to `DurationFormat`) and `ElapsedScale` (which can inherit the shared `TimeScale`). There is deliberately no separate deadline scale. Two other differences from `Elapsed`:

- Rounding uses the **ceiling**, not nearest: remaining time displays as one full selected scale/rounding step, so the default countdown from `15s` steps `15s, 14s, ... 1s` and shows `0s` only when truly expired.
- `ElapsedMinimum` never hides a deadline field - hiding a countdown as it approaches expiry would defeat its purpose.

### Gradient

The gradient position is the consumed fraction `(from - remaining) / from`, so `from` is always the gradient maximum - there is no `GradientMax` knob. The stops default to the logger's `DeadlineGradient` (green → yellow → red); override per field with options from the `deadline` package:

```go
import "github.com/gechr/clog/field/deadline"

e := clog.Info().Deadline("timeout", 15*time.Second,
  deadline.WithGradient(
    style.ColorStop{Position: 0, Color: blue},
    style.ColorStop{Position: 1, Color: magenta},
  ),
  deadline.WithGradientMode(style.GradientModeStep),
)
waitForConfirmation()
e.Msg("confirmed")
```
