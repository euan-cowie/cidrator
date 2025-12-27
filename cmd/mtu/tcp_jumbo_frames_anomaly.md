# The "9216 Problem": Why TCP Reports Jumbo Frames

During verification of the `cidrator` tool, we observed a discrepancy between UDP and TCP MTU discovery results over the same internet path:

- **UDP Discovery:** `1472` (+28 headers = **1500 bytes**) ✅ Correct
- **TCP Discovery:** `9216` (+40 headers = **9256 bytes**) ⚠️ Anomaly

This document explains why this happens and why UDP is generally more reliable for *Path* MTU Discovery (PMTUD).

## The Anomaly

The value **9216** is significant: it is the standard MTU for **Jumbo Frames** in AWS VPCs and many data centers.

However, the path between a residential/office internet connection and AWS traverses the public internet, which has a hard limit of **1500 bytes**. It is physically impossible for a 9216-byte packet to traverse the internet without fragmentation.

## Cause 1: TCP Segmentation Offload (TSO/GSO)

Modern network interface cards (NICs) and operating systems use **TCP Segmentation Offload (TSO)** or **Generic Segmentation Offload (GSO)** to reduce CPU load.

1.  **Application View:** The application (cidrator) writes a large buffer (e.g., 9000 bytes) to the TCP socket.
2.  **OS Kernel View:** The kernel accepts this large payload because the virtual network interface claims to support it.
3.  **Hardware Reality:** The NIC (or the virtualized driver in EC2/Localhost) breaks this large segment into valid 1500-byte packets *transparently*.

Because `cidrator` successfully executed `conn.Write(9000_bytes)` and received an echo response (which was also automatically reassembled by the receiver's stack), the tool **falsely believes** a 9000-byte packet traversed the network intact.

## Cause 2: Middlebox Reassembly

Stateful firewalls and NAT gateways often perform **Transparent Reassembly**:

1.  Sender transmits fragmented IP packets (or segmented TCP).
2.  Middlebox (Firewall/NAT) receives them all.
3.  Middlebox reassembles the stream to inspect the content (DPI).
4.  Middlebox re-fragments or re-segments the data to forward it to the destination.

This hides the Path MTU from the endpoints. The TCP connection stays alive and transfers data, but the "atomic" property of the packet size is lost.

## Why UDP is Different

UDP is message-oriented, not stream-oriented.

1.  **Atomic Packets:** When `cidrator` sets the **Don't Fragment (DF)** bit on a UDP packet, TSO/GSO logic typically applies differently or not at all (depending on OS).
2.  **Drop Behavior:** If a 9000-byte UDP packet with `DF=1` hits a 1500-byte link:
    - The router **Must Drop** it.
    - It sends (or should send) an ICMP "Fragmentation Needed" error.
    - `cidrator` sees the timeout or ICMP error and correctly marks that size as "failed".

## Case Study: Local Client to EC2 Server

We performed a verified test to demonstrate this behavior despite implementing RFC-compliant mitigation.

**Setup:**
- **Server:** AWS EC2 Instance (`t3.micro`, Amazon Linux 2023) running in `eu-west-1`.
  - Command: `mtu server --port 4821 --proto udp,tcp`
- **Client:** Local Mac Accessing via Public Internet.
  - Fix Applied: `TCP_MAXSEG` socket option set to match probe size.
  - Command: `mtu discover 54.154.159.28 --port 4821 --proto tcp`

**Results:**

| Protocol | Path MTU Result | Verdict |
|----------|----------------|---------|
| **UDP**  | `1472` (+28 = 1500) | ✅ **Accurate** (Reflects physical wire limit) |
| **TCP**  | `9216` (+40 = 9256) | ⚠️ **Obscured** (TSO/GSO hid the fragmentation) |

**Analysis:**
Even with the explicit `TCP_MAXSEG` socket option set by `cidrator`, the modern virtualization stack (AWS Nitro / host OS) or the client OS performed "Super-packet" optimization. The kernel accepted the large segments and handled the segmentation/reassembly transparently, confirming that **Application-Layer TCP Probing is insufficient for measuring Physical Path MTU** in cloud environments.

## Conclusion

The "9216 Problem" is a false positive caused by modern network stack optimizations (Off-loading) that are beneficial for throughput but detrimental for accurate *Path* MTU diagnosis.

**Recommendation:** Always prefer **UDP** (or **ICMP**) for Path MTU Discovery to measure the physical properties of the network path. Use TCP only if you specifically want to test the MSS (Maximum Segment Size) negotiation or application-layer throughput behavior.
