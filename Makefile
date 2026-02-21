GO       ?= go
GO_TOOLS ?= $(shell $(GO) tool | grep /)

.PHONY: all
all: fmt lint test

.PHONY: demo
demo:
	@vhs ./assets/demo.tape

.PHONY: examples
examples:
	@$(GO) run ./examples

.PHONY: fmt
fmt:
	@rumdl fmt --quiet
	@$(GO) fix ./...
	@$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt --enable=gci,golines,gofumpt
	@$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run --fix --enable-only tagalign

.PHONY: gen
gen:
	@$(GO) generate

.PHONY: lint
lint:
	@$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

.PHONY: spinners
spinners:
	@$(GO) run ./examples -spinners="$(SPINNERS)"

.PHONY: test
test:
	@$(GO) test -timeout 2m -race ./...

.PHONY: update
update:
	@$(GO) get $(GO_TOOLS) $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	@$(GO) mod tidy
