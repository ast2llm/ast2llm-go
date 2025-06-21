# Variables
BINARY_NAME=mcp-server
LINTER=golangci-lint

.PHONY: check build test lint help

check: test lint

# Build the application binary
build:
	@echo "Building binary..."
	go build -o $(BINARY_NAME) ./cmd/server

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run linter
# You might need to install golangci-lint first:
# go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
lint:
	@echo "Running linter..."
	@if ! command -v $(LINTER) &> /dev/null; then \
		echo "$(LINTER) is not installed. Please run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		exit 1; \
	fi
	$(LINTER) run ./...

# Default target
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@echo "  check    - Test and lint the project"
	@echo "  build  - Build the application binary '$(BINARY_NAME)'"
	@echo "  test   - Run all tests"
	@echo "  lint   - Run the linter (golangci-lint)"
	@echo "  help   - Show this help message"

.DEFAULT_GOAL := help 