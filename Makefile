# Guvnor Makefile

.PHONY: build clean test docs docs-serve docs-build install help

# Go build settings
BINARY_NAME=guvnor
BINARY_ALIAS=gv
BUILD_DIR=build
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Documentation settings
DOCS_DIR=docs
DOCS_BUILD_DIR=docs/site
DOCS_PORT=8000

build: ## Build the guvnor binary
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ./cmd/guvnor
	@ln -sf ${BINARY_NAME} ${BUILD_DIR}/${BINARY_ALIAS}
	@echo "Built ${BUILD_DIR}/${BINARY_NAME} and ${BUILD_DIR}/${BINARY_ALIAS}"

install: build ## Install to /usr/local/bin
	@echo "Installing ${BINARY_NAME} to /usr/local/bin..."
	@sudo cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	@sudo ln -sf /usr/local/bin/${BINARY_NAME} /usr/local/bin/${BINARY_ALIAS}
	@echo "Installed ${BINARY_NAME} and ${BINARY_ALIAS}"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf ${BUILD_DIR}
	@rm -rf ${DOCS_BUILD_DIR}

test: ## Run all tests
	@echo "Running tests..."
	go test -v ./internal/...

docs-serve: docs-build ## Build and serve documentation
	@echo "Serving documentation at http://localhost:${DOCS_PORT}"
	@cd ${DOCS_BUILD_DIR} && python3 -m http.server ${DOCS_PORT}

docs-build: ## Build documentation site
	@echo "Building documentation site..."
	@mkdir -p ${DOCS_BUILD_DIR}
	@./scripts/build-docs.sh ${DOCS_DIR} ${DOCS_BUILD_DIR}
	@echo "Documentation built in ${DOCS_BUILD_DIR}"

docs-clean: ## Clean documentation build
	@echo "Cleaning documentation build..."
	@rm -rf ${DOCS_BUILD_DIR}

docs: docs-clean docs-build ## Generate static documentation site
	@echo "Documentation ready at ${DOCS_BUILD_DIR}/index.html"

help: ## Display this help message
	@echo "Guvnor Build & Documentation"
	@echo "============================"
	@echo ""
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
