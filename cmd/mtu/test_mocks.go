package mtu

import (
	"context"
	"fmt"
	"net"
	"time"
)

// MockNetworkProber provides a configurable mock for network probing
type MockNetworkProber struct {
	responses   map[int]*ProbeResult
	failAfter   int
	callCount   int
	simulateRTT time.Duration
	closed      bool
}

// NewMockNetworkProber creates a new mock network prober
func NewMockNetworkProber() *MockNetworkProber {
	return &MockNetworkProber{
		responses:   make(map[int]*ProbeResult),
		simulateRTT: 10 * time.Millisecond,
	}
}

// SetResponse configures the response for a specific packet size
func (m *MockNetworkProber) SetResponse(size int, success bool, icmpErr *ICMPError) {
	m.responses[size] = &ProbeResult{
		Size:    size,
		Success: success,
		RTT:     m.simulateRTT,
		ICMPErr: icmpErr,
	}
}

// SetMTUFragmentationPoint configures responses to simulate MTU discovery
func (m *MockNetworkProber) SetMTUFragmentationPoint(mtu int) {
	// Clear existing responses
	m.responses = make(map[int]*ProbeResult)

	// Set responses for common test sizes
	for _, size := range []int{576, 1000, 1200, 1400, 1472, 1500, 1600, 9000} {
		success := size <= mtu
		var icmpErr *ICMPError
		if !success {
			icmpErr = &ICMPError{
				Type:    3, // Destination Unreachable
				Code:    4, // Fragmentation Needed
				Message: "Fragmentation Needed and Don't Fragment was Set",
			}
		}
		m.SetResponse(size, success, icmpErr)
	}
}

// SetFailAfter configures the mock to fail after a certain number of calls
func (m *MockNetworkProber) SetFailAfter(count int) {
	m.failAfter = count
}

// Probe implements the NetworkProber interface
func (m *MockNetworkProber) Probe(ctx context.Context, size int) *ProbeResult {
	if m.closed {
		return &ProbeResult{
			Size:    size,
			Success: false,
			Error:   fmt.Errorf("prober is closed"),
		}
	}

	m.callCount++

	// Check if we should fail
	if m.failAfter > 0 && m.callCount > m.failAfter {
		return &ProbeResult{
			Size:    size,
			Success: false,
			Error:   fmt.Errorf("mock failure after %d calls", m.failAfter),
		}
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return &ProbeResult{
			Size:    size,
			Success: false,
			Error:   ctx.Err(),
		}
	default:
	}

	// Return configured response or default success
	if response, exists := m.responses[size]; exists {
		return response
	}

	// Default to success for unlisted sizes
	return &ProbeResult{
		Size:    size,
		Success: true,
		RTT:     m.simulateRTT,
	}
}

// Close implements the NetworkProber interface
func (m *MockNetworkProber) Close() error {
	m.closed = true
	return nil
}

// GetCallCount returns the number of times Probe was called
func (m *MockNetworkProber) GetCallCount() int {
	return m.callCount
}

// MockMTUDiscoverer provides a configurable mock for MTU discovery
type MockMTUDiscoverer struct {
	target      string
	protocol    string
	pmtu        int
	failureMode string
	elapsedTime time.Duration
	hops        int
	closed      bool
}

// NewMockMTUDiscoverer creates a new mock MTU discoverer
func NewMockMTUDiscoverer(target, protocol string, pmtu int) *MockMTUDiscoverer {
	return &MockMTUDiscoverer{
		target:      target,
		protocol:    protocol,
		pmtu:        pmtu,
		elapsedTime: 150 * time.Millisecond,
		hops:        8,
	}
}

// SetFailureMode configures the mock to simulate different failure scenarios
func (m *MockMTUDiscoverer) SetFailureMode(mode string) {
	m.failureMode = mode
}

