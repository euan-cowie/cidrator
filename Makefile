VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse HEAD)
DATE ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GO ?= go
BINARY ?= bin/cidrator
LDFLAGS = -ldflags "-X github.com/euan-cowie/cidrator/cmd.Version=$(VERSION) -X github.com/euan-cowie/cidrator/cmd.Commit=$(COMMIT) -X github.com/euan-cowie/cidrator/cmd.Date=$(DATE)"

.DEFAULT_GOAL := help

.PHONY: setup
setup: ## Bootstrap the local development environment
	@./scripts/setup.sh

.PHONY: setup-tools
setup-tools: ## Bootstrap the project and install optional development tools
	@./scripts/setup.sh --install-tools

.PHONY: dev
dev: build test-quick ## Build and run the fast test suite

.PHONY: build
build: ## Build the cidrator binary
	@mkdir -p bin
	@$(GO) build $(LDFLAGS) -o $(BINARY) .

.PHONY: build-all
build-all: ## Build release binaries for supported targets
	@mkdir -p bin
	@GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/cidrator-linux-amd64 .
	@GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/cidrator-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/cidrator-darwin-arm64 .

.PHONY: test
test: ## Run the full test suite with race detection
	@$(GO) test -race ./...

.PHONY: test-quick
test-quick: ## Run the test suite without race detection
	@$(GO) test ./...

.PHONY: test-integration
test-integration: build ## Run basic CLI integration checks
	@./bin/cidrator version >/dev/null
	@./bin/cidrator --help >/dev/null
	@./bin/cidrator cidr explain 192.168.1.0/24 >/dev/null

.PHONY: test-lab
test-lab: build ## Run the Linux namespace MTU lab
	@bash ./test/labs/mtu-namespaces.sh ./bin/cidrator

.PHONY: test-lab-hops
test-lab-hops: build ## Run the Linux hop-by-hop MTU lab
	@bash ./test/labs/mtu-hop-by-hop.sh ./bin/cidrator

.PHONY: test-lab-plpmtud
test-lab-plpmtud: build ## Run the Linux ICMP black-hole PLPMTUD lab
	@bash ./test/labs/mtu-plpmtud-blackhole.sh ./bin/cidrator

.PHONY: fmt
fmt: ## Format Go source files
	@$(GO) fmt ./...

.PHONY: vet
vet: ## Run go vet
	@$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint
	@golangci-lint run

.PHONY: lint-if-available
lint-if-available: ## Run golangci-lint when it is installed
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found; skipping"; \
	fi

.PHONY: check
check: fmt vet test lint-if-available ## Run the standard local verification suite

.PHONY: install-tools
install-tools: ## Install optional development tools
	@./scripts/setup.sh --install-tools --skip-bootstrap

.PHONY: deps
deps: ## Download and tidy Go modules
	@$(GO) mod download
	@$(GO) mod tidy

.PHONY: install
install: ## Install cidrator into GOPATH/bin
	@$(GO) install $(LDFLAGS) .

.PHONY: run
run: build ## Build and run the binary (set ARGS="...")
	@./bin/cidrator $(ARGS)

.PHONY: examples
examples: build ## Run a small set of example commands
	@./bin/cidrator cidr explain 192.168.1.0/24
	@./bin/cidrator cidr explain 10.0.0.0/16 --format json
	@./bin/cidrator cidr contains 192.168.1.0/24 192.168.1.100

.PHONY: clean
clean: ## Remove build artifacts and cached coverage files
	@rm -rf bin/ coverage.out coverage.html
	@$(GO) clean

.PHONY: help
help: ## Show available make targets
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
