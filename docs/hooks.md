# Hooks

Write hooks let you run code immediately before or after each log line is written. This is useful for coordinating log output with other terminal activity such as spinners or progress bars.

## `SetHookBeforeWrite`

Called just before each log line is written to the output. For example, clearing a spinner line so log output isn't visually corrupted:

```go
clog.SetHookBeforeWrite(func() {
    fmt.Print("\r\033[K") // clear current line
})

// Later, remove the hook:
clog.SetHookBeforeWrite(nil)
```

## `SetHookAfterWrite`

Called just after each log line is written:

```go
clog.SetHookAfterWrite(func() {
    fmt.Print(">>> ") // restore a prompt
})
```

## Notes

- Both hooks are called under the logger's mutex, so they must not call back into the same logger.
- Pass `nil` to clear a hook.
- Hooks fire for all log levels (debug through fatal) and for both the built-in formatter and custom [handlers](handlers.md).
- Per-logger hooks are available via `logger.SetHookBeforeWrite(fn)` and `logger.SetHookAfterWrite(fn)`.
