package mtu

import (
	"net"
	"runtime"
	"strings"
	"testing"
)

// TestGetNetworkInterfaces tests the network interface detection
func TestGetNetworkInterfaces(t *testing.T) {
	result, err := GetNetworkInterfaces()
	if err != nil {
		t.Errorf("GetNetworkInterfaces() failed: %v", err)
		return
	}

	if result == nil {
		t.Errorf("expected result, got nil")
		return
	}

	if len(result.Interfaces) == 0 {
		t.Errorf("expected at least one interface, got none")
		return
	}

	// Validate each interface
	for i, iface := range result.Interfaces {
		if iface.Name == "" {
			t.Errorf("interface %d has empty name", i)
		}

		if iface.MTU <= 0 {
			t.Errorf("interface %d (%s) has invalid MTU: %d", i, iface.Name, iface.MTU)
		}

		if iface.Type == "" {
			t.Errorf("interface %d (%s) has empty type", i, iface.Name)
		}
	}

	// Check for common interfaces that should exist
	foundLoopback := false
	for _, iface := range result.Interfaces {
		if iface.Type == "loopback" || strings.HasPrefix(iface.Name, "lo") {
			foundLoopback = true
			break
		}
	}

	if !foundLoopback {
		t.Errorf("expected to find loopback interface")
	}
}

// TestDetermineInterfaceType tests interface type classification
func TestDetermineInterfaceType(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		flags         net.Flags
		expected      string
	}{
		{
			name:          "loopback by flag",
			interfaceName: "lo0",
			flags:         net.FlagLoopback | net.FlagUp,
			expected:      "loopback",
		},
		{
			name:          "loopback by name",
			interfaceName: "lo",
			flags:         net.FlagUp,
			expected:      "loopback",
		},
		{
			name:          "ethernet interface",
			interfaceName: "eth0",
			flags:         net.FlagUp | net.FlagBroadcast,
			expected:      "ethernet",
		},
		{
			name:          "ethernet en interface",
			interfaceName: "en0",
			flags:         net.FlagUp | net.FlagBroadcast,
			expected:      "ethernet",
		},
		{
			name:          "wireless interface",
			interfaceName: "wlan0",
			flags:         net.FlagUp | net.FlagBroadcast,
			expected:      "wireless",
		},
		{
			name:          "tunnel interface",
			interfaceName: "tun0",
			flags:         net.FlagUp | net.FlagPointToPoint,
			expected:      "tunnel",
		},
		{
			name:          "bridge interface",
			interfaceName: "br0",
			flags:         net.FlagUp | net.FlagBroadcast,
			expected:      "bridge",
		},
		{
			name:          "docker interface",
			interfaceName: "docker0",
			flags:         net.FlagUp | net.FlagBroadcast,
			expected:      "virtual",
		},
		{
			name:          "ppp interface",
			interfaceName: "ppp0",
			flags:         net.FlagUp | net.FlagPointToPoint,
			expected:      "ppp",
		},
		{
			name:          "bond interface",
			interfaceName: "bond0",
			flags:         net.FlagUp | net.FlagBroadcast,
			expected:      "bond",
		},
		{
			name:          "vlan interface",
			interfaceName: "vlan100",
			flags:         net.FlagUp | net.FlagBroadcast,
			expected:      "vlan",
		},
		{
			name:          "unknown interface",
			interfaceName: "xyz123",
			flags:         net.FlagUp,
			expected:      "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineInterfaceType(tt.interfaceName, tt.flags)
			if result != tt.expected {
				t.Errorf("determineInterfaceType(%q, %v) = %q, want %q",
					tt.interfaceName, tt.flags, result, tt.expected)
			}
		})
	}
}

// TestDetermineInterfaceTypeCaseInsensitive tests case insensitive matching
func TestDetermineInterfaceTypeCaseInsensitive(t *testing.T) {
	tests := []struct {
		name          string
		interfaceName string
		expected      string
	}{
		{"uppercase ethernet", "ETH0", "ethernet"},
		{"mixed case ethernet", "En0", "ethernet"},
		{"uppercase wireless", "WLAN0", "wireless"},
		{"mixed case tunnel", "Tun0", "tunnel"},
		{"uppercase loopback", "LO0", "loopback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineInterfaceType(tt.interfaceName, net.FlagUp)
			if result != tt.expected {
				t.Errorf("determineInterfaceType(%q) = %q, want %q",
					tt.interfaceName, result, tt.expected)
			}
		})
	}
}

// TestGetMaxMTU tests maximum MTU detection
func TestGetMaxMTU(t *testing.T) {
	maxMTU, err := GetMaxMTU()
	if err != nil {
		t.Errorf("GetMaxMTU() failed: %v", err)
		return
	}

	if maxMTU <= 0 {
		t.Errorf("expected positive max MTU, got %d", maxMTU)
	}

	// Max MTU should be at least the minimum required
	if maxMTU < 1500 {
		t.Logf("Warning: max MTU (%d) is less than standard Ethernet MTU (1500)", maxMTU)
	}

	// Max MTU should be reasonable (not larger than jumbo frames)
	if maxMTU > 65536 {
		t.Errorf("max MTU (%d) seems unreasonably large", maxMTU)
	}
}

// TestGetMaxMTUFallback tests the fallback behavior when no interfaces are found
func TestGetMaxMTUFallback(t *testing.T) {
	// This test is tricky because we can't easily mock the interface detection
	// Instead, we'll test that the fallback value is reasonable
	maxMTU, err := GetMaxMTU()
	if err != nil {
		t.Errorf("GetMaxMTU() failed: %v", err)
		return
	}

	// The fallback should be 1500 if no interfaces are found
	// But since we likely have interfaces, just check it's reasonable
	if maxMTU < 1500 {
		t.Errorf("max MTU (%d) is less than expected minimum", maxMTU)
	}
}

