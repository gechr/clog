# TOML

`TOML` marshals any Go value; `RawTOML` accepts pre-serialized bytes:

```go
clog.Print().TOML(configStruct)
clog.Print().RawTOML(configBytes)
```

## Styling

Token colors are configured via styles:

```go
custom := style.DefaultTOML()
custom.Key = new(lipgloss.NewStyle().Foreground(lipgloss.Color("#50fa7b")))
clog.SetStyles(&style.Config{TOML: custom})
```

| Style field   | Tokens                                           |
| ------------- | ------------------------------------------------ |
| `BoolTrue`    | `true`                                           |
| `BoolFalse`   | `false`                                          |
| `Comment`     | `# text`                                         |
| `DateTime`    | dates, times, datetimes                          |
| `Float`       | floating point, inf, nan                         |
| `Integer`     | integers, hex, octal, binary                     |
| `Key`         | bare and dotted keys                             |
| `Punctuation` | structural tokens (`=`, `[`, `]`, `{`, `}`, `,`) |
| `String`      | basic and literal strings                        |
| `TableKey`    | `[table]` and `[[array]]` header keys            |
