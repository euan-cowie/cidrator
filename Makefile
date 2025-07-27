# Cidrator - Simple Developer Makefile
# Everything you need, nothing you don't

# Build info
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS = -ldflags "-X github.com/euan-cowie/cidrator/cmd.Version=$(VERSION) -X github.com/euan-cowie/cidrator/cmd.Commit=$(COMMIT) -X github.com/euan-cowie/cidrator/cmd.Date=$(DATE)"

# Colors for pretty output
GREEN = \033[0;32m
BLUE = \033[0;34m
YELLOW = \033[0;33m
NC = \033[0m

.DEFAULT_GOAL := help

# === ESSENTIAL COMMANDS ===

.PHONY: setup
setup: ## 🚀 First-time setup (run this once)
	@echo "$(BLUE)Setting up cidrator development...$(NC)"
	@./scripts/setup.sh

.PHONY: dev
dev: build test-quick ## 🛠️ Quick development loop (build + test)
	@echo "$(GREEN)✅ Ready! Try: make run ARGS=\"cidr explain 192.168.1.0/24\"$(NC)"

.PHONY: build
build: ## 🔨 Build the binary
	@echo "$(BLUE)Building cidrator...$(NC)"
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/cidrator .
	@echo "$(GREEN)✅ Built: bin/cidrator$(NC)"

.PHONY: test
test: ## 🧪 Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -race ./...

.PHONY: test-quick
test-quick: ## ⚡ Quick tests (no race detection)
	@go test ./...

.PHONY: run
run: build ## 🏃 Build and run (use: make run ARGS="cidr explain 10.0.0.0/8")
	@./bin/cidrator $(ARGS)

.PHONY: check
check: fmt vet test lint-if-available ## ✅ Full quality check (run before PR)
	@echo "$(GREEN)✅ All checks passed!$(NC)"

# === QUALITY TOOLS ===

.PHONY: fmt
fmt: ## 📝 Format code
	@go fmt ./...

.PHONY: vet
vet: ## 🔍 Check for issues
	@go vet ./...

.PHONY: lint
lint: ## 🔍 Run linter (requires golangci-lint)
	@golangci-lint run

.PHONY: lint-if-available
lint-if-available: ## 🔍 Run linter if available
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(BLUE)Running linter...$(NC)"; \
		if golangci-lint run; then \
			echo "$(GREEN)✅ Linting passed$(NC)"; \
		else \
			echo "$(RED)❌ Linting failed$(NC)"; \
			exit 1; \
		fi \
	else \
		echo "$(YELLOW)⚠️ golangci-lint not found in PATH$(NC)"; \
		echo "$(BLUE)💡 Quick fixes:$(NC)"; \
		echo "  1. Restart your terminal"; \
		echo "  2. Or run: source ~/.zshrc"; \
		echo "  3. Or install: make install-tools"; \
		echo "$(BLUE)📖 See CONTRIBUTING.md for more help$(NC)"; \
	fi

# === OPTIONAL TOOLS ===

.PHONY: install-tools
install-tools: ## 🔧 Install development tools
	@echo "$(BLUE)Installing development tools...$(NC)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin latest
	@echo "$(GREEN)✅ Tools installed!$(NC)"

.PHONY: install-precommit
install-precommit: ## 🎣 Install pre-commit hooks
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "$(GREEN)✅ Pre-commit hooks installed!$(NC)"; \
	else \
		echo "$(YELLOW)Install pre-commit first: pip install pre-commit$(NC)"; \
	fi

# === BUILD VARIANTS ===

.PHONY: build-all
build-all: ## 🏗️ Build for all platforms
	@echo "$(BLUE)Building for all platforms...$(NC)"
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/cidrator-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/cidrator-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/cidrator-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/cidrator-windows-amd64.exe .
	@echo "$(GREEN)✅ All builds complete!$(NC)"

# === MAINTENANCE ===

.PHONY: clean
clean: ## 🧹 Clean build artifacts
	@rm -rf bin/ coverage.out coverage.html
	@go clean

.PHONY: deps
deps: ## 📦 Update dependencies
	@go mod download && go mod tidy

.PHONY: install
install: ## 📦 Install to $GOPATH/bin
	@go install $(LDFLAGS) .

# === EXAMPLES & TESTING ===

.PHONY: examples
examples: build ## 📋 Run example commands
	@echo "$(BLUE)Example commands:$(NC)"
	@echo "$(YELLOW)CIDR explanation:$(NC)"
	@./bin/cidrator cidr explain 192.168.1.0/24
	@echo "\n$(YELLOW)JSON output:$(NC)"
	@./bin/cidrator cidr explain 10.0.0.0/16 --format json
	@echo "\n$(YELLOW)Contains check:$(NC)"
	@./bin/cidrator cidr contains 192.168.1.0/24 192.168.1.100

.PHONY: test-integration
test-integration: build ## 🧪 Integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@./bin/cidrator version >/dev/null
	@./bin/cidrator --help >/dev/null
	@./bin/cidrator cidr explain 192.168.1.0/24 >/dev/null
	@echo "$(GREEN)✅ Integration tests passed$(NC)"

# === HELP ===

.PHONY: help
help: ## ❓ Show this help
	@echo "$(BLUE)Cidrator Development Commands$(NC)"
	@echo ""
	@echo "$(GREEN)🚀 Get Started:$(NC)"
	@echo "  make setup          One-time setup"
	@echo "  make dev            Quick build + test"
	@echo "  make run ARGS=\"...\" Test a command"
	@echo ""
	@echo "$(GREEN)📖 All Commands:$(NC)"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(GREEN)💡 Examples:$(NC)"
	@echo "  make run ARGS=\"cidr explain 10.0.0.0/8\""
	@echo "  make run ARGS=\"cidr contains 192.168.1.0/24 192.168.1.100\""
	@echo "  make check                    # Before submitting PR"
	@echo ""
	@echo "$(YELLOW)📖 More info: CONTRIBUTING.md$(NC)"
