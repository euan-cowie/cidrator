#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
NC='\033[0m' # No Color

echo -e "${BLUE}${BOLD}🚀 Cidrator Development Setup${NC}"
echo -e "${BLUE}Setting up your development environment...${NC}"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo -e "${RED}❌ Go is not installed!${NC}"
    echo -e "${YELLOW}Please install Go from: https://golang.org/dl/${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Go found: $(go version)${NC}"

# Check if we're in the right directory
if [[ ! -f "go.mod" ]] || [[ ! -f "main.go" ]]; then
    echo -e "${RED}❌ Please run this script from the cidrator project root${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Project directory confirmed${NC}"

# Download dependencies and build
echo -e "${BLUE}📦 Downloading dependencies...${NC}"
go mod download
go mod tidy

echo -e "${BLUE}🔨 Building cidrator...${NC}"
make build

echo -e "${BLUE}🧪 Running tests...${NC}"
make test-quick

echo ""
echo -e "${GREEN}${BOLD}🎉 Setup complete!${NC}"
echo ""
echo -e "${YELLOW}Quick start commands:${NC}"
echo -e "  ${BOLD}make help${NC}          - Show all available commands"
echo -e "  ${BOLD}make dev${NC}           - Quick development workflow"
echo -e "  ${BOLD}make run ARGS=\"...\"${NC}  - Test a command"
echo -e "  ${BOLD}make check${NC}         - Run all quality checks"
echo ""
echo -e "${YELLOW}Optional tools (recommended for active contributors):${NC}"
echo -e "  ${BOLD}make install-tools${NC}     - Install golangci-lint and other dev tools"
echo -e "  ${BOLD}make install-precommit${NC} - Install optional pre-commit hooks"
echo ""
echo -e "${GREEN}Example command to try:${NC}"
echo -e "  ${BOLD}./bin/cidrator cidr explain 192.168.1.0/24${NC}"
echo ""
echo -e "${BLUE}Happy coding! 🎉${NC}" 