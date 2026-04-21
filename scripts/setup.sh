#!/usr/bin/env bash
set -euo pipefail

install_tools=0
skip_bootstrap=0

usage() {
	cat <<'EOF'
Usage: ./scripts/setup.sh [options]

Bootstrap the local development environment for cidrator.

Options:
  --install-tools   Install optional development tools after bootstrap
  --skip-bootstrap  Skip module download, build, and test steps
  -h, --help        Show this help text
EOF
}

log() {
	printf '%s\n' "$*"
}

warn() {
	printf 'warning: %s\n' "$*" >&2
}

die() {
	printf 'error: %s\n' "$*" >&2
	exit 1
}

have_command() {
	command -v "$1" >/dev/null 2>&1
}

require_project_root() {
	if [[ ! -f "go.mod" ]] || [[ ! -f "main.go" ]]; then
		die "run this script from the repository root"
	fi
}

ensure_go_bin_on_path_notice() {
	local gobin
	gobin="$(go env GOPATH)/bin"

	if [[ ":$PATH:" != *":$gobin:"* ]]; then
		warn "Go tool binaries may not be on PATH: $gobin"
		warn "Add it to your shell profile if installed tools are not available in new shells."
	fi
}

bootstrap_project() {
	log "Downloading Go modules"
	go mod download
	go mod tidy

	log "Building cidrator"
	make build

	log "Running quick test pass"
	make test-quick
}

install_golangci_lint() {
	local gobin
	gobin="$(go env GOPATH)/bin"

	if have_command golangci-lint; then
		log "golangci-lint already installed"
		return
	fi

	if ! have_command curl; then
		warn "curl not found; skipping golangci-lint installation"
		return
	fi

	mkdir -p "$gobin"
	log "Installing golangci-lint"
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$gobin" latest
	ensure_go_bin_on_path_notice
}

install_pre_commit() {
	if have_command pre-commit; then
		log "pre-commit already installed"
	else
		if have_command pip3; then
			log "Installing pre-commit with pip3"
			pip3 install --user pre-commit
		elif have_command pip; then
			log "Installing pre-commit with pip"
			pip install --user pre-commit
		else
			warn "pip not found; skipping pre-commit installation"
			return
		fi
	fi

	if have_command pre-commit; then
		log "Installing pre-commit hooks"
		pre-commit install
	else
		warn "pre-commit installed but not available in the current shell"
		warn "Reload your shell or ensure your Python user bin directory is on PATH."
	fi
}

install_optional_tools() {
	log "Installing optional development tools"
	install_golangci_lint
	install_pre_commit
}

show_next_steps() {
	cat <<'EOF'

Setup complete.

Common commands:
  make build
  make test
  make check
  make run ARGS="cidr explain 192.168.1.0/24"

Additional guidance:
  - CONTRIBUTING.md explains contribution standards and review expectations.
  - DEVELOPMENT.md documents local workflows and Linux MTU lab requirements.
EOF
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	--install-tools)
		install_tools=1
		;;
	--skip-bootstrap)
		skip_bootstrap=1
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		die "unknown option: $1"
		;;
	esac
	shift
done

if ! have_command go; then
	die "Go is required. Install it from https://go.dev/dl/"
fi

if ! have_command make; then
	die "make is required"
fi

require_project_root
log "Using $(go version)"

if [[ "$skip_bootstrap" -eq 0 ]]; then
	bootstrap_project
fi

if [[ "$install_tools" -eq 1 ]]; then
	install_optional_tools
fi

show_next_steps
