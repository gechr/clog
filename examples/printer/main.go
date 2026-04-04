package main

import (
	"fmt"

	"github.com/gechr/clog"
)

func main() {
	// JSON: multiline pretty-printed with nested objects and arrays.
	clog.Print().RawJSON([]byte(`{
		"user": {
			"name": "alice",
			"age": 30,
			"active": true,
			"email": null,
			"roles": ["admin", "editor"],
			"settings": {
				"theme": "dark",
				"notifications": false,
				"limits": {"maxRetries": 3, "timeout": 30.5}
			}
		},
		"tags": ["prod", "us-east-1"]
	}`))

	fmt.Println()

	// JSON: inline mode.
	clog.Print().Mode(clog.JSONFlat).RawJSON([]byte(`{"status":"ok","count":42}`))

	// JSON: marshal a Go value.
	clog.Print().JSON(map[string]any{
		"name":   "alice",
		"active": true,
		"scores": []int{98, 87, 95},
	})

	fmt.Println()

	// Compare with a regular log event that includes JSON as a field.
	clog.Info().
		RawJSON("response", []byte(`{"status":"ok","count":42,"active":true}`)).
		Msg("API response")

	fmt.Println()

	// YAML: marshal a deeply nested Go value.
	clog.Print().YAML(map[string]any{
		"database": map[string]any{
			"host":     "localhost",
			"port":     5432,
			"name":     "myapp",
			"replicas": []string{"db1.internal", "db2.internal"},
			"pool": map[string]any{
				"min":     5,
				"max":     20,
				"timeout": "30s",
			},
		},
		"debug":   false,
		"version": 2.1,
	})

	fmt.Println()

	// YAML: pre-serialized bytes with inline comments.
	clog.Print().RawYAML([]byte(`server:
  host: 0.0.0.0 # bind address
  port: 8080    # listen port
  tls:
    enabled: true  # require TLS
    cert: /etc/ssl/cert.pem
# Feature flags
features:
  - name: dark-mode
    enabled: true
  - name: beta-api
    enabled: false  # not yet GA
`))
}
