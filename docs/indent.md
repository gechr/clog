# Indentation

Create visually nested output where the message text shifts right while level, timestamp, and prefix columns stay fixed.

## SetIndent

Set the indent depth directly on any logger:

```go
logger.SetIndent(1)
logger.Info().Msg("indented")
// INF ℹ️   indented

logger.SetIndent(0) // back to normal
```

## Sub-logger Indent

Use `Indent()` on a context builder to add one indent level (2 spaces by default). It's chainable:

```go
clog.Info().Msg("Building project")

build := clog.With().Indent().Logger()
build.Info().Msg("Compiling main.go")
build.Info().Msg("Compiling util.go")

link := build.With().Indent().Logger()
link.Info().Msg("Linking binary")

clog.Info().Msg("Build complete")
```

```text
INF ℹ️ Building project
INF ℹ️   Compiling main.go
INF ℹ️   Compiling util.go
INF ℹ️     Linking binary
INF ℹ️ Build complete
```

## Dedent

Use `Dedent()` to remove one indent level from a context builder. It is the
mirror of `Indent()` and clamps at zero - calling it on an unindented logger
is a no-op:

```go
build := clog.With().Depth(3).Logger()

// Step back one level for a less-nested sub-logger.
back := build.With().Dedent().Logger()
back.Info().Msg("one step back")
// INF ℹ️     one step back  (depth 2, not 3)

// Indent and Dedent cancel out.
same := clog.With().Indent().Dedent().Logger()
same.Info().Msg("no change")
// INF ℹ️ no change
```

`Dedent()` is also available on animation builders:

```go
clog.Spinner("loading").Indent().Indent().Dedent(). // net depth 1
    Wait(ctx, work).Msg("done")
```

## Depth

Use `Depth(n)` to add multiple indent levels at once:

```go
sub := clog.With().Depth(3).Logger()
sub.Info().Msg("Deeply nested")
// INF ℹ️       Deeply nested
```

## Indent Width

The default indent width is 2 spaces per level. Change it with `SetIndentWidth`:

```go
clog.SetIndentWidth(4)
```

## Indent Prefixes

Add decorations before the message at each indent level with `SetIndentPrefixes`. Prefixes cycle through the slice and a separator (default `" "`) is appended automatically:

```go
clog.SetIndentPrefixes([]string{"│"})

sub := clog.With().Indent().Logger()
sub.Info().Msg("hello")
// INF ℹ️   │ hello
```

With multiple prefixes they cycle per depth:

```go
clog.SetIndentPrefixes([]string{"├─", "└─"})

d1 := clog.With().Indent().Logger()
d1.Info().Msg("first")
// INF ℹ️   ├─ first

d2 := clog.With().Depth(2).Logger()
d2.Info().Msg("second")
// INF ℹ️     └─ second

d3 := clog.With().Depth(3).Logger()
d3.Info().Msg("third")  // wraps back to first prefix
// INF ℹ️       ├─ third
```

## Prefix Separator

Change the separator between the prefix and message with `SetIndentPrefixSeparator`:

```go
clog.SetIndentPrefixSeparator("")   // no space
clog.SetIndentPrefixSeparator("  ") // double space
```

## Handlers

Custom handlers receive the indent level via `Entry.Indent`:

```go
clog.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
    fmt.Printf("indent=%d msg=%s\n", e.Indent, e.Message)
}))
```
