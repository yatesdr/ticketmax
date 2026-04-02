# thermal-printer Makefile
# Cross-platform builds for ESC/POS thermal printer CLI

BINARY_NAME=thermal-printer
VERSION?=dev
BUILD_DIR=build

LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION)"

.PHONY: all clean linux windows macos build run test fmt vet help

# Default: build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Build for all platforms
all: clean linux windows macos

# Run tests
test:
	go test -v ./...

# Run the application
run:
	go run . $(ARGS)

# Linux builds
linux: linux-amd64 linux-arm64

linux-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .

linux-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

# Windows builds
windows: windows-amd64

windows-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

# macOS builds
macos: macos-amd64 macos-arm64

macos-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .

macos-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Format code
fmt:
	go fmt ./...

# Static analysis
vet:
	go vet ./...

# Show help
help:
	@echo "thermal-printer build targets:"
	@echo "  make build     - Build for current platform"
	@echo "  make all       - Build for all platforms (linux, windows, macos)"
	@echo "  make linux     - Build for Linux (amd64, arm64)"
	@echo "  make windows   - Build for Windows (amd64)"
	@echo "  make macos     - Build for macOS (amd64, arm64)"
	@echo "  make test      - Run tests"
	@echo "  make clean     - Remove build artifacts"
	@echo "  make fmt       - Format code"
	@echo "  make vet       - Run static analysis"
	@echo ""
	@echo "Override version: make build VERSION=1.2.3"
