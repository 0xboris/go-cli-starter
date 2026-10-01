BIN      := bin/tool
PKG      := example.com/tool
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo DEV)
DATE     ?= $(shell date -u +%Y-%m-%d)
LDFLAGS  := -X $(PKG)/internal/build.Version=$(VERSION) -X $(PKG)/internal/build.Date=$(DATE)

.PHONY: help build test lint docs completions check-docs clean

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary with version info
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/tool

test: ## Run all tests (unit + acceptance) with the race detector
	go test -race ./...

lint: ## Run linters
	go mod tidy -diff
	go vet ./...
	golangci-lint run

docs: ## Generate man pages and markdown docs from the command tree
	go run ./cmd/gen-docs --man-page --doc-path share/man/man1
	go run ./cmd/gen-docs --website --doc-path docs/commands

completions: build ## Generate shell completion scripts
	mkdir -p share/completions
	$(BIN) completion bash > share/completions/tool.bash
	$(BIN) completion zsh  > share/completions/_tool
	$(BIN) completion fish > share/completions/tool.fish

check-docs: docs ## Fail if generated docs are out of date (CI)
	git diff --exit-code -- docs/commands

clean: ## Remove build outputs
	rm -rf bin share
