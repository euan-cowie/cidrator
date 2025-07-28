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
	MTU     int // MTU value from ICMP error (0 if not available)
}

// HopInfo represents information about a single hop in the path
type HopInfo struct {
	Hop     int           `json:"hop"`
	Addr    net.IP        `json:"addr,omitempty"`
	MTU     int           `json:"mtu,omitempty"` // 0 if unchanged from previous hop
	RTT     time.Duration `json:"rtt"`
	Timeout bool          `json:"timeout,omitempty"`
	Error   string        `json:"error,omitempty"`
}

// HopMTUResult represents the result of hop-by-hop MTU discovery
type HopMTUResult struct {
	Target       string     `json:"target"`
	Protocol     string     `json:"protocol"`
	MaxProbeSize int        `json:"max_probe_size"`
	FinalPMTU    int        `json:"final_pmtu"`
	Hops         []*HopInfo `json:"hops"`
	ElapsedMS    int        `json:"elapsed_ms"`
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
		// Use standard ICMP socket and rely on socket options for DF flag
		network = "ip4:icmp"
	}

	conn, err := net.ListenPacket(network, "")
	if err != nil {
		return err
	}

	d.conn = conn

	// Set DF flag for MTU discovery using proper socket options
	if err := d.setDontFragmentSocket(); err != nil {
		// Don't fail completely, but warn user
		fmt.Printf("Warning: Failed to set DF flag via socket options: %v\n", err)
	}

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

// DiscoverHopByHopMTU performs hop-by-hop MTU discovery using TTL variation
func (d *MTUDiscoverer) DiscoverHopByHopMTU(ctx context.Context, maxTTL int, maxProbeSize int) (*HopMTUResult, error) {
	if d.protocol != "icmp" {
		return nil, fmt.Errorf("hop-by-hop discovery only supported for ICMP protocol")
	}

	start := time.Now()

	// Use standard connection
	if d.conn == nil {
		if err := d.resolveTarget(); err != nil {
			return nil, fmt.Errorf("failed to resolve target: %w", err)
		}
		if err := d.setupConnection(); err != nil {
			return nil, fmt.Errorf("failed to setup connection: %w", err)
		}
	}

	var hops []*HopInfo
	finalPMTU := 0

	// Probe each hop to discover router addresses and basic connectivity
	for ttl := 1; ttl <= maxTTL; ttl++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// First, do a basic probe to identify the hop
		hop := d.probeHop(ctx, ttl, 1000)

		// If we get a response, discover the maximum MTU to this hop
		if hop.Addr != nil && hop.Error == "" {
			// Discover MTU to this specific hop
			hopMTU := d.discoverMTUToHop(ctx, ttl, 576, 1600)
			if hopMTU > 0 {
				hop.MTU = hopMTU
				fmt.Printf("Hop %d: %s (Path MTU to this hop: %d bytes)\n", ttl, hop.Addr, hopMTU)
			} else {
				fmt.Printf("Hop %d: %s (MTU discovery failed)\n", ttl, hop.Addr)
			}
		}

		hops = append(hops, hop)

		// Check if we've reached the destination
		if d.isDestinationReached(hop) {
			// For the final hop (destination), use regular PMTU discovery for more accurate results
			result, err := d.DiscoverPMTU(ctx, 576, maxProbeSize)
			if err == nil {
				finalPMTU = result.PMTU
				hop.MTU = result.PMTU
				fmt.Printf("Reached destination at hop %d with PMTU: %d bytes\n", ttl, result.PMTU)
			}
			break
		}

		// If we timeout consistently, we might have reached the end or hit a firewall
		if hop.Timeout {
			// Try a few more hops to see if we can get through
			consecutiveTimeouts := 1
			for i := ttl + 1; i <= ttl+3 && i <= maxTTL; i++ {
				extraHop := d.probeHop(ctx, i, 1000)
				hops = append(hops, extraHop)
				if extraHop.Timeout {
					consecutiveTimeouts++
				} else {
					consecutiveTimeouts = 0
					if d.isDestinationReached(extraHop) {
						// Reached destination after timeouts
						result, err := d.DiscoverPMTU(ctx, 576, maxProbeSize)
						if err == nil {
							finalPMTU = result.PMTU
							extraHop.MTU = result.PMTU
						}
						ttl = i // Update ttl for the loop exit
						break
					}
				}
			}
			if consecutiveTimeouts >= 3 {
				break // Assume we've reached the end
			}
			ttl += 3 // Skip the extra hops we already probed
		}
	}

	elapsed := time.Since(start)

	// Use the actual path MTU as discovered by regular PMTU discovery
	if finalPMTU == 0 {
		result, err := d.DiscoverPMTU(ctx, 576, maxProbeSize)
		if err == nil {
			finalPMTU = result.PMTU
		}
	}

	return &HopMTUResult{
		Target:       d.target,
		Protocol:     d.protocol,
		MaxProbeSize: maxProbeSize,
		FinalPMTU:    finalPMTU,
		Hops:         hops,
		ElapsedMS:    int(elapsed.Milliseconds()),
	}, nil
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

