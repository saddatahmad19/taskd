## taskd Makefile
## ─────────────
## Run `make help` for a full list of targets.

.PHONY: build install test lint clean run-add run-list run-version snapshot release-dry-run help

BINARY     := taskd
OUTPUT_DIR := bin
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X github.com/saddatahmad19/taskd/internal/commands.Version=$(VERSION)"
MAIN       := ./cmd/taskd

build: ## Build the binary to ./bin/taskd
	@mkdir -p $(OUTPUT_DIR)
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY) $(MAIN)
	@echo "Built: $(OUTPUT_DIR)/$(BINARY)  (version=$(VERSION))"

install: ## Install to $$GOPATH/bin
	go install $(LDFLAGS) $(MAIN)
	@echo "Installed: $(shell go env GOPATH)/bin/$(BINARY)"

test: ## Run all tests
	go test ./... -v -count=1

lint: ## Run golangci-lint
	@which golangci-lint >/dev/null 2>&1 || \
		(echo "golangci-lint not found. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

clean: ## Remove build artefacts
	rm -rf $(OUTPUT_DIR)

run-add: build ## Quick smoke-test: taskd add
	./$(OUTPUT_DIR)/$(BINARY) add

run-list: build ## Quick smoke-test: taskd list
	./$(OUTPUT_DIR)/$(BINARY) list

run-version: build ## Quick smoke-test: taskd version
	./$(OUTPUT_DIR)/$(BINARY) version

snapshot: ## Build a local release snapshot (no publish, requires goreleaser)
	@which goreleaser >/dev/null 2>&1 || \
		(echo "goreleaser not found. Install: https://goreleaser.com/install/" && exit 1)
	goreleaser release --snapshot --clean

release-dry-run: ## Validate the .goreleaser.yaml config without publishing
	@which goreleaser >/dev/null 2>&1 || \
		(echo "goreleaser not found. Install: https://goreleaser.com/install/" && exit 1)
	goreleaser check

help:
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
