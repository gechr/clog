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

Use a different character for both sides by passing the same rune twice:

```go
clog.SetQuoteChars('\'', '\'')
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

## Smart Quoting

Enable smart quoting to pick a delimiter per value instead of escaping. Each quoted value is wrapped in the first delimiter that does not occur in it, so the output stays escape-free:

```go
clog.SetSmartQuotes(true)
clog.Info().Str("msg", `plain value`).Msg("test")
// INF ℹ️ test msg="plain value"

clog.Info().Str("msg", `say "hi"`).Msg("test")
// INF ℹ️ test msg='say "hi"'

clog.Info().Str("msg", `it's a "test"`).Msg("test")
// INF ℹ️ test msg=`it's a "test"`
```

The default preference order is `"`, then `'`, then `` ` ``. When a value contains all of them (or a backslash or a non-printable character, which cannot be wrapped literally), smart quoting falls back to Go-style escaped quoting via `strconv.Quote`.

### Custom Preference Order

Override the order with `SetSmartQuoteChars`. Each `QuotePair` may use distinct opening and closing runes (a zero `Close` mirrors `Open`):

```go
clog.SetSmartQuotes(true)
clog.SetSmartQuoteChars(
	clog.QuotePair{Open: '«', Close: '»'},
	clog.QuotePair{Open: '['},
)
clog.Info().Str("msg", "hello world").Msg("test")
// INF ℹ️ test msg=«hello world»

clog.Info().Str("msg", "a » b").Msg("test")
// INF ℹ️ test msg=[a » b]
```

Passing no pairs restores the default order. Smart quoting takes precedence over `SetQuoteChars`.

## Styling Delimiters

By default the quote delimiters share the styling of the value they wrap. Set a `FieldQuote` style (see [Styles](styles.md)) to color them independently of the value body.

Quoting applies to individual field values and to elements within string and `[]any` slices. All quoting settings are inherited by sub-loggers. Pass `0` to reset to the default (`strconv.Quote`).
