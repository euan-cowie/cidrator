package mtu

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// ProbeResult represents the result of a single MTU probe
type ProbeResult struct {
	Size    int
	Success bool
	RTT     time.Duration
	Error   error
	ICMPErr *ICMPError
}

// ICMPError contains details about ICMP error messages
type ICMPError struct {
	Type    int
	Code    int
	Message string
}

// MTUDiscoverer handles Path-MTU discovery
type MTUDiscoverer struct {
	target     string
	ipv6       bool
	protocol   string
	timeout    time.Duration
	ttl        int
	conn       net.PacketConn
	targetAddr net.Addr
	security   *SecurityConfig
}

// NewMTUDiscoverer creates a new MTU discovery instance
func NewMTUDiscoverer(target string, ipv6 bool, protocol string, timeout time.Duration, ttl int) (*MTUDiscoverer, error) {
	d := &MTUDiscoverer{
		target:   target,
		ipv6:     ipv6,
		protocol: protocol,
		timeout:  timeout,
		ttl:      ttl,
		security: NewSecurityConfig(10), // Default 10 pps
	}

	// For non-ICMP protocols, we don't need to setup raw sockets immediately
	if protocol == "icmp" {
		// Resolve target address
		if err := d.resolveTarget(); err != nil {
			return nil, fmt.Errorf("failed to resolve target: %w", err)
		}

		// Setup network connection
		if err := d.setupConnection(); err != nil {
			return nil, fmt.Errorf("failed to setup connection: %w", err)
		}
	}

	return d, nil
}

// resolveTarget resolves the target hostname to an IP address
func (d *MTUDiscoverer) resolveTarget() error {

	// Try to parse as IP first
	if ip := net.ParseIP(d.target); ip != nil {
		if d.ipv6 && ip.To4() != nil {
			return fmt.Errorf("IPv4 address provided but IPv6 requested")
		}
		if !d.ipv6 && ip.To4() == nil {
			return fmt.Errorf("IPv6 address provided but IPv4 requested")
		}
		d.targetAddr = &net.IPAddr{IP: ip}
		return nil
	}

	// Resolve hostname
	addrs, err := net.LookupIP(d.target)
	if err != nil {
		return err
	}

	// Find appropriate address
	for _, addr := range addrs {
		if d.ipv6 && addr.To4() == nil {
			d.targetAddr = &net.IPAddr{IP: addr}
			return nil
		}
		if !d.ipv6 && addr.To4() != nil {
			d.targetAddr = &net.IPAddr{IP: addr}
			return nil
		}
	}

	if d.ipv6 {
		return fmt.Errorf("no IPv6 address found for %s", d.target)
	}
	return fmt.Errorf("no IPv4 address found for %s", d.target)
}

// setupConnection establishes the network connection for probing
func (d *MTUDiscoverer) setupConnection() error {
	var network string
	if d.ipv6 {
		network = "ip6:ipv6-icmp"
	} else {
		network = "ip4:icmp"
	}

	conn, err := net.ListenPacket(network, "")
	if err != nil {
		return err
	}

	d.conn = conn
	return nil
}

// Close closes the discoverer and releases resources
func (d *MTUDiscoverer) Close() error {
	if d.conn != nil {
		return d.conn.Close()
	}
	return nil
}

// DiscoverPMTU performs binary search to find the Path-MTU using the specified protocol
func (d *MTUDiscoverer) DiscoverPMTU(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
	switch d.protocol {
	case "icmp":
		return d.discoverICMP(ctx, minMTU, maxMTU)
	case "tcp":
		return d.discoverTCP(ctx, minMTU, maxMTU)
	case "udp":
		return d.discoverUDP(ctx, minMTU, maxMTU)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", d.protocol)
	}
}

// discoverICMP performs ICMP-based MTU discovery
func (d *MTUDiscoverer) discoverICMP(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
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

		result := d.probe(ctx, mid)
		hops++

		if result.Success {
			lastWorking = mid
			low = mid + 1
		} else {
			// Check if it's an ICMP "Packet Too Big" or "Fragmentation Needed"
			if result.ICMPErr != nil && d.isFragmentationError(result.ICMPErr) {
				high = mid - 1
			} else {
				// Timeout or other error - try smaller size
				high = mid - 1
			}
		}
	}

	if lastWorking == 0 {
		return nil, fmt.Errorf("no working MTU found in range %d-%d", minMTU, maxMTU)
	}

	elapsed := time.Since(start)

	// Calculate MSS based on IP version
	mss := lastWorking - 40 // IPv4 headers (20) + TCP headers (20)
	if d.ipv6 {
		mss = lastWorking - 60 // IPv6 headers (40) + TCP headers (20)
	}

	return &MTUResult{
		Target:    d.target,
		Protocol:  d.protocol,
		PMTU:      lastWorking,
		MSS:       mss,
		Hops:      hops,
		ElapsedMS: int(elapsed.Milliseconds()),
	}, nil
}

// discoverTCP performs TCP-based MTU discovery
func (d *MTUDiscoverer) discoverTCP(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
	prober, err := NewTCPProber(d.target, d.ipv6, d.timeout)
	if err != nil {
		return nil, err
	}

	return prober.DiscoverPMTUTCP(ctx, minMTU, maxMTU)
}

// discoverUDP performs UDP-based MTU discovery
func (d *MTUDiscoverer) discoverUDP(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
	prober, err := NewUDPProber(d.target, d.ipv6, d.timeout)
	if err != nil {
		return nil, err
	}

	return prober.DiscoverPMTUUDP(ctx, minMTU, maxMTU)
}

