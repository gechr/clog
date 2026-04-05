# HCL

`RawHCL` accepts pre-serialized HCL bytes (there is no marshal method since HCL has no standard Go marshal API):

```go
clog.Print().RawHCL(terraformConfig)
```

## Styling

Token colors are configured via styles:

```go
custom := style.DefaultHCL()
custom.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
clog.SetStyles(&style.Config{HCL: custom})
```

| Style field   | Tokens                                                                |
| ------------- | --------------------------------------------------------------------- |
| `BlockType`   | block type identifiers (`resource`, `variable`)                       |
| `BoolFalse`   | `false`                                                               |
| `BoolTrue`    | `true`                                                                |
| `Comment`     | `#`, `//`, `/* */` comments                                           |
| `Key`         | attribute keys (identifier before `=`)                                |
| `NestedKey`   | attribute keys inside nested blocks (depth >= 2); falls back to `Key` |
| `Null`        | `null`                                                                |
| `Number`      | numeric literals                                                      |
| `Punctuation` | structural tokens (`=`, `{`, `}`, `[`, `]`)                           |
| `String`      | string values and quote markers                                       |
