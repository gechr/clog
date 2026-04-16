# Quoting

By default, field values containing spaces or special characters are wrapped in Go-style double quotes (`"hello world"`). This behaviour can be customised with `SetQuote`.

## Quote Modes

| Mode          | Description                                                                   |
| ------------- | ----------------------------------------------------------------------------- |
| `QuoteAuto`   | Quote only when needed - spaces, unprintable chars, embedded quotes (default) |
| `QuoteAlways` | Always quote string, error, and default-kind values                           |
| `QuoteNever`  | Never quote                                                                   |

```go
// Default: only quote when needed
clog.Info().Str("reason", "timeout").Str("msg", "hello world").Msg("test")
// INF ℹ️ test reason=timeout msg="hello world"

// Always quote string values
clog.SetQuote(clog.QuoteAlways)
clog.Info().Str("reason", "timeout").Msg("test")
// INF ℹ️ test reason="timeout"

// Never quote
clog.SetQuote(clog.QuoteNever)
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg=hello world
```

## Custom Quote Character

Use a different character for both sides:

```go
clog.SetQuoteChar('\'')
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg='hello world'
```

## Asymmetric Quote Characters

Use different opening and closing characters:

```go
clog.SetQuoteChars('«', '»')
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg=«hello world»

clog.SetQuoteChars('[', ']')
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg=[hello world]
```

Quoting applies to individual field values and to elements within string and `[]any` slices. All quoting settings are inherited by sub-loggers. Pass `0` to reset to the default (`strconv.Quote`).
