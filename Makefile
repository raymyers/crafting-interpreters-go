# Makefile for codecrafters-interpreter-go

# Variables
BINARY_NAME=interpreter
EYG_BINARY_NAME=eyg-interpreter
BUILD_DIR=build
APP_DIR=app
EYG_DIR=eyg-interpreter
GO_FILES=$(wildcard $(APP_DIR)/*.go)
EYG_GO_FILES=$(wildcard $(EYG_DIR)/*.go)

# Default target
.DEFAULT_GOAL := build-all

# Build all applications
.PHONY: build-all
build-all: build build-eyg

# Build the main Lox interpreter
.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME)

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(APP_DIR)/*.go

# Build the EYG interpreter
.PHONY: build-eyg
build-eyg: $(BUILD_DIR)/$(EYG_BINARY_NAME)

$(BUILD_DIR)/$(EYG_BINARY_NAME): $(EYG_GO_FILES)
	mkdir -p $(BUILD_DIR)
	cd $(EYG_DIR) && go build -o ../$(BUILD_DIR)/$(EYG_BINARY_NAME) .

# Create build directory
$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# Run all tests
.PHONY: test-all
test-all: test test-eyg

# Run main interpreter tests
.PHONY: test
test:
	go test ./$(APP_DIR)

# Run EYG interpreter tests
.PHONY: test-eyg
test-eyg:
	cd $(EYG_DIR) && go test

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	go test ./$(APP_DIR) -cover

# Run EYG tests with coverage
.PHONY: test-eyg-coverage
test-eyg-coverage:
	cd $(EYG_DIR) && go test -cover

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

# Run the main Lox interpreter (requires arguments)
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Run the EYG interpreter
.PHONY: run-eyg
run-eyg: build-eyg
	./$(BUILD_DIR)/$(EYG_BINARY_NAME)

# Install dependencies for all modules
.PHONY: deps
deps:
	go mod download
	go mod tidy
	cd $(EYG_DIR) && go mod download && go mod tidy

# Format all code
.PHONY: fmt
fmt:
	go fmt ./$(APP_DIR)/...
	cd $(EYG_DIR) && go fmt .

# Lint all code (requires golangci-lint)
.PHONY: lint
lint:
	golangci-lint run ./$(APP_DIR)/...
	cd $(EYG_DIR) && golangci-lint run .

# Vet all code
.PHONY: vet
vet:
	go vet ./$(APP_DIR)/...
	cd $(EYG_DIR) && go vet .

# Check code quality for all modules (fmt, vet, test)
.PHONY: check
check: fmt vet test-all

# Build for production (optimized)
.PHONY: build-prod
build-prod: build-prod-lox build-prod-eyg

# Build main interpreter for production
.PHONY: build-prod-lox
build-prod-lox: $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) $(APP_DIR)/*.go

# Build EYG interpreter for production
.PHONY: build-prod-eyg
build-prod-eyg: $(BUILD_DIR)
	cd $(EYG_DIR) && CGO_ENABLED=0 go build -ldflags="-w -s" -o ../$(BUILD_DIR)/$(EYG_BINARY_NAME) .

# Help target
.PHONY: help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build-all    - Build both interpreters (default)"
	@echo "  build        - Build the main Lox interpreter"
	@echo "  build-eyg    - Build the EYG interpreter"
	@echo "  build-prod   - Build both interpreters optimized for production"
	@echo ""
	@echo "Test targets:"
	@echo "  test-all     - Run all tests"
	@echo "  test         - Run main interpreter tests"
	@echo "  test-eyg     - Run EYG interpreter tests"
	@echo "  test-coverage- Run main interpreter tests with coverage"
	@echo "  test-eyg-coverage - Run EYG interpreter tests with coverage"
	@echo ""
	@echo "Run targets:"
	@echo "  run          - Run the main Lox interpreter (use ARGS='...' for arguments)"
	@echo "  run-eyg      - Run the EYG interpreter"
	@echo ""
	@echo "Development targets:"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Install/update dependencies for all modules"
	@echo "  fmt          - Format all code"
	@echo "  lint         - Lint all code (requires golangci-lint)"
	@echo "  vet          - Vet all code"
	@echo "  check        - Run fmt, vet, and test-all"
	@echo "  help         - Show this help message"