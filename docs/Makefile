MDBOOK_VERSION ?= 0.5.2
HLJS_VERSION   ?= 11.11.1
HLJS_URL       := https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@$(HLJS_VERSION)/build
HLJS_LANGS     := go bash json

MDBOOK ?= mdbook

.PHONY: all
all: serve

.PHONY: build
build:
	@$(MDBOOK) build

.PHONY: clean
clean:
	@$(MDBOOK) clean

.PHONY: fmt
fmt:
	@rumdl fmt --quiet

.PHONY: serve
serve:
	@$(MDBOOK) serve --open

.PHONY: update
update: update-mdbook update-hljs

.PHONY: update-mdbook
update-mdbook:
	@curl -sL "https://github.com/rust-lang/mdBook/releases/download/v$(MDBOOK_VERSION)/mdbook-v$(MDBOOK_VERSION)-aarch64-apple-darwin.tar.gz" | tar xz -C "$$(dirname "$$(which mdbook)")" mdbook
	@awk '/MDBOOK_VERSION:/{$$0 = "  MDBOOK_VERSION: \"$(MDBOOK_VERSION)\""}1' ../.github/workflows/docs.yaml > ../.github/workflows/docs.yaml.tmp && mv ../.github/workflows/docs.yaml.tmp ../.github/workflows/docs.yaml
	@mdbook --version

.PHONY: update-hljs
update-hljs:
	@curl -sL "$(HLJS_URL)/highlight.min.js" -o theme/highlight.js
	@$(foreach lang,$(HLJS_LANGS),curl -sL "$(HLJS_URL)/languages/$(lang).min.js" >> theme/highlight.js;)
	@echo "highlight.js $(HLJS_VERSION)"
