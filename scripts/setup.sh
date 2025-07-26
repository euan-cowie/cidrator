#!/bin/bash
# Cidrator Development Setup - Simple and Reliable
set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

print_header() {
    echo -e "${BLUE}${BOLD}"
    echo "🚀 Cidrator Development Setup"
    echo "================================"
    echo -e "${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

check_requirements() {
    print_info "Checking requirements..."

    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed!"
        echo "Install from: https://golang.org/dl/"
        exit 1
    fi
    print_success "Go $(go version | awk '{print $3}')"

    # Check we're in the right place
    if [[ ! -f "go.mod" ]] || [[ ! -f "main.go" ]]; then
        print_error "Run this from the cidrator project root"
        exit 1
    fi
    print_success "Project directory confirmed"
}

setup_project() {
    print_info "Setting up project..."

    # Clean any previous builds
    rm -rf bin/ 2>/dev/null || true

    # Get dependencies
    go mod download
    go mod tidy
    print_success "Dependencies ready"

    # Build
    make build
    print_success "Build complete"

    # Quick test
    if make test-quick >/dev/null 2>&1; then
        print_success "Tests passing"
    else
        print_warning "Some tests failed (you can fix this later)"
    fi
}

detect_shell() {
    # Detect the user's shell
    case "$SHELL" in
        */zsh)
            echo "zsh"
            ;;
        */bash)
            echo "bash"
            ;;
        *)
            # Default to zsh for modern macOS
            echo "zsh"
            ;;
    esac
}

check_and_fix_path() {
    local tool_name="$1"
    local install_path="$(go env GOPATH)/bin"

    # Check if tool is already in PATH
    if command -v "$tool_name" &> /dev/null; then
        print_success "$tool_name is accessible in PATH"
        return 0
    fi

    # Check if tool exists in Go bin directory
    if [[ ! -f "$install_path/$tool_name" ]]; then
        print_error "$tool_name not found in $install_path"
        return 1
    fi

    print_warning "$tool_name installed but not in PATH"

    # Auto-fix PATH issue
    local shell_type=$(detect_shell)
    local config_file=""

    case "$shell_type" in
        "zsh")
            config_file="$HOME/.zshrc"
            ;;
        "bash")
            config_file="$HOME/.bash_profile"
            ;;
    esac

    print_info "Detected shell: $shell_type"
    print_info "Config file: $config_file"

    # Check if Go bin path is already in the config
    local go_path_export='export PATH="$PATH:$(go env GOPATH)/bin"'

    if [[ -f "$config_file" ]] && grep -q 'GOPATH.*bin' "$config_file"; then
        print_info "Go PATH already configured in $config_file"
    else
        print_info "Adding Go bin directory to PATH in $config_file"
        echo "" >> "$config_file"
        echo "# Added by cidrator setup - Go tools" >> "$config_file"
        echo "$go_path_export" >> "$config_file"
        print_success "PATH updated in $config_file"
    fi

    # Test if it works now
    if [[ ":$PATH:" == *":$install_path:"* ]] || command -v "$tool_name" &> /dev/null; then
        print_success "$tool_name should now be accessible"
    else
        print_warning "You may need to restart your terminal or run:"
        echo -e "  ${BLUE}source $config_file${NC}"
        echo -e "  ${BLUE}export PATH=\"\$PATH:\$(go env GOPATH)/bin\"${NC}"
    fi
}

install_optional_tools() {
    print_info "Installing optional development tools..."

    # Install golangci-lint if not present
    if ! command -v golangci-lint &> /dev/null; then
        print_info "Installing golangci-lint..."
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin latest

        # Check and fix PATH for golangci-lint
        check_and_fix_path "golangci-lint"
    else
        print_success "golangci-lint already installed"
    fi

    # Install pre-commit if python is available
    if command -v python3 &> /dev/null || command -v python &> /dev/null; then
        if ! command -v pre-commit &> /dev/null; then
            print_info "Installing pre-commit..."
            if command -v pip3 &> /dev/null; then
                pip3 install pre-commit --user
            elif command -v pip &> /dev/null; then
                pip install pre-commit --user
            fi
            print_success "pre-commit installed"
        else
            print_success "pre-commit already installed"
        fi

        # Install hooks
        if command -v pre-commit &> /dev/null; then
            pre-commit install >/dev/null 2>&1 || true
            print_success "pre-commit hooks installed"
        fi
    fi
}

show_next_steps() {
    echo
    echo -e "${GREEN}${BOLD}🎉 Setup Complete!${NC}"
    echo

    # Verify tools are working
    echo -e "${BOLD}🔧 Tool Status:${NC}"
    if command -v golangci-lint &> /dev/null; then
        echo -e "  ${GREEN}✅ golangci-lint: $(golangci-lint --version | head -1)${NC}"
    else
        echo -e "  ${YELLOW}⚠️  golangci-lint: Restart terminal or run: source ~/.zshrc${NC}"
    fi

    echo
    echo -e "${BOLD}Try these commands:${NC}"
    echo -e "  ${BLUE}./bin/cidrator cidr explain 192.168.1.0/24${NC}  # Test the CLI"
    echo -e "  ${BLUE}make dev${NC}                                    # Quick build+test"
    echo -e "  ${BLUE}make help${NC}                                   # Show all commands"
    echo
    echo -e "${BOLD}Development workflow:${NC}"
    echo -e "  1. ${BLUE}make dev${NC}      # Build and test your changes"
    echo -e "  2. ${BLUE}make check${NC}    # Run full quality checks"
    echo -e "  3. ${BLUE}git commit${NC}    # Commit with conventional message"
    echo
    echo -e "${YELLOW}📖 See CONTRIBUTING.md for detailed guide${NC}"
    echo
}

main() {
    print_header

    check_requirements
    setup_project

    # Ask about optional tools
    echo
    read -p "Install optional dev tools? (golangci-lint, pre-commit) [Y/n]: " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        install_optional_tools
    fi

    show_next_steps
}

main "$@"
