# `klog` Integration

Use `kloghandler.SetKlog` to route [`k8s.io/klog/v2`](https://pkg.go.dev/k8s.io/klog/v2) through a clog logger. Once set, klog stops writing to its own files and streams, so its flags need no further attention.

```go
import "github.com/gechr/clog/kloghandler"

kloghandler.SetKlog(clog.Default(), nil)

klog.InfoS("reconciled", "name", "example")
// INF ℹ️ reconciled name=example
```

## `logr`

klog has no handler interface of its own - `klog.SetLogger` accepts a [`logr.Logger`](https://pkg.go.dev/github.com/go-logr/logr#Logger), so the bridge is really a `logr.LogSink`. That makes it usable by anything built on logr, not just klog:

```go
import "github.com/gechr/clog/kloghandler"

ctrl.SetLogger(kloghandler.NewLogger(clog.Default(), nil))
```

`New` returns the bare `logr.LogSink` when a consumer wants to wrap it itself.

## Options

```go
kloghandler.SetKlog(clog.Default(), &kloghandler.Options{
  AddSource:    true,                      // include source file:line in each entry
  Verbosity:    &verbosity,                // cap the V-level (nil = use the logger's level)
  NameKey:      "component",               // field key for WithName (default "logger")
  KlogSeverity: true,                      // keep the severity of unstructured klog calls
  LevelFor:     kloghandler.VerbosityLevel, // custom V-level mapping
})
```

## Level Mapping

logr has no levels, only verbosity: `Info` carries a V-level and `Error` does not. The default mapping follows logr's own convention.

| logr call  | clog level        |
| ---------- | ----------------- |
| `V(0)`     | `clog.LevelInfo`  |
| `V(1)`     | `clog.LevelDebug` |
| `V(2)`+    | `clog.LevelTrace` |
| `Error`    | `clog.LevelError` |

Kubernetes components tend to treat `V(4)` as debug and `V(5)` as trace instead. Set `LevelFor` to follow that or any other scheme:

```go
opts := &kloghandler.Options{
  LevelFor: func(verbosity int) clog.Level {
    switch {
    case verbosity < 4:
      return clog.LevelInfo
    case verbosity == 4:
      return clog.LevelDebug
    default:
      return clog.LevelTrace
    }
  },
}
```

## Verbosity

klog applies its own `-v` flag before a verbose call ever reaches the sink, so a trace-level clog logger alone will not surface `klog.V(2)` entries. `Verbosity` raises klog's `-v` to match, wiring both ends of the bridge to the same ceiling:

```go
verbosity := 4
kloghandler.SetKlog(clog.Default(), &kloghandler.Options{Verbosity: &verbosity})

klog.V(4).InfoS("cache synced")
// TRC 🔍 cache synced
```

Leave it nil to keep whatever `-v` the program already parsed.

## Severity

klog only forwards a severity for its structured calls. Its unstructured ones - `Info`, `Warning`, `Error`, `Fatal` - are formatted into a line first and arrive as plain `Info`, so `klog.Warning` is logged at `clog.LevelInfo`.

`KlogSeverity` recovers the real severity by reading klog's own formatted line instead:

```go
kloghandler.SetKlog(clog.Default(), &kloghandler.Options{KlogSeverity: true})

klog.Warning("careful")
// WRN ⚠️ careful
```

That carries two costs:

- Unstructured verbose calls such as `klog.V(4).Info` arrive as `Info`, because klog stamps them with an `Info` header.
- klog hands the formatted line straight to the callback without consulting the sink, so `Verbosity` and `LevelFor` do not apply to it. klog's own `-v` gate and the logger's level still do.

Structured calls are unaffected - they bypass the formatted line entirely, so their fields, V-level and sink gating all survive.

Because the severity lives in the header, `SetKlog` forces `-skip_headers` back off when `KlogSeverity` is set. Without a header the severity is not merely lost: a message that happens to be header-shaped would be read as one and forge its own.

Records mapped to `clog.LevelFatal` do not call `os.Exit` from clog's side; `klog.Fatal` terminates the process itself, as it always has - writing goroutine stacks directly to stderr on its way out, whatever logger is installed.

## Names and Values

`WithName` segments are joined with a dot and emitted as a single field, the convention logr backends share. `WithValues` pairs are preset on every entry:

```go
logger := kloghandler.NewLogger(clog.Default(), nil)

logger.WithName("controller").WithName("pod").WithValues("namespace", "default").Info("synced")
// INF ℹ️ synced logger=controller.pod namespace=default
```

A value implementing [`logr.Marshaler`](https://pkg.go.dev/github.com/go-logr/logr#Marshaler) is resolved through `MarshalLog` first. A key that is not a string is formatted with `fmt.Sprint`, and a trailing value with no key is recorded under `!BADKEY`.

## Restoring klog

`ClearKlog` hands logging back to klog itself:

```go
kloghandler.ClearKlog()
```

klog's logger is process-wide and not thread-safe to swap, so set it during initialization, before other goroutines log.