// probeHop sends a single probe with specified TTL for hop-by-hop discovery
func (d *MTUDiscoverer) probeHop(ctx context.Context, ttl int, size int) *HopInfo {
	start := time.Now()

	// Apply rate limiting
	d.security.RateLimiter.Wait()

	hop := &HopInfo{
		Hop: ttl,
	}

	// Create packet connection with TTL control
	var pconn interface{}
	if d.ipv6 {
		p := ipv6.NewPacketConn(d.conn)
		if err := p.SetHopLimit(ttl); err != nil {
			hop.Error = fmt.Sprintf("failed to set hop limit: %v", err)
			hop.RTT = time.Since(start)
			return hop
		}
		if err := p.SetControlMessage(ipv6.FlagHopLimit, true); err != nil {
			hop.Error = fmt.Sprintf("failed to set control message: %v", err)
			hop.RTT = time.Since(start)
			return hop
		}
		pconn = p
	} else {
		p := ipv4.NewPacketConn(d.conn)
		if err := p.SetTTL(ttl); err != nil {
			hop.Error = fmt.Sprintf("failed to set TTL: %v", err)
			hop.RTT = time.Since(start)
			return hop
		}
		if err := p.SetControlMessage(ipv4.FlagTTL, true); err != nil {
			hop.Error = fmt.Sprintf("failed to set control message: %v", err)
			hop.RTT = time.Since(start)
			return hop
		}
		pconn = p
	}

	// Create ICMP packet with DF flag
	packet, err := d.createICMPPacket(size)
	if err != nil {
		hop.Error = fmt.Sprintf("failed to create packet: %v", err)
		hop.RTT = time.Since(start)
		return hop
	}

	// Send packet
	_, err = d.conn.WriteTo(packet, d.targetAddr)
	if err != nil {
		hop.Error = fmt.Sprintf("failed to send packet: %v", err)
		hop.RTT = time.Since(start)
		return hop
	}

	// Set read deadline
	deadline := time.Now().Add(d.timeout)
	if err := d.conn.SetReadDeadline(deadline); err != nil {
		hop.Error = fmt.Sprintf("failed to set read deadline: %v", err)
		hop.RTT = time.Since(start)
		return hop
	}

	// Read response with control message
	response := make([]byte, 1500)
	var n int
	var addr net.Addr

	if d.ipv6 {
		p := pconn.(*ipv6.PacketConn)
		var cm *ipv6.ControlMessage
		n, cm, addr, err = p.ReadFrom(response)
		_ = cm // For now, we don't use the control message info
	} else {
		p := pconn.(*ipv4.PacketConn)
		var cm *ipv4.ControlMessage
		n, cm, addr, err = p.ReadFrom(response)
		_ = cm // For now, we don't use the control message info
	}

	hop.RTT = time.Since(start)

	if err != nil {
		// Check if it's a timeout
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			hop.Timeout = true
			return hop
		}
		hop.Error = fmt.Sprintf("read error: %v", err)
		return hop
	}

	// Extract source IP from response
	if ipAddr, ok := addr.(*net.IPAddr); ok {
		hop.Addr = ipAddr.IP
	}

	// Parse ICMP response to get MTU information
	icmpErr := d.parseICMPResponseWithMTU(response[:n], addr)
	if icmpErr != nil {
		if icmpErr.MTU > 0 {
			hop.MTU = icmpErr.MTU
		}

		// Check if this is a TTL exceeded error (normal for traceroute)
		if (d.ipv6 && icmpErr.Type == int(ipv6.ICMPTypeTimeExceeded)) ||
			(!d.ipv6 && icmpErr.Type == int(ipv4.ICMPTypeTimeExceeded)) {
			// This is normal - router responded with TTL exceeded
			return hop
		}

		// Check if this is an MTU-related error
		if d.isFragmentationError(icmpErr) {
			return hop
		}

		// Other ICMP error
		hop.Error = icmpErr.Message
		return hop
	}

	// If we get here, we got an echo reply, meaning we reached the destination
	return hop
}

