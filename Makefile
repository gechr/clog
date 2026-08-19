GO                      ?= go
GO_TOOLS                ?= $(shell $(GO) list tool)

DOCS_DIR                ?= ./docs
EXAMPLES_DIR            ?= ./examples

TAPE_banner             := banner/banner
TAPE_spinner            := spinner/spinner
TAPE_pulse              := pulse/pulse
TAPE_shimmer            := shimmer/shimmer
TAPE_shimmer-directions := shimmer/directions
TAPE_bar                := bar/bar
TAPE_bar-styles         := bar/styles
TAPE_group              := group/group

GIF_banner              := banner.gif
GIF_spinner             := spinner.gif
GIF_pulse               := pulse.gif
GIF_shimmer             := shimmer.gif
GIF_shimmer-directions  := shimmer-directions.gif
GIF_bar                 := bar.gif
GIF_bar-styles          := bar-styles.gif
GIF_group               := group.gif

TAPE_NAMES   := banner spinner pulse shimmer shimmer-directions bar bar-styles group
TAPE_TARGETS := $(addprefix tape-,$(TAPE_NAMES))

.PHONY: all
all: gen fmt lint test

.PHONY: banner
banner:
	@vhs -o assets/banner.gif $(EXAMPLES_DIR)/banner/banner.tape > /dev/null
	@/bin/cp -f assets/banner.gif $(DOCS_DIR)/assets/banner.gif

.PHONY: demos
demos: $(TAPE_TARGETS) tape-spinner-styles tape-json tape-styles
	@cp $(DOCS_DIR)/assets/banner.gif assets/banner.gif

.PHONY: docs
docs:
	@$(MAKE) --no-print-directory -C $(DOCS_DIR)

.PHONY: examples
examples:
	@$(GO) run $(EXAMPLES_DIR)

.PHONY: fmt
fmt:
	@$(MAKE) --no-print-directory -C $(DOCS_DIR) fmt
	@clover format
	@rumdl fmt --quiet
	@$(GO) fix ./...
	@$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint fmt --enable=gci,golines,gofumpt
	@$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run --fix --enable-only tagalign
	@$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run --fix -c .golangci.ruleguard.yml

.PHONY: gen
gen:
	@$(GO) generate ./...

.PHONY: lint
lint:
	@$(GO) tool github.com/golangci/golangci-lint/v2/cmd/golangci-lint run

.PHONY: spinners
spinners:
	@$(GO) run $(EXAMPLES_DIR) -spinners="$(SPINNERS)"

.PHONY: tape-json
tape-json:
	@mkdir -p $(DOCS_DIR)/assets
	@vhs $(EXAMPLES_DIR)/json/json.tape > /dev/null
	@mv json.png $(DOCS_DIR)/assets/json.png

SPINNER_STYLE_PAGES := 1 2 3 4 5 6 7 8 9
SPINNER_STYLE_TARGETS := $(addprefix tape-spinner-styles-,$(SPINNER_STYLE_PAGES))

.PHONY: tape-spinner-styles $(SPINNER_STYLE_TARGETS)
tape-spinner-styles: $(SPINNER_STYLE_TARGETS)

$(SPINNER_STYLE_TARGETS): tape-spinner-styles-%:
	@mkdir -p $(DOCS_DIR)/assets
	@sed "s/PAGE/$*/" $(EXAMPLES_DIR)/spinner/styles.tape > /tmp/spinner-styles-$*.tape
	@vhs -o $(DOCS_DIR)/assets/spinner-styles-$*.gif /tmp/spinner-styles-$*.tape > /dev/null

.PHONY: tape-styles
tape-styles:
	@mkdir -p $(DOCS_DIR)/assets
	@vhs $(EXAMPLES_DIR)/styles/styles.tape > /dev/null
	@mv styles.png $(DOCS_DIR)/assets/styles.png

.PHONY: $(TAPE_TARGETS)
$(TAPE_TARGETS): tape-%:
	@mkdir -p $(DOCS_DIR)/assets
	@vhs -o $(DOCS_DIR)/assets/$(GIF_$*) $(EXAMPLES_DIR)/$(TAPE_$*).tape > /dev/null

.PHONY: test
test:
	@$(GO) test -timeout 2m -race ./...

.PHONY: update
update:
	@clover run
	@$(GO) get $(GO_TOOLS) $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	@$(GO) mod tidy
