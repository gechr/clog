# Part Order

Control which parts appear in log output and in what order. The default order is: timestamp, level, symbol, message, fields.

```go
// Reorder: show message before level
clog.SetParts(clog.PartMessage, clog.PartLevel, clog.PartSymbol, clog.PartFields)

// Hide parts by omitting them
clog.SetParts(clog.PartLevel, clog.PartMessage, clog.PartFields) // no symbol or timestamp

// Fields before message
clog.SetParts(clog.PartLevel, clog.PartFields, clog.PartMessage)
```

Available parts: `PartTimestamp`, `PartLevel`, `PartSymbol`, `PartMessage`, `PartFields`.

Use `DefaultParts()` to get the default ordering. Parts omitted from the list are hidden.

## Per-Event Override

Override the part order for a single log event without mutating the logger:

```go
// This event shows only the symbol and message - no level label or fields.
clog.Info().Parts(clog.PartSymbol, clog.PartMessage).Msg("hello")

// Other events still use the logger's default parts.
clog.Info().Msg("world")
```

This is useful when a particular message needs different formatting (e.g. a banner or status line) without creating a separate logger or calling `SetParts`.

The same `.Parts()` method is available on animations and their results:

```go
// Spinner - both animation and completion use overridden parts.
clog.Spinner("loading").
  Parts(clog.PartSymbol, clog.PartMessage).
  Wait(ctx, fn).
  Msg("done")

// Override only on the completion message.
clog.Spinner("loading").
  Wait(ctx, fn).
  Parts(clog.PartMessage).
  Msg("done")
```

See [Spinner](spinner.md) and [Group](group.md) for more details.
