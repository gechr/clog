CLOG_DOCS_DIR ?= ../clog-docs
GO            ?= go
GO_TOOLS      ?= $(shell $(GO) tool | grep /)

.PHONY: all
all: fmt lint test

TAPE_banner             := banner/banner
TAPE_spinner            := spinner/spinner
TAPE_pulse              := pulse/pulse
TAPE_shimmer            := shimmer/shimmer
TAPE_shimmer-directions := shimmer/directions
TAPE_bar                := bar/bar
TAPE_bar-styles         := bar/styles
TAPE_group              := group/group

GIF_banner              := demo.gif
GIF_spinner             := spinner.gif
GIF_pulse               := pulse.gif
GIF_shimmer             := shimmer.gif
GIF_shimmer-directions  := shimmer-directions.gif
GIF_bar                 := bar.gif
GIF_bar-styles          := bar-styles.gif
GIF_group               := group.gif

TAPE_NAMES   := banner spinner pulse shimmer shimmer-directions bar bar-styles group
TAPE_TARGETS := $(addprefix tape-,$(TAPE_NAMES))

.PHONY: demo
demo:
	@vhs -o assets/demo.gif examples/banner/banner.tape > /dev/null

.PHONY: demos
demos: $(TAPE_TARGETS) tape-spinner-styles tape-json tape-styles
	@cp $(CLOG_DOCS_DIR)/assets/demo.gif assets/demo.gif

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

.PHONY: tape-json
tape-json:
	@mkdir -p $(CLOG_DOCS_DIR)/assets
	@vhs examples/json/json.tape > /dev/null
	@mv json.png $(CLOG_DOCS_DIR)/assets/json.png

SPINNER_STYLE_PAGES := 1 2 3 4 5 6 7 8 9
SPINNER_STYLE_TARGETS := $(addprefix tape-spinner-styles-,$(SPINNER_STYLE_PAGES))

.PHONY: tape-spinner-styles $(SPINNER_STYLE_TARGETS)
tape-spinner-styles: $(SPINNER_STYLE_TARGETS)

$(SPINNER_STYLE_TARGETS): tape-spinner-styles-%:
	@mkdir -p $(CLOG_DOCS_DIR)/assets
	@sed "s/PAGE/$*/" examples/spinner/styles.tape > /tmp/spinner-styles-$*.tape
	@vhs -o $(CLOG_DOCS_DIR)/assets/spinner-styles-$*.gif /tmp/spinner-styles-$*.tape > /dev/null

.PHONY: tape-styles
tape-styles:
	@mkdir -p $(CLOG_DOCS_DIR)/assets
	@vhs examples/styles/styles.tape > /dev/null
	@mv styles.png $(CLOG_DOCS_DIR)/assets/styles.png

.PHONY: $(TAPE_TARGETS)
$(TAPE_TARGETS): tape-%:
	@mkdir -p $(CLOG_DOCS_DIR)/assets
	@vhs -o $(CLOG_DOCS_DIR)/assets/$(GIF_$*) examples/$(TAPE_$*).tape > /dev/null

.PHONY: test
test:
	@$(GO) test -timeout 2m -race ./...

.PHONY: update
update:
	@$(GO) get $(GO_TOOLS) $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	@$(GO) mod tidy