// DiscoverPMTU implements the MTUDiscoveryInterface interface
func (m *MockMTUDiscoverer) DiscoverPMTU(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
	if m.closed {
		return nil, fmt.Errorf("discoverer is closed")
	}

	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Handle failure modes
	switch m.failureMode {
	case "network_unreachable":
		return nil, fmt.Errorf("network unreachable")
	case "permission_denied":
		return nil, fmt.Errorf("operation not permitted")
	case "timeout":
		return nil, fmt.Errorf("operation timed out")
	case "no_working_mtu":
		return nil, fmt.Errorf("no working MTU found in range %d-%d", minMTU, maxMTU)
	}

	// Validate range
	if minMTU > maxMTU {
		return nil, fmt.Errorf("invalid MTU range: min %d > max %d", minMTU, maxMTU)
	}

	// Use configured PMTU or clamp to range
	discoveredMTU := m.pmtu
	if discoveredMTU < minMTU {
		discoveredMTU = minMTU
	}
	if discoveredMTU > maxMTU {
		discoveredMTU = maxMTU
	}

	// Calculate MSS
	mss := CalculateExpectedMSS(discoveredMTU, false)

	return &MTUResult{
		Target:    m.target,
		Protocol:  m.protocol,
		PMTU:      discoveredMTU,
		MSS:       mss,
		Hops:      m.hops,
		ElapsedMS: int(m.elapsedTime.Milliseconds()),
	}, nil
}

// Close implements the MTUDiscoveryInterface interface
func (m *MockMTUDiscoverer) Close() error {
	m.closed = true
	return nil
}

// MockNetworkResolver provides a configurable mock for address resolution
type MockNetworkResolver struct {
	addresses map[string]net.Addr
	failures  map[string]error
}

// NewMockNetworkResolver creates a new mock network resolver
func NewMockNetworkResolver() *MockNetworkResolver {
	return &MockNetworkResolver{
		addresses: make(map[string]net.Addr),
		failures:  make(map[string]error),
	}
}

// SetAddress configures the mock to return a specific address for a target
func (m *MockNetworkResolver) SetAddress(target string, addr net.Addr) {
	m.addresses[target] = addr
}

// SetFailure configures the mock to fail for a specific target
func (m *MockNetworkResolver) SetFailure(target string, err error) {
	m.failures[target] = err
}

// ResolveTarget implements the NetworkResolver interface
func (m *MockNetworkResolver) ResolveTarget(target string, ipv6 bool) (net.Addr, error) {
	// Check for configured failure
	if err, exists := m.failures[target]; exists {
		return nil, err
	}

	// Check for configured address
	if addr, exists := m.addresses[target]; exists {
		return addr, nil
	}

	// Default behavior: parse as IP or return default addresses
	if ip := net.ParseIP(target); ip != nil {
		return &net.IPAddr{IP: ip}, nil
	}

	// Return default test addresses
	if ipv6 {
		ip := net.ParseIP(TestTargets.ValidIPv6)
		return &net.IPAddr{IP: ip}, nil
	}
	ip := net.ParseIP(TestTargets.ValidIPv4)
	return &net.IPAddr{IP: ip}, nil
}

// MockInterfaceDetector provides a configurable mock for interface detection
type MockInterfaceDetector struct {
	interfaces []NetworkInterface
	maxMTU     int
	failure    error
}

// NewMockInterfaceDetector creates a new mock interface detector
func NewMockInterfaceDetector() *MockInterfaceDetector {
	return &MockInterfaceDetector{
		interfaces: []NetworkInterface{
			{Name: "lo0", MTU: 16384, Type: "loopback"},
			{Name: "en0", MTU: 1500, Type: "ethernet"},
		},
		maxMTU: 16384,
	}
}

// SetInterfaces configures the mock interfaces
func (m *MockInterfaceDetector) SetInterfaces(interfaces []NetworkInterface) {
	m.interfaces = interfaces

	// Update max MTU
	m.maxMTU = 0
	for _, iface := range interfaces {
		if iface.MTU > m.maxMTU {
			m.maxMTU = iface.MTU
		}
	}
}

// SetFailure configures the mock to fail
func (m *MockInterfaceDetector) SetFailure(err error) {
	m.failure = err
}

// GetNetworkInterfaces implements the InterfaceDetector interface
func (m *MockInterfaceDetector) GetNetworkInterfaces() (*InterfaceResult, error) {
	if m.failure != nil {
		return nil, m.failure
	}

	return &InterfaceResult{
		Interfaces: m.interfaces,
	}, nil
}

// GetMaxMTU implements the InterfaceDetector interface
func (m *MockInterfaceDetector) GetMaxMTU() (int, error) {
	if m.failure != nil {
		return 0, m.failure
	}

	if m.maxMTU == 0 {
		return 1500, nil // Default fallback
	}

	return m.maxMTU, nil
}
