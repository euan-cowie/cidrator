# Security Policy

## 🛡️ Reporting Security Vulnerabilities

We take the security of Cidrator seriously. If you discover a security vulnerability, please follow our responsible disclosure process.

### 📧 Private Disclosure (Preferred)

For security vulnerabilities, please email us directly:

- **Email**: [security@cidrator.dev](mailto:security@cidrator.dev)
- **Subject**: `[SECURITY] Vulnerability in Cidrator`
- **Response Time**: We aim to respond within 24 hours

### 🔒 GitHub Security Advisories

You can also report vulnerabilities through GitHub's private vulnerability reporting:

1. Go to [Security Advisories](https://github.com/euan-cowie/cidrator/security/advisories)
2. Click "Report a vulnerability"
3. Fill out the private advisory form

### ⚠️ What NOT to Do

- **Do NOT** open public GitHub issues for security vulnerabilities
- **Do NOT** discuss security issues in public forums or social media
- **Do NOT** share vulnerability details until we've had time to address them

## 🎯 What Qualifies as a Security Vulnerability

We consider the following as security vulnerabilities:

### **High Priority**
- Remote code execution
- Local privilege escalation
- Authentication bypass
- Unauthorized data access
- Memory corruption vulnerabilities
- Injection attacks (command injection, etc.)

### **Medium Priority**
- Information disclosure (sensitive data)
- Denial of service vulnerabilities
- Path traversal attacks
- Input validation issues leading to unexpected behavior

### **Low Priority**
- Information disclosure (non-sensitive data)
- Rate limiting issues
- Minor input validation issues

### **Not Security Issues**
- Feature requests
- Performance issues (unless DoS-related)
- Configuration issues
- User error
- Issues requiring physical access to the system

## 🔍 Security Features

Cidrator includes several security features:

### **Input Validation**
- Comprehensive CIDR format validation
- IP address format validation
- Safe limits on operations (preventing memory exhaustion)
- Sanitized error messages (no sensitive data leakage)

### **Safe Defaults**
- Automatic safety limits on large range operations
- Memory-safe operations using Go's built-in protections
- No external network requests by default
- Read-only operations on network data

### **Build Security**
- Dependencies are regularly updated
- Security scanning in CI/CD pipeline
- Reproducible builds
- Code signing for releases

## 📋 Supported Versions

We provide security updates for the following versions:

| Version | Supported | End of Life |
|---------|-----------|-------------|
| 1.x.x   | ✅ Yes    | TBD         |
| 0.x.x   | ⚠️ Limited | When 1.0 releases |

**Note**: We generally support the latest major version and may provide critical security fixes for the previous major version.

## 🚨 Security Response Process

### **1. Report Received**
- We acknowledge receipt within 24 hours
- Initial assessment within 48 hours
- Severity classification assigned

### **2. Investigation**
- Detailed analysis of the vulnerability
- Impact assessment
- Reproducibility verification
- Affected versions identification

### **3. Fix Development**
- Develop and test security fix
- Create minimal reproduction case
- Prepare security advisory
- Coordinate with reporter

### **4. Disclosure**
- Private notification to reporter
- Security advisory published
- CVE assignment (if applicable)
- Public release with fix

### **5. Timeline**
- **Initial Response**: 24 hours
- **Assessment**: 48-72 hours  
- **Fix Development**: 1-7 days (depending on severity)
- **Public Disclosure**: After fix is available

## 🏆 Security Hall of Fame

We recognize security researchers who help improve Cidrator's security:

<!-- Security researchers who have reported vulnerabilities will be listed here -->

*Be the first to help secure Cidrator!*

## 🔧 Security Best Practices for Users

### **General Usage**
- Always use the latest stable version
- Validate input when using Cidrator in scripts
- Don't run Cidrator with unnecessary privileges
- Be cautious with large CIDR ranges

### **In Production**
- Monitor resource usage when processing large ranges
- Implement timeouts for automated scripts
- Log and monitor Cidrator usage
- Use configuration management for consistent deployments

### **For Developers**
- Follow input validation best practices
- Use Cidrator's JSON output for programmatic access
- Handle errors appropriately in your applications
- Don't expose raw Cidrator output to untrusted users

## 🔗 Security Resources

### **External Resources**
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Guide](https://github.com/securego/gosec)

### **Project Resources**
- [Contributing Guidelines](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Issue Templates](.github/ISSUE_TEMPLATE/)

## 📞 Contact Information

### **Security Team**
- **Email**: [security@cidrator.dev](mailto:security@cidrator.dev)
- **GPG Key**: Available on request

### **General Contact**
- **Issues**: [GitHub Issues](https://github.com/euan-cowie/cidrator/issues)
- **Discussions**: [GitHub Discussions](https://github.com/euan-cowie/cidrator/discussions)

## 🏅 Acknowledgments

We appreciate the security research community and thank all researchers who responsibly disclose vulnerabilities to help make Cidrator more secure.

---

**Last Updated**: January 2025
**Policy Version**: 1.0 