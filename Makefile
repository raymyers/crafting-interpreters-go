# Makefile for codecrafters-interpreter-go

# Variables
BINARY_NAME=interpreter
BUILD_DIR=build
APP_DIR=app
GO_FILES=$(wildcard $(APP_DIR)/*.go)

# Default target
.DEFAULT_GOAL := build

# Build the application
.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(APP_DIR)/*.go

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Run tests
.PHONY: test
test:
	go test ./$(APP_DIR) -v

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test ./$(APP_DIR) -v -cover

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# Run the application (requires arguments)
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Install dependencies
.PHONY: deps
deps:
	go mod download
	go mod tidy

# Format code
.PHONY: fmt
fmt:
	go fmt ./$(APP_DIR)/...

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run ./$(APP_DIR)/...

# Vet code
.PHONY: vet
vet:
	go vet ./$(APP_DIR)/...

# Check code quality (fmt, vet, test)
.PHONY: check
check: fmt vet test

# Build for production (optimized)
.PHONY: build-prod
build-prod: $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(APP_DIR)/*.go

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage"
	@echo "  run          - Run the application (use ARGS='...' for arguments)"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install/update dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Lint code (requires golangci-lint)"
	@echo "  vet          - Vet code"
	@echo "  check        - Run fmt, vet, and test"
	@echo "  build-prod   - Build optimized production binary"
	@echo "  help         - Show this help message"