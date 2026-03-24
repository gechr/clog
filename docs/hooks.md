# Hooks

Hooks let you run code at specific points in the log write lifecycle. Register them with `AddHook` and a `HookPoint`.

## Hook Points

| Point             | When                                 |
| ----------------- | ------------------------------------ |
| `HookBeforeWrite` | Just before each log line is written |
| `HookAfterWrite`  | Just after each log line is written  |

## Usage

```go
// Clear a spinner line before log output
clog.AddHook(clog.HookBeforeWrite, func() {
    fmt.Print("\r\033[K")
})

// Restore a prompt after log output
clog.AddHook(clog.HookAfterWrite, func() {
    fmt.Print(">>> ")
})
```

Multiple hooks per point run in registration order.

## Clearing Hooks

```go
clog.ClearHooks(clog.HookBeforeWrite) // clear one point
clog.ClearAllHooks()                  // clear all points
```

## Notes

- Hooks are called under the logger's mutex - they must not call back into the same logger.
- Hooks fire for all log levels and for both the built-in formatter and custom [handlers](handlers.md).
- Sub-loggers inherit parent hooks.
- Per-logger methods: `logger.AddHook(point, fn)`, `logger.ClearHooks(point)`, `logger.ClearAllHooks()`.
