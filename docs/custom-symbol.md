# Custom Symbol

Override the default emoji symbol per-event, per-logger, or globally:

```go
// Per-event
clog.Info().Symbol("📦").Str("pkg", "clog").Msg("Installed")

// Per-logger (via sub-logger)
logger := clog.With().Symbol("🛡️").Str("component", "auth").Logger()
logger.Info().Msg("Ready")

// Global (changes defaults for all levels)
clog.SetSymbols(clog.LabelMap{
  clog.LevelInfo:  ">>",
  clog.LevelWarn:  "!!",
  clog.LevelError: "XX",
})
```

During animations, `SetSymbol` on the `Update` changes the icon mid-task:

```go
update.SetSymbol("📡").Str("stage", "receiving").Send()
```

Symbol resolution order: event override > logger preset > default emoji for level.

Missing levels in `SetSymbols` fall back to the defaults. Use `DefaultSymbols()` to get a copy of the default symbol map.

## Styling Symbols

Symbols can be any string - not just emojis. Use `Styles.Symbols` to apply a [lipgloss](https://charm.land/lipgloss/v2) style per level:

```go
clog.SetStyles(&style.Config{
  Symbols: style.LevelMap{
    clog.LevelInfo:  new(lipgloss.NewStyle().Foreground(lipgloss.Color("2"))), // green
    clog.LevelWarn:  new(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))), // yellow
    clog.LevelError: new(lipgloss.NewStyle().Foreground(lipgloss.Color("1"))), // red
  },
})
```

Symbol styles also apply to spinner animation frames, so spinners inherit the color of their level.

`Styles.Symbols` is a `LevelMap`. Entries for levels not in the map render unstyled (the default).
