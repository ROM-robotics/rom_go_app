.PHONY: build run clean dev fmt test

APP_NAME := rom_dynamics_web
BUILD_DIR := ./build

# Build the application
build:
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) .

# Run in development mode
run: build
	$(BUILD_DIR)/$(APP_NAME)

# Run directly with go run
dev:
	go run .

# Format code
fmt:
	gofmt -s -w .

# Run tests
test:
	go test ./...

# Tidy dependencies
tidy:
	go mod tidy

# Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)

# Cross-compile for ARM (Raspberry Pi / robot)
build-arm:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(APP_NAME)_arm64 .

# Build for current platform with optimizations
build-release:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) .
