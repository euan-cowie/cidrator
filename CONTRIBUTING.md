# Contributing to Cidrator

🎉 **Welcome! Thank you for considering contributing to Cidrator!** 🎉

We've designed the development experience to be as smooth and simple as possible. Whether you're fixing a typo or adding a major feature, we want contributing to be enjoyable.

## 🚀 Quick Start (2 minutes)

### **Option 1: Automated Setup (Recommended)**

```bash
# Clone the repository
git clone https://github.com/YOUR_USERNAME/cidrator.git
cd cidrator

# Run the setup script
./scripts/setup.sh
```

### **Option 2: Manual Setup**

```bash
# Clone and setup
git clone https://github.com/YOUR_USERNAME/cidrator.git
cd cidrator

# One-time setup
make setup

# Quick development check
make dev
```

**That's it!** You're ready to contribute. ✨

## 📖 Essential Commands

Everything you need is in the `Makefile`:

```bash
make help          # Show all available commands
make dev           # Quick build + test workflow  
make run ARGS="..."# Test a command (e.g., make run ARGS="cidr explain 10.0.0.0/8")
make check         # Full quality checks before committing
```

## 🔄 Development Workflow

### **1. Create Your Feature**

```bash
# Sync with main
git checkout main && git pull upstream main

# Create your branch
git checkout -b feature/awesome-feature

# Make your changes...

# Quick test
make dev
```

### **2. Test Your Changes**

```bash
# Run tests
make test

# Test your specific changes
make run ARGS="cidr explain 192.168.1.0/24"

# Full quality check
make check
```

### **3. Commit and Push**

```bash
# Stage your changes
git add .

# Commit with descriptive message
git commit -m "feat: add awesome new feature"

# Push to your fork
git push origin feature/awesome-feature
```

### **4. Create Pull Request**

Open a PR on GitHub! Our CI will run all checks automatically.

## 📝 Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/) for automatic changelog generation:

```bash
feat: add new DNS lookup command
fix: resolve memory leak in CIDR expansion
docs: update installation instructions
test: add tests for IPv6 support
```

**Types:**
- `feat:` - New features
- `fix:` - Bug fixes  
- `docs:` - Documentation changes
- `test:` - Adding tests
- `refactor:` - Code refactoring
- `chore:` - Maintenance tasks

## 🧪 Testing

### **Quick Tests**
```bash
make test-quick    # Fast tests, no coverage
make test          # Full tests with coverage
```

### **Manual Testing**
```bash
# Test the CLI directly
make run ARGS="cidr explain 10.0.0.0/16"
make run ARGS="cidr count 192.168.1.0/24"

# Run examples
make examples
```

### **Writing Tests**

We use table-driven tests:

```go
func TestYourFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid case",
            input:    "192.168.1.0/24", 
            expected: "256",
            wantErr:  false,
        },
        // ... more cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := YourFunction(tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                return
            }
            
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

## 🛠️ Optional Advanced Tools

These are **completely optional** but helpful for active contributors:

### **Linting Tools**
```bash
make install-tools     # Install golangci-lint, gosec, etc.
make lint             # Run advanced linting
```

### **Pre-commit Hooks**
```bash
make install-precommit # Install optional git hooks
make remove-precommit  # Remove them if you don't like them
```

Pre-commit hooks are **optional**. Our CI catches everything, so you can contribute successfully without them.

## 🎯 What to Work On

### **Good First Issues**
- Look for [`good first issue`](https://github.com/euan-cowie/cidrator/labels/good%20first%20issue) label
- Documentation improvements
- Adding tests
- Small bug fixes

### **High-Impact Areas**
- **DNS Tools** (`cmd/dns/`) - Implement DNS lookup features
- **Network Scanning** (`cmd/scan/`) - Add port scanning capabilities
- **Firewall Tools** (`cmd/fw/`) - Rule generation and analysis
- **Output Formats** - New export formats (CSV, XML, etc.)
- **Performance** - Optimize large network operations

## 📁 Project Structure

```
cidrator/
├── cmd/                    # CLI commands (keep these thin)
│   ├── cidr/              # CIDR analysis commands
│   ├── dns/               # DNS tools (coming soon)
│   ├── scan/              # Network scanning (coming soon)
│   └── fw/                # Firewall tools (coming soon)
├── internal/              # Core logic
│   ├── cidr/              # CIDR calculations
│   └── validation/        # Input validation
├── scripts/               # Development scripts
└── Makefile              # Primary developer interface
```

**Design Philosophy:**
- **Simple first** - Easy to understand and modify
- **Test everything** - Comprehensive test coverage
- **Clear errors** - Helpful error messages for users
- **Fast feedback** - Quick build and test cycles

## 🤝 Code Standards

### **Go Standards**
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `make fmt` to format code
- Add tests for new functionality
- Write clear, descriptive function names

### **Package Organization**
- `cmd/` - CLI interface only, delegate to `internal/`
- `internal/` - Business logic and algorithms
- Keep functions small and focused
- Use interfaces for testing

### **Error Handling**
```go
// ✅ Good: Descriptive errors
return nil, fmt.Errorf("failed to parse CIDR %q: %w", cidr, err)

// ❌ Bad: Generic errors  
return nil, fmt.Errorf("error")
```

## 🚨 Before Submitting

Run our quality checks:

```bash
make check    # Runs fmt, vet, test, and optional linting
```

This ensures:
- ✅ Code is formatted correctly
- ✅ No suspicious constructs (`go vet`)
- ✅ All tests pass
- ✅ Optional linting passes (if tools installed)

## 🆘 Getting Help

**Stuck? We're here to help!**

- 🗣️ **Ask Questions**: [GitHub Discussions](https://github.com/euan-cowie/cidrator/discussions)
- 🐛 **Report Issues**: [GitHub Issues](https://github.com/euan-cowie/cidrator/issues)
- 💬 **Chat**: Comment on any issue or PR
- 📖 **Documentation**: Check the [Wiki](https://github.com/euan-cowie/cidrator/wiki)

## 🎉 Recognition

All contributors are valued and will be:
- ✨ Listed in our contributors page
- 🏷️ Mentioned in release notes  
- 👏 Recognized in project documentation

## ❤️ Thank You

Contributing to open source makes the development community stronger. Every contribution, no matter how small, makes a difference.

**Ready to contribute? Here's your first command:**

```bash
./scripts/setup.sh && make dev
```

Happy coding! 🚀
