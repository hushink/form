# Makefile for form library

GO ?= go
GOFMT ?= gofmt
LICENSE_EYE ?= license-eye
GOLANGCI_LINT ?= golangci-lint

COVERAGE_OUT := coverage.out
COVERAGE_HTML := coverage.html

.PHONY: all test test-race test-cover bench fmt fmt-check vet lint license-dep license-fix license-check clean check ci fuzz help tools deps

# Default target
all: fmt vet test ## Run fmt, vet, and test

# ============================================================================
# Testing
# ============================================================================

test: ## Run all tests
	$(GO) test -v ./...

test-race: ## Run tests with race detection
	$(GO) test -race -v ./...

test-cover: ## Run tests and generate coverage report
	$(GO) test -coverprofile=$(COVERAGE_OUT) ./...
	$(GO) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

bench: ## Run benchmarks
	$(GO) test -bench=. -benchmem ./...

# ============================================================================
# Code Quality
# ============================================================================

fmt: ## Format code
	$(GOFMT) -w .

fmt-check: ## Check code format (for CI)
	@if [ -n "$$($(GOFMT) -l .)" ]; then \
		echo "Code is not formatted. Run 'make fmt' to fix."; \
		$(GOFMT) -l .; \
		exit 1; \
	fi

vet: ## Run go vet
	$(GO) vet ./...

lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run ./...

# ============================================================================
# License
# ============================================================================

license-dep: ## Generate dependency license files
	$(LICENSE_EYE) dep resolve -l LICENSE

license-fix: ## Fix source file license headers
	$(LICENSE_EYE) header fix

license-check: ## Check source file license headers (for CI)
	$(LICENSE_EYE) header check

# ============================================================================
# Cleanup
# ============================================================================

clean: ## Clean generated files
	@rm -f $(COVERAGE_OUT) $(COVERAGE_HTML)
	@echo "Cleaned generated files."

# ============================================================================
# Combined Targets
# ============================================================================

check: fmt-check vet lint license-check test ## Run fmt-check, vet, lint, license-check, and test (for CI)

ci: fmt-check vet lint test-race fuzz ## Run the full CI suite locally

fuzz: ## Run short fuzz smoke tests
	$(GO) test -run '^$$' -fuzz=FuzzParseKeyPath -fuzztime=10s .
	$(GO) test -run '^$$' -fuzz=FuzzDecode -fuzztime=10s .

# ============================================================================
# Tools
# ============================================================================

tools: ## Install required tools (license-eye, golangci-lint)
	@echo Installing tools...
	$(GO) install github.com/apache/skywalking-eyes/cmd/license-eye@latest
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@echo Tools installed successfully.

deps: ## Download project dependencies
	$(GO) mod download
	@echo Dependencies downloaded successfully.
	@$(GO) mod tidy
	@echo Dependencies tidied successfully.

# ============================================================================
# Help
# ============================================================================

help: ## Show this help message
	@echo "Available targets:"
	@echo "  all             Run fmt, vet, and test"
	@echo "  bench           Run benchmarks"
	@echo "  check           Run fmt-check, vet, lint, license-check, and test (for CI)"
	@echo "  ci              Run fmt-check, vet, lint, race tests, and fuzz smoke tests"
	@echo "  clean           Clean generated files"
	@echo "  deps            Download project dependencies"
	@echo "  fmt             Format code"
	@echo "  fmt-check       Check code format (for CI)"
	@echo "  fuzz            Run short fuzz smoke tests"
	@echo "  help            Show this help message"
	@echo "  license-check   Check source file license headers (for CI)"
	@echo "  license-dep     Generate dependency license files"
	@echo "  license-fix     Fix source file license headers"
	@echo "  lint            Run golangci-lint"
	@echo "  test            Run all tests"
	@echo "  test-cover      Run tests and generate coverage report"
	@echo "  test-race       Run tests with race detection"
	@echo "  tools           Install required tools"
	@echo "  vet             Run go vet"
