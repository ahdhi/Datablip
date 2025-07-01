# Project configuration
PROJECT_NAME := datablip
MAIN_PATH := cmd/datablip
OUTPUT_DIR := bin
BUILD_DIR := build

# Version information
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)
BUILD_FLAGS := -ldflags "$(LDFLAGS)"

# Colors
COLOR_RESET := \033[0m
COLOR_BLUE := \033[0;34m
COLOR_GREEN := \033[0;32m

.PHONY: all build clean test run install cross-compile docker help

all: build

## Build the application
build:
	@echo -e "$(COLOR_BLUE)Building $(PROJECT_NAME)...$(COLOR_RESET)"
	@mkdir -p $(OUTPUT_DIR)
	@go build $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(PROJECT_NAME) ./$(MAIN_PATH)
	@bash scripts/post-build.sh --build-dir $(OUTPUT_DIR) --binary $(PROJECT_NAME)
	@echo -e "$(COLOR_GREEN)Build completed: $(OUTPUT_DIR)/$(PROJECT_NAME)$(COLOR_RESET)"

## Build with verbose output
build-verbose:
	@echo -e "$(COLOR_BLUE)Building $(PROJECT_NAME) (verbose)...$(COLOR_RESET)"
	@mkdir -p $(OUTPUT_DIR)
	@go build -v $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(PROJECT_NAME) ./$(MAIN_PATH)
	@bash scripts/post-build.sh --build-dir $(OUTPUT_DIR) --binary $(PROJECT_NAME) --verbose

## Cross-compile for multiple platforms
cross-compile:
	@echo -e "$(COLOR_BLUE)Cross-compiling $(PROJECT_NAME)...$(COLOR_RESET)"
	@bash scripts/cross-compile.sh

## Run tests
test:
	@echo -e "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	@go test -v ./...

## Run the application with default parameters
run: build
	@echo -e "$(COLOR_BLUE)Running $(PROJECT_NAME)...$(COLOR_RESET)"
	@./$(OUTPUT_DIR)/$(PROJECT_NAME) --help

## Install the application to GOPATH/bin
install:
	@echo -e "$(COLOR_BLUE)Installing $(PROJECT_NAME)...$(COLOR_RESET)"
	@go install $(BUILD_FLAGS) ./$(MAIN_PATH)

## Clean build artifacts
clean:
	@echo -e "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	@rm -rf $(OUTPUT_DIR)
	@rm -rf $(BUILD_DIR)/dist
	@go clean

## Format code
fmt:
	@echo -e "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@go fmt ./...

## Run linter
lint:
	@echo -e "$(COLOR_BLUE)Running linter...$(COLOR_RESET)"
	@golangci-lint run

## Tidy dependencies
tidy:
	@echo -e "$(COLOR_BLUE)Tidying dependencies...$(COLOR_RESET)"
	@go mod tidy

## Create release package
release: cross-compile
	@echo -e "$(COLOR_BLUE)Creating release package...$(COLOR_RESET)"
	@bash scripts/release.sh

## Build Docker image
docker:
	@echo -e "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	@docker build -t $(PROJECT_NAME):$(VERSION) -f build/docker/Dockerfile .

## Show help
help:
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)