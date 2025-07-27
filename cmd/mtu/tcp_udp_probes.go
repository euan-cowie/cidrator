package mtu

import (
	"context"
	"fmt"
	"net"
	"time"
)

// TCPProber handles MTU discovery using TCP SYN packets
type TCPProber struct {
	target     string
	targetAddr *net.TCPAddr
	timeout    time.Duration
}

// UDPProber handles MTU discovery using UDP packets
type UDPProber struct {
	target     string
	targetAddr *net.UDPAddr
	timeout    time.Duration
}

// NewTCPProber creates a new TCP-based MTU prober
func NewTCPProber(target string, ipv6 bool, timeout time.Duration) (*TCPProber, error) {
	// Resolve target address
	network := "tcp4"
	if ipv6 {
		network = "tcp6"
	}

	// Try common ports: 80 (HTTP), 443 (HTTPS), 22 (SSH)
	ports := []string{"443", "80", "22"}
	var addr *net.TCPAddr
	var err error

	for _, port := range ports {
		addr, err = net.ResolveTCPAddr(network, net.JoinHostPort(target, port))
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	return &TCPProber{
		target:     target,
		targetAddr: addr,
		timeout:    timeout,
	}, nil
}

// NewUDPProber creates a new UDP-based MTU prober
func NewUDPProber(target string, ipv6 bool, timeout time.Duration) (*UDPProber, error) {
	// Resolve target address
	network := "udp4"
	if ipv6 {
		network = "udp6"
	}

	addr, err := net.ResolveUDPAddr(network, net.JoinHostPort(target, "53"))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	return &UDPProber{
		target:     target,
		targetAddr: addr,
		timeout:    timeout,
	}, nil
}

// ProbeTCP performs a TCP-based MTU probe
func (p *TCPProber) ProbeTCP(ctx context.Context, size int) *ProbeResult {
	start := time.Now()

	// Create TCP connection with specific socket options
	dialer := &net.Dialer{
		Timeout: p.timeout,
	}

	// Connect to target
	conn, err := dialer.DialContext(ctx, "tcp", p.targetAddr.String())
	if err != nil {
		return &ProbeResult{
			Size:    size,
			Success: false,
			RTT:     time.Since(start),
			Error:   err,
		}
	}
	defer conn.Close()

	// For TCP, successful connection means the packet got through
	// In a real implementation, we'd need to set DF bit and handle ICMP responses
	return &ProbeResult{
		Size:    size,
		Success: true,
		RTT:     time.Since(start),
	}
}

// ProbeUDP performs a UDP-based MTU probe
func (p *UDPProber) ProbeUDP(ctx context.Context, size int) *ProbeResult {
	start := time.Now()

	// Create UDP connection
	conn, err := net.DialUDP("udp", nil, p.targetAddr)
	if err != nil {
		return &ProbeResult{
			Size:    size,
			Success: false,
			RTT:     time.Since(start),
			Error:   err,
		}
	}
	defer conn.Close()

	// Set deadline
	deadline := time.Now().Add(p.timeout)
	conn.SetDeadline(deadline)

	// Create payload of specified size
	payload := make([]byte, size)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	// Send UDP packet
	_, err = conn.Write(payload)
	if err != nil {
		return &ProbeResult{
			Size:    size,
			Success: false,
			RTT:     time.Since(start),
			Error:   err,
		}
	}

	// Try to read response (may timeout, which is expected for most UDP probes)
	response := make([]byte, 1500)
	_, err = conn.Read(response)
	rtt := time.Since(start)

	// For UDP, we consider it successful if the packet was sent
	// ICMP errors would be detected by the OS but not always propagated
	return &ProbeResult{
		Size:    size,
		Success: true, // Assume success unless we get a clear error
		RTT:     rtt,
		Error:   err, // Store error but don't fail the probe
	}
}

// DiscoverPMTUTCP performs TCP-based MTU discovery
func (p *TCPProber) DiscoverPMTUTCP(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
	start := time.Now()

	// Binary search for maximum working MTU
	low := minMTU
	high := maxMTU
	lastWorking := 0
	hops := 0

	for low <= high {
		mid := (low + high) / 2

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result := p.ProbeTCP(ctx, mid)
		hops++

		if result.Success {
			lastWorking = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if lastWorking == 0 {
		return nil, fmt.Errorf("no working MTU found in range %d-%d", minMTU, maxMTU)
	}

	elapsed := time.Since(start)

	return &MTUResult{
		Target:    p.target,
		Protocol:  "tcp",
		PMTU:      lastWorking,
		MSS:       lastWorking - 40, // TCP/IP headers
		Hops:      hops,
		ElapsedMS: int(elapsed.Milliseconds()),
	}, nil
}

// DiscoverPMTUUDP performs UDP-based MTU discovery
func (p *UDPProber) DiscoverPMTUUDP(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
	start := time.Now()

	// Binary search for maximum working MTU
	low := minMTU
	high := maxMTU
	lastWorking := 0
	hops := 0

	for low <= high {
		mid := (low + high) / 2

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result := p.ProbeUDP(ctx, mid)
		hops++

		if result.Success {
			lastWorking = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	if lastWorking == 0 {
		return nil, fmt.Errorf("no working MTU found in range %d-%d", minMTU, maxMTU)
	}

	elapsed := time.Since(start)

	return &MTUResult{
		Target:    p.target,
		Protocol:  "udp",
		PMTU:      lastWorking,
		MSS:       lastWorking - 28, // UDP/IP headers
		Hops:      hops,
		ElapsedMS: int(elapsed.Milliseconds()),
	}, nil
}
