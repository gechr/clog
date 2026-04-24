# Custom Levels

Define custom log levels at any numeric value between the built-in levels. Built-in levels use gaps of 5, so there is plenty of room.

## Registering a Custom Level

```go
const SuccessLevel clog.Level = clog.LevelDry + 1

func init() {
    clog.RegisterLevel(SuccessLevel, clog.LevelConfig{
        Name:   "success", // required: canonical name for ParseLevel/MarshalText
        Label:  "SCS", // short display label (default: uppercase Name, max 3 chars)
        Symbol: "✅", // emoji symbol (default: "")
        Style:  new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))),
    })
}
```

## Logging with a Custom Level

Use `Log(level)` instead of the named methods:

```go
clog.Log(SuccessLevel).Str("pkg", "api").Msg("Build completed")
// SCS ✅ Build completed pkg=api
```

## Level Filtering

Custom levels respect the same filtering rules as built-in levels. A custom level with value `3` (between `LevelDry` at `2` and `LevelWarn` at `5`) is visible when the minimum level is `LevelInfo` but hidden when the minimum level is `LevelWarn` or higher.

## ParseLevel and Marshalling

Registered custom levels work with `ParseLevel`, `MarshalText`, and `UnmarshalText`:

```go
level, err := clog.ParseLevel("success") // returns SuccessLevel
```

## Iterating All Levels

Since built-in levels use gaps, `level++` iteration will not work. Use `Levels()` instead:

```go
for _, level := range clog.Levels() {
    fmt.Println(level) // prints all levels in ascending severity order
}
```
