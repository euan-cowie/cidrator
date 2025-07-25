# Contributing to Cidrator

🎉 **Thank you for considering contributing to Cidrator!** 🎉

We welcome contributions from the community and are excited to see what you'll build with us. This document outlines the process for contributing to make it as smooth as possible for everyone involved.

## 📋 Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Contributing Guidelines](#contributing-guidelines)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Release Process](#release-process)

## 🤝 Code of Conduct

This project adheres to a [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code. Please report unacceptable behavior to [security@cidrator.dev](mailto:security@cidrator.dev).

## 🚀 Getting Started

### **Prerequisites**

- **Go 1.21+** - [Download and install Go](https://golang.org/dl/)
- **Make** - For build automation
- **Git** - For version control
- **golangci-lint** - For linting (optional, installed via make)
- **gosec** - For security scanning (optional, installed via make)

### **Fork and Clone**

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/cidrator.git
   cd cidrator
   ```

3. **Add upstream** remote:
   ```bash
   git remote add upstream https://github.com/euan-cowie/cidrator.git
   ```

### **Development Setup**

```bash
# Install dependencies
go mod download

# Install development tools
make deps

# Build the project
make build

# Run tests
make test

# Verify everything works
./bin/cidrator cidr explain 192.168.1.0/24
```

## 🔄 Development Workflow

### **Branch Strategy**

- `main` - Production-ready code
- `develop` - Integration branch for new features
- `feature/*` - Feature development branches
- `bugfix/*` - Bug fix branches
- `hotfix/*` - Critical production fixes

### **Starting New Work**

1. **Sync with upstream**:
   ```bash
   git checkout main
   git pull upstream main
   git push origin main
   ```

2. **Create feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes** (see [Code Standards](#code-standards))

4. **Test thoroughly**:
   ```bash
   make test
   make lint
   make security
   ```

5. **Commit and push**:
   ```bash
   git add .
   git commit -m "feat: add awesome new feature"
   git push origin feature/your-feature-name
   ```

## 📝 Contributing Guidelines

### **Types of Contributions**

We welcome all types of contributions:

- 🐛 **Bug Reports** - Help us identify and fix issues
- 💡 **Feature Requests** - Suggest new functionality
- 📖 **Documentation** - Improve guides, examples, and API docs
- 🧪 **Tests** - Add test coverage for existing features
- 🔧 **Code** - Bug fixes, new features, performance improvements
- 🎨 **Design** - UI/UX improvements for CLI output
- 🌍 **Localization** - Help make Cidrator accessible worldwide

### **What to Work On**

**Good First Issues**: Look for issues labeled [`good first issue`](https://github.com/euan-cowie/cidrator/labels/good%20first%20issue)

**Priority Areas**:
- DNS tools implementation (`cmd/dns/`, `internal/dns/`)
- Network scanning features (`cmd/scan/`, `internal/scan/`)
- Firewall management tools (`cmd/fw/`, `internal/fw/`)
- Performance optimizations
- Additional output formats
- Cross-platform compatibility improvements

### **Before You Start**

- 🔍 **Check existing issues** to avoid duplicating work
- 💬 **Discuss major changes** in an issue or discussion first
- 📋 **Follow the project roadmap** for aligned contributions
- 🧪 **Write tests** for new functionality
- 📚 **Update documentation** for user-facing changes

## 🛠️ Code Standards

### **Go Style Guide**

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

### **Code Organization**

```
- cmd/           # CLI commands (keep thin, delegate to internal/)
- internal/      # Private application code
  - <domain>/    # Domain-specific packages (cidr, dns, scan, fw)
  - validation/  # Input validation logic
- examples/      # Usage examples
- docs/          # Documentation
```

### **Naming Conventions**

- **Packages**: Short, lowercase, no underscores (`cidr`, `dns`, `validation`)
- **Files**: Lowercase with underscores (`network_validator.go`)
- **Functions**: CamelCase, exported functions start with capital
- **Variables**: camelCase for local, CamelCase for exported
- **Constants**: CamelCase or UPPER_CASE for package-level

### **Error Handling**

```go
// ✅ Good: Use typed errors with context
func ParseCIDR(cidr string) (*NetworkInfo, error) {
    _, network, err := net.ParseCIDR(cidr)
    if err != nil {
        return nil, NewCIDRError("parse", cidr, ErrInvalidCIDR)
    }
    // ...
}

// ❌ Bad: Generic errors without context
func ParseCIDR(cidr string) (*NetworkInfo, error) {
    _, network, err := net.ParseCIDR(cidr)
    if err != nil {
        return nil, fmt.Errorf("failed to parse CIDR")
    }
    // ...
}
```

### **Function Design**

- **Keep functions small** (5-15 lines typically)
- **Single responsibility** principle
- **Max 2-3 parameters** (use structs for options)
- **Early returns** to reduce nesting
- **Clear names** that describe what the function does

### **Comments**

```go
// ✅ Good: Explains why, not what
// ValidateCIDR checks if a CIDR string is valid and can be parsed.
// It returns a descriptive error for invalid formats to help users
// understand what went wrong.
func ValidateCIDR(cidr string) error {
    // ...
}

// ❌ Bad: States the obvious
// ParseCIDR parses a CIDR string
func ParseCIDR(cidr string) error {
    // ...
}
```

## 🧪 Testing

### **Test Requirements**

- **Unit tests** for all new functions
- **Integration tests** for CLI commands
- **Table-driven tests** for multiple scenarios
- **Error cases** must be tested
- **Target 95%+ coverage** for new code

### **Test Structure**

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid IPv4 CIDR",
            input:    "192.168.1.0/24",
            expected: "256",
            wantErr:  false,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionName(tt.input)
            
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

### **Running Tests**

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific package tests
go test ./internal/cidr -v

# Run with race detection
go test -race ./...

# Benchmark tests
go test -bench=. ./...
```

## 📚 Documentation

### **Code Documentation**

- **Public APIs** must have godoc comments
- **Complex algorithms** need explanatory comments
- **Examples** in godoc when helpful
- **Package docs** explaining purpose and usage

### **User Documentation**

- **README** updates for new features
- **Command help** text for new commands
- **Examples** in documentation
- **Migration guides** for breaking changes

### **Documentation Testing**

```bash
# Test documentation examples
go test -run Example

# Generate and check docs locally
godoc -http=:6060
# Visit http://localhost:6060/pkg/github.com/euan-cowie/cidrator/
```

## 📤 Submitting Changes

### **Commit Message Format**

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples**:
```
feat(cidr): add IPv6 support for expand command

Add complete IPv6 support to the expand command including:
- Large address space handling with safety limits
- Proper formatting for IPv6 addresses
- Updated tests and documentation

Closes #123

fix: prevent memory exhaustion in large CIDR expansion

The expand command could consume excessive memory for large ranges.
Added safety limits and proper error handling.

Breaking change: Large ranges now return an error instead of
consuming all available memory.
```

### **Pull Request Process**

1. **Create descriptive PR**:
   - Clear title and description
   - Link to related issues
   - Describe changes and motivation
   - Include testing details

2. **Ensure CI passes**:
   - All tests pass
   - Linting succeeds
   - Security scans clean
   - Documentation builds

3. **Request review**:
   - Tag relevant maintainers
   - Respond to feedback promptly
   - Make requested changes

4. **Merge requirements**:
   - ✅ CI passing
   - ✅ Code review approval
   - ✅ Documentation updated
   - ✅ Tests added/updated
   - ✅ No conflicts with main

### **PR Template**

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing performed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex code
- [ ] Documentation updated
- [ ] No new warnings introduced
```

## 🚀 Release Process

**For Maintainers Only**

1. **Version Planning**:
   - Follow [Semantic Versioning](https://semver.org/)
   - Document breaking changes
   - Update CHANGELOG.md

2. **Release Steps**:
   ```bash
   # Create release branch
   git checkout -b release/v1.2.0
   
   # Update version in code
   # Update CHANGELOG.md
   # Commit changes
   
   # Create tag
   git tag -a v1.2.0 -m "Release v1.2.0"
   
   # Push tag (triggers automated release)
   git push origin v1.2.0
   ```

3. **Post-Release**:
   - Update documentation
   - Announce on discussions
   - Close related issues

## 🆘 Getting Help

- 💬 **Discussions**: [GitHub Discussions](https://github.com/euan-cowie/cidrator/discussions)
- 🐛 **Issues**: [GitHub Issues](https://github.com/euan-cowie/cidrator/issues)
- 📧 **Security**: [security@cidrator.dev](mailto:security@cidrator.dev)
- 📖 **Documentation**: [Project Wiki](https://github.com/euan-cowie/cidrator/wiki)

## 🙏 Recognition

Contributors will be:
- Listed in CONTRIBUTORS.md
- Mentioned in release notes
- Added to GitHub contributors
- Recognized in project documentation

---

**Thank you for contributing to Cidrator! 🎉**

Your contributions help make networking tools better for everyone in the community. 