# MTU Discovery & Path-MTU Analysis

> **Comprehensive Path-MTU discovery toolkit with RFC-compliant algorithms and advanced monitoring capabilities**

The `cidrator mtu` command group provides a complete solution for Path-MTU discovery, network interface analysis, and continuous MTU monitoring. Built on RFC 1191 (IPv4), RFC 8201 (IPv6), and RFC 4821 (PLPMTUD) standards.

## 🎯 Overview

Path-MTU discovery answers three critical questions:
- **What MTU can I safely send to that host?**
- **Did today's change introduce an MTU black-hole?**
- **What MSS or VPN segment size should I configure?**

## 📋 Command Reference

### `cidrator mtu discover`

Performs binary-search MTU discovery to find the largest packet size that reaches the destination without fragmentation.

```bash
cidrator mtu discover <destination> [flags]
```

#### **Global Flags**
- `--4` / `--6` - Force IPv4 or IPv6
- `--proto icmp|udp|tcp` - Probe method (default: icmp)
- `--min <size>` - Lower bound (IPv4: 576, IPv6: 1280)
- `--max <size>` - Upper bound (default: 9216)
- `--step <size>` - Granularity for linear sweep fallback (default: 16)
- `--timeout <duration>` - Wait per probe (default: 2s)
- `--ttl <hops>` - Initial hop limit (default: 64)
- `--pps <rate>` - Rate limit probes per second (default: 10)
- `--json` - Structured output
- `--quiet` - Suppress progress information

#### **Examples**

```bash
# Basic ICMP discovery
cidrator mtu discover google.com

# TCP-based discovery (useful for firewalled networks)
cidrator mtu discover example.com --proto tcp

# UDP discovery for VPN scenarios
cidrator mtu discover vpn-server.corp.com --proto udp

# IPv6 with custom range
cidrator mtu discover 2001:4860:4860::8888 --6 --min 1280 --max 1500

# JSON output for automation
cidrator mtu discover 8.8.8.8 --json
```

#### **Output Format**

**Human-readable:**
```
Target: google.com
Protocol: icmp
Path MTU: 1500
TCP MSS: 1460
Hops: 12
Elapsed: 234ms
```

**JSON:**
```json
{
  "target": "google.com",
  "protocol": "icmp",
  "pmtu": 1500,
  "mss": 1460,
  "hops": 12,
  "elapsed_ms": 234
}
```

### `cidrator mtu watch`

Continuously monitors Path-MTU and alerts on changes. Essential for detecting network configuration changes and MTU black holes.

```bash
cidrator mtu watch <destination> [flags]
```

#### **Specific Flags**
- `--interval <duration>` - Check interval (default: 10s)
- `--mss-only` - Only alert on MSS changes
- `--syslog` - Send alerts to syslog

#### **Examples**

```bash
# Basic monitoring
cidrator mtu watch critical-service.com

# Custom interval with JSON output
cidrator mtu watch example.com --interval 30s --json

# MSS-only monitoring for application tuning
cidrator mtu watch api.service.com --mss-only

# Production monitoring with syslog
cidrator mtu watch production-db.corp.com --syslog --interval 60s
```

#### **Exit Behavior**
- Exit code `0` - Normal operation
- Exit code `1` - PMTU decreased (indicates potential network issue)

### `cidrator mtu interfaces`

Lists all network interfaces with their configured MTU values. Useful for baseline analysis and auto-detection of maximum MTU.

```bash
cidrator mtu interfaces [flags]
```

#### **Examples**

```bash
# Human-readable table
cidrator mtu interfaces

# JSON for automation
cidrator mtu interfaces --json
```

#### **Output Format**

**Table:**
```
Interface       MTU    Type
--------------- ------ --------
lo0             16384  loopback
en0             1500   ethernet
utun0           1380   tunnel
bridge0         1500   bridge
```

**JSON:**
```json
{
  "interfaces": [
    {"name": "lo0", "mtu": 16384, "type": "loopback"},
    {"name": "en0", "mtu": 1500, "type": "ethernet"},
    {"name": "utun0", "mtu": 1380, "type": "tunnel"}
  ]
}
```

### `cidrator mtu suggest`

Calculates optimal frame sizes for various protocols based on the discovered Path-MTU.

```bash
cidrator mtu suggest <destination> [flags]
```

#### **Examples**

```bash
# Protocol recommendations
cidrator mtu suggest vpn-server.corp.com

# JSON for configuration automation
cidrator mtu suggest example.com --json
```

#### **Calculations**

- **TCP MSS (IPv4):** PMTU - 40 (IP header: 20, TCP header: 20)
- **TCP MSS (IPv6):** PMTU - 60 (IPv6 header: 40, TCP header: 20)
- **WireGuard payload:** PMTU - 60 (WireGuard overhead)
- **IPSec ESP+UDP:** PMTU - 84 (ESP + UDP + IP overhead)

## 🔬 Technical Implementation

### **Discovery Algorithms**

#### **Binary Search (Default)**
1. Start at `--max` MTU size
2. If successful, try larger size (binary search up)
3. If ICMP "Too Big" received, try smaller size (binary search down)
4. Continue until optimal size found

#### **Linear Sweep (Fallback)**
- Used when ICMP is filtered or unreliable
- Increments by `--step` size from `--min` to `--max`
- More thorough but slower than binary search

### **Probe Protocols**

