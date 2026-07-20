# Sub-loggers & Context

## Sub-loggers

Create sub-loggers with preset fields using the `With()` context builder:

```go
logger := clog.With().Str("component", "auth").Logger()
logger.Info().Str("user", "john").Msg("Authenticated")
// INF ℹ️ Authenticated component=auth user=john
```

Context fields support the same typed methods as events.

## Context Propagation

Store a logger in a `context.Context` and retrieve it deeper in the call stack:

```go
logger := clog.With().Str("request_id", "abc-123").Logger()
ctx := logger.WithContext(ctx)

// later, in any function that receives ctx:
clog.Ctx(ctx).Info().Msg("Handling request")
// INF ℹ️ Handling request request_id=abc-123
```

`Ctx` returns `clog.Default()` when the context is `nil` or contains no logger, so it is always safe to call.

A package-level `WithContext` convenience stores `clog.Default()`:

```go
ctx := clog.WithContext(ctx) // stores clog.Default()
```
