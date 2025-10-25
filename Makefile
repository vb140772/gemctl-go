# Makefile for gemctl-go

# Variables
BINARY_NAME=gemctl
VERSION=1.0.0
BUILD_DIR=build
GO_VERSION=1.21

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME) .

# Build for multiple platforms
.PHONY: build-all
build-all:
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=windows GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

# Install the binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install -ldflags "-X main.version=$(VERSION)" .

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -race ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Run vet
.PHONY: vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	go clean

# Run the CLI with help
.PHONY: help
help: build
	@echo "Running CLI help..."
	./$(BUILD_DIR)/$(BINARY_NAME) --help

# Run the CLI with engines help
.PHONY: help-engines
help-engines: build
	@echo "Running engines help..."
	./$(BUILD_DIR)/$(BINARY_NAME) engines --help

# Run the CLI with data-stores help
.PHONY: help-datastores
help-datastores: build
	@echo "Running data-stores help..."
	./$(BUILD_DIR)/$(BINARY_NAME) data-stores --help

# Check dependencies
.PHONY: deps
deps:
	@echo "Checking dependencies..."
	go mod tidy
	go mod verify

# Update dependencies
.PHONY: deps-update
deps-update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Run the CLI (for testing)
.PHONY: run
run: build
	@echo "Running CLI..."
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Development setup
.PHONY: dev-setup
dev-setup:
	@echo "Setting up development environment..."
	go mod download
	go mod tidy

# Check if required tools are installed
.PHONY: check-tools
check-tools:
	@echo "Checking required tools..."
	@command -v go >/dev/null 2>&1 || { echo "Go is required but not installed. Aborting." >&2; exit 1; }
	@go version | grep -q "go$(GO_VERSION)" || { echo "Go $(GO_VERSION) is required. Current version: $$(go version)"; exit 1; }

# Show version
.PHONY: version
version: build
	@echo "Version: $(VERSION)"
	@./$(BUILD_DIR)/$(BINARY_NAME) --version

# Create release packages
.PHONY: release
release: build-all
	@echo "Creating release packages..."
	@mkdir -p $(BUILD_DIR)/release
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	@cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	@cd $(BUILD_DIR) && zip release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

# Show all available targets
.PHONY: targets
targets:
	@echo "Available targets:"
	@echo "  build          - Build the binary"
	@echo "  build-all      - Build for all platforms"
	@echo "  install        - Install the binary"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  test-race      - Run tests with race detection"
	@echo "  fmt            - Format code"
	@echo "  lint           - Run linter"
	@echo "  vet            - Run go vet"
	@echo "  clean          - Clean build artifacts"
	@echo "  help           - Show CLI help"
	@echo "  help-engines   - Show engines help"
	@echo "  help-datastores - Show data-stores help"
	@echo "  deps           - Check dependencies"
	@echo "  deps-update    - Update dependencies"
	@echo "  dev-setup      - Setup development environment"
	@echo "  check-tools    - Check required tools"
	@echo "  version        - Show version"
	@echo "  release        - Create release packages"
	@echo "  targets        - Show this help"
