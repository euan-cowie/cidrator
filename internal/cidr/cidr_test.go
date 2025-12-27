package cidr

import (
	"context"
	"math/big"
	"net"
	"strings"
	"testing"
)

func assertError(t *testing.T, got error, want bool) {
	t.Helper()
	if (got != nil) != want {
		t.Errorf("got error %v, want error=%v", got, want)
	}
}

func TestParseCIDR(t *testing.T) {
	tests := []struct {
		name          string
		cidr          string
		expectError   bool
		expectedIPv6  bool
		expectedBits  int
		expectedTotal string
	}{
		{
			name:          "IPv4 /24",
			cidr:          "192.168.1.0/24",
			expectError:   false,
			expectedIPv6:  false,
			expectedBits:  8,
			expectedTotal: "256",
		},
		{
			name:          "IPv4 /16",
			cidr:          "10.0.0.0/16",
			expectError:   false,
			expectedIPv6:  false,
			expectedBits:  16,
			expectedTotal: "65,536",
		},
		{
			name:          "IPv4 /32",
			cidr:          "192.168.1.1/32",
			expectError:   false,
			expectedIPv6:  false,
			expectedBits:  0,
			expectedTotal: "1",
		},
		{
			name:          "IPv4 /31",
			cidr:          "192.168.1.0/31",
			expectError:   false,
			expectedIPv6:  false,
			expectedBits:  1,
			expectedTotal: "2",
		},
		{
			name:          "IPv6 /64",
			cidr:          "2001:db8::/64",
			expectError:   false,
			expectedIPv6:  true,
			expectedBits:  64,
			expectedTotal: "18,446,744,073,709,551,616",
		},
		{
			name:          "IPv6 /128",
			cidr:          "2001:db8::1/128",
			expectError:   false,
			expectedIPv6:  true,
			expectedBits:  0,
			expectedTotal: "1",
		},
		{
			name:        "Invalid CIDR",
			cidr:        "invalid",
			expectError: true,
		},
		{
			name:        "Invalid prefix",
			cidr:        "192.168.1.0/33",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseCIDR(tt.cidr)

			assertError(t, err, tt.expectError)
			if tt.expectError {
				return
			}

			if info.IsIPv6 != tt.expectedIPv6 {
				t.Errorf("Expected IPv6=%v, got %v", tt.expectedIPv6, info.IsIPv6)
			}

			if info.HostBits != tt.expectedBits {
				t.Errorf("Expected %d host bits, got %d", tt.expectedBits, info.HostBits)
			}

			if FormatBigInt(info.TotalAddresses) != tt.expectedTotal {
				t.Errorf("Expected %s total addresses, got %s", tt.expectedTotal, FormatBigInt(info.TotalAddresses))
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		ip       string
		expected bool
		hasError bool
	}{
		{
			name:     "IPv4 contained",
			cidr:     "192.168.1.0/24",
			ip:       "192.168.1.100",
			expected: true,
		},
		{
			name:     "IPv4 not contained",
			cidr:     "192.168.1.0/24",
			ip:       "192.168.2.100",
			expected: false,
		},
		{
			name:     "IPv4 network address",
			cidr:     "192.168.1.0/24",
			ip:       "192.168.1.0",
			expected: true,
		},
		{
			name:     "IPv4 broadcast address",
			cidr:     "192.168.1.0/24",
			ip:       "192.168.1.255",
			expected: true,
		},
		{
			name:     "IPv6 contained",
			cidr:     "2001:db8::/32",
			ip:       "2001:db8:1::1",
			expected: true,
		},
		{
			name:     "IPv6 not contained",
			cidr:     "2001:db8::/32",
			ip:       "2001:db9::1",
			expected: false,
		},
		{
			name:     "Invalid CIDR",
			cidr:     "invalid",
			ip:       "192.168.1.1",
			hasError: true,
		},
		{
			name:     "Invalid IP",
			cidr:     "192.168.1.0/24",
			ip:       "invalid",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Contains(tt.cidr, tt.ip)

			assertError(t, err, tt.hasError)
			if tt.hasError {
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCount(t *testing.T) {
	tests := []struct {
		name     string
		cidr     string
		expected string
		hasError bool
	}{
		{
			name:     "IPv4 /24",
			cidr:     "192.168.1.0/24",
			expected: "256",
		},
		{
			name:     "IPv4 /16",
			cidr:     "10.0.0.0/16",
			expected: "65536",
		},
		{
			name:     "IPv4 /32",
			cidr:     "192.168.1.1/32",
			expected: "1",
		},
		{
			name:     "IPv6 /127",
			cidr:     "2001:db8::/127",
			expected: "2",
		},
		{
			name:     "Invalid CIDR",
			cidr:     "invalid",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Count(tt.cidr)

			assertError(t, err, tt.hasError)
			if tt.hasError {
				return
			}

			if result.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.String())
			}
		})
	}
}

func TestOverlaps(t *testing.T) {
	tests := []struct {
		name     string
		cidr1    string
		cidr2    string
		expected bool
		hasError bool
	}{
		{
			name:     "IPv4 overlapping",
			cidr1:    "192.168.1.0/24",
			cidr2:    "192.168.1.128/25",
			expected: true,
		},
		{
			name:     "IPv4 non-overlapping",
			cidr1:    "192.168.1.0/24",
			cidr2:    "192.168.2.0/24",
			expected: false,
		},
		{
			name:     "IPv4 identical",
			cidr1:    "192.168.1.0/24",
			cidr2:    "192.168.1.0/24",
			expected: true,
		},
		{
			name:     "IPv4 one contains other",
			cidr1:    "192.168.0.0/16",
			cidr2:    "192.168.1.0/24",
			expected: true,
		},
		{
			name:     "IPv6 overlapping",
			cidr1:    "2001:db8::/32",
			cidr2:    "2001:db8:1::/48",
			expected: true,
		},
		{
			name:     "IPv6 non-overlapping",
			cidr1:    "2001:db8::/32",
			cidr2:    "2001:db9::/32",
			expected: false,
		},
		{
			name:     "Invalid first CIDR",
			cidr1:    "invalid",
			cidr2:    "192.168.1.0/24",
			hasError: true,
		},
		{
			name:     "Invalid second CIDR",
			cidr1:    "192.168.1.0/24",
			cidr2:    "invalid",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Overlaps(tt.cidr1, tt.cidr2)

			assertError(t, err, tt.hasError)
			if tt.hasError {
				return
			}

			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDivide(t *testing.T) {
	tests := []struct {
		name          string
		cidr          string
		parts         int
		expectedCount int
		expectedFirst string
		hasError      bool
	}{
		{
			name:          "IPv4 /24 into 4 parts",
			cidr:          "192.168.1.0/24",
			parts:         4,
			expectedCount: 4,
			expectedFirst: "192.168.1.0/26",
		},
		{
			name:          "IPv4 /16 into 256 parts",
			cidr:          "10.0.0.0/16",
			parts:         256,
			expectedCount: 256,
			expectedFirst: "10.0.0.0/24",
		},
		{
			name:          "IPv6 /64 into 4 parts",
			cidr:          "2001:db8::/64",
			parts:         4,
			expectedCount: 4,
			expectedFirst: "2001:db8::/66",
		},
		{
			name:     "Zero parts",
			cidr:     "192.168.1.0/24",
			parts:    0,
			hasError: true,
		},
		{
			name:     "Negative parts",
			cidr:     "192.168.1.0/24",
			parts:    -1,
			hasError: true,
		},
		{
			name:     "Too many parts",
			cidr:     "192.168.1.0/30",
			parts:    16,
			hasError: true,
		},
		{
			name:     "Invalid CIDR",
			cidr:     "invalid",
			parts:    4,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := DivisionOptions{Parts: tt.parts}
			result, err := Divide(tt.cidr, opts)

			assertError(t, err, tt.hasError)
			if tt.hasError {
				return
			}

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d subnets, got %d", tt.expectedCount, len(result))
			}

			if len(result) > 0 && result[0] != tt.expectedFirst {
				t.Errorf("Expected first subnet %s, got %s", tt.expectedFirst, result[0])
			}
		})
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name          string
		cidr          string
		limit         int
		expectedCount int
		expectedFirst string
		expectedLast  string
		hasError      bool
	}{
		{
			name:          "IPv4 /30",
			cidr:          "192.168.1.0/30",
			limit:         0,
			expectedCount: 4,
			expectedFirst: "192.168.1.0",
			expectedLast:  "192.168.1.3",
		},
		{
			name:          "IPv4 /29 with limit",
			cidr:          "10.0.0.0/29",
			limit:         5,
			expectedCount: 5, // Now respects limit, doesn't error
			expectedFirst: "10.0.0.0",
			expectedLast:  "10.0.0.4",
		},
		{
			name:          "IPv4 /32",
			cidr:          "192.168.1.1/32",
			limit:         0,
			expectedCount: 1,
			expectedFirst: "192.168.1.1",
			expectedLast:  "192.168.1.1",
		},
		{
			name:          "IPv4 /16 streams without OOM",
			cidr:          "10.0.0.0/16",
			limit:         10, // Just get first 10 to prove streaming works
			expectedCount: 10,
			expectedFirst: "10.0.0.0",
			expectedLast:  "10.0.0.9",
		},
		{
			name:          "IPv6 /126",
			cidr:          "2001:db8::/126",
			limit:         0,
			expectedCount: 4,
			expectedFirst: "2001:db8::",
			expectedLast:  "2001:db8::3",
		},
		{
			name:     "Invalid CIDR",
			cidr:     "invalid",
			limit:    0,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ExpansionOptions{Limit: tt.limit}
			resultChan := Expand(context.Background(), tt.cidr, opts)

			// Collect results from channel
			var result []string
			var err error
			for r := range resultChan {
				if r.Err != nil {
					err = r.Err
					break
				}
				result = append(result, r.IP)
			}

			assertError(t, err, tt.hasError)
			if tt.hasError {
				return
			}

			if len(result) != tt.expectedCount {
				t.Errorf("Expected %d IPs, got %d", tt.expectedCount, len(result))
			}

			if len(result) > 0 {
				if result[0] != tt.expectedFirst {
					t.Errorf("Expected first IP %s, got %s", tt.expectedFirst, result[0])
				}

				if result[len(result)-1] != tt.expectedLast {
					t.Errorf("Expected last IP %s, got %s", tt.expectedLast, result[len(result)-1])
				}
			}
		})
	}
}

func TestNetworkInfoOutput(t *testing.T) {
	tests := []struct {
		name   string
		cidr   string
		format string
	}{
		{
			name:   "IPv4 JSON output",
			cidr:   "192.168.1.0/24",
			format: "json",
		},
		{
			name:   "IPv4 YAML output",
			cidr:   "192.168.1.0/24",
			format: "yaml",
		},
		{
			name:   "IPv6 JSON output",
			cidr:   "2001:db8::/64",
			format: "json",
		},
		{
			name:   "IPv6 YAML output",
			cidr:   "2001:db8::/64",
			format: "yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseCIDR(tt.cidr)
			if err != nil {
				t.Fatalf("Failed to parse CIDR: %v", err)
			}

			switch tt.format {
			case "json":
				output, err := info.ToJSON()
				if err != nil {
					t.Errorf("Failed to generate JSON: %v", err)
				}
				if !strings.Contains(output, "base_address") {
					t.Errorf("JSON output missing expected fields")
				}

			case "yaml":
				output, err := info.ToYAML()
				if err != nil {
					t.Errorf("Failed to generate YAML: %v", err)
				}
				if !strings.Contains(output, "base_address:") {
					t.Errorf("YAML output missing expected fields")
				}
			}
		})
	}
}

func TestFormatBigInt(t *testing.T) {
	tests := []struct {
		name     string
		input    *big.Int
		expected string
	}{
		{
			name:     "Small number",
			input:    big.NewInt(123),
			expected: "123",
		},
		{
			name:     "Thousand",
			input:    big.NewInt(1000),
			expected: "1,000",
		},
		{
			name:     "Million",
			input:    big.NewInt(1000000),
			expected: "1,000,000",
		},
		{
			name:     "Large number",
			input:    big.NewInt(12345678901),
			expected: "12,345,678,901",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBigInt(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestIncrementIP(t *testing.T) {
	tests := []struct {
		name     string
		input    net.IP
		expected string
	}{
		{
			name:     "IPv4 simple increment",
			input:    net.ParseIP("192.168.1.1"),
			expected: "192.168.1.2",
		},
		{
			name:     "IPv4 carry over",
			input:    net.ParseIP("192.168.1.255"),
			expected: "192.168.2.0",
		},
		{
			name:     "IPv6 simple increment",
			input:    net.ParseIP("2001:db8::1"),
			expected: "2001:db8::2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := make(net.IP, len(tt.input))
			copy(ip, tt.input)

			incrementIP(ip)

			if ip.String() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, ip.String())
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test getBroadcastAddress
	_, network, _ := net.ParseCIDR("192.168.1.0/24")
	broadcast := getBroadcastAddress(network)
	if broadcast.String() != "192.168.1.255" {
		t.Errorf("Expected broadcast 192.168.1.255, got %s", broadcast.String())
	}

	// Test getFirstUsable
	first := getFirstUsable(network)
	if first.String() != "192.168.1.1" {
		t.Errorf("Expected first usable 192.168.1.1, got %s", first.String())
	}

	// Test getLastUsable
	last := getLastUsable(network)
	if last.String() != "192.168.1.254" {
		t.Errorf("Expected last usable 192.168.1.254, got %s", last.String())
	}

	// Test /31 network (point-to-point)
	_, network31, _ := net.ParseCIDR("192.168.1.0/31")
	first31 := getFirstUsable(network31)
	last31 := getLastUsable(network31)
	if first31.String() != "192.168.1.0" {
		t.Errorf("Expected /31 first usable 192.168.1.0, got %s", first31.String())
	}
	if last31.String() != "192.168.1.1" {
		t.Errorf("Expected /31 last usable 192.168.1.1, got %s", last31.String())
	}

	// Test /32 network (host route)
	_, network32, _ := net.ParseCIDR("192.168.1.1/32")
	first32 := getFirstUsable(network32)
	if first32.String() != "192.168.1.1" {
		t.Errorf("Expected /32 first usable 192.168.1.1, got %s", first32.String())
	}
}
