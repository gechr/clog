# Handlers

Implement the `Handler` interface for custom output formats:

```go
type Handler interface {
  Log(Entry)
}
```

The `Entry` struct provides `Level`, `Time`, `Message`, `Symbol`, `Indent`, `Tree`, and `Fields`. The logger handles level filtering, field accumulation, timestamps, and locking - the handler only formats and writes.

```go
// Using HandlerFunc adapter
clog.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
  data, _ := json.Marshal(e)
  fmt.Println(string(data))
}))
```

Example output:

```json
{"level":"info","symbol":"ℹ️","message":"Server started","fields":[{"key":"port","value":"8080"}]}
```

`Level` serializes as a human-readable string (e.g. `"info"`, `"error"`). `Time` is omitted when timestamps are disabled. `Symbol`, `Indent`, `Tree`, and `Fields` are omitted when empty/zero.
