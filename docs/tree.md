# Tree Indentation

Draw tree-like structures with box-drawing connectors (`├──`, `└──`, `│`). Each call to `Tree()` adds one nesting level, and ancestor continuation lines are handled automatically.

## Basic Usage

Use `Tree()` on a context builder with `TreeFirst`, `TreeMiddle`, or `TreeLast`:

```go
clog.Info().Msg("Project")

src := clog.With().Tree(clog.TreeMiddle).Logger()
src.Info().Msg("src/")

main := src.With().Tree(clog.TreeMiddle).Logger()
main.Info().Msg("main.go")

util := src.With().Tree(clog.TreeLast).Logger()
util.Info().Msg("util.go")

mod := clog.With().Tree(clog.TreeLast).Logger()
mod.Info().Msg("go.mod")
```

```text
INF ℹ️ Project
INF ℹ️ ├── src/
INF ℹ️ │   ├── main.go
INF ℹ️ │   └── util.go
INF ℹ️ └── go.mod
```

## Positions

| Position     | Connector | Meaning                |
| ------------ | --------- | ---------------------- |
| `TreeFirst`  | `├──`     | First sibling          |
| `TreeMiddle` | `├──`     | Middle sibling         |
| `TreeLast`   | `└──`     | Last sibling (no more) |

`TreeFirst` and `TreeMiddle` render identically by default but can be distinguished with custom characters (see below).

## Combining with Indent

Tree and indent are orthogonal. When both are present, indent spaces render first, then tree connectors:

```go
sub := clog.With().Indent().Tree(clog.TreeMiddle).Logger()
sub.Info().Msg("hello")
// INF ℹ️   ├── hello
```

## Custom Characters

Override the default box-drawing characters with `SetTreeChars`:

```go
clog.SetTreeChars(clog.TreeChars{
    First:    "┌─ ",
    Middle:   "├─ ",
    Last:     "└─ ",
    Continue: "│  ",
    Blank:    "   ",
})
```

Defaults include trailing whitespace for alignment (see `DefaultTreeChars()`).

| Field      | Default | Purpose                                   |
| ---------- | ------- | ----------------------------------------- |
| `First`    | `├──`   | Connector for `TreeFirst`                 |
| `Middle`   | `├──`   | Connector for `TreeMiddle`                |
| `Last`     | `└──`   | Connector for `TreeLast`                  |
| `Continue` | `│`     | Ancestor line when parent is First/Middle |
| `Blank`    |         | Ancestor line when parent is Last         |

## Animations

Tree positions work on animation builders too:

```go
clog.Spinner("loading").Tree(clog.TreeMiddle).Wait(ctx, work).Msg("done")
```

## Handlers

Custom handlers receive the tree positions via `Entry.Tree`:

```go
clog.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
    fmt.Printf("tree=%v msg=%s\n", e.Tree, e.Message)
}))
```