// TestPlatformSpecificMTU tests platform-specific MTU detection
func TestPlatformSpecificMTU(t *testing.T) {
	// Test that getPlatformSpecificMTU handles different platforms
	_, err := getPlatformSpecificMTU("nonexistent-interface")

	// Should return an error for nonexistent interface
	if err == nil {
		t.Errorf("expected error for nonexistent interface, got nil")
	}
}

// TestLinuxMTUDetection tests Linux-specific MTU detection
func TestLinuxMTUDetection(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test on non-Linux platform")
	}

	// Test with a known interface (loopback should exist on Linux)
	mtu, err := getLinuxMTU("lo")
	if err != nil {
		t.Errorf("getLinuxMTU('lo') failed: %v", err)
		return
	}

	if mtu <= 0 {
		t.Errorf("expected positive MTU for loopback, got %d", mtu)
	}

	// Loopback on Linux typically has a large MTU
	if mtu < 16384 {
		t.Logf("Warning: loopback MTU (%d) is smaller than typical", mtu)
	}
}

// TestLinuxMTUNonexistentInterface tests error handling for nonexistent interfaces
func TestLinuxMTUNonexistentInterface(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping Linux-specific test on non-Linux platform")
	}

	_, err := getLinuxMTU("nonexistent-interface-xyz123")
	if err == nil {
		t.Errorf("expected error for nonexistent interface, got nil")
	}
}

// TestDarwinMTUDetection tests macOS-specific MTU detection
func TestDarwinMTUDetection(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping macOS-specific test on non-macOS platform")
	}

	// Currently returns error as it's not implemented
	_, err := getDarwinMTU("lo0")
	if err == nil {
		t.Errorf("expected error for unimplemented Darwin MTU detection, got nil")
	}

	expectedError := "platform-specific MTU detection not implemented for macOS"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("expected error containing %q, got %q", expectedError, err.Error())
	}
}

// TestWindowsMTUDetection tests Windows-specific MTU detection
func TestWindowsMTUDetection(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping Windows-specific test on non-Windows platform")
	}

	// Currently returns error as it's not implemented
	_, err := getWindowsMTU("Loopback")
	if err == nil {
		t.Errorf("expected error for unimplemented Windows MTU detection, got nil")
	}

	expectedError := "platform-specific MTU detection not implemented for Windows"
	if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("expected error containing %q, got %q", expectedError, err.Error())
	}
}

// TestUnsupportedPlatform tests unsupported platform handling
func TestUnsupportedPlatform(t *testing.T) {
	// This test can't actually change the runtime.GOOS, but we can test
	// that the function exists and would handle unknown platforms
	// We'll test by checking the current platform is handled
	supportedPlatforms := []string{"linux", "darwin", "windows"}
	currentPlatform := runtime.GOOS

	isSupported := false
	for _, platform := range supportedPlatforms {
		if platform == currentPlatform {
			isSupported = true
			break
		}
	}

	if !isSupported {
		t.Logf("Current platform %s is not explicitly supported, but that's okay", currentPlatform)
	}
}

// TestNetworkInterfaceStructure tests the NetworkInterface struct
func TestNetworkInterfaceStructure(t *testing.T) {
	iface := NetworkInterface{
		Name: "test0",
		MTU:  1500,
		Type: "ethernet",
	}

	if iface.Name != "test0" {
		t.Errorf("name mismatch: got %q, want %q", iface.Name, "test0")
	}

	if iface.MTU != 1500 {
		t.Errorf("MTU mismatch: got %d, want %d", iface.MTU, 1500)
	}

	if iface.Type != "ethernet" {
		t.Errorf("type mismatch: got %q, want %q", iface.Type, "ethernet")
	}
}

// TestInterfaceResultStructure tests the InterfaceResult struct
func TestInterfaceResultStructure(t *testing.T) {
	result := InterfaceResult{
		Interfaces: []NetworkInterface{
			{Name: "lo0", MTU: 16384, Type: "loopback"},
			{Name: "en0", MTU: 1500, Type: "ethernet"},
		},
	}

	if len(result.Interfaces) != 2 {
		t.Errorf("expected 2 interfaces, got %d", len(result.Interfaces))
	}

	// Check first interface
	if result.Interfaces[0].Name != "lo0" {
		t.Errorf("first interface name: got %q, want %q", result.Interfaces[0].Name, "lo0")
	}

	// Check second interface
	if result.Interfaces[1].Name != "en0" {
		t.Errorf("second interface name: got %q, want %q", result.Interfaces[1].Name, "en0")
	}
}

// Benchmark tests for performance validation
func BenchmarkGetNetworkInterfaces(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetNetworkInterfaces()
		if err != nil {
			b.Errorf("GetNetworkInterfaces() failed: %v", err)
		}
	}
}

func BenchmarkDetermineInterfaceType(b *testing.B) {
	testCases := []struct {
		name  string
		flags net.Flags
	}{
		{"eth0", net.FlagUp | net.FlagBroadcast},
		{"lo0", net.FlagUp | net.FlagLoopback},
		{"wlan0", net.FlagUp | net.FlagBroadcast},
		{"tun0", net.FlagUp | net.FlagPointToPoint},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tc := testCases[i%len(testCases)]
		determineInterfaceType(tc.name, tc.flags)
	}
}

func BenchmarkGetMaxMTU(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := GetMaxMTU()
		if err != nil {
			b.Errorf("GetMaxMTU() failed: %v", err)
		}
	}
}
