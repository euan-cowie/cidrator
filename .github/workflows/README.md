# Cidrator Workflows Documentation

This document describes the comprehensive CI/CD workflow system for Cidrator, including automated versioning, building, testing, and releasing.

## 🏗️ Workflow Overview

### 1. **Continuous Integration (CI) - `.github/workflows/ci.yml`**

Runs on every push to `main`/`develop` and pull requests to `main`.

**Features:**
- ✅ **Multi-platform testing** (Ubuntu, macOS, Windows)
- ✅ **Multi-version Go testing** (1.21, 1.22)
- ✅ **Code coverage with threshold** (85% minimum)
- ✅ **Advanced linting** with golangci-lint
- ✅ **Security scanning** with gosec, Nancy, and Trivy
- ✅ **Performance benchmarking** with trend tracking
- ✅ **Build verification** for all platforms
- ✅ **Integration testing** with real CLI

### 2. **Automated Versioning - `.github/workflows/version.yml`**

Automatically manages semantic versioning based on conventional commits.

**Features:**
- ✅ **Semantic versioning** with conventional commits
- ✅ **Automatic changelog generation**
- ✅ **Version bumping** in source code
- ✅ **Tag creation** and release notes

### 3. **Release Pipeline - `.github/workflows/release.yml`**

Comprehensive release automation triggered by version tags.

**Features:**
- ✅ **Multi-platform binaries** (Linux, macOS, Windows, FreeBSD)
- ✅ **Multiple architectures** (amd64, arm64, arm)
- ✅ **Package managers** (Homebrew, APT, RPM, AUR, Snap)
- ✅ **Security signing** with GPG
- ✅ **SBOM generation** for supply chain security
- ✅ **Shell completions** (bash, zsh, fish)
- ✅ **Binary security verification**

## 🚀 Getting Started

### Prerequisites

1. **Repository Secrets** - Configure these in GitHub Settings → Secrets:

```bash
# Required for releases
GITHUB_TOKEN                 # Automatically provided by GitHub
HOMEBREW_TAP_GITHUB_TOKEN   # Token for homebrew tap repository
GPG_PRIVATE_KEY             # GPG private key for signing (optional)
GPG_PASSPHRASE              # GPG key passphrase (optional)

# Optional for additional package managers
AUR_KEY                     # AUR SSH private key for Arch Linux packages
FURY_TOKEN                  # Fury.io token for package publishing
FURY_ACCOUNT                # Fury.io account name
```

2. **Branch Protection Rules** - Recommended settings:

```yaml
main:
  required_status_checks:
    - "Test (ubuntu-latest, 1.22)"
    - "Test (macos-latest, 1.22)"
    - "Test (windows-latest, 1.22)"
    - "Lint"
    - "Security"
    - "Build"
  enforce_admins: true
  required_pull_request_reviews:
    required_approving_review_count: 1
    dismiss_stale_reviews: true
```

### Setup Instructions

1. **Enable workflows** by committing the workflow files to your repository

2. **Configure semantic-release** by ensuring your commits follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Examples of properly formatted commits
git commit -m "feat: add IPv6 support for expand command"
git commit -m "fix: prevent memory exhaustion in large CIDR expansion"
git commit -m "feat!: change default output format to JSON"  # Breaking change
git commit -m "docs: update installation instructions"
```

3. **Set up GPG signing** (optional but recommended):

```bash
# Generate GPG key
gpg --full-generate-key

# Export private key
gpg --armor --export-secret-keys YOUR_KEY_ID

# Add to GitHub secrets as GPG_PRIVATE_KEY
```



## 📋 Conventional Commits Guide

Our automated versioning system uses conventional commits to determine version bumps:

| Commit Type | Version Bump | Example |
|-------------|--------------|---------|
| `feat:` | Minor (1.0.0 → 1.1.0) | `feat: add new DNS lookup feature` |
| `fix:` | Patch (1.0.0 → 1.0.1) | `fix: resolve memory leak in scanner` |
| `perf:` | Patch (1.0.0 → 1.0.1) | `perf: optimize CIDR parsing performance` |
| `feat!:` | Major (1.0.0 → 2.0.0) | `feat!: change CLI interface structure` |
| `fix!:` | Major (1.0.0 → 2.0.0) | `fix!: remove deprecated --legacy flag` |
| `docs:` | No bump | `docs: update API documentation` |
| `style:` | No bump | `style: fix code formatting` |
| `refactor:` | Patch (1.0.0 → 1.0.1) | `refactor: improve error handling` |
| `test:` | No bump | `test: add unit tests for CIDR parsing` |
| `chore:` | No bump | `chore: update dependencies` |

### Breaking Changes

Use `!` after the type or add `BREAKING CHANGE:` in the footer:

```bash
# Method 1: Using !
git commit -m "feat!: change default port from 8080 to 3000"

# Method 2: Using footer
git commit -m "feat: add new configuration format

