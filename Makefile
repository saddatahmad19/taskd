## taskd Makefile
## ─────────────
## Targets:
##   make build     Build the binary to ./bin/taskd
##   make install   Install to $GOPATH/bin
##   make test      Run all tests
##   make lint      Run golangci-lint
##   make clean     Remove build artefacts
##   make run-add   Quick smoke-test: taskd add

.PHONY: build install test lint clean run-add run-list run-version help

BINARY     := taskd
OUTPUT_DIR := bin
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -ldflags "-X github.com/saddatahmad19/taskd/internal/commands.Version=$(VERSION)"
MAIN       := ./cmd/taskd

build:
	@mkdir -p $(OUTPUT_DIR)
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY) $(MAIN)
	@echo "Built: $(OUTPUT_DIR)/$(BINARY)  (version=$(VERSION))"

install:
	go install $(LDFLAGS) $(MAIN)
	@echo "Installed: $(shell go env GOPATH)/bin/$(BINARY)"

test:
	go test ./... -v -count=1

lint:
	@which golangci-lint >/dev/null 2>&1 || \
		(echo "golangci-lint not found. Install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

clean:
	rm -rf $(OUTPUT_DIR)

run-add: build
	./$(OUTPUT_DIR)/$(BINARY) add

run-list: build
	./$(OUTPUT_DIR)/$(BINARY) list

run-version: build
	./$(OUTPUT_DIR)/$(BINARY) version

help:
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
