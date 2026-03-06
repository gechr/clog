# Handlers

Implement the `Handler` interface for custom output formats:

```go
type Handler interface {
  Log(Entry)
}
```

The `Entry` struct provides `Level`, `Time`, `Message`, `Symbol`, and `Fields`. The logger handles level filtering, field accumulation, timestamps, and locking - the handler only formats and writes.

```go
// Using HandlerFunc adapter
clog.SetHandler(clog.HandlerFunc(func(e clog.Entry) {
  data, _ := json.Marshal(e)
  fmt.Println(string(data))
}))
```

Example output:

```json
{"fields":[{"key":"port","value":"8080"}],"level":"info","message":"Server started"}
```

`Level` serializes as a human-readable string (e.g. `"info"`, `"error"`). `Time` is omitted when timestamps are disabled. `Fields` and `Symbol` are omitted when empty.
