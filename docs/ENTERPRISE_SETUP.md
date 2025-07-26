# 🏢 Enterprise Release Setup Guide

This guide explains how to upgrade your simple CLI release workflow to enterprise-grade with security, compliance, and multi-platform distribution features.

## 📊 Current State: Simple CLI
Your current workflow provides:
- ✅ Cross-platform binary builds via GoReleaser
- ✅ GitHub Releases with automatic changelogs
- ✅ Basic caching for faster builds
- ✅ Automated testing before release

## 🚀 Enterprise Upgrade Path

### Phase 1: Security & Signing 🔐

**Why:** Supply chain security, binary authenticity, compliance requirements

**Add these steps to `.github/workflows/release.yml`:**

```yaml
# After "Set up Go" step
- name: Install Cosign
  uses: sigstore/cosign-installer@v3
  with:
    cosign-release: 'v2.2.2'

- name: Install Syft (for SBOM generation)
  run: |
    curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

- name: Import GPG key
  if: env.GPG_PRIVATE_KEY != ''
  env:
    GPG_PRIVATE_KEY: ${{ secrets.GPG_PRIVATE_KEY }}
    GPG_PASSPHRASE: ${{ secrets.GPG_PASSPHRASE }}
  run: |
    echo "$GPG_PRIVATE_KEY" | gpg --batch --import
```

**Required Secrets:**
- `COSIGN_PRIVATE_KEY` - Generate: `cosign generate-key-pair`
- `COSIGN_PASSWORD` - Password for Cosign key
- `GPG_PRIVATE_KEY` - Export: `gpg --export-secret-keys --armor KEY_ID`
- `GPG_PASSPHRASE` - GPG key passphrase

**Add to GoReleaser environment:**
```yaml
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  COSIGN_PRIVATE_KEY: ${{ secrets.COSIGN_PRIVATE_KEY }}
  COSIGN_PASSWORD: ${{ secrets.COSIGN_PASSWORD }}
```

### Phase 2: Multi-Platform Distribution 📦

**Why:** Easier installation for users across different package managers

**Required Secrets:**
- `HOMEBREW_TAP_GITHUB_TOKEN` - Personal Access Token with repo scope
- `AUR_KEY` - SSH private key for AUR access
- `FURY_TOKEN` + `FURY_ACCOUNT` - Gemfury.com credentials

**Update GoReleaser config** to include:
```yaml
# .goreleaser.yaml
brews:
  - name: cidrator
    tap:
      owner: euan-cowie
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_GITHUB_TOKEN }}"
    homepage: "https://github.com/euan-cowie/cidrator"
    description: "CIDR manipulation and network analysis CLI tool"

archives:
  - id: default
    files:
      - completions/*
```

### Phase 3: Verification & Compliance 🛡️

**Add verification steps:**
```yaml
- name: Verify shell completions exist
  run: |
    ls -la completions/
    [ -f completions/cidrator.bash ] || { echo "Missing bash completion"; exit 1; }
    [ -f completions/cidrator.zsh ] || { echo "Missing zsh completion"; exit 1; }
    [ -f completions/cidrator.fish ] || { echo "Missing fish completion"; exit 1; }

- name: Verify release artifacts
  run: |
    cd dist
    sha256sum -c checksums.txt
    if [ -f checksums.txt.sig ]; then
      gpg --verify checksums.txt.sig checksums.txt
    fi
```

### Phase 4: Binary Optimization 🗜️

**Add UPX compression:**
```yaml
- name: Install UPX
  run: |
    sudo apt-get update
    sudo apt-get install -y upx-ucl
```

**Update `.goreleaser.yaml`:**
```yaml
builds:
  - binary: cidrator
    flags:
      - -trimpath
    ldflags:
      - -s -w -X cmd.Version={{.Version}} -X cmd.Commit={{.Commit}} -X cmd.Date={{.Date}}
    hooks:
      post: upx "{{ .Path }}"
```

### Phase 5: Notifications & Integrations 🔔

**Add post-release job:**
```yaml
post-release:
  runs-on: ubuntu-latest
  needs: release
  steps:
  - name: Checkout code
    uses: actions/checkout@v4

  - name: Notify success
    run: |
      echo "🎉 Release ${{ github.ref_name }} completed!"
      echo "📦 Download: https://github.com/${{ github.repository }}/releases/tag/${{ github.ref_name }}"
      echo "🍺 Homebrew: brew install euan-cowie/tap/cidrator"

  - name: Trigger documentation update
    uses: peter-evans/repository-dispatch@v2
    with:
      token: ${{ secrets.GITHUB_TOKEN }}
      event-type: release-published
      client-payload: '{"version": "${{ github.ref_name }}"}'
```

## 🎯 Incremental Adoption Strategy

**Recommended order:**
1. **Start here:** Phase 1 (Security) - Most important for trust
2. **High impact:** Phase 2 (Distribution) - Easier user adoption
3. **Compliance:** Phase 3 (Verification) - For enterprise customers
4. **Optimization:** Phase 4 (UPX) - Performance improvement
5. **Nice-to-have:** Phase 5 (Notifications) - Developer experience

## 📋 Enterprise Checklist

- [ ] **Security**: Cosign + GPG signing implemented
- [ ] **SBOM**: Software Bill of Materials generated
- [ ] **Distribution**: Homebrew tap configured
- [ ] **Verification**: Checksums and signatures verified
- [ ] **Compliance**: All binaries signed and documented
- [ ] **Monitoring**: Release notifications configured
- [ ] **Documentation**: User installation guides updated

## 🔍 Monitoring & Maintenance

**Monthly tasks:**
- Rotate signing keys if needed
- Update package manager metadata
- Review security scan results
- Update dependencies in workflows

**Quarterly tasks:**
- Audit secret permissions
- Review distribution analytics
- Update compliance documentation
- Security audit of release process

## 🆘 Troubleshooting

**Common issues:**
- **Cosign failures**: Check key format and permissions
- **Homebrew tap failures**: Verify token scope includes repo access
- **GPG signing issues**: Ensure passphrase is correct
- **AUR publishing**: Check SSH key has proper access

## 📚 Additional Resources

- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)
- [GoReleaser Documentation](https://goreleaser.com/intro/)
- [GitHub Actions Security](https://docs.github.com/en/actions/security-guides)
- [Supply Chain Security Best Practices](https://slsa.dev/)

---

> **Note**: Enterprise features add complexity but provide significant value for:
> - Corporate environments requiring signed binaries
> - Open source projects seeking wide distribution
> - Applications requiring compliance documentation
> - High-security environments needing full audit trails
