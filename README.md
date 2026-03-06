<h1 align="center"><code>clog</code></h1>

A highly customizable structured logger for command-line tools with a [zerolog](https://github.com/rs/zerolog)-inspired fluent API, terminal-aware colors, hyperlinks, animations, and custom log levels.

## Demo

<p align="center">
  <img src="./assets/banner.gif" alt="clog demo" />
</p>

## Installation

```sh
go get github.com/gechr/clog
```

## Quick Start

```go
package main

import (
  "fmt"

  "github.com/gechr/clog"
)

func main() {
  clog.Info().Str("port", "8080").Msg("Server started")
  clog.Warn().Str("path", "/old").Msg("Deprecated endpoint")
  err := fmt.Errorf("connection refused")
  clog.Error().Err(err).Msg("Connection failed")
}
```

Output:

```text
INF ℹ️ Server started port=8080
WRN ⚠️ Deprecated endpoint path=/old
ERR ❌ Connection failed error=connection refused
```

## Documentation

Full documentation is available at [gechr.github.io/clog](https://gechr.github.io/clog/).
