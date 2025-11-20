# Makefile for Backtide backup utility

.PHONY: build test clean install install-man version help

# Default target
help:
	@echo "Backtide Development Build Targets:"
	@echo "  build       - Build development binary"
	@echo "  test        - Run tests"
	@echo "  clean       - Remove build artifacts"
	@echo "  install     - Install to system"
	@echo "  install-man - Install man page to system"
	@echo "  version     - Show current version"
	@echo "  help        - Show this help"

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
install: build install-man
	@echo "Installing Backtide to /usr/local/bin..."
	sudo mv backtide /usr/local/bin/

# Install man page (requires sudo)
install-man:
	@echo "Installing man page to /usr/local/share/man/man1/..."
	@if [ -f man/backtide.1 ]; then \
		sudo mkdir -p /usr/local/share/man/man1/; \
		sudo cp man/backtide.1 /usr/local/share/man/man1/; \
		sudo mandb >/dev/null 2>&1 || true; \
		echo "Man page installed successfully"; \
	else \
		echo "Warning: man/backtide.1 not found - skipping man page installation"; \
	fi



# Show current version
version:
	@./backtide version 2>/dev/null || echo "Not built yet"

# Production build target
build-linux:
	@echo "Building production binary for Linux..."
	GOOS=linux GOARCH=amd64 go build -o backtide-linux-amd64

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
