# Custom Parts

A log line is assembled from parts (see [Part Order](part-order.md)). Beyond the five built-in parts, any other `Part` value can carry a renderer of your own, letting a line show something clog knows nothing about - a worker id, a hostname, a build number.

## Registering a Custom Part

```go
const PartWorker clog.Part = 100

func init() {
    clog.RegisterPart(PartWorker, func(e clog.Entry, styles *style.Config, noColor bool) string {
        if noColor {
            return workerID
        }
        return styles.KeyDefault.Render(workerID)
    })
}

clog.SetParts(clog.PartLevel, PartWorker, clog.PartMessage, clog.PartFields)
clog.Info().Str("queue", "ingest").Msg("draining")
// INF worker-3 draining queue=ingest
```

Pick any value outside the built-in range (`0`-`4`); `RegisterPart` panics if you pass a built-in part. Registering the same custom part again replaces its renderer, and `UnregisterPart` (or a nil renderer) removes it.

The registry is package-global, like [custom levels](custom-levels.md): a registered part works across every logger in the process, and two libraries choosing the same numeric value will collide.

## The Renderer

The renderer receives the completed [`Entry`](handlers.md) - the same snapshot a `Handler` gets - so it can react to the level, message, or fields:

```go
clog.RegisterPart(PartWorker, func(e clog.Entry, _ *style.Config, _ bool) string {
    if e.Level < clog.LevelWarn {
        return "" // an empty string omits the part for this line
    }
    return workerID
})
```

`e.Time` is set only when timestamp reporting is enabled and the event carries a time. It is the same instant the built-in `PartTimestamp` renders, so a part deriving from it cannot disagree with the timestamp on its own line.

Treat `styles` as read-only: it is the logger's live configuration, so mutating it changes subsequent output, and holding on to the pointer past the call is unsafe.

Renderers run while the logger is locked: do not log through the same logger from inside one, and keep the work cheap - it runs for every line whose order includes the part.

## Per-Event Override

Custom parts work with `.Parts()` just like built-in ones:

```go
clog.Info().Parts(PartWorker, clog.PartMessage).Msg("tagged")
```

A part with no registered renderer is skipped, so an order can name parts that are registered conditionally.

## Animations

Custom parts appear on lines written through the logger, which includes an animation's completion line:

```go
clog.Spinner("loading").Wait(ctx, fn).Msg("done")
// loading          <- task row, painted by the animation renderer: no custom parts
// worker-3 done    <- completion line, logged normally: custom parts included
```

Task rows are painted by the animation renderer, which is handed pre-rendered strings rather than an `Entry`, so it omits parts it does not know about. That covers live frames, the final row an animation leaves behind, and every row inside a [Group](group.md).
