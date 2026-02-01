# Fuzzyrepo Makefile
# Usage: make help

BINARY_NAME=fuzzyrepo
BUILD_DIR=./bin
CONFIG_DIR=$(HOME)/.config/fuzzyrepo
CACHE_DIR=$(HOME)/.local/share/fuzzyrepo

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOVET=$(GOCMD) vet
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod

.PHONY: help build build-local run clean clean-all clean-config clean-cache clean-build reset vet test tidy check install

## help: Show this help message
help:
	@echo "Fuzzyrepo Development Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build targets:"
	@echo "  build        Build the binary to ./bin/fuzzyrepo"
	@echo "  build-local  Build and install to /usr/local/bin"
	@echo "  run          Build and run fuzzyrepo"
	@echo "  install      Install to GOPATH/bin"
	@echo ""
	@echo "Test targets:"
	@echo "  test         Run Go tests"
	@echo "  vet          Run go vet"
	@echo "  check        Run vet + build (pre-commit check)"
	@echo ""
	@echo "Clean targets:"
	@echo "  clean        Remove build artifacts"
	@echo "  clean-config Remove config file (~/.config/fuzzyrepo/)"
	@echo "  clean-cache  Remove cache files (~/.local/share/fuzzyrepo/)"
	@echo "  clean-all    Remove build + config + cache"
	@echo "  reset        Full reset: clean-all (simulates fresh install)"
	@echo ""
	@echo "Other targets:"
	@echo "  tidy         Run go mod tidy"
	@echo "  show-config  Show current config file"
	@echo "  show-cache   Show cache directory contents"
	@echo "  show-meta    Show metadata file"
	@echo ""
	@echo "Data locations:"
	@echo "  Config: $(CONFIG_DIR)"
	@echo "  Cache:  $(CACHE_DIR)"
	@echo ""
	@echo "Get Repository path to clipboard"
	@echo "  copy-repopath Copy repopath"

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-local: Build and copy to /usr/local/bin
build-local: build
	@echo "Installing to /usr/local/bin/$(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installed!"

## run: Build and run
run: build
	@$(BUILD_DIR)/$(BINARY_NAME)

## install: Install to GOPATH/bin
install:
	@echo "Installing to GOPATH/bin..."
	$(GOCMD) install .
	@echo "Installed!"

## test: Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## check: Run vet and build (pre-commit check)
check: vet build
	@echo "All checks passed!"

## tidy: Run go mod tidy
tidy:
	@echo "Running go mod tidy..."
	$(GOMOD) tidy

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)
	@echo "Build artifacts cleaned."

## clean-config: Remove config directory
clean-config:
	@echo "Removing config directory: $(CONFIG_DIR)"
	@rm -rf $(CONFIG_DIR)
	@echo "Config removed."

## clean-cache: Remove cache directory
clean-cache:
	@echo "Removing cache directory: $(CACHE_DIR)"
	@rm -rf $(CACHE_DIR)
	@echo "Cache removed."

## clean-all: Remove build + config + cache
clean-all: clean clean-config clean-cache
	@echo "All cleaned!"

## reset: Full reset (simulates fresh install)
reset: clean-all
	@echo ""
	@echo "=========================================="
	@echo "Full reset complete!"
	@echo "Next 'make run' will trigger first-run experience."
	@echo "=========================================="

## show-config: Display current config file
show-config:
	@echo "Config file: $(CONFIG_DIR)/config.yaml"
	@echo "---"
	@cat $(CONFIG_DIR)/config.yaml 2>/dev/null || echo "(no config file found)"

## show-cache: Display cache directory contents
show-cache:
	@echo "Cache directory: $(CACHE_DIR)"
	@echo "---"
	@ls -la $(CACHE_DIR) 2>/dev/null || echo "(no cache directory found)"
	@echo ""
	@echo "Repo count in cache:"
	@cat $(CACHE_DIR)/repos.json 2>/dev/null | grep -c '"full_name"' || echo "0"

## show-meta: Display metadata file
show-meta:
	@echo "Metadata file: $(CACHE_DIR)/metadata.json"
	@echo "---"
	@watch -n 1 cat $(CACHE_DIR)/metadata.json 2>/dev/null | python3 -m json.tool 2>/dev/null || cat $(CACHE_DIR)/metadata.json 2>/dev/null || echo "(no metadata file found)"

## Add repopath to clipboard
copy-repopath: 
	@echo "$(HOME)/Documents/Repositories" | pbcopy