BREAKING CHANGE: Configuration files now use YAML instead of JSON"
```

## 🔄 Workflow Triggers

### CI Workflow
- **Push** to `main` or `develop` branches
- **Pull Request** to `main` branch

### Version Workflow
- **Push** to `main` branch (only creates versions/tags)

### Release Workflow
- **Tag push** with pattern `v*` (e.g., `v1.2.3`)

## 📦 Release Process

### Automatic Release (Recommended)

1. **Create feature branch:**
```bash
git checkout -b feature/new-awesome-feature
```

2. **Make changes and commit with conventional format:**
```bash
git commit -m "feat: add awesome new feature"
```

3. **Create pull request** and get it reviewed/approved

4. **Merge to main** - this triggers:
   - CI checks run
   - Version workflow creates new tag
   - Release workflow builds and publishes

### Manual Release (If needed)

1. **Create and push tag manually:**
```bash
git tag -a v1.2.3 -m "Release v1.2.3"
git push origin v1.2.3
```

## 🛡️ Security Features

### Code Security
- **Static analysis** with gosec
- **Dependency scanning** with Nancy and Trivy
- **SARIF reports** uploaded to GitHub Security tab
- **Vulnerability database** checks

### Release Security
- **Binary signing** with GPG
- **SBOM generation** for supply chain transparency
- **Checksums** for all artifacts
- **Provenance** attestation

### Supply Chain Security
- **Dependabot** for automated dependency updates
- **Pin action versions** to specific commits
- **SLSA compliance** through GitHub's OIDC
- **Reproducible builds** with locked dependencies

## 📊 Quality Gates

### Coverage Requirements
- **Minimum 85% code coverage** for new code
- **Coverage reports** uploaded to Codecov
- **Coverage trending** to track improvements

### Performance Monitoring
- **Benchmark regression detection** (200% threshold)
- **Performance trends** tracked over time
- **Alerts** on significant performance degradation

### Code Quality
- **Comprehensive linting** with golangci-lint
- **Format checking** with gofmt
- **Vet analysis** for suspicious constructs
- **Import organization** verification

## 📚 Package Distribution

### Homebrew (macOS/Linux)
```bash
brew install euan-cowie/tap/cidrator
```

### APT (Debian/Ubuntu)
```bash
# Add repository and install
curl -fsSL https://packagecloud.io/euan-cowie/cidrator/gpgkey | sudo apt-key add -
echo "deb https://packagecloud.io/euan-cowie/cidrator/ubuntu focal main" | sudo tee /etc/apt/sources.list.d/cidrator.list
sudo apt update && sudo apt install cidrator
```

### RPM (CentOS/RHEL/Fedora)
```bash
# Add repository and install
curl -s https://packagecloud.io/install/repositories/euan-cowie/cidrator/script.rpm.sh | sudo bash
sudo yum install cidrator
```

### AUR (Arch Linux)
```bash
# Using yay
yay -S cidrator-bin

# Using pacman
git clone https://aur.archlinux.org/cidrator-bin.git
cd cidrator-bin && makepkg -si
```

### Snap (Universal Linux)
```bash
sudo snap install cidrator
```

## 🔧 Troubleshooting

### Common Issues

1. **Version workflow not triggering:**
   - Check commit message follows conventional format
   - Ensure push is to `main` branch
   - Verify GitHub Actions are enabled

2. **Release workflow failing:**
   - Check all required secrets are configured
   - Verify tag format matches `v*` pattern
   - Review workflow logs for specific errors

3. **Package publishing failures:**
   - Verify repository access tokens
   - Check package repository status
   - Review third-party service documentation

### Debug Tips

1. **Enable debug logging:**
```yaml
env:
  ACTIONS_STEP_DEBUG: true
  ACTIONS_RUNNER_DEBUG: true
```

2. **Test releases on fork:**
   - Fork repository
   - Configure secrets in fork
   - Test with different tag patterns

3. **Local testing:**
```bash
# Test GoReleaser locally
goreleaser release --snapshot --rm-dist

# Test semantic-release
npx semantic-release --dry-run
```

## 📈 Monitoring and Metrics

### Available Metrics
- **Build success rate** across platforms
- **Test coverage trends** over time
- **Performance benchmark** comparisons
- **Release frequency** and patterns
- **Security scan results** and trends

### Dashboards
- **GitHub Insights** for repository metrics
- **Actions tab** for workflow status
- **Security tab** for vulnerability reports
- **Codecov** for coverage analysis

## 🤝 Contributing to Workflows

### Workflow Development
1. **Test locally** using act or similar tools
2. **Create feature branch** for workflow changes
3. **Test on fork** before submitting PR
4. **Document changes** in this README

### Best Practices
- **Pin action versions** to specific commits
- **Use caching** for better performance
- **Implement proper error handling**
- **Add comprehensive logging**
- **Test failure scenarios**

---

## 📞 Support

- **Documentation Issues**: Create issue with `documentation` label
- **Workflow Problems**: Create issue with `ci/cd` label  
- **Security Concerns**: Email security@cidrator.dev
- **General Help**: Check GitHub Discussions 