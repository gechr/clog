# Omit Empty / Zero

## OmitEmpty

**OmitEmpty** omits fields that are semantically "nothing": `nil`, empty strings `""`, and nil or empty slices and maps.

```go
clog.SetOmitEmpty(true)
clog.Info().
  Str("name", "alice").
  Str("nickname", "").   // omitted
  Any("role", nil).      // omitted
  Int("age", 0).         // kept (zero but not empty)
  Bool("admin", false).  // kept (zero but not empty)
  Msg("User")
// INF ℹ️ User name=alice age=0 admin=false
```

## OmitZero

**OmitZero** is a superset of `OmitEmpty` - it additionally omits `0`, `false`, `0.0`, zero durations, and any other typed zero value.

```go
clog.SetOmitZero(true)
clog.Info().
  Str("name", "alice").
  Str("nickname", "").   // omitted
  Any("role", nil).      // omitted
  Int("age", 0).         // omitted
  Bool("admin", false).  // omitted
  Msg("User")
// INF ℹ️ User name=alice
```

Both settings are inherited by sub-loggers created with `With()`. When both are enabled, `OmitZero` takes precedence.
