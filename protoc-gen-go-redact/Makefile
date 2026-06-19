# Makefile for protoc-gen-redact
# Copyright 2020 Shivam Rathore
# Copyright 2025 Contributors
# Licensed under the Apache License, Version 2.0

.DEFAULT_GOAL := help

# Variables
BINARY_NAME := protoc-gen-redact
BIN_DIR := bin
GO := go
GOFLAGS := -v
LDFLAGS := -s -w
COVERAGE_DIR := coverage
COVERAGE_FILE := $(COVERAGE_DIR)/coverage.out
COVERAGE_HTML := $(COVERAGE_DIR)/coverage.html

# Buf configuration
BUF := buf
BUF_MODULE := buf.build/menta2k/redact

# Linting tools
GOLANGCI_LINT := golangci-lint
STATICCHECK := staticcheck

# Proto files
PROTO_FILES := $(shell find . -name "*.proto" -not -path "./vendor/*" -not -path "./testdata/*")
REDACT_PROTO := redact/v3/redact.proto
EXAMPLE_PROTOS := examples/user/pb/user.proto examples/tests/message.proto

# Go files
GO_FILES := $(shell find . -name "*.go" -not -path "./vendor/*" -not -path "./testdata/*")

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: install-tools
install-tools: ## Install required development tools
	@echo "Installing development tools..."
	@command -v buf > /dev/null || go install github.com/bufbuild/buf/cmd/buf@v1.47.2
	@command -v golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@command -v staticcheck > /dev/null || go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "✓ All tools installed"

.PHONY: deps
deps: ## Download Go module dependencies
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod verify
	@echo "✓ Dependencies downloaded"

.PHONY: tidy
tidy: ## Tidy Go module dependencies
	@echo "Tidying dependencies..."
	$(GO) mod tidy
	@echo "✓ Dependencies tidied"

##@ Building

.PHONY: build
build: clean-bin ## Build the protoc-gen-redact plugin
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "✓ Built $(BIN_DIR)/$(BINARY_NAME)"

.PHONY: build-debug
build-debug: clean-bin ## Build with debug symbols
	@echo "Building $(BINARY_NAME) with debug symbols..."
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "✓ Built $(BIN_DIR)/$(BINARY_NAME) (debug)"

.PHONY: install
install: ## Install the plugin to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(GOFLAGS) -ldflags "$(LDFLAGS)" .
	@echo "✓ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

##@ Testing

.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	$(GO) test -v -race -timeout 5m ./...
	@echo "✓ All tests passed"

.PHONY: test-short
test-short: ## Run short tests only
	@echo "Running short tests..."
	$(GO) test -v -short -race ./...
	@echo "✓ Short tests passed"

.PHONY: test-integration
test-integration: build ## Run integration tests
	@echo "Running integration tests..."
	$(GO) test -v -run TestIntegration ./...
	@echo "✓ Integration tests passed"

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@mkdir -p $(COVERAGE_DIR)
	$(GO) test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "✓ Coverage report: $(COVERAGE_HTML)"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | tail -1

.PHONY: test-bench
test-bench: ## Run benchmarks
	@echo "Running benchmarks..."
	$(GO) test -v -run=^$$ -bench=. -benchmem ./...

##@ Code Quality

.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	@gofmt -s -w $(GO_FILES)
	@$(GO) fmt ./...
	@echo "✓ Code formatted"

.PHONY: vet
vet: ## Run go vet
	@echo "Running go vet..."
	$(GO) vet ./...
	@echo "✓ go vet passed"

.PHONY: lint
lint: fmt vet ## Run all linters
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) run --timeout=5m ./...
	@echo "Running staticcheck..."
	$(STATICCHECK) -checks "all,-SA1019,-ST1000" ./...
	@echo "✓ All linters passed"

.PHONY: lint-fix
lint-fix: ## Run linters and auto-fix issues
	@echo "Running golangci-lint with auto-fix..."
	$(GOLANGCI_LINT) run --fix --timeout=5m ./...
	@echo "✓ Linting complete with fixes applied"

##@ Protocol Buffers

.PHONY: buf-lint
buf-lint: ## Lint proto files with buf
	@echo "Linting proto files..."
	$(BUF) lint --path redact/v3 || (echo "⚠ Buf linting found issues (examples/testdata excluded)" && exit 0)
	@echo "✓ Proto files linted"

.PHONY: buf-format
buf-format: ## Format proto files with buf
	@echo "Formatting proto files..."
	$(BUF) format -w
	@echo "✓ Proto files formatted"

.PHONY: buf-breaking
buf-breaking: ## Check for breaking changes in proto files
	@echo "Checking for breaking changes..."
	$(BUF) breaking --against '.git#branch=main'
	@echo "✓ No breaking changes detected"

.PHONY: buf-generate
buf-generate: ## Generate code from proto files using buf
	@echo "Generating code from proto files..."
	$(BUF) generate
	@echo "✓ Code generated"

.PHONY: buf-push
buf-push: buf-lint ## Push proto files to buf.build
	@echo "Pushing to $(BUF_MODULE)..."
	$(BUF) push
	@echo "✓ Pushed to $(BUF_MODULE)"