// discoverMTUToHop performs MTU discovery to a specific hop by testing forwarding capacity
func (d *MTUDiscoverer) discoverMTUToHop(ctx context.Context, hopTTL int, minMTU, maxMTU int) int {
	// Binary search for maximum packet size that can reach this hop
	low := minMTU
	high := maxMTU
	lastWorking := 0

	for low <= high {
		mid := (low + high) / 2

		select {
		case <-ctx.Done():
			return lastWorking
		default:
		}

		// Test if a packet of this size can reach the target hop
		if d.canReachHopWithSize(ctx, hopTTL, mid) {
			lastWorking = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return lastWorking
}

// canReachHopWithSize tests if a packet of given size can reach the specified hop
func (d *MTUDiscoverer) canReachHopWithSize(ctx context.Context, hopTTL int, size int) bool {
	hop := d.probeHop(ctx, hopTTL, size)

	// If we get a response from this hop (TTL exceeded), the packet reached it successfully
	// If we get timeout or fragmentation error, the packet was too big for some hop in the path
	return hop.Addr != nil && hop.Error == ""
}

// isDestinationReached checks if we've reached our intended destination
func (d *MTUDiscoverer) isDestinationReached(hop *HopInfo) bool {
	if hop.Addr == nil {
		return false
	}

	// Get the target IP to compare
	var targetIP net.IP
	if ipAddr, ok := d.targetAddr.(*net.IPAddr); ok {
		targetIP = ipAddr.IP
	} else {
		return false
	}

	// Compare IPs
	return hop.Addr.Equal(targetIP)
}

// createICMPPacket creates an ICMP Echo Request packet (DF flag set via socket options)
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

// parseICMPResponseWithMTU parses ICMP response to get MTU information
func (d *MTUDiscoverer) parseICMPResponseWithMTU(data []byte, addr net.Addr) *ICMPError {
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
			mtu := 0
			// Extract MTU from ICMP packet too big message
			if packetTooBig, ok := msg.Body.(*icmp.PacketTooBig); ok {
				mtu = packetTooBig.MTU
			}
			return &ICMPError{
				Type:    int(ipv6.ICMPTypePacketTooBig),
				Code:    msg.Code,
				Message: "Packet Too Big",
				MTU:     mtu,
			}
		case ipv6.ICMPTypeDestinationUnreachable:
			return &ICMPError{
				Type:    int(ipv6.ICMPTypeDestinationUnreachable),
				Code:    msg.Code,
				Message: "Destination Unreachable",
			}
		case ipv6.ICMPTypeTimeExceeded:
			return &ICMPError{
				Type:    int(ipv6.ICMPTypeTimeExceeded),
				Code:    msg.Code,
				Message: "Time Exceeded",
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
			mtu := 0
			if msg.Code == 4 {
				errMsg = "Fragmentation Needed and Don't Fragment was Set"
				// Extract MTU from ICMP destination unreachable message
				if destUnreach, ok := msg.Body.(*icmp.DstUnreach); ok && destUnreach.Data != nil && len(destUnreach.Data) >= 6 {
					// MTU is in bytes 6-7 of the ICMP data (after the unused 4 bytes)
					mtu = int(destUnreach.Data[4])<<8 | int(destUnreach.Data[5])
				}
			}
			return &ICMPError{
				Type:    int(ipv4.ICMPTypeDestinationUnreachable),
				Code:    msg.Code,
				Message: errMsg,
				MTU:     mtu,
			}
		case ipv4.ICMPTypeTimeExceeded:
			return &ICMPError{
				Type:    int(ipv4.ICMPTypeTimeExceeded),
				Code:    msg.Code,
				Message: "Time Exceeded",
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

// setDontFragmentSocket sets the DF flag using proper socket options
func (d *MTUDiscoverer) setDontFragmentSocket() error {
	if d.ipv6 {
		// IPv6: Try to set IPV6_DONTFRAG
		return d.setIPv6DontFragment()
	} else {
		// IPv4: Use ipv4.PacketConn wrapper for proper DF flag control
		return d.setIPv4DontFragment()
	}
}

// setIPv4DontFragment sets DF flag for IPv4 using platform-specific constants
func (d *MTUDiscoverer) setIPv4DontFragment() error {
	switch conn := d.conn.(type) {
	case *net.IPConn:
		return setIPv4DontFragment(conn)
	default:
		return fmt.Errorf("unsupported connection type: %T", conn)
	}
}

// setIPv6DontFragment sets DF flag for IPv6
func (d *MTUDiscoverer) setIPv6DontFragment() error {
	switch conn := d.conn.(type) {
	case *net.IPConn:
		return setIPv6DontFragment(conn)
	default:
		return fmt.Errorf("unsupported connection type: %T", conn)
	}
}
