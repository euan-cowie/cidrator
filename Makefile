# Cidrator Development Makefile
# Simple, fast, reliable developer experience

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X github.com/euan-cowie/cidrator/cmd.Version=$(VERSION) -X github.com/euan-cowie/cidrator/cmd.Commit=$(COMMIT) -X github.com/euan-cowie/cidrator/cmd.Date=$(DATE)"

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod
BINARY_NAME = cidrator

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[0;33m
BLUE = \033[0;34m
NC = \033[0m # No Color

# Default target
.DEFAULT_GOAL := help

# === CORE DEVELOPMENT COMMANDS ===

.PHONY: setup
setup: ## 🚀 Initial setup for new contributors (one-time)
	@echo "$(BLUE)Setting up Cidrator development environment...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "$(GREEN)✅ Setup complete! Try: make dev$(NC)"

.PHONY: dev
dev: ## 🛠️  Quick development workflow (build + test + run)
	@echo "$(BLUE)Running development workflow...$(NC)"
	@$(MAKE) --no-print-directory build
	@$(MAKE) --no-print-directory test-quick
	@echo "$(GREEN)✅ Development check passed! Binary ready at bin/$(BINARY_NAME)$(NC)"

.PHONY: build
build: ## 🔨 Build the binary for current platform
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p bin
	@$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) .
	@echo "$(GREEN)✅ Built: bin/$(BINARY_NAME)$(NC)"

.PHONY: test-quick
test-quick: ## ⚡ Run tests (fast, no coverage)
	@echo "$(BLUE)Running tests...$(NC)"
	@$(GOTEST) -v ./...

.PHONY: test
test: ## 🧪 Run full test suite with coverage
	@echo "$(BLUE)Running full test suite...$(NC)"
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✅ Tests complete! Coverage: coverage.html$(NC)"

.PHONY: run
run: build ## 🏃 Build and run with arguments (use: make run ARGS="cidr explain 192.168.1.0/24")
	@./bin/$(BINARY_NAME) $(ARGS)

# === QUALITY CHECKS ===

.PHONY: check
check: ## ✅ Run all quality checks (recommended before committing)
	@echo "$(BLUE)Running all quality checks...$(NC)"
	@$(MAKE) --no-print-directory fmt
	@$(MAKE) --no-print-directory vet
	@$(MAKE) --no-print-directory test
	@$(MAKE) --no-print-directory lint-if-available
	@echo "$(GREEN)✅ All checks passed!$(NC)"

.PHONY: fmt
fmt: ## 📝 Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	@$(GOCMD) fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

.PHONY: vet
vet: ## 🔍 Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@$(GOCMD) vet ./...
	@echo "$(GREEN)✅ Vet checks passed$(NC)"

.PHONY: lint
lint: ## 🔍 Run golangci-lint (requires installation)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(BLUE)Running golangci-lint...$(NC)"; \
		golangci-lint run; \
		echo "$(GREEN)✅ Linting passed$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed. Install with: make install-tools$(NC)"; \
		exit 1; \
	fi

.PHONY: lint-if-available
lint-if-available: ## 🔍 Run golangci-lint if available, warn if not
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(BLUE)Running golangci-lint...$(NC)"; \
		golangci-lint run; \
		echo "$(GREEN)✅ Linting passed$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not installed (optional). Install with: make install-tools$(NC)"; \
	fi

# === BUILD VARIANTS ===

.PHONY: build-all
build-all: ## 🏗️  Build for all platforms
	@echo "$(BLUE)Building for all platforms...$(NC)"
	@$(MAKE) --no-print-directory build-linux
	@$(MAKE) --no-print-directory build-darwin
	@$(MAKE) --no-print-directory build-windows
	@echo "$(GREEN)✅ All platform builds complete!$(NC)"

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 .

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 .

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe .
	@GOOS=windows GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-arm64.exe .

# === TOOLS AND SETUP ===

.PHONY: install-tools
install-tools: ## 🔧 Install development tools (optional but recommended)
	@echo "$(BLUE)Installing development tools...$(NC)"
	@echo "Installing golangci-lint..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin latest
	@echo "Installing gosec..."
	@$(GOCMD) install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	@echo "$(GREEN)✅ Development tools installed!$(NC)"

.PHONY: install-precommit
install-precommit: ## 🎣 Install pre-commit hooks (optional)
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "$(BLUE)Installing pre-commit hooks...$(NC)"; \
		pre-commit install; \
		echo "$(GREEN)✅ Pre-commit hooks installed!$(NC)"; \
		echo "$(YELLOW)ℹ️  Run 'make remove-precommit' to disable$(NC)"; \
	else \
		echo "$(YELLOW)⚠️  pre-commit not installed. Install with: pip install pre-commit$(NC)"; \
		echo "$(YELLOW)   Then run: make install-precommit$(NC)"; \
	fi

.PHONY: remove-precommit
remove-precommit: ## 🗑️  Remove pre-commit hooks
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "$(BLUE)Removing pre-commit hooks...$(NC)"; \
		pre-commit uninstall; \
		echo "$(GREEN)✅ Pre-commit hooks removed$(NC)"; \
	else \
		echo "$(YELLOW)pre-commit not installed, nothing to remove$(NC)"; \
	fi

# === MAINTENANCE ===

.PHONY: deps
deps: ## 📦 Download and tidy dependencies
	@echo "$(BLUE)Managing dependencies...$(NC)"
	@$(GOMOD) download
	@$(GOMOD) tidy
	@echo "$(GREEN)✅ Dependencies updated$(NC)"

.PHONY: clean
clean: ## 🧹 Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@$(GOCLEAN)
	@rm -rf bin/
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)✅ Cleaned$(NC)"

.PHONY: install
install: ## 📦 Install binary to $GOPATH/bin
	@echo "$(BLUE)Installing to GOPATH...$(NC)"
	@$(GOCMD) install $(LDFLAGS) .
	@echo "$(GREEN)✅ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)$(NC)"

# === TESTING HELPERS ===

.PHONY: test-integration
test-integration: build ## 🧪 Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@./bin/$(BINARY_NAME) version
	@./bin/$(BINARY_NAME) --help >/dev/null
	@./bin/$(BINARY_NAME) cidr explain 192.168.1.0/24 >/dev/null
	@./bin/$(BINARY_NAME) cidr count 10.0.0.0/16 >/dev/null
	@echo "$(GREEN)✅ Integration tests passed$(NC)"

.PHONY: benchmark
benchmark: ## 📊 Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(NC)"
	@$(GOTEST) -bench=. -benchmem ./...

# === EXAMPLES ===

.PHONY: examples
examples: build ## 📋 Run example commands
	@echo "$(BLUE)Running example commands...$(NC)"
	@echo "\n$(YELLOW)Example: CIDR explanation$(NC)"
	@./bin/$(BINARY_NAME) cidr explain 192.168.1.0/24
	@echo "\n$(YELLOW)Example: JSON output$(NC)"
	@./bin/$(BINARY_NAME) cidr explain 10.0.0.0/16 --format json
	@echo "\n$(YELLOW)Example: IP contains check$(NC)"
	@./bin/$(BINARY_NAME) cidr contains 192.168.1.0/24 192.168.1.100

# === HELP ===

.PHONY: help
help: ## ❓ Show this help message
	@echo "$(BLUE)Cidrator Development Commands$(NC)"
	@echo ""
	@echo "$(GREEN)🚀 Getting Started:$(NC)"
	@echo "  make setup          - One-time setup for new contributors"
	@echo "  make dev            - Quick development workflow (build + test)"
	@echo ""
	@echo "$(GREEN)📖 Available Commands:$(NC)"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)💡 Quick Examples:$(NC)"
	@echo "  make dev                              # Build, test, and verify"
	@echo "  make run ARGS=\"cidr explain 10.0.0.0/8\"  # Test a command"
	@echo "  make check                            # Full quality checks"
	@echo "  make install-tools                    # Install optional tools"
	@echo ""
	@echo "$(GREEN)🔗 More Info:$(NC)"
	@echo "  📖 Contributing: CONTRIBUTING.md"
	@echo "  🐛 Issues: https://github.com/euan-cowie/cidrator/issues"
	@echo "" 