VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

.PHONY: build clean install test fmt lint check help set-version tag publish release tidy deps

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

check: build test lint ## Run all checks
	@printf "All checks passed!\n"

# Version for publishing (usage: make set-version v=0.1.0)
v ?=

# Set version by creating git tag
set-version:
ifndef v
	@echo "Error: version not specified. Usage: make set-version v=0.1.0"
	@exit 1
endif
	@echo "Creating tag v$(v)..."
	@git tag -a v$(v) -m "Release v$(v)"
	@echo "Tag v$(v) created successfully!"

# Create and push git tag (Go modules use git tags for versioning)
tag: set-version
ifndef v
	@echo "Error: version not specified. Usage: make tag v=0.1.0"
	@exit 1
endif
	@echo "Pushing tag v$(v) to origin..."
	@git push origin v$(v)
	@echo "Tag v$(v) pushed. Go proxy will index the new version automatically."

# Publish to Go proxy (just push the tag)
publish: check tag
	@echo "Published version v$(v) successfully!"
	@echo "Users can now use: go install github.com/sandbox0-ai/s0@v$(v)"

# Full release workflow
release: publish
	@echo "Release v$(v) completed!"

help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