.PHONY: buf-push-tag
buf-push-tag: buf-lint ## Push proto files with a tag to buf.build
	@if [ -z "$(TAG)" ]; then echo "Error: TAG is required. Usage: make buf-push-tag TAG=v1.0.0"; exit 1; fi
	@echo "Pushing to $(BUF_MODULE) with tag $(TAG)..."
	$(BUF) push --tag $(TAG)
	@echo "✓ Pushed to $(BUF_MODULE) with tag $(TAG)"

##@ Proto Generation (legacy)

.PHONY: generate
generate: ## Generate Go code from redact.proto
	@echo "Generating code from $(REDACT_PROTO)..."
	protoc -I . \
		--go_out=. \
		--go_opt=paths=source_relative \
		$(REDACT_PROTO)
	@echo "✓ Generated code from $(REDACT_PROTO)"

.PHONY: generate-examples
generate-examples: build ## Generate code for examples
	@echo "Generating example code..."
	protoc -I . \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		--redact_out=. \
		--redact_opt=paths=source_relative \
		--plugin=$(BIN_DIR)/$(BINARY_NAME) \
		$(EXAMPLE_PROTOS)
	@echo "✓ Generated example code"

.PHONY: generate-testdata
generate-testdata: build ## Regenerate test data
	@echo "Regenerating test data..."
	protoc --experimental_allow_proto3_optional \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		-I=. \
		testdata/integration/test.proto
	protoc --experimental_allow_proto3_optional \
		--plugin=$(BIN_DIR)/$(BINARY_NAME) \
		--redact_out=. \
		--redact_opt=paths=source_relative \
		-I=. \
		testdata/integration/test.proto
	@echo "✓ Test data regenerated"

##@ Cleaning

.PHONY: clean-bin
clean-bin: ## Remove binary artifacts
	@echo "Cleaning binary artifacts..."
	@rm -rf $(BIN_DIR)
	@echo "✓ Binary artifacts cleaned"

.PHONY: clean-coverage
clean-coverage: ## Remove coverage reports
	@echo "Cleaning coverage reports..."
	@rm -rf $(COVERAGE_DIR)
	@echo "✓ Coverage reports cleaned"

.PHONY: clean-generated
clean-generated: ## Remove generated Go files (careful!)
	@echo "WARNING: This will remove generated .pb.go files!"
	@echo "Press Ctrl+C to cancel, or wait 3 seconds..."
	@sleep 3
	@find . -name "*.pb.go" -not -path "./vendor/*" -delete
	@find . -name "*.pb.redact.go" -not -path "./vendor/*" -delete
	@echo "✓ Generated files cleaned"

.PHONY: clean
clean: clean-bin clean-coverage ## Clean all artifacts
	@echo "✓ All artifacts cleaned"

##@ CI/CD

.PHONY: ci
ci: deps lint test ## Run CI pipeline (lint + test)
	@echo "✓ CI pipeline complete"

.PHONY: ci-full
ci-full: deps lint test-coverage buf-lint buf-breaking ## Run full CI pipeline
	@echo "✓ Full CI pipeline complete"

.PHONY: pre-commit
pre-commit: fmt lint test-short ## Run pre-commit checks
	@echo "✓ Pre-commit checks passed"

##@ Release

.PHONY: version
version: ## Display version information
	@echo "protoc-gen-redact version information:"
	@$(GO) version
	@echo "Module: $(shell $(GO) list -m)"

.PHONY: check-git-clean
check-git-clean: ## Check if git working directory is clean
	@echo "Checking git status..."
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Git working directory is not clean"; \
		git status --short; \
		exit 1; \
	fi
	@echo "✓ Git working directory is clean"

##@ Docker (future)

.PHONY: docker-build
docker-build: ## Build Docker image (placeholder)
	@echo "Docker support not yet implemented"

##@ Information

.PHONY: info
info: ## Display project information
	@echo "Project Information:"
	@echo "  Name:        protoc-gen-redact"
	@echo "  Module:      $(shell $(GO) list -m)"
	@echo "  Go Version:  $(shell $(GO) version)"
	@echo "  Buf Module:  $(BUF_MODULE)"
	@echo ""
	@echo "Directories:"
	@echo "  Binary:      $(BIN_DIR)"
	@echo "  Coverage:    $(COVERAGE_DIR)"
	@echo ""
	@echo "Tools:"
	@echo "  Go:          $(shell which go)"
	@echo "  Buf:         $(shell which buf 2>/dev/null || echo 'not installed')"
	@echo "  golangci-lint: $(shell which golangci-lint 2>/dev/null || echo 'not installed')"
	@echo "  staticcheck: $(shell which staticcheck 2>/dev/null || echo 'not installed')"

.PHONY: list
list: ## List all available targets
	@$(MAKE) -pRrq -f $(firstword $(MAKEFILE_LIST)) : 2>/dev/null | \
		awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | \
		sort | grep -E -v -e '^[^[:alnum:]]' -e '^$@$$'

# Special targets
.PHONY: all
all: deps build test ## Build and test everything
	@echo "✓ Build and test complete"
