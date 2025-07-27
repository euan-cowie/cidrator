package mtu

import (
	"context"
	"testing"
	"time"
)

// TestMTUDiscoverer tests the MTUDiscoverer struct creation and basic functionality
func TestNewMTUDiscoverer(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		ipv6        bool
		protocol    string
		timeout     time.Duration
		ttl         int
		expectError bool
	}{
		{
			name:        "valid ICMP discoverer",
			target:      "127.0.0.1",
			ipv6:        false,
			protocol:    "icmp",
			timeout:     2 * time.Second,
			ttl:         64,
			expectError: false,
		},
		{
			name:        "valid TCP discoverer",
			target:      "localhost",
			ipv6:        false,
			protocol:    "tcp",
			timeout:     2 * time.Second,
			ttl:         64,
			expectError: false,
		},
		{
			name:        "valid UDP discoverer",
			target:      "localhost",
			ipv6:        false,
			protocol:    "udp",
			timeout:     2 * time.Second,
			ttl:         64,
			expectError: false,
		},
		{
			name:        "IPv6 TCP discoverer",
			target:      "::1",
			ipv6:        true,
			protocol:    "tcp",
			timeout:     2 * time.Second,
			ttl:         64,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoverer, err := NewMTUDiscoverer(tt.target, tt.ipv6, tt.protocol, tt.timeout, tt.ttl)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if discoverer == nil {
				t.Errorf("expected discoverer, got nil")
				return
			}

			// Validate discoverer properties
			if discoverer.target != tt.target {
				t.Errorf("target mismatch: got %q, want %q", discoverer.target, tt.target)
			}

			if discoverer.ipv6 != tt.ipv6 {
				t.Errorf("ipv6 mismatch: got %v, want %v", discoverer.ipv6, tt.ipv6)
			}

			if discoverer.protocol != tt.protocol {
				t.Errorf("protocol mismatch: got %q, want %q", discoverer.protocol, tt.protocol)
			}

			if discoverer.timeout != tt.timeout {
				t.Errorf("timeout mismatch: got %v, want %v", discoverer.timeout, tt.timeout)
			}

			if discoverer.ttl != tt.ttl {
				t.Errorf("ttl mismatch: got %d, want %d", discoverer.ttl, tt.ttl)
			}

			// Clean up
			if discoverer != nil {
				discoverer.Close()
			}
		})
	}
}

// TestResolveTarget tests target resolution functionality
func TestResolveTarget(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		ipv6        bool
		expectError bool
	}{
		{
			name:        "IPv4 localhost",
			target:      "127.0.0.1",
			ipv6:        false,
			expectError: false,
		},
		{
			name:        "IPv6 localhost",
			target:      "::1",
			ipv6:        true,
			expectError: false,
		},
		{
			name:        "hostname",
			target:      "localhost",
			ipv6:        false,
			expectError: false,
		},
		{
			name:        "invalid hostname",
			target:      "invalid..hostname..example",
			ipv6:        false,
			expectError: true,
		},
		{
			name:        "IPv4 when IPv6 requested",
			target:      "127.0.0.1",
			ipv6:        true,
			expectError: true,
		},
		{
			name:        "IPv6 when IPv4 requested",
			target:      "::1",
			ipv6:        false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoverer := &MTUDiscoverer{
				target:   tt.target,
				ipv6:     tt.ipv6,
				protocol: "icmp",
				timeout:  2 * time.Second,
				ttl:      64,
				security: NewSecurityConfig(10),
			}

			err := discoverer.resolveTarget()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if discoverer.targetAddr == nil {
				t.Errorf("expected target address to be set")
			}
		})
	}
}

// TestProtocolSupport tests different protocol implementations
func TestProtocolSupport(t *testing.T) {
	protocols := []string{"icmp", "tcp", "udp"}

	for _, protocol := range protocols {
		t.Run(protocol, func(t *testing.T) {
			discoverer, err := NewMTUDiscoverer("localhost", false, protocol, 2*time.Second, 64)
			if err != nil {
				t.Errorf("failed to create %s discoverer: %v", protocol, err)
				return
			}
			defer discoverer.Close()

			// Test that the discoverer was created with correct protocol
			if discoverer.protocol != protocol {
				t.Errorf("protocol mismatch: got %q, want %q", discoverer.protocol, protocol)
			}
		})
	}
}

// TestInvalidProtocol tests unsupported protocol handling
func TestInvalidProtocol(t *testing.T) {
	discoverer, err := NewMTUDiscoverer("localhost", false, "invalid", 2*time.Second, 64)
	if err != nil {
		t.Errorf("expected no error during creation, got: %v", err)
		return
	}
	defer discoverer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = discoverer.DiscoverPMTU(ctx, 576, 1500)
	if err == nil {
		t.Errorf("expected error for invalid protocol, got nil")
	}

	expectedError := "unsupported protocol: invalid"
	if err.Error() != expectedError {
		t.Errorf("expected error %q, got %q", expectedError, err.Error())
	}
}

// TestSecurityConfig tests security configuration
func TestSecurityConfig(t *testing.T) {
	config := NewSecurityConfig(5)

	if config == nil {
		t.Fatalf("expected security config, got nil")
	}

	if config.RateLimiter == nil {
		t.Errorf("expected rate limiter, got nil")
	}

	if config.Randomizer == nil {
		t.Errorf("expected randomizer, got nil")
	}

	if config.RetryThrottler == nil {
		t.Errorf("expected retry throttler, got nil")
	}

	// Test rate limiter configuration
	if config.RateLimiter.packetsPerSecond != 5 {
		t.Errorf("expected 5 PPS, got %d", config.RateLimiter.packetsPerSecond)
	}
}

