# Custom Labels

Override the default level labels with `SetLabels`:

```go
clog.SetLabels(clog.LabelMap{
  clog.LevelInfo:  "INFO",
  clog.LevelWarn:  "WARNING",
  clog.LevelError: "ERROR",
})
```

Missing levels fall back to the defaults. Use `DefaultLabels()` to get a copy of the default label map.

## Level Alignment

When custom labels have different widths, control alignment with `SetLevelAlign`:

```go
clog.SetLevelAlign(clog.AlignRight)   // default: "   INFO", "WARNING", "  ERROR"
clog.SetLevelAlign(clog.AlignLeft)    //          "INFO   ", "WARNING", "ERROR  "
clog.SetLevelAlign(clog.AlignCenter)  //          " INFO  ", "WARNING", " ERROR "
clog.SetLevelAlign(clog.AlignNone)    //          "INFO",    "WARNING", "ERROR"
```
