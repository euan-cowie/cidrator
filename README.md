# Cidrator

[![Go Version](https://img.shields.io/github/go-mod/go-version/euan-cowie/cidrator)](https://golang.org/)
[![Release](https://img.shields.io/github/v/release/euan-cowie/cidrator)](https://github.com/euan-cowie/cidrator/releases)
[![License](https://img.shields.io/github/license/euan-cowie/cidrator)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/euan-cowie/cidrator/ci.yml?branch=main&label=CI)](https://github.com/euan-cowie/cidrator/actions/workflows/ci.yml)
[![Security Scan](https://img.shields.io/github/actions/workflow/status/euan-cowie/cidrator/ci.yml?branch=main&label=Security)](https://github.com/euan-cowie/cidrator/security/code-scanning)
[![Coverage](https://codecov.io/gh/euan-cowie/cidrator/branch/main/graph/badge.svg)](https://codecov.io/gh/euan-cowie/cidrator)
[![Go Report Card](https://goreportcard.com/badge/github.com/euan-cowie/cidrator)](https://goreportcard.com/report/github.com/euan-cowie/cidrator)

> **Comprehensive network analysis and manipulation toolkit built with Go**

Cidrator is a modern, fast, and feature-rich CLI tool for IPv4/IPv6 CIDR network analysis, DNS operations, network scanning, and firewall management. Designed with a clean `kubectl`-style interface and built for both interactive use and automation.

## ✨ Features

### 🌐 **CIDR Network Analysis** (Production Ready)
- **📊 Explain CIDR ranges** - Detailed network information with multiple output formats (table, JSON, YAML)
- **🔍 IP membership check** - Verify if IP addresses belong to CIDR ranges
- **🔢 Address counting** - Count total addresses in CIDR ranges with large number support
- **⚡ Overlap detection** - Check if two CIDR ranges overlap
- **✂️ Subnet division** - Split CIDR ranges into smaller subnets intelligently
- **📋 IP expansion** - List all individual IP addresses with safety limits
- **🌍 Full IPv6 support** - Complete feature parity between IPv4 and IPv6

### 🛣️ **Path-MTU Discovery & MTU Toolbox** (Production Ready)
- **🔍 Smart MTU Discovery** - RFC-compliant Path-MTU discovery using ICMP, TCP, or UDP probes
- **👀 Continuous Monitoring** - Watch mode with real-time change detection and alerting
- **🖥️ Interface Analysis** - Cross-platform network interface MTU enumeration
- **💡 Protocol Suggestions** - Calculate optimal frame sizes for TCP, VPN protocols, and more
- **🛡️ ICMP-Filtered Fallback** - PLPMTUD (RFC 4821) fallback for restrictive networks
- **🔒 Security Features** - Rate limiting, packet randomization, and retry throttling
- **🌐 Full Dual-Stack** - Complete IPv4 and IPv6 support across all probe methods

### 🚀 **Planned Features** (Coming Soon)
- **🔍 DNS Tools** - DNS lookups, reverse DNS, zone analysis, performance testing
- **🔎 Network Scanning** - Port scanning, ping sweeps, host discovery, service detection
- **🛡️ Firewall Management** - Rule generation, analysis, optimization, format conversion

## 🚀 Installation

### **Quick Install**

```bash
# Using Go (recommended)
go install github.com/euan-cowie/cidrator@latest

# Or download pre-built binaries
curl -sSL https://github.com/euan-cowie/cidrator/releases/latest/download/cidrator-$(uname -s)-$(uname -m).tar.gz | tar xz
```

### **Package Managers**

```bash
# Homebrew (macOS/Linux)
brew install euan-cowie/tap/cidrator

# Arch Linux (AUR)
yay -S cidrator-bin

### **From Source**

```bash
git clone https://github.com/euan-cowie/cidrator.git
cd cidrator
make build
sudo mv bin/cidrator /usr/local/bin/
```

## 📖 Quick Start

```bash
# Show all available command groups
cidrator --help

# Analyze a CIDR range
cidrator cidr explain 192.168.1.0/24

# Get JSON output for automation
cidrator cidr explain 10.0.0.0/16 --format json

# Check if IP is in range
cidrator cidr contains 192.168.1.0/24 192.168.1.100

# Discover Path-MTU to a destination
cidrator mtu discover google.com

# Watch for MTU changes (alerts on drops)
cidrator mtu watch example.com --interval 30s

# List network interfaces and their MTUs
cidrator mtu interfaces

# Get protocol-specific recommendations
cidrator mtu suggest 8.8.8.8 --json
```

## 📚 Documentation

### **Command Structure**

Cidrator uses a clean, `kubectl`-style subcommand structure:

```bash
cidrator <command-group> <command> [arguments] [flags]
```

**Available Command Groups:**
- `cidr` - IPv4/IPv6 CIDR network analysis and manipulation
- `mtu` - Path-MTU discovery and MTU analysis toolkit
- `dns` - DNS analysis and lookup tools *(coming soon)*
- `scan` - Network scanning and discovery *(coming soon)*
- `fw` - Firewall rule generation and analysis *(coming soon)*

### **MTU Commands**

#### **🔍 Path-MTU Discovery**

Discover the maximum packet size that can reach a destination without fragmentation:

```bash
# Basic ICMP-based discovery
$ cidrator mtu discover google.com
Discovering MTU to google.com...
Protocol: icmp, Range: 576-9216, Timeout: 2s
Target: google.com
Protocol: icmp
Path MTU: 1500
TCP MSS: 1460
Hops: 12
Elapsed: 234ms

# TCP-based discovery with JSON output
$ cidrator mtu discover example.com --proto tcp --json
{
  "target": "example.com",
  "protocol": "tcp",
  "pmtu": 1500,
  "mss": 1460,
  "hops": 10,
  "elapsed_ms": 190
}

# UDP-based discovery for VPN scenarios
$ cidrator mtu discover 8.8.8.8 --proto udp --max 1472
Target: 8.8.8.8
Protocol: udp
Path MTU: 1472
TCP MSS: 1432
Hops: 8
Elapsed: 145ms

# IPv6 discovery
$ cidrator mtu discover 2001:4860:4860::8888 --6
Target: 2001:4860:4860::8888
Protocol: icmp
Path MTU: 1500
TCP MSS: 1440
Hops: 11
Elapsed: 278ms
```

#### **👀 Continuous MTU Monitoring**

Monitor Path-MTU changes over time with alerting:

```bash
# Basic monitoring
$ cidrator mtu watch example.com --interval 30s
Watching MTU to example.com every 30s...
Press Ctrl+C to stop

[15:30:15]  MTU: 1500, MSS: 1460
[15:30:45]  MTU: 1500, MSS: 1460
[15:31:15]! MTU: 1472, MSS: 1432 (was 1500) ← CHANGED

# JSON output for logging/monitoring systems
$ cidrator mtu watch 8.8.8.8 --interval 10s --json
{"timestamp":"2023-12-01T15:30:15Z","target":"8.8.8.8","pmtu":1500,"mss":1460,"changed":false,"mss_changed":false}
{"timestamp":"2023-12-01T15:30:25Z","target":"8.8.8.8","pmtu":1500,"mss":1460,"changed":false,"mss_changed":false}

# Alert only on MSS changes
$ cidrator mtu watch corporate-vpn.example.com --mss-only --syslog
```

#### **🖥️ Interface MTU Analysis**

List network interfaces and their MTU configurations:

```bash
# Human-readable table
$ cidrator mtu interfaces
Interface       MTU    Type
--------------- ------ --------
lo0             16384  loopback
en0             1500   ethernet
en1             1500   ethernet
utun0           1380   tunnel
bridge0         1500   bridge

# JSON for automation
$ cidrator mtu interfaces --json
{
  "interfaces": [
    {"name": "lo0", "mtu": 16384, "type": "loopback"},
    {"name": "en0", "mtu": 1500, "type": "ethernet"},
    {"name": "utun0", "mtu": 1380, "type": "tunnel"}
  ]
}
```

#### **💡 Protocol Frame Size Suggestions**

Get optimal frame sizes for various protocols based on discovered Path-MTU:

```bash
# Protocol recommendations
$ cidrator mtu suggest example.com
Suggestions for example.com (PMTU: 1500):

TCP MSS (IPv4):      1460
TCP MSS (IPv6):      1440
WireGuard payload:   1440
IPSec ESP+UDP:       1416

# JSON output for configuration management
$ cidrator mtu suggest vpn-server.corp.com --json
{
  "target": "vpn-server.corp.com",
  "pmtu": 1472,
  "suggestions": {
    "tcp_mss_ipv4": 1432,
    "tcp_mss_ipv6": 1412,
    "wireguard_payload": 1412,
    "ipsec_esp_udp": 1388
  }
}
```

### **CIDR Commands**

#### **📊 Explain Network Details**

Get comprehensive information about CIDR ranges:

```bash
# Human-readable table (default)
$ cidrator cidr explain 10.0.0.0/16
Property              Value
--------              -----
Base Address          10.0.0.0
Usable Address Range  10.0.0.1 to 10.0.255.254 (65,534)
Broadcast Address     10.0.255.255
Total Addresses       65,536
Network Mask          255.255.0.0 (/16 bits)
Host Mask            0.0.255.255
Prefix Length         /16
Host Bits             16
IPv6                  false

# JSON for automation/scripting
$ cidrator cidr explain 10.0.0.0/24 --format json
{
  "base_address": "10.0.0.0",
  "broadcast_address": "10.0.0.255",
  "first_usable": "10.0.0.1",
  "last_usable": "10.0.0.254",
  "netmask": "255.255.255.0",
  "total_addresses": "256",
  "usable_addresses": "254",
  "is_ipv6": false
}

# YAML for configuration
$ cidrator cidr explain 2001:db8::/64 --format yaml
base_address: "2001:db8::"
first_usable: "2001:db8::"
last_usable: "2001:db8::ffff:ffff:ffff:ffff"
netmask: "ffff:ffff:ffff:ffff::"
total_addresses: "18,446,744,073,709,551,616"
is_ipv6: true
```

#### **📋 Expand IP Addresses**

List individual IP addresses with built-in safety limits:

```bash
# List all IPs in range
$ cidrator cidr expand 192.168.1.0/30
192.168.1.0
192.168.1.1
192.168.1.2
192.168.1.3

# Comma-separated output
$ cidrator cidr expand 192.168.1.0/30 --one-line
192.168.1.0, 192.168.1.1, 192.168.1.2, 192.168.1.3

# Safety limits prevent memory exhaustion
$ cidrator cidr expand 10.0.0.0/8
Error: CIDR range too large (16,777,216 addresses). Use a smaller range

# Custom limits for controlled output
$ cidrator cidr expand 10.0.0.0/28 --limit 5
Error: CIDR range contains 16 addresses, exceeds limit of 5
```

#### **🔍 IP Membership & Analysis**

```bash
# Check IP membership
$ cidrator cidr contains 10.0.0.0/16 10.0.14.5
true

$ cidrator cidr contains 192.168.1.0/24 10.0.0.1
false

# Count addresses
$ cidrator cidr count 10.0.0.0/16
65,536

$ cidrator cidr count 2001:db8::/64
18,446,744,073,709,551,616

# Check overlaps
$ cidrator cidr overlaps 10.0.0.0/16 10.0.14.0/22
true

$ cidrator cidr overlaps 192.168.1.0/24 172.16.0.0/16
false
```

#### **✂️ Subnet Division**

Intelligently split networks into smaller subnets:

```bash
# Divide IPv4 network
$ cidrator cidr divide 10.0.0.0/16 4
10.0.0.0/18
10.0.64.0/18
10.0.128.0/18
10.0.192.0/18

# Divide IPv6 network
$ cidrator cidr divide 2001:db8::/64 8
2001:db8::/67
2001:db8::800:0:0:0/67
2001:db8::1000:0:0:0/67
2001:db8::1800:0:0:0/67
2001:db8::2000:0:0:0/67
2001:db8::2800:0:0:0/67
2001:db8::3000:0:0:0/67
2001:db8::3800:0:0:0/67

# Automatic validation
$ cidrator cidr divide 192.168.1.0/30 8
Error: cannot divide 192.168.1.0/30 into 8 parts: insufficient host bits
```

## 🎯 Use Cases

### **Network Engineering**
- **Subnet planning** - Divide large networks into manageable subnets
- **IP inventory** - Count and list available addresses
- **Network validation** - Verify CIDR configurations
- **MTU optimization** - Discover and monitor optimal packet sizes
- **Performance troubleshooting** - Identify MTU misconfigurations causing fragmentation
- **Documentation** - Generate network documentation with JSON/YAML export

### **DevOps & Automation**
- **Infrastructure as Code** - Validate CIDR ranges in Terraform/CloudFormation
- **CI/CD pipelines** - Automate network validation and MTU testing
- **Monitoring** - Continuous MTU monitoring with alerting integration
- **VPN Configuration** - Automate optimal MTU settings for VPN tunnels
- **Container networking** - Validate network configurations in Kubernetes/Docker
- **Scripting** - JSON output for easy parsing in automation scripts

### **Security & Compliance**
- **Network segmentation** - Plan and validate network boundaries
- **MTU black hole detection** - Identify network changes affecting connectivity
- **Performance audits** - Ensure optimal network performance settings
- **Penetration testing** - Network reconnaissance and path analysis
- **Firewall rules** - Generate and validate IP ranges for rules *(coming soon)*
- **Security monitoring** - Track network configuration changes

## 🔧 Advanced Usage

### **Automation & Scripting**

```bash
# Parse JSON output with jq
cidrator cidr explain 10.0.0.0/24 --format json | jq '.total_addresses'

# Check multiple IPs against a range
for ip in 10.0.0.1 10.0.0.100 10.1.0.1; do
  echo "$ip: $(cidrator cidr contains 10.0.0.0/16 $ip)"
done

# Automated MTU testing for multiple hosts
hosts=(google.com cloudflare.com example.com)
for host in "${hosts[@]}"; do
  mtu=$(cidrator mtu discover "$host" --json | jq -r '.pmtu')
  echo "$host: MTU $mtu"
done

# VPN configuration automation
mtu=$(cidrator mtu discover vpn-server.corp.com --proto udp --json | jq -r '.pmtu')
wireguard_mtu=$((mtu - 60))
echo "WireGuard MTU: $wireguard_mtu"

# Monitor network health in CI/CD
if ! cidrator mtu discover critical-service.com --quiet --timeout 5s; then
  echo "MTU discovery failed - network issue detected"
  exit 1
fi
```

### **Performance & Limits**

- **Memory efficient** - Streaming algorithms for large ranges
- **Safety limits** - Automatic protection against memory exhaustion
- **Fast operations** - Optimized for both small and large CIDR ranges
- **Large number support** - Handles IPv6 address counts with `big.Int`

## 🏗️ Architecture

### **Project Structure**

```
cidrator/
├── cmd/                    # CLI commands organized by functionality
│   ├── root.go            # Root command and global configuration
│   ├── version.go         # Version information
│   ├── cidr/              # CIDR command group
│   │   ├── cidr.go        # CIDR parent command
│   │   ├── config.go      # Configuration management
│   │   ├── explain.go     # Network analysis with multiple formats
│   │   ├── expand.go      # IP address expansion with safety limits
│   │   ├── contains.go    # IP membership checking
│   │   ├── count.go       # Address counting with big number support
│   │   ├── overlaps.go    # Network overlap detection
│   │   └── divide.go      # Intelligent subnet division
│   ├── mtu/               # MTU command group
│   │   ├── mtu.go         # MTU parent command
│   │   ├── discover.go    # MTU discovery command interface
│   │   ├── discovery.go   # Core MTU discovery algorithms
│   │   ├── watch.go       # Continuous monitoring
│   │   ├── interfaces.go  # Interface enumeration
│   │   ├── suggest.go     # Protocol recommendations
│   │   ├── interface_detector.go # Cross-platform interface detection
│   │   ├── tcp_udp_probes.go     # TCP/UDP probe implementations
│   │   ├── plpmtud.go     # RFC 4821 PLPMTUD fallback
│   │   └── security.go    # Rate limiting & packet randomization
│   ├── dns/               # DNS command group (scaffold)
│   ├── scan/              # Scanning command group (scaffold)
│   └── fw/                # Firewall command group (scaffold)
├── internal/
│   ├── cidr/              # Core CIDR functionality
│   │   ├── cidr.go        # Network calculations & data structures
│   │   ├── errors.go      # Typed error handling
│   │   └── formatter.go   # Output formatting interfaces
│   └── validation/        # Input validation layer
│       └── network.go     # Centralized network input validation
├── .github/               # GitHub workflows and issue templates
│   ├── workflows/         # CI/CD pipelines
│   └── ISSUE_TEMPLATE/    # Issue templates
├── docs/                  # Additional documentation
├── examples/              # Usage examples and scripts
└── scripts/               # Build and development scripts
```

### **Design Principles**

- **🧩 Modular Architecture** - Clean separation of concerns with interfaces
- **🎯 Single Responsibility** - Each package has a focused purpose
- **🔒 Type Safety** - Comprehensive error handling with typed errors
- **⚡ Performance** - Optimized algorithms with safety limits
- **🧪 Test Coverage** - Comprehensive test suite with 95%+ coverage
- **📚 Documentation** - Self-documenting code with clear interfaces

## 🚀 What Makes Cidrator Different

### **🎨 Modern Architecture**
- **kubectl-style commands** - Intuitive, organized command structure
- **Type-safe operations** - Comprehensive error handling and validation
- **Memory efficient** - Smart algorithms prevent resource exhaustion
- **Future-ready** - Extensible design for new networking tools

### **🔧 Advanced CIDR Features**
- **Multiple output formats** - Human-readable tables, JSON, YAML
- **Safety-first design** - Built-in protection against large range expansion
- **IPv6 excellence** - Complete feature parity with IPv4
- **Big number support** - Handles massive IPv6 address spaces correctly

### **🛠️ Developer Experience**
- **Clean JSON output** - Perfect for automation and scripting
- **Comprehensive validation** - Clear error messages for invalid inputs
- **Cross-platform** - Native binaries for Linux, macOS
- **Zero dependencies** - Single binary with no external requirements

## 📊 Comparison

| Feature | Cidrator | `ping` | `tracepath` | `iperf3` |
|---------|----------|--------|-------------|----------|
| **Path-MTU Discovery** | ✅ Multi-protocol | ⚠️ Basic ICMP | ⚠️ IPv4 only | ❌ No |
| **Continuous Monitoring** | ✅ Built-in | ❌ Manual | ❌ Manual | ❌ No |
| **JSON Output** | ✅ Structured | ❌ No | ❌ No | ⚠️ Limited |
| **Multiple Protocols** | ✅ ICMP/TCP/UDP | ❌ ICMP only | ❌ UDP only | ✅ TCP/UDP |
| **ICMP Fallback** | ✅ PLPMTUD | ❌ No | ❌ No | ❌ No |
| **IPv6 Support** | ✅ Full | ✅ Yes | ⚠️ Limited | ✅ Yes |
| **Security Features** | ✅ Rate limiting | ❌ Basic | ❌ No | ❌ No |
| **Cross-platform** | ✅ All platforms | ✅ Universal | ⚠️ Linux/BSD | ✅ All platforms |

## 🛣️ Roadmap

### **Phase 1: Core CIDR** ✅ *Complete*
- ✅ IPv4/IPv6 CIDR analysis
- ✅ Multiple output formats
- ✅ Safety limits and validation
- ✅ Comprehensive test coverage

### **Phase 2: Path-MTU Discovery** ✅ *Complete*
- ✅ Multi-protocol MTU discovery (ICMP, TCP, UDP)
- ✅ Continuous monitoring with change detection
- ✅ Cross-platform interface enumeration
- ✅ Protocol-specific frame size recommendations
- ✅ PLPMTUD fallback for ICMP-filtered networks
- ✅ Security features and rate limiting

### **Phase 3: DNS Tools** 🚧 *In Progress*
- 🔄 DNS record lookups (A, AAAA, MX, TXT, etc.)
- 🔄 Reverse DNS lookups
- 🔄 DNS server performance testing
- 🔄 Zone file analysis

### **Phase 4: Network Scanning** 📅 *Planned*
- 📅 Port scanning with multiple techniques
- 📅 Host discovery and ping sweeps
- 📅 Service detection and OS fingerprinting
- 📅 Network topology mapping

### **Phase 5: Firewall Management** 📅 *Future*
- 📅 Multi-format rule generation (iptables, pf, cisco)
- 📅 Configuration analysis and optimization
- 📅 Security policy validation
- 📅 Rule conflict detection

## 🤝 Contributing

**Want to contribute? It's super easy!**

```bash
# 1. Fork and clone
git clone https://github.com/YOUR_USERNAME/cidrator.git
cd cidrator

# 2. One-time setup
make setup

# 3. Make changes and test
make dev

# 4. Submit PR
git commit -m "feat: your change"
git push origin your-branch
```

**That's it!** See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

**Looking for something to work on?**
- 🏷️ [`good first issue`](https://github.com/euan-cowie/cidrator/labels/good%20first%20issue) - Perfect for newcomers
- 🚀 [`help wanted`](https://github.com/euan-cowie/cidrator/labels/help%20wanted) - Ready for contributors
- 💡 [Discussions](https://github.com/euan-cowie/cidrator/discussions) - Share ideas

## 📝 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Inspired by**: [bschaatsbergen/cidr](https://github.com/bschaatsbergen/cidr), [pda/cidrinfo](https://github.com/pda/cidrinfo)
- **CLI framework**: [spf13/cobra](https://github.com/spf13/cobra) and [spf13/viper](https://github.com/spf13/viper)
- **Architecture inspiration**: [kubectl](https://kubernetes.io/docs/reference/kubectl/) command structure

## 📞 Support

- **🐛 Bug Reports**: [GitHub Issues](https://github.com/euan-cowie/cidrator/issues)
- **💡 Feature Requests**: [GitHub Discussions](https://github.com/euan-cowie/cidrator/discussions)
- **📖 Documentation**: [Wiki](https://github.com/euan-cowie/cidrator/wiki)
- **💬 Community**: [Discussions](https://github.com/euan-cowie/cidrator/discussions)

---

<div align="center">

**⭐ Star us on GitHub — it motivates us a lot!**

[Report Bug](https://github.com/euan-cowie/cidrator/issues) · [Request Feature](https://github.com/euan-cowie/cidrator/discussions) · [Contribute](CONTRIBUTING.md)

Made with ❤️ by [Euan Cowie](https://github.com/euan-cowie) and [contributors](https://github.com/euan-cowie/cidrator/contributors)

</div>
