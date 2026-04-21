package mtu

import "time"

// Test constants for protocol overhead calculations
const (
	IPv4HeaderSize      = 20
	IPv6HeaderSize      = 40
	TCPHeaderSize       = 20
	UDPHeaderSize       = 8
	ICMPHeaderSize      = 8
	WireGuardOverhead   = 60
	IPSecESPUDPOverhead = 84
)

// Standard MTU values for testing
var (
	StandardMTUs = []int{576, 1280, 1500, 9000}

	TestMTUScenarios = []struct {
		Name        string
		MTU         int
		Description string
	}{
		{"IPv4 Minimum", 576, "Minimum IPv4 MTU per RFC 791"},
		{"IPv6 Minimum", 1280, "Minimum IPv6 MTU per RFC 8200"},
		{"Ethernet Standard", 1500, "Standard Ethernet MTU"},
		{"Jumbo Frame", 9000, "Jumbo frame MTU"},
	}

	TestTargets = struct {
		ValidHostname   string
		ValidIPv4       string
		ValidIPv6       string
		InvalidHostname string
		Localhost       string
	}{
		ValidHostname:   "example.com",
		ValidIPv4:       "192.0.2.1",   // RFC 5737 test address
		ValidIPv6:       "2001:db8::1", // RFC 3849 test address
		InvalidHostname: "invalid..hostname..test",
		Localhost:       "localhost",
	}

	TestTimeouts = struct {
		Short  time.Duration
		Normal time.Duration
		Long   time.Duration
	}{
		Short:  100 * time.Millisecond,
		Normal: 2 * time.Second,
		Long:   10 * time.Second,
	}
)

// CalculateExpectedMSS calculates expected MSS for testing
func CalculateExpectedMSS(mtu int, ipv6 bool) int {
	if ipv6 {
		return mtu - IPv6HeaderSize - TCPHeaderSize
	}
	return mtu - IPv4HeaderSize - TCPHeaderSize
}

// CalculateExpectedPayload calculates expected payload sizes for different protocols
func CalculateExpectedPayload(mtu int, protocol string, ipv6 bool) int {
	baseHeaderSize := IPv4HeaderSize
	if ipv6 {
		baseHeaderSize = IPv6HeaderSize
	}

	switch protocol {
	case "tcp":
		return mtu - baseHeaderSize - TCPHeaderSize
	case "udp":
		return mtu - baseHeaderSize - UDPHeaderSize
	case "icmp":
		return mtu - baseHeaderSize - ICMPHeaderSize
	case "wireguard":
		return mtu - WireGuardOverhead
	case "ipsec":
		return mtu - IPSecESPUDPOverhead
	default:
		return mtu - baseHeaderSize
	}
}
