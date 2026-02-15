VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

.PHONY: build clean install test fmt lint help

build: ## Build the s0 binary
	go build $(LDFLAGS) -o bin/s0 ./cmd/s0

clean: ## Clean build artifacts
	rm -rf bin/

install: build ## Install s0 to /usr/local/bin
	cp bin/s0 /usr/local/bin/

test: ## Run tests
	go test -v ./...

fmt: ## Format code
	go fmt ./...

lint: ## Run linter
	golangci-lint run ./...

tidy: ## Tidy go modules
	go mod tidy

deps: ## Download dependencies
	go mod download

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