#### **ICMP (Default)**
- Uses ICMP Echo Request packets
- Requires raw socket access (may need privileges)
- Most accurate for true Path-MTU discovery
- Handles ICMP "Packet Too Big" responses correctly

#### **TCP**
- Establishes TCP connections with varying MSS
- Works through most firewalls (ports 443, 80, 22)
- No raw socket privileges required
- Success = connection established, failure = RST or timeout

#### **UDP**
- Sends UDP packets to DNS port (53)
- Simulates VPN handshake MTU testing
- Good fallback when ICMP is blocked
- No privileges required

### **PLPMTUD Fallback (RFC 4821)**

When ICMP is completely filtered:
1. Switch to "raise size in-band" strategy
2. Gradually increase packet sizes with application data
3. Confirm successful delivery through application layer
4. More reliable in ICMP-hostile networks

### **Security Features**

#### **Rate Limiting**
- Configurable packets-per-second limit
- Prevents network flooding
- Default: 10 PPS for politeness

#### **Packet Randomization**
- Random packet IDs and sequence numbers
- Cryptographically random payload data
- Prevents fingerprinting and detection

#### **Retry Throttling**
- Exponential backoff with jitter
- Prevents burst retries
- Respects network conditions

## 🚀 Use Cases

### **Network Operations**

```bash
# Pre-deployment MTU validation
cidrator mtu discover new-service.corp.com --json > mtu-baseline.json

# Post-change verification
cidrator mtu discover service.corp.com --json | jq '.pmtu' > new-mtu.txt
if [ "$(cat new-mtu.txt)" != "$(cat baseline-mtu.txt)" ]; then
  echo "MTU changed after deployment!"
fi
```

### **VPN Configuration**

```bash
# WireGuard MTU optimization
PMTU=$(cidrator mtu discover vpn-server.example.com --proto udp --json | jq -r '.pmtu')
WG_MTU=$((PMTU - 60))
echo "MTU = $WG_MTU" >> wg0.conf
```

### **Application Tuning**

```bash
# TCP MSS clamping for iptables
MSS=$(cidrator mtu discover app-server.corp.com --json | jq -r '.mss')
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --set-mss $MSS
```

### **Monitoring & Alerting**

```bash
#!/bin/bash
# Production MTU monitoring script
SERVICES=(web-app.corp.com api.corp.com db.corp.com)

for service in "${SERVICES[@]}"; do
  if ! cidrator mtu watch "$service" --quiet --timeout 10s >/dev/null 2>&1; then
    echo "ALERT: MTU issue detected for $service" | mail -s "MTU Alert" ops@corp.com
  fi
done
```

## 🔧 Troubleshooting

### **Common Issues**

#### **Permission Denied (ICMP)**
```bash
# Solution 1: Use non-ICMP protocols
cidrator mtu discover target.com --proto tcp

# Solution 2: Run with appropriate privileges
sudo cidrator mtu discover target.com
```

#### **Timeouts**
```bash
# Increase timeout for slow networks
cidrator mtu discover target.com --timeout 5s

# Use UDP for faster probing
cidrator mtu discover target.com --proto udp
```

#### **ICMP Filtered Networks**
```bash
# TCP fallback
cidrator mtu discover target.com --proto tcp

# PLPMTUD for completely filtered networks
# (Requires application-layer cooperation)
```

### **Debugging**

```bash
# Verbose output
cidrator mtu discover target.com --json | jq '.'

# Interface analysis first
cidrator mtu interfaces --json

# Check with different protocols
for proto in icmp tcp udp; do
  echo "Testing $proto:"
  cidrator mtu discover target.com --proto $proto
done
```

## 📊 Performance Considerations

### **Speed vs Accuracy Trade-offs**

- **ICMP:** Most accurate, may require privileges
- **TCP:** Good compromise, widely accepted
- **UDP:** Fastest, least accurate for true PMTU

### **Network Politeness**

- Default rate limiting (10 PPS) prevents network abuse
- Exponential backoff respects network conditions
- Randomization prevents detection/blocking

### **Memory Usage**

- Streaming algorithms for efficiency
- No large buffer allocations
- Suitable for resource-constrained environments

## 🌐 Cross-Platform Support

### **Linux**
- Full feature support
- Raw socket ICMP
- Interface enumeration via `/sys/class/net`

### **macOS**
- Full feature support
- Raw socket ICMP (may need sudo)
- Interface enumeration via system calls

### **BSD/Others**
- Core functionality supported
- May need platform-specific tuning

## 🔗 Related RFCs

- **RFC 1191** - Path MTU Discovery (IPv4)
- **RFC 8201** - Path MTU Discovery for IP version 6
- **RFC 4821** - Packetization Layer Path MTU Discovery
- **RFC 1981** - Path MTU Discovery for IPv6 (obsoleted by RFC 8201)

## 💡 Best Practices

1. **Use appropriate protocols** - ICMP for accuracy, TCP for compatibility
2. **Monitor continuously** - Use watch mode for critical services
3. **Automate configuration** - Use JSON output for VPN/application tuning
4. **Respect networks** - Keep default rate limits unless necessary
5. **Test both protocols** - IPv4 and IPv6 may have different MTUs
6. **Consider security** - Some networks filter/block MTU discovery

---

For more examples and advanced usage, see the main [README.md](../README.md) and [examples](../examples/) directory.
