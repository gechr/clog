# `log/slog` Integration

Use `sloghandler.New` to create a [`slog.Handler`](https://pkg.go.dev/log/slog#Handler) backed by a clog logger. This lets any code that accepts `slog.Handler` or `*slog.Logger` produce clog-formatted output.

```go
import "github.com/gechr/clog/sloghandler"

h := sloghandler.New(clog.Default, nil)
logger := slog.New(h)

logger.Info("request handled", "method", "GET", "status", 200)
// INF ℹ️ request handled method=GET status=200
```

## Options

```go
h := sloghandler.New(clog.Default, &sloghandler.Options{
  AddSource: true,           // include source file:line in each entry
  Level:     slog.LevelWarn, // override minimum level (nil = use logger's level)
})
```

## Level Mapping

| slog level     | clog level        |
| -------------- | ----------------- |
| < `LevelDebug` | `clog.LevelTrace` |
| `LevelDebug`   | `clog.LevelDebug` |
| `LevelInfo`    | `clog.LevelInfo`  |
| `LevelWarn`    | `clog.LevelWarn`  |
| `LevelError`   | `clog.LevelError` |
| > `LevelError` | `clog.LevelFatal` |

Records mapped to `LevelFatal` are logged but do **not** call `os.Exit` - only clog's own `Fatal().Msg()` does that.

## Groups and Attrs

`WithGroup` and `WithAttrs` work as expected. Groups use dot-notation keys matching clog's `Dict()` pattern:

```go
import "github.com/gechr/clog/sloghandler"

h := sloghandler.New(clog.Default, nil)
logger := slog.New(h).WithGroup("req")

logger.Info("handled", "method", "GET", "status", 200)
// INF ℹ️ handled req.method=GET req.status=200
```
