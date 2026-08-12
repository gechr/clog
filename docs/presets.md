# Presets

A `Preset` bundles logger configuration - part order, alignment, wrapping, labels, symbols, styles, and spinner defaults - into one declarative value applied in a single call, replacing dozens of imperative `Set*` calls at startup.

```go
clog.ApplyPreset(clog.TersePreset())

clog.Info().Str("service", "billing").Msg("Starting")
// · Starting service=billing
clog.Success().Msg("Deployed")
// ✔︎ Deployed
```

## Fields

Fields are applied in the order listed. Nil or empty fields are skipped, so sparse presets layer over the logger's current configuration, and applying the same preset twice is harmless.

| Field             | Type               | Applied via          | Semantics                             |
| ----------------- | ------------------ | -------------------- | ------------------------------------- |
| `Parts`           | `[]Part`           | `SetParts`           | Replaces the part order               |
| `LevelAlign`      | `*Align`           | `SetLevelAlign`      | Replaces label alignment              |
| `Wrap`            | `*Wrap`            | `SetWrap`            | Replaces the wrap mode                |
| `Labels`          | `LabelMap`         | `SetLabels`          | Merges over the default labels        |
| `Symbols`         | `LabelMap`         | `SetSymbols`         | Merges over the default symbols       |
| `Styles`          | `*style.Config`    | `SetStyles`          | Merges into the current styles        |
| `SpinnerDefaults` | `[]spinner.Option` | `SetSpinnerDefaults` | Replaces the spinner defaults         |

`ApplyPreset` is a thin orchestrator over the corresponding setters, so each field keeps its setter's exact semantics. Application is not atomic - each setter locks separately - so apply presets at startup, before logging begins.

Presets exist per-logger (`logger.ApplyPreset`) and for the `Default` logger (`clog.ApplyPreset`).

## Built-in Presets

### Terse

`TersePreset` renders symbol-first lines with no level labels or timestamps, minimal single-character glyphs, ANSI colors, soft wrapping, and a bouncing-dots spinner. It suits compact CLI tools. Each call returns a fresh value, safe to modify before applying.

| Level     | Symbol | Symbol style | Message style |
| --------- | ------ | ------------ | ------------- |
| `Info`    | `·`    | yellow       | green         |
| `Success` | `✔︎`    | green        | green         |
| `Dry`     | `$`    | yellow bold  | yellow        |
| `Notice`  | `›`    | yellow       | yellow        |
| `Warn`    | `!`    | yellow       | yellow        |
| `Error`   | `✘`    | red          | red           |
| `Fatal`   | `✘`    | red          | plain         |

Messages are bold across all levels. Because the part order omits `PartLevel`, level labels never render - but names still work everywhere else (`ParseLevel("success")`, marshalling, `CLOG_LOG_LEVEL`).

```go
clog.ApplyPreset(clog.TersePreset())

clog.Dry().Msg("rm -rf ./cache")
// $ rm -rf ./cache
clog.Warn().Str("region", "emea").Msg("Degraded upstream")
// ! Degraded upstream region=emea
```

Terse leaves the printer and gradient styles untouched, so background-adaptive theming (see [Styles](styles.md)) keeps working.

## Custom Presets

Define your own preset as a literal and apply it the same way. Unset fields inherit the logger's current configuration:

```go
preset := &clog.Preset{
    Parts: []clog.Part{clog.PartTimestamp, clog.PartMessage, clog.PartFields},
    Wrap:  new(clog.WrapSoft),
    Symbols: clog.LabelMap{
        clog.LevelInfo: ">>",
    },
}
clog.ApplyPreset(preset)
```

Presets layer: applying a second preset only changes the fields it sets, so a tool can ship a base preset and let subcommands apply sparse tweaks on top.
