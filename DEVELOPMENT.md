# 🚀 Quick Development Reference

> **TL;DR**: `make setup` → `make dev` → `make check` → commit

## ⚡ Essential Commands

```bash
# First time setup
make setup              # One-time setup (installs everything)

# Daily development
make dev                # Build + test (use this most)
make run ARGS="..."     # Test your changes
make check              # Full quality check before PR

# Quick reference
make help               # Show all commands
```

## 🔄 Development Loop

```bash
# 1. Start feature
git checkout -b feature/awesome-thing

# 2. Make changes and test constantly
make dev                # Quick build + test

# 3. Test manually
make run ARGS="cidr explain 192.168.1.0/24"

# 4. Full check before commit
make check

# 5. Commit and push
git commit -m "feat: add awesome thing"
git push origin feature/awesome-thing
```

## 🧪 Testing

```bash
make test-quick         # Fast tests during development
make test               # Full tests with race detection
make test-integration   # Integration tests
make examples           # See examples in action
```

## 🛠️ Building

```bash
make build              # Build for current platform
make build-all          # Build for all platforms
make clean              # Clean build artifacts
```

## 🔧 Code Quality

```bash
make fmt                # Format code
make vet                # Check for issues
make lint               # Run linter (if installed)
make lint-if-available  # Run linter if available (safe)
```

## 📦 Optional Tools

```bash
make install-tools      # Install golangci-lint, etc.
make install-precommit  # Install pre-commit hooks
```

## 🎯 Common Tasks

### **Test a specific command**
```bash
make run ARGS="cidr explain 10.0.0.0/8"
make run ARGS="cidr contains 192.168.1.0/24 192.168.1.100"
make run ARGS="--help"
```

### **Add a new feature**
1. Add code to `cmd/` (CLI) and `internal/` (logic)
2. Add tests to `*_test.go` files
3. Test: `make dev`
4. Quality check: `make check`

### **Fix a bug**
1. Write a failing test first
2. Fix the bug
3. Verify: `make test`
4. Quality check: `make check`

## 📂 Project Layout

```
cmd/           # CLI commands (add new commands here)
├── cidr/      # CIDR analysis ✅ (complete)
├── dns/       # DNS tools 🚧 (needs implementation)
├── scan/      # Network scanning 🚧 (needs implementation)
└── fw/        # Firewall tools 🚧 (needs implementation)

internal/      # Core business logic
├── cidr/      # CIDR calculations
└── validation/ # Input validation

scripts/       # Development scripts
```

## 💡 Tips

- **Use `make dev` constantly** during development
- **Run `make check` before every commit**
- **Test manually with `make run`** to verify behavior
- **Add tests for new features** - it's required
- **Keep commits atomic** and use conventional format

## 🐛 Troubleshooting

### Build fails?
```bash
make clean
go mod tidy
make build
```

### Tests fail?
```bash
make test-quick          # See specific failures
make run ARGS="--help"   # Test if binary works
```

### Linting errors?
```bash
make fmt                 # Fix formatting
make vet                 # Check for issues
```

### Need help?
- 💬 [GitHub Discussions](https://github.com/euan-cowie/cidrator/discussions)
- 🐛 [Issues](https://github.com/euan-cowie/cidrator/issues)
- 📖 [Contributing Guide](CONTRIBUTING.md)

---

**⚡ Quick start: `make setup && make dev`**
