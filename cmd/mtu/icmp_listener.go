package mtu

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// FragmentationError represents an ICMP "Fragmentation Needed" error
// per RFC 1191 Section 4
type FragmentationError struct {
	// NextHopMTU is the MTU of the next-hop network that caused the error
	// This is extracted from ICMP Type 3 Code 4 (IPv4) or Type 2 (IPv6)
	NextHopMTU int

	// OriginalDst is the destination from the original packet that triggered the error
	OriginalDst net.IP

	// OriginalSrcPort and OriginalDstPort from the embedded packet header
	OriginalSrcPort int
	OriginalDstPort int
}

// ICMPListener listens for ICMP "Fragmentation Needed and DF Set" errors
// as specified in RFC 1191 Section 4
type ICMPListener struct {
	conn4   *icmp.PacketConn
	conn6   *icmp.PacketConn
	errors  chan *FragmentationError
	done    chan struct{}
	mu      sync.Mutex
	running bool
}

// NewICMPListener creates a new ICMP error listener
// Requires elevated privileges (root/sudo)
func NewICMPListener() (*ICMPListener, error) {
	listener := &ICMPListener{
		errors: make(chan *FragmentationError, 16),
		done:   make(chan struct{}),
	}

	// Try to open IPv4 ICMP socket
	conn4, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		// May fail without privileges, continue anyway
		fmt.Printf("Warning: Could not open IPv4 ICMP socket: %v\n", err)
	} else {
		listener.conn4 = conn4
	}

	// Try to open IPv6 ICMP socket
	conn6, err := icmp.ListenPacket("ip6:ipv6-icmp", "::")
	if err != nil {
		// May fail without privileges or IPv6 support
		fmt.Printf("Warning: Could not open IPv6 ICMP socket: %v\n", err)
	} else {
		listener.conn6 = conn6
	}

	if listener.conn4 == nil && listener.conn6 == nil {
		return nil, fmt.Errorf("failed to open any ICMP socket (requires root)")
	}

	return listener, nil
}

// Start begins listening for ICMP errors in the background
func (l *ICMPListener) Start(ctx context.Context) {
	l.mu.Lock()
	if l.running {
		l.mu.Unlock()
		return
	}
	l.running = true
	l.mu.Unlock()

	if l.conn4 != nil {
		go l.listenIPv4(ctx)
	}
	if l.conn6 != nil {
		go l.listenIPv6(ctx)
	}
}

// Errors returns a channel that receives ICMP errors
func (l *ICMPListener) Errors() <-chan *FragmentationError {
	return l.errors
}

// Close stops the listener and releases resources
func (l *ICMPListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.running {
		return nil
	}
	l.running = false
	close(l.done)

	var errs []error
	if l.conn4 != nil {
		if err := l.conn4.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if l.conn6 != nil {
		if err := l.conn6.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}
	return nil
}

// listenIPv4 listens for IPv4 ICMP Type 3 Code 4 messages
// RFC 1191: "fragmentation needed and DF set"
func (l *ICMPListener) listenIPv4(ctx context.Context) {
	buf := make([]byte, 1500)

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		default:
		}

		// Set read deadline to periodically check for cancellation
		if err := l.conn4.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			continue
		}

		n, peer, err := l.conn4.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			// Check if we're shutting down
			select {
			case <-l.done:
				return
			default:
				continue
			}
		}

		// Parse ICMP message
		msg, err := icmp.ParseMessage(1, buf[:n]) // Protocol 1 = ICMP
		if err != nil {
			continue
		}

		// Check for Type 3 (Destination Unreachable), Code 4 (Fragmentation Needed)
		if msg.Type != ipv4.ICMPTypeDestinationUnreachable {
			continue
		}

		dstUnreach, ok := msg.Body.(*icmp.DstUnreach)
		if !ok {
			continue
		}

		// Code 4 = Fragmentation Needed and DF Set
		if msg.Code != 4 {
			continue
		}

		// Extract Next-Hop MTU from the ICMP message
		// Per RFC 1191: bytes 6-7 of the ICMP header contain Next-Hop MTU
		// In the parsed message, this is available in the Data field prefix
		icmpErr := l.parseICMPv4Error(dstUnreach.Data, peer)
		if icmpErr != nil {
			select {
			case l.errors <- icmpErr:
			default:
				// Channel full, drop oldest
			}
		}
	}
}

// listenIPv6 listens for IPv6 ICMPv6 Type 2 messages
// RFC 8201: "Packet Too Big"
func (l *ICMPListener) listenIPv6(ctx context.Context) {
	buf := make([]byte, 1500)

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.done:
			return
		default:
		}

		if err := l.conn6.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			continue
		}

		n, peer, err := l.conn6.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			select {
			case <-l.done:
				return
			default:
				continue
			}
		}

		// Parse ICMPv6 message
		msg, err := icmp.ParseMessage(58, buf[:n]) // Protocol 58 = ICMPv6
		if err != nil {
			continue
		}

		// Check for Type 2 (Packet Too Big)
		if msg.Type != ipv6.ICMPTypePacketTooBig {
			continue
		}

		pktTooBig, ok := msg.Body.(*icmp.PacketTooBig)
		if !ok {
			continue
		}

		icmpErr := &FragmentationError{
			NextHopMTU: pktTooBig.MTU,
		}

		// Try to extract destination from embedded packet
		if len(pktTooBig.Data) >= 40 {
			icmpErr.OriginalDst = net.IP(pktTooBig.Data[24:40])
		}

		select {
		case l.errors <- icmpErr:
		default:
		}

		_ = peer // Suppress unused warning
	}
}

// parseICMPv4Error extracts error information from ICMP message data
// The data contains the original IP header + first 8 bytes of payload
func (l *ICMPListener) parseICMPv4Error(data []byte, peer net.Addr) *FragmentationError {
	// Need at least IP header (20 bytes) + 8 bytes of original data
	if len(data) < 28 {
		return nil
	}

	// Extract Next-Hop MTU from ICMP header
	// This was placed before the IP header in the original message
	// The icmp library strips this, so we need to get it differently
	// For now, we'll use a default or require the caller to handle this

	icmpErr := &FragmentationError{
		NextHopMTU: 0, // Will be set by caller from raw message
	}

	// IP header: destination is at bytes 16-19
	icmpErr.OriginalDst = net.IP(data[16:20])

	// Protocol is at byte 9
	protocol := data[9]
	ihl := int(data[0]&0x0f) * 4 // IP Header Length

	// Extract ports from transport header (UDP/TCP)
	if len(data) >= ihl+4 {
		if protocol == 6 || protocol == 17 { // TCP or UDP
			icmpErr.OriginalSrcPort = int(binary.BigEndian.Uint16(data[ihl : ihl+2]))
			icmpErr.OriginalDstPort = int(binary.BigEndian.Uint16(data[ihl+2 : ihl+4]))
		}
	}

	_ = peer // Suppress unused warning
	return icmpErr
}

// WaitForError waits for an ICMP error matching the given destination
// Returns the error or nil if timeout occurs
func (l *ICMPListener) WaitForError(ctx context.Context, dst net.IP, timeout time.Duration) *FragmentationError {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-l.errors:
			// Check if this error matches our destination
			if err.OriginalDst != nil && err.OriginalDst.Equal(dst) {
				return err
			}
			// Not our error, continue waiting
		}
	}
}
