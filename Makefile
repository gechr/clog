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

TAPE_NAMES   := banner spinner pulse shimmer shimmer-directions bar bar-styles group
TAPE_TARGETS := $(addprefix tape-,$(TAPE_NAMES))
GROUP_TARGETS := tape-bar-styles tape-group tape-shimmer-directions

.PHONY: all
all: gen fmt lint test

.PHONY: banner
banner: tape-banner
	@cp $(DOCS_DIR)/assets/banner.gif assets/banner.gif

.PHONY: demos
demos: banner $(TAPE_TARGETS) tape-spinner-styles tape-json tape-styles

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

.PHONY: demo-binaries
demo-binaries:
	@$(GO) build -o .demo-bin/ $(addprefix $(EXAMPLES_DIR)/,banner bar group json pulse shimmer spinner styles)

.PHONY: tape-json tape-styles
tape-json tape-styles: tape-%: | demo-binaries
	@mkdir -p $(DOCS_DIR)/assets
	@reel run $(EXAMPLES_DIR)/$*/$*.tape -o $(DOCS_DIR)/assets/$*.png

SPINNER_STYLE_PAGES := 1 2 3 4 5 6 7 8 9
SPINNER_STYLE_TARGETS := $(addprefix tape-spinner-styles-,$(SPINNER_STYLE_PAGES))

.PHONY: tape-spinner-styles $(SPINNER_STYLE_TARGETS)
tape-spinner-styles: $(SPINNER_STYLE_TARGETS)

$(SPINNER_STYLE_TARGETS): tape-spinner-styles-%: | demo-binaries
	@mkdir -p $(DOCS_DIR)/assets
	@DEMO_PAGE=$* sh $(EXAMPLES_DIR)/render-group.sh $(EXAMPLES_DIR)/spinner/styles.tape $(DOCS_DIR)/assets/spinner-styles-$*.gif $(if $(filter 9,$*),2,1) --font-size 18

.PHONY: $(TAPE_TARGETS)
$(GROUP_TARGETS): tape-%: | demo-binaries
	@mkdir -p $(DOCS_DIR)/assets
	@sh $(EXAMPLES_DIR)/render-group.sh $(EXAMPLES_DIR)/$(TAPE_$*).tape $(DOCS_DIR)/assets/$*.gif 1 --hold-last=3s

$(filter-out $(GROUP_TARGETS),$(TAPE_TARGETS)): tape-%: | demo-binaries
	@mkdir -p $(DOCS_DIR)/assets
	@reel run $(EXAMPLES_DIR)/$(TAPE_$*).tape --hold-last=3s -o $(DOCS_DIR)/assets/$*.gif

.PHONY: test
test:
	@$(GO) test -timeout 2m -race ./...

.PHONY: update
update:
	@clover run
	@$(GO) get $(GO_TOOLS) $(shell $(GO) list -f '{{if not (or .Main .Indirect)}}{{.Path}}{{end}}' -m all)
	@$(GO) mod tidy
