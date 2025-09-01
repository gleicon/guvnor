# Guv'nor Makefile

.PHONY: test build clean help

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test

# Binary name
BINARY_NAME=build/guvnor
BINARY_ALIAS=build/gv

help: ## Display this help message
	@echo "Guv'nor Build & Test"
	@echo "===================="
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

test: ## Run all tests
	$(GOTEST) -v ./internal/...

build: ## Build the guvnor binary
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/guvnor
	@ln -s $(BINARY_NAME) $(BINARY_ALIAS)

clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME) $(BINARY_ALIAS)

.DEFAULT_GOAL := help
