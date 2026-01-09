# Makefile for go-pve-vdi
# Build cross-platform releases

APP_NAME=go-pve-vdi
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DIR=build
LDFLAGS=-ldflags "-s -w"

.PHONY: all clean linux linux-amd64 linux-386 run help

# Default target
all: linux

# Build all Linux releases
linux: linux-amd64 linux-386

# Build Linux 64-bit
linux-amd64:
	@echo "Building Linux AMD64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)-linux-amd64"

# Build Linux 32-bit
linux-386:
	@echo "Building Linux 386..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-386
	@echo "Built: $(BUILD_DIR)/$(APP_NAME)-linux-386"

# Build for current platform
build:
	@echo "Building for current platform..."
	go build $(LDFLAGS) -o $(APP_NAME)
	@echo "Built: $(APP_NAME)"

# Run the application (current platform)
run: build
	./$(APP_NAME) -cli -config_location ./vdiclient.ini

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -f $(APP_NAME)
	@echo "Clean complete"

# Show help
help:
	@echo "Makefile for $(APP_NAME)"
	@echo ""
	@echo "Usage:"
	@echo "  make              Build all Linux releases (32-bit and 64-bit)"
	@echo "  make linux        Build all Linux releases"
	@echo "  make linux-amd64  Build Linux 64-bit"
	@echo "  make linux-386    Build Linux 32-bit"
	@echo "  make build        Build for current platform"
	@echo "  make run          Build and run (CLI mode)"
	@echo "  make clean        Remove build artifacts"
	@echo "  make help         Show this help message"
