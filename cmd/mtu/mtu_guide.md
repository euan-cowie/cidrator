# Deep Dive: MTU, Fragmentation, and PMTUD

This guide explains the fundamental networking concepts behind Maximum Transmission Unit (MTU), IP Fragmentation, and Path MTU Discovery (PMTUD). It also details how `cidrator` implements these standards to robustly verify network path characteristics.

## 1. What is MTU?

**MTU (Maximum Transmission Unit)** is the size (in bytes) of the largest Protocol Data Unit (PDU) that can be communicated in a single network layer transaction.

- **Ethernet Default**: The standard MTU for Ethernet is **1500 bytes**.
- **Jumbo Frames**: Some networks support larger frames, typically up to **9000 bytes**.
- **Tunneling Overhead**: Protocols like GRE, IPsec, and WireGuard add headers to packets. If a 1500-byte packet is encapsulated in a GRE tunnel, the resulting packet exceeds 1500 bytes, which may not fit on the physical wire.

When a packet is larger than the MTU of the next hop in its path, one of two things must happen:
1. **Fragmentation**: The router splits the packet into smaller chunks (IPv4 only, if allowed).
2. **Drop**: The router discards the packet and (ideally) sends an error message back to the sender.

## 2. IP Fragmentation

Fragmentation occurs when an IP packet is too large to be transmitted across a link.

### IPv4
If the **DF (Don't Fragment)** bit in the IP header is **NOT** set (0), a router can split the packet into fragments. The destination host reassembles them.
- **Pros**: Connectivity is maintained even if MTU is mismatched.
- **Cons**: High CPU overhead on routers/hosts; if one fragment is lost, the entire packet must be retransmitted; firewalls may drop fragments.

### IPv6
**IPv6 routers DO NOT fragment packets.** Only the sender can perform fragmentation. If a packet is too big, the router must drop it and send an ICMPv6 "Packet Too Big" message back to the source.

## 3. Path MTU Discovery (PMTUD)

PMTUD is the standardized technique for determining the maximum packet size that can traverse a path without fragmentation.

### Classical PMTUD (RFC 1191 for IPv4, RFC 8201 for IPv6)
1. **The Sender** sets the **DF (Don't Fragment)** bit on all outgoing packets.
2. **The Goal**: Send packets as large as the local interface supports.
3. **The Bottleneck**: If a router encounters a packet larger than the next link's MTU:
    - It sees `DF=1`, so it cannot fragment.
    - It **drops** the packet.
    - It sends an **ICMP Type 3, Code 4** (IPv4) or **ICMPv6 Type 2** (IPv6) message back to the sender: "Destination Unreachable: Fragmentation Needed".
    - Crucially, this message contains the **Next-Hop MTU**—the size limit of the constriction.
4. **The Adjustment**: The sender receives this ICMP message, lowers its PMTU estimate, and retransmits.

### The Problem: PMTUD Black Holes
A "Black Hole" occurs when ICMP messages are blocked by firewalls or misconfigured routers.
1. Sender sends valid large packet (DF=1).
2. Router drops it (too big).
3. Router tries to send ICMP "Fragmentation Needed".
4. **Firewall blocks the ICMP message.**
5. Sender never knows why the packet was lost. It retransmits, fails again, and the connection hangs (e.g., HTTPS loads headers but images spin forever).

## 4. How `cidrator` Addresses These Issues

`cidrator` provides a complete toolkit for analyzing and solving MTU issues, implementing modern RFC standards to handle edge cases like Black Holes.

### A. Strict RFC 1191/8201 Compliance (ICMP Mode)
When you run `cidrator mtu discover target.com`:
1. It opens a raw socket to listen for **ICMP Fragmentation Needed** messages.
2. It sends probe packets with the **DF bit explicitly set** (using platform-specific syscalls).
3. If it receives an ICMP error, it reads the **Next-Hop MTU** directly from the packet, exactly as an operating system kernel would, to instantly find the bottleneck.

### B. Robust PLPMTUD (Packetization Layer PMTUD - RFC 4821)
To defeat Black Holes, `cidrator` implements **PLPMTUD**.
- Instead of relying on ICMP error messages (which might be blocked), it treats **packet loss** as a sign that the MTU was too large.
- It uses a **Binary Search** or **Linear Sweep** strategy:
    - Send large probe → Timeout? → Assume MTU too big → Try smaller.
    - Send small probe → Success? → MTU is at least this big → Try larger.
- This works even if the network is completely silent/filtered.

### C. Protocol-Specific Probing (RFC 8899)
MTU can vary by protocol (e.g., some middleboxes treat UDP differently than ICMP).
- **`--proto udp`**: Implements **RFC 8899**. It sends UDP probes to a `cidrator mtu server` which echoes them back. This verifies the path is valid **application-to-application**.
- **`--proto tcp`**: Establishes a real TCP connection and pushes data segments with DF=1 to verify the TCP Path MTU, avoiding the common pitfall where "TCP Ping" (SYN packets) falsely reports a high MTU because SYN packets are small.

### D. Practical Suggestions
Because overhead matters, `cidrator` calculates the safe **MSS (Maximum Segment Size)** and tunnel MTUs for you:
- **WireGuard**: `PMTU - 60 bytes` (IPv4) or `PMTU - 80 bytes` (IPv6).
- **IPsec ESP**: `PMTU - 70+ bytes` (variable).
- **TCP MSS**: `PMTU - 40 bytes`.

## Summary Table

| Standard | Description | `cidrator` Support |
| :--- | :--- | :--- |
| **RFC 791** | IPv4 standard, defining DF bit and Fragmentation. | ✅ Enforces DF bit on all probes. |
| **RFC 1191** | Classical PMTUD for IPv4 (ICMP-based). | ✅ Full listener implementation. |
| **RFC 8201** | PMTUD for IPv6 (Packet Too Big). | ✅ Full IPv6 support. |
| **RFC 4821** | PLPMTUD (Probing without ICMP). | ✅ Used for Black Hole detection. |
| **RFC 8899** | Datagram PLPMTUD (UDP). | ✅ Via `cidrator mtu server`. |

## Further Reading
- [Cisco: Resolve IPv4 Fragmentation, MTU, MSS, and PMTUD Issues](https://www.cisco.com/c/en/us/support/docs/ip/generic-routing-encapsulation-gre/25885-pmtud-ipfrag.html)
- [Cloudflare: Path MTU Discovery in Practice](https://blog.cloudflare.com/path-mtu-discovery-in-practice/)
- [Packet Pushers: IP Fragmentation in Detail](https://packetpushers.net/blog/ip-fragmentation-in-detail/)
