# Makefile for AddrMint

# Variables
BINARY_NAME=addrmint
GO=go
BUILD_DIR=build
MAIN_FILE=main.go
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
GOARCH=$(shell go env GOARCH)
GOOS=$(shell go env GOOS)

# Default build flags
GOFLAGS=-v

# Default target
.PHONY: all
all: clean build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Build complete. Binary available at $(BUILD_DIR)/$(BINARY_NAME)"

# Build with optimizations for production
.PHONY: build-prod
build-prod:
	@echo "Building optimized binary for production..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -v -ldflags "-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_FILE)
	@echo "Production build complete. Binary available at $(BUILD_DIR)/$(BINARY_NAME)"

# Cross-compile for multiple platforms
.PHONY: build-all
build-all: build-linux build-windows build-darwin

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)/linux
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/linux/$(BINARY_NAME) $(MAIN_FILE)

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)/windows
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/windows/$(BINARY_NAME).exe $(MAIN_FILE)

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)/darwin
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/darwin/$(BINARY_NAME) $(MAIN_FILE)

# Run the application
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) --network ethereum --count 10

# Install dependencies
.PHONY: deps
deps:
	$(GO) mod download
	$(GO) mod tidy

# Verify dependencies
.PHONY: verify
verify:
	$(GO) mod verify

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."

# Format the code
.PHONY: fmt
fmt:
	$(GO) fmt ./...

# Run tests
.PHONY: test
test:
	$(GO) test -v ./...

# Check for lint errors
.PHONY: lint
lint:
	$(GO) vet ./...
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "golint not installed. Run: go install golang.org/x/lint/golint@latest"; \
	fi

# CI pipeline target for continuous integration
.PHONY: ci
ci: deps verify fmt build test lint
	@echo "CI pipeline completed successfully."

# Build and install the binary
.PHONY: install
install:
	$(GO) install $(LDFLAGS) ./...

# Generate example outputs
.PHONY: examples
examples:
	@echo "Generating example outputs..."
	@mkdir -p examples
	./$(BUILD_DIR)/$(BINARY_NAME) --network ethereum --count 100 --seed 42 > examples/ethereum_addresses.txt
	./$(BUILD_DIR)/$(BINARY_NAME) --network bitcoin --count 100 --seed 42 > examples/bitcoin_addresses.txt
	./$(BUILD_DIR)/$(BINARY_NAME) --network solana --count 100 --seed 42 > examples/solana_addresses.txt
	./$(BUILD_DIR)/$(BINARY_NAME) --network ethereum --count 100 --seed 42 --generate-hash > examples/ethereum_addresses_with_hash.txt
	@echo "Example outputs generated in examples/ directory."

# Help target for command documentation
.PHONY: help
help:
	@echo "AddrMint - Makefile targets:"
	@echo "  all           - Clean and build the project"
	@echo "  build         - Build the binary"
	@echo "  build-prod    - Build optimized binary for production"
	@echo "  build-all     - Cross-compile for Linux, Windows and macOS"
	@echo "  run           - Build and run with sample parameters"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  verify        - Verify dependencies"
	@echo "  clean         - Remove build artifacts"
	@echo "  fmt           - Format code"
	@echo "  test          - Run tests"
	@echo "  lint          - Run linter"
	@echo "  ci            - Run continuous integration pipeline (deps, verify, fmt, build, test, lint)"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  examples      - Generate example output files"
	@echo "  help          - Show this help message" 