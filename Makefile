# Makefile for Backtide backup utility

.PHONY: build test clean install version help

# Default target
help:
	@echo "Backtide Development Build Targets:"
	@echo "  build     - Build development binary"
	@echo "  test      - Run tests"
	@echo "  clean     - Remove build artifacts"
	@echo "  install   - Install to system"
	@echo "  version   - Show current version"
	@echo "  help      - Show this help"

# Build the binary
build:
	@echo "Building Backtide..."
	go build -o backtide

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f backtide
	rm -f backtide-*
	rm -f *.test

# Install to system (requires sudo)
install: build
	@echo "Installing Backtide to /usr/local/bin..."
	sudo mv backtide /usr/local/bin/



# Show current version
version:
	@./backtide version 2>/dev/null || echo "Not built yet"

# Cross-compilation targets
build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 go build -o backtide-linux-amd64

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 go build -o backtide-darwin-amd64

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 go build -o backtide-windows-amd64.exe

# Build all platforms
build-all: build-linux build-darwin build-windows

# Development targets
dev: build
	@echo "Running development build..."
	./backtide --help

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Vet code
vet:
	@echo "Vetting code..."
	go vet ./...

# Lint and format
lint: fmt vet

# Dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod verify

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy
