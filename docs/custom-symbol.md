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
styles := clog.DefaultStyles()

// Render the warn symbol in bold yellow
styles.Symbols[clog.LevelWarn] = new(
  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")), // yellow
)

clog.SetStyles(styles)

// Both "warning" and "!!" are printed in bold yellow
clog.Warn().Symbol("warning").Msg("Low disk space")
clog.Warn().Symbol("!!").Msg("Low disk space")
```

`Styles.Symbols` is a `LevelStyleMap`. Entries for levels not in the map render unstyled (the default). Use `nil` for a specific level to explicitly disable styling for that level.