// probe sends a single MTU probe packet
func (d *MTUDiscoverer) probe(ctx context.Context, size int) *ProbeResult {
	start := time.Now()

	// Apply rate limiting
	d.security.RateLimiter.Wait()

	// Create ICMP packet
	packet, err := d.createICMPPacket(size)
	if err != nil {
		return &ProbeResult{
			Size:    size,
			Success: false,
			Error:   err,
		}
	}

	// Send packet
	_, err = d.conn.WriteTo(packet, d.targetAddr)
	if err != nil {
		return &ProbeResult{
			Size:    size,
			Success: false,
			Error:   err,
		}
	}

	// Set read deadline
	deadline := time.Now().Add(d.timeout)
	if err := d.conn.SetReadDeadline(deadline); err != nil {
		return &ProbeResult{
			Size:    size,
			Success: false,
			RTT:     time.Since(start),
			Error:   fmt.Errorf("failed to set read deadline: %w", err),
		}
	}

	// Read response
	response := make([]byte, 1500)
	n, addr, err := d.conn.ReadFrom(response)
	rtt := time.Since(start)

	if err != nil {
		// Check if it's a timeout
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return &ProbeResult{
				Size:    size,
				Success: false,
				RTT:     rtt,
				Error:   err,
			}
		}
		return &ProbeResult{
			Size:    size,
			Success: false,
			RTT:     rtt,
			Error:   err,
		}
	}

	// Parse ICMP response
	icmpErr := d.parseICMPResponse(response[:n], addr)

	return &ProbeResult{
		Size:    size,
		Success: icmpErr == nil,
		RTT:     rtt,
		ICMPErr: icmpErr,
	}
}

// createICMPPacket creates an ICMP Echo Request packet with the specified payload size
func (d *MTUDiscoverer) createICMPPacket(payloadSize int) ([]byte, error) {
	// Calculate payload size (subtract ICMP header)
	dataSize := payloadSize - 8 // ICMP header is 8 bytes
	if dataSize < 0 {
		dataSize = 0
	}

	// Create payload with security randomization
	payload := d.security.Randomizer.GenerateRandomPayload(dataSize)

	var msg *icmp.Message
	if d.ipv6 {
		msg = &icmp.Message{
			Type: ipv6.ICMPTypeEchoRequest,
			Code: 0,
			Body: &icmp.Echo{
				ID:   d.security.Randomizer.GenerateRandomID(),
				Seq:  d.security.Randomizer.GenerateRandomSeq(),
				Data: payload,
			},
		}
	} else {
		msg = &icmp.Message{
			Type: ipv4.ICMPTypeEcho,
			Code: 0,
			Body: &icmp.Echo{
				ID:   d.security.Randomizer.GenerateRandomID(),
				Seq:  d.security.Randomizer.GenerateRandomSeq(),
				Data: payload,
			},
		}
	}

	return msg.Marshal(nil)
}

// parseICMPResponse parses ICMP response to check for errors
func (d *MTUDiscoverer) parseICMPResponse(data []byte, addr net.Addr) *ICMPError {
	var proto int
	if d.ipv6 {
		proto = 58 // ICMPv6 protocol number
	} else {
		proto = 1 // ICMP protocol number
	}

	msg, err := icmp.ParseMessage(proto, data)
	if err != nil {
		return &ICMPError{
			Type:    -1,
			Code:    -1,
			Message: "Unable to parse ICMP response",
		}
	}

	// Check message type
	if d.ipv6 {
		switch msg.Type {
		case ipv6.ICMPTypeEchoReply:
			// Success
			return nil
		case ipv6.ICMPTypePacketTooBig:
			return &ICMPError{
				Type:    int(ipv6.ICMPTypePacketTooBig),
				Code:    msg.Code,
				Message: "Packet Too Big",
			}
		case ipv6.ICMPTypeDestinationUnreachable:
			return &ICMPError{
				Type:    int(ipv6.ICMPTypeDestinationUnreachable),
				Code:    msg.Code,
				Message: "Destination Unreachable",
			}
		default:
			// Try to get type as int
			typeInt := 0
			if icmpType, ok := msg.Type.(ipv6.ICMPType); ok {
				typeInt = int(icmpType)
			}
			return &ICMPError{
				Type:    typeInt,
				Code:    msg.Code,
				Message: fmt.Sprintf("ICMPv6 Type %v Code %d", msg.Type, msg.Code),
			}
		}
	} else {
		switch msg.Type {
		case ipv4.ICMPTypeEchoReply:
			// Success
			return nil
		case ipv4.ICMPTypeDestinationUnreachable:
			errMsg := "Destination Unreachable"
			if msg.Code == 4 {
				errMsg = "Fragmentation Needed and Don't Fragment was Set"
			}
			return &ICMPError{
				Type:    int(ipv4.ICMPTypeDestinationUnreachable),
				Code:    msg.Code,
				Message: errMsg,
			}
		default:
			// Try to get type as int
			typeInt := 0
			if icmpType, ok := msg.Type.(ipv4.ICMPType); ok {
				typeInt = int(icmpType)
			}
			return &ICMPError{
				Type:    typeInt,
				Code:    msg.Code,
				Message: fmt.Sprintf("ICMP Type %v Code %d", msg.Type, msg.Code),
			}
		}
	}
}

// isFragmentationError checks if the ICMP error indicates fragmentation needed
func (d *MTUDiscoverer) isFragmentationError(icmpErr *ICMPError) bool {
	if d.ipv6 {
		// IPv6 Packet Too Big
		return icmpErr.Type == int(ipv6.ICMPTypePacketTooBig)
	} else {
		// IPv4 Destination Unreachable with Fragmentation Needed
		return icmpErr.Type == int(ipv4.ICMPTypeDestinationUnreachable) && icmpErr.Code == 4
	}
}
