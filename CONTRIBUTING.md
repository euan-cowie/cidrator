# Contributing to Cidrator

🎉 **Thank you for contributing!** 🎉

We've made contributing as simple as possible. You can be productive in under 5 minutes.

## 🚀 Quick Start

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/cidrator.git
cd cidrator

# 2. One-time setup
make setup

# 3. Make your changes, then test
make dev

# 4. Run quality checks
make check

# 5. Commit and push
git commit -m "feat: your awesome change"
git push origin your-branch-name
```

**That's it!** Create a PR and we'll review it.

## 🛠️ Development Commands

```bash
make help           # Show all commands
make dev            # Quick build + test (use this most)
make run ARGS="..." # Test your changes
make check          # Full checks before PR
```

## 🆘 Troubleshooting

### golangci-lint: command not found

If you see this error after running `make setup`:

**Quick fix:**
```bash
# Restart your terminal, OR
source ~/.zshrc    # for zsh (macOS default)
source ~/.bash_profile  # for bash
```

**Manual fix if needed:**
```bash
# Add Go's bin directory to your PATH
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

**Why this happens:** golangci-lint installs to `$(go env GOPATH)/bin` (usually `~/go/bin`), but this directory might not be in your shell's PATH. Our setup script tries to fix this automatically, but sometimes requires a terminal restart.

### Other Common Issues

**Tests failing after setup:**
```bash
make clean && make build && make test
```

**Dependencies out of sync:**
```bash
go mod download && go mod tidy
```

**Can't find `make` command:**
- **macOS:** Install Xcode Command Line Tools: `xcode-select --install`
- **Linux:** Install build-essential: `sudo apt-get install build-essential`

**Wrong Go version:**
- Cidrator requires Go 1.19+
- Check: `go version`
- Update: [https://golang.org/dl/](https://golang.org/dl/)

## 📝 Commit Messages

Use [conventional commits](https://www.conventionalcommits.org/):

```bash
feat: add awesome feature
fix: resolve memory leak
docs: update readme
test: add unit tests
```

**Types:** `feat`, `fix`, `docs`, `test`, `refactor`, `chore`

## 🧪 Testing

```bash
# Quick test during development
make test-quick

# Full tests before PR
make test

# Test your changes manually
make run ARGS="cidr explain 192.168.1.0/24"
```

## 🎯 What to Work On

**Good first issues:** Look for the [`good first issue`](https://github.com/euan-cowie/cidrator/labels/good%20first%20issue) label

**High-impact areas:**
- **DNS Tools** (`cmd/dns/`) - DNS lookup features
- **Network Scanning** (`cmd/scan/`) - Port scanning
- **Firewall Tools** (`cmd/fw/`) - Rule generation
- **Output Formats** - CSV, XML support
- **Tests** - Always appreciated!

## 📁 Project Structure

```
cidrator/
├── cmd/                 # CLI commands (add new features here)
│   ├── cidr/           # CIDR analysis
│   ├── dns/            # DNS tools (needs implementation)
│   ├── scan/           # Network scanning (needs implementation)
│   └── fw/             # Firewall tools (needs implementation)
├── internal/           # Core logic
│   ├── cidr/          # CIDR calculations
│   └── validation/    # Input validation
└── scripts/           # Development scripts
```

**Keep it simple:**
- `cmd/` = CLI interface (thin layer)
- `internal/` = Business logic
- Add tests for new features
- Follow Go conventions

## ✅ Before Submitting

1. **Run checks:** `make check`
2. **Test manually:** `make run ARGS="your command"`
3. **Write tests** for new features
4. **Use conventional commits**

## 🆘 Need Help?

- 🗣️ **Questions:** [GitHub Discussions](https://github.com/euan-cowie/cidrator/discussions)
- 🐛 **Issues:** [GitHub Issues](https://github.com/euan-cowie/cidrator/issues)
- 💬 **Chat:** Comment on any issue/PR

## 🎨 Code Style

- **Follow `go fmt`** (automatic with `make fmt`)
- **Add tests** for new functionality
- **Clear error messages** for users
- **Keep functions small** and focused

## 🚀 Advanced Setup (Optional)

```bash
# Install optional development tools
make install-tools      # golangci-lint
make install-precommit  # pre-commit hooks

# Build for all platforms
make build-all

# Run examples
make examples
```

## ❤️ Recognition

All contributors are listed in our contributors page and mentioned in release notes.

---

**Ready to contribute?**

```bash
make setup && make dev
```

Happy coding! 🚀