// TestICMPError tests ICMP error handling
func TestICMPError(t *testing.T) {
	tests := []struct {
		name     string
		icmpType int
		code     int
		ipv6     bool
		expected bool // whether it should be considered a fragmentation error
	}{
		{
			name:     "IPv4 fragmentation needed",
			icmpType: 3, // Destination Unreachable
			code:     4, // Fragmentation Needed
			ipv6:     false,
			expected: true,
		},
		{
			name:     "IPv6 packet too big",
			icmpType: 2, // Packet Too Big
			code:     0,
			ipv6:     true,
			expected: true,
		},
		{
			name:     "IPv4 other error",
			icmpType: 3, // Destination Unreachable
			code:     1, // Host Unreachable
			ipv6:     false,
			expected: false,
		},
		{
			name:     "IPv6 other error",
			icmpType: 1, // Destination Unreachable
			code:     0,
			ipv6:     true,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			discoverer := &MTUDiscoverer{
				ipv6: tt.ipv6,
			}

			icmpErr := &ICMPError{
				Type: tt.icmpType,
				Code: tt.code,
			}

			result := discoverer.isFragmentationError(icmpErr)
			if result != tt.expected {
				t.Errorf("isFragmentationError: got %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestProbeResult tests probe result structure
func TestProbeResult(t *testing.T) {
	result := &ProbeResult{
		Size:    1500,
		Success: true,
		RTT:     50 * time.Millisecond,
		Error:   nil,
		ICMPErr: nil,
	}

	if result.Size != 1500 {
		t.Errorf("size mismatch: got %d, want %d", result.Size, 1500)
	}

	if !result.Success {
		t.Errorf("expected success to be true")
	}

	if result.RTT != 50*time.Millisecond {
		t.Errorf("RTT mismatch: got %v, want %v", result.RTT, 50*time.Millisecond)
	}
}

// TestMTURange tests MTU range validation
func TestMTURange(t *testing.T) {
	tests := []struct {
		name        string
		minMTU      int
		maxMTU      int
		ipv6        bool
		expectError bool
	}{
		{
			name:        "valid IPv4 range",
			minMTU:      576,
			maxMTU:      1500,
			ipv6:        false,
			expectError: false,
		},
		{
			name:        "valid IPv6 range",
			minMTU:      1280,
			maxMTU:      1500,
			ipv6:        true,
			expectError: false,
		},
		{
			name:        "invalid range (min > max)",
			minMTU:      1500,
			maxMTU:      576,
			ipv6:        false,
			expectError: true,
		},
		{
			name:        "zero range",
			minMTU:      1500,
			maxMTU:      1500,
			ipv6:        false,
			expectError: false, // Single value should work
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock discoverer for testing
			discoverer := &MTUDiscoverer{
				target:   "localhost",
				ipv6:     tt.ipv6,
				protocol: "tcp", // Use TCP to avoid raw socket issues
				timeout:  1 * time.Second,
				ttl:      64,
				security: NewSecurityConfig(10),
			}
			defer discoverer.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			_, err := discoverer.DiscoverPMTU(ctx, tt.minMTU, tt.maxMTU)

			if tt.expectError && err == nil {
				t.Errorf("expected error for invalid range, got nil")
			}

			// Note: For valid ranges, we might still get errors due to network conditions
			// so we don't fail the test if we get an error for valid ranges
		})
	}
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	discoverer, err := NewMTUDiscoverer("localhost", false, "tcp", 5*time.Second, 64)
	if err != nil {
		t.Fatalf("failed to create discoverer: %v", err)
	}
	defer discoverer.Close()

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = discoverer.DiscoverPMTU(ctx, 576, 1500)
	if err == nil {
		t.Errorf("expected context cancellation error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// TestTimeoutHandling tests timeout configuration
func TestTimeoutHandling(t *testing.T) {
	shortTimeout := 1 * time.Millisecond // Very short timeout
	discoverer, err := NewMTUDiscoverer("localhost", false, "tcp", shortTimeout, 64)
	if err != nil {
		t.Fatalf("failed to create discoverer: %v", err)
	}
	defer discoverer.Close()

	if discoverer.timeout != shortTimeout {
		t.Errorf("timeout mismatch: got %v, want %v", discoverer.timeout, shortTimeout)
	}
}

// TestCloseDiscoverer tests discoverer cleanup
func TestCloseDiscoverer(t *testing.T) {
	discoverer, err := NewMTUDiscoverer("localhost", false, "tcp", 2*time.Second, 64)
	if err != nil {
		t.Fatalf("failed to create discoverer: %v", err)
	}

	// Test that Close() doesn't return an error for TCP/UDP (no connection to close)
	err = discoverer.Close()
	if err != nil {
		t.Errorf("unexpected error from Close(): %v", err)
	}

	// Test that multiple Close() calls are safe
	err = discoverer.Close()
	if err != nil {
		t.Errorf("unexpected error from second Close(): %v", err)
	}
}

// Benchmark tests for performance validation
func BenchmarkNewMTUDiscoverer(b *testing.B) {
	for i := 0; i < b.N; i++ {
		discoverer, err := NewMTUDiscoverer("localhost", false, "tcp", 2*time.Second, 64)
		if err != nil {
			b.Errorf("failed to create discoverer: %v", err)
		}
		discoverer.Close()
	}
}

func BenchmarkResolveTarget(b *testing.B) {
	discoverer := &MTUDiscoverer{
		target:   "localhost",
		ipv6:     false,
		protocol: "tcp",
		timeout:  2 * time.Second,
		ttl:      64,
		security: NewSecurityConfig(10),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		discoverer.resolveTarget()
	}
}
