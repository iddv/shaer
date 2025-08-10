# File Sharing App - Makefile

APP_NAME := file-sharing-app
VERSION := 1.0.0
BUILD_DIR := dist
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Default target
.PHONY: all
all: clean build

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# Build for all platforms
.PHONY: build
build: build-windows build-darwin build-linux

# Build for Windows
.PHONY: build-windows
build-windows:
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build \
		-ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe \
		./cmd/main.go

# Build for macOS
.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS (amd64)..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build \
		-ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 \
		./cmd/main.go
	@echo "Building for macOS (arm64)..."
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build \
		-ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 \
		./cmd/main.go

# Build for Linux
.PHONY: build-linux
build-linux:
	@echo "Building for Linux (amd64)..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build \
		-ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 \
		./cmd/main.go
	@echo "Building for Linux (arm64)..."
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build \
		-ldflags="$(LDFLAGS)" \
		-o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 \
		./cmd/main.go

# Build for current platform only
.PHONY: build-local
build-local:
	@echo "Building for current platform..."
	@go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/main.go

# Generate checksums
.PHONY: checksums
checksums:
	@echo "Generating checksums..."
	@cd $(BUILD_DIR) && sha256sum * > checksums.txt

# Package for distribution
.PHONY: package
package: build checksums package-windows package-darwin package-linux

# Package for Windows
.PHONY: package-windows
package-windows:
	@echo "Packaging for Windows..."
	@mkdir -p $(BUILD_DIR)/packages
	@cp $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(BUILD_DIR)/packages/
	@cp README.md $(BUILD_DIR)/packages/
	@cp LICENSE $(BUILD_DIR)/packages/ 2>/dev/null || echo "LICENSE file not found, skipping..."
	@cd $(BUILD_DIR)/packages && zip -r ../$(APP_NAME)-windows-amd64.zip .
	@rm -rf $(BUILD_DIR)/packages

# Package for macOS
.PHONY: package-darwin
package-darwin:
	@echo "Packaging for macOS..."
	@mkdir -p $(BUILD_DIR)/packages
	@cp $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(BUILD_DIR)/packages/
	@cp $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(BUILD_DIR)/packages/
	@cp README.md $(BUILD_DIR)/packages/
	@cp LICENSE $(BUILD_DIR)/packages/ 2>/dev/null || echo "LICENSE file not found, skipping..."
	@cd $(BUILD_DIR)/packages && tar -czf ../$(APP_NAME)-darwin.tar.gz .
	@rm -rf $(BUILD_DIR)/packages

# Package for Linux
.PHONY: package-linux
package-linux:
	@echo "Packaging for Linux..."
	@mkdir -p $(BUILD_DIR)/packages
	@cp $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(BUILD_DIR)/packages/
	@cp $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(BUILD_DIR)/packages/
	@cp README.md $(BUILD_DIR)/packages/
	@cp LICENSE $(BUILD_DIR)/packages/ 2>/dev/null || echo "LICENSE file not found, skipping..."
	@cd $(BUILD_DIR)/packages && tar -czf ../$(APP_NAME)-linux.tar.gz .
	@rm -rf $(BUILD_DIR)/packages

# Test the build
.PHONY: test-build
test-build: build-local
	@echo "Testing local build..."
	@$(BUILD_DIR)/$(APP_NAME) --version 2>/dev/null || echo "Version flag not implemented yet"

# Install locally (Linux/macOS)
.PHONY: install
install: build-local
	@echo "Installing $(APP_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(APP_NAME)
	@echo "Installation complete!"

# Show help
.PHONY: help
help:
	@echo "File Sharing App - Build System"
	@echo "================================"
	@echo ""
	@echo "Available targets:"
	@echo "  all           - Clean and build for all platforms"
	@echo "  clean         - Clean build artifacts"
	@echo "  build         - Build for all platforms"
	@echo "  build-windows - Build for Windows"
	@echo "  build-darwin  - Build for macOS"
	@echo "  build-linux   - Build for Linux"
	@echo "  build-local   - Build for current platform"
	@echo "  checksums     - Generate checksums for built binaries"
	@echo "  package       - Package for distribution"
	@echo "  test-build    - Test the local build"
	@echo "  install       - Install locally (Linux/macOS)"
	@echo "  help          - Show this help message"