package main

import (
	"github.com/gechr/clog"
)

func main() {
	// Default: multiline pretty-printed output with syntax highlighting.
	clog.Print().RawJSON([]byte(`{"status":"ok","count":42,"active":true,"tags":["prod","staging"]}`))

	// Inline mode: flatten to a single line.
	clog.Print().Mode(clog.PrintInline).RawJSON([]byte(`{"status":"ok","count":42}`))

	// Marshal a Go value.
	clog.Print().JSON(map[string]any{
		"name":   "alice",
		"active": true,
	})

	// Compare with a regular log event that includes JSON as a field.
	clog.Info().
		RawJSON("response", []byte(`{"status":"ok","count":42,"active":true}`)).
		Msg("API response")

	// Custom indent.
	clog.SetPrintIndent("\t")
	clog.Print().RawJSON([]byte(`{"a":1,"b":2}`))
}
