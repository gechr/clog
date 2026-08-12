# Custom Levels

Define custom log levels at any numeric value between the built-in levels. Built-in levels use uniform gaps of 10, so there is plenty of room.

## Registering a Custom Level

```go
const AuditLevel clog.Level = clog.LevelDry + 5

func init() {
    clog.RegisterLevel(AuditLevel, clog.LevelConfig{
        Name:   "audit", // required: canonical name for ParseLevel/MarshalText
        Label:  "AUD", // short display label (default: uppercase Name, max 3 chars)
        Symbol: "📋", // emoji symbol (default: "")
        Style:  new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))),
    })
}
```

## Logging with a Custom Level

Use `Log(level)` instead of the named methods:

```go
clog.Log(AuditLevel).Str("actor", "deploy-bot").Msg("Config changed")
// AUD 📋 Config changed actor=deploy-bot
```

## Level Filtering

Custom levels respect the same filtering rules as built-in levels. A custom level with value `25` (between `LevelDry` at `20` and `LevelSuccess` at `30`) is visible when the minimum level is `LevelInfo` but hidden when the minimum level is `LevelWarn` or higher.

## ParseLevel and Marshalling

Registered custom levels work with `ParseLevel`, `MarshalText`, and `UnmarshalText`:

```go
level, err := clog.ParseLevel("audit") // returns AuditLevel
```

## Iterating All Levels

Since built-in levels use gaps, `level++` iteration will not work. Use `Levels()` instead:

```go
for _, level := range clog.Levels() {
    fmt.Println(level) // prints all levels in ascending severity order
}
```
