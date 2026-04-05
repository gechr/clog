# YAML

`YAML` marshals any Go value; `RawYAML` accepts pre-serialized bytes:

```go
clog.Print().YAML(configStruct)
clog.Print().RawYAML(responseBody)
```

## Styling

Token colors are configured via styles:

```go
custom := style.DefaultYAML()
custom.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
clog.SetStyles(&style.Config{YAML: custom})
```

| Style field   | Tokens                                                |
| ------------- | ----------------------------------------------------- |
| `Anchor`      | `&name`                                               |
| `Alias`       | `*name`                                               |
| `BoolTrue`    | `true`, `yes`, `on`                                   |
| `BoolFalse`   | `false`, `no`, `off`                                  |
| `Comment`     | `# text`                                              |
| `Key`         | mapping keys                                          |
| `Null`        | `null`, `~`                                           |
| `Number`      | integers, floats, hex, octal, binary, inf, nan        |
| `Punctuation` | structural tokens (`:`, `-`, `[`, `]`, `{`, `}`, `,`) |
| `String`      | plain, single-quoted, double-quoted string values     |
| `Tag`         | `!!str`, `!!int`, `!custom`                           |
