# Wrapping

When log lines exceed the terminal width, the terminal hard-wraps them at arbitrary column positions, potentially breaking mid-word. `SetWrap` enables ANSI-aware wrapping that respects word boundaries and preserves escape sequences (colors, hyperlinks).

## Wrap Modes

| Mode       | Description                                                          |
| ---------- | -------------------------------------------------------------------- |
| `WrapNone` | No wrapping; lines are written as-is (default)                       |
| `WrapHard` | Break at the terminal width, even mid-word                           |
| `WrapSoft` | Break at word boundaries, falling back to hard breaks for long words |

```go
clog.SetWrap(clog.WrapSoft)
clog.Info().Strs("repos", repos).Int("total", len(repos)).Msg("Cloned")
```

Wrapping uses the detected terminal width ([Output.Width]) and has no effect on non-TTY outputs.
