package mtu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// Test helpers following existing patterns

// createFreshMTUCommand creates a fresh MTU command instance for testing
func createFreshMTUCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mtu",
		Short: "Path-MTU discovery & MTU toolbox",
		Long: `MTU subcommand provides Path-MTU discovery and MTU analysis tools.

The mtu sub-command is a smart wrapper around the techniques in RFC 1191 (IPv4),
RFC 8201 (IPv6) and RFC 4821 (PLPMTUD). It answers three everyday questions:
• What MTU can I safely send to that host?
• Did today's change introduce an MTU black-hole?
• What MSS or VPN segment size should I configure?

Available operations:
- discover: Binary-search to the largest size that gets through (default)
- watch: Re-run discover every N seconds and notify on change
- interfaces: List local interfaces + configured MTU
- suggest: Print TCP MSS / IPSec ESP / WireGuard frame sizes for the path

All commands support both IPv4 and IPv6 with multiple probe protocols.`,
	}

	// Add subcommands
	cmd.AddCommand(discoverCmd)
	cmd.AddCommand(watchCmd)
	cmd.AddCommand(interfacesCmd)
	cmd.AddCommand(suggestCmd)

	// Global flags for MTU commands
	cmd.PersistentFlags().Bool("4", false, "Force IPv4")
	cmd.PersistentFlags().Bool("6", false, "Force IPv6")
	cmd.PersistentFlags().String("proto", "icmp", "Probe method (icmp|udp|tcp)")
	cmd.PersistentFlags().Int("min", 0, "Lower bound (IPv4 default: 576, IPv6: 1280)")
	cmd.PersistentFlags().Int("max", 9216, "Upper bound")
	cmd.PersistentFlags().Int("step", 16, "Granularity for linear sweep mode")
	cmd.PersistentFlags().Duration("timeout", 0, "Wait per probe (default: 2s)")
	cmd.PersistentFlags().Int("ttl", 64, "Initial hop limit")
	cmd.PersistentFlags().Bool("json", false, "Structured output")
	cmd.PersistentFlags().Bool("quiet", false, "Suppress progress bar")
	cmd.PersistentFlags().Int("pps", 10, "Rate limit probes per second")

	return cmd
}

// captureCommandOutput executes a command and captures its stdout output
func captureCommandOutput(t *testing.T, cmd *cobra.Command, args []string) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Use fresh MTU command for all subcommands
	var cmdToRun *cobra.Command
	switch cmd {
	case discoverCmd:
		cmdToRun = createFreshMTUCommand()
		args = append([]string{"discover"}, args...)
	case interfacesCmd:
		cmdToRun = createFreshMTUCommand()
		args = append([]string{"interfaces"}, args...)
	case suggestCmd:
		cmdToRun = createFreshMTUCommand()
		args = append([]string{"suggest"}, args...)
	case watchCmd:
		cmdToRun = createFreshMTUCommand()
		args = append([]string{"watch"}, args...)
	default:
		cmdToRun = cmd
	}

	cmdToRun.SetArgs(args)
	err := cmdToRun.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, readErr := buf.ReadFrom(r)
	return strings.TrimSpace(buf.String()), errors.Join(err, readErr)
}

// assertTestResult validates test results with common patterns
func assertTestResult(t *testing.T, err error, output string, expectErr bool, expectedSubstring string) {
	t.Helper()

	if expectErr {
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		return
	}

	if err != nil {
		t.Errorf("unexpected error: %v", err)
		return
	}

	if expectedSubstring != "" && !strings.Contains(output, expectedSubstring) {
		t.Errorf("expected output to contain %q, got %q", expectedSubstring, output)
	}
}

// TestMTUCommand tests the main MTU command
func TestMTUCommand(t *testing.T) {
	output, err := captureCommandOutput(t, MTUCmd, []string{"--help"})
	assertTestResult(t, err, output, false, "Path-MTU discovery")
	assertTestResult(t, err, output, false, "discover")
	assertTestResult(t, err, output, false, "watch")
	assertTestResult(t, err, output, false, "interfaces")
	assertTestResult(t, err, output, false, "suggest")
}

// TestDiscoverCommand tests the discover subcommand
func TestDiscoverCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectedSubstr string
	}{
		{
			name:           "no arguments",
			args:           []string{},
			expectError:    true,
			expectedSubstr: "",
		},
		{
			name:           "help flag",
			args:           []string{"--help"},
			expectError:    false,
			expectedSubstr: "Path-MTU discovery using binary search",
		},
		{
			name:           "invalid destination",
			args:           []string{"invalid..destination"},
			expectError:    false, // Command accepts any string as destination, validation happens during execution
			expectedSubstr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := captureCommandOutput(t, discoverCmd, tt.args)
			assertTestResult(t, err, output, tt.expectError, tt.expectedSubstr)
		})
	}
}

// TestInterfacesCommand tests the interfaces subcommand
func TestInterfacesCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectedSubstr string
	}{
		{
			name:           "default output",
			args:           []string{},
			expectError:    false,
			expectedSubstr: "Interface",
		},
		{
			name:           "json output",
			args:           []string{"--json"},
			expectError:    false,
			expectedSubstr: "interfaces",
		},
		{
			name:           "help flag",
			args:           []string{"--help"},
			expectError:    false,
			expectedSubstr: "local network interfaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := captureCommandOutput(t, interfacesCmd, tt.args)
			assertTestResult(t, err, output, tt.expectError, tt.expectedSubstr)
		})
	}
}

// TestInterfacesJSONOutput tests JSON output format
func TestInterfacesJSONOutput(t *testing.T) {
	// Skip if we can't build the binary
	binary := "../../bin/cidrator"
	if _, err := os.Stat(binary); os.IsNotExist(err) {
		// Try to build it
		buildCmd := exec.Command("go", "build", "-o", binary, "../../main.go")
		if err := buildCmd.Run(); err != nil {
			t.Skip("Cannot build binary for integration test")
		}
	}

	cmd := exec.Command(binary, "mtu", "interfaces", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("interfaces command failed: %v, output: %s", err, string(output))
	}

	var result struct {
		Interfaces []struct {
			Name string `json:"name"`
			MTU  int    `json:"mtu"`
			Type string `json:"type"`
		} `json:"interfaces"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		t.Errorf("failed to parse JSON output: %v\nOutput: %s", err, string(output))
		return
	}

	if len(result.Interfaces) == 0 {
		t.Error("expected at least one interface in JSON output")
	}

	// Validate interface structure
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
}

// TestSuggestCommand tests the suggest subcommand
func TestSuggestCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectedSubstr string
	}{
		{
			name:           "no arguments",
			args:           []string{},
			expectError:    true,
			expectedSubstr: "",
		},
		{
			name:           "help flag",
			args:           []string{"--help"},
			expectError:    false,
			expectedSubstr: "optimal frame sizes",
		},
		{
			name:           "localhost suggestion",
			args:           []string{"localhost"},
			expectError:    false,
			expectedSubstr: "TCP MSS",
		},
		{
			name:           "localhost json",
			args:           []string{"localhost", "--json"},
			expectError:    false,
			expectedSubstr: "", // Skip this test case - covered by TestSuggestJSONOutput
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := captureCommandOutput(t, suggestCmd, tt.args)
			assertTestResult(t, err, output, tt.expectError, tt.expectedSubstr)
		})
	}
}

// TestSuggestJSONOutput tests suggest JSON output format
func TestSuggestJSONOutput(t *testing.T) {
	// Skip if we can't build the binary
	binary := "../../bin/cidrator"
	if _, err := os.Stat(binary); os.IsNotExist(err) {
		// Try to build it
		buildCmd := exec.Command("go", "build", "-o", binary, "../../main.go")
		if err := buildCmd.Run(); err != nil {
			t.Skip("Cannot build binary for integration test")
		}
	}

	cmd := exec.Command(binary, "mtu", "suggest", "localhost", "--json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("suggest command failed: %v, output: %s", err, string(output))
	}

	var result struct {
		Target      string `json:"target"`
		PMTU        int    `json:"pmtu"`
		Suggestions struct {
			TCPMSSv4         int `json:"tcp_mss_ipv4"`
			TCPMSSv6         int `json:"tcp_mss_ipv6"`
			WireGuardPayload int `json:"wireguard_payload"`
			IPSecESPUDP      int `json:"ipsec_esp_udp"`
		} `json:"suggestions"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		t.Errorf("failed to parse JSON output: %v\nOutput: %s", err, string(output))
		return
	}

	if result.Target != "localhost" {
		t.Errorf("expected target 'localhost', got %q", result.Target)
	}

	if result.PMTU <= 0 {
		t.Errorf("expected positive PMTU, got %d", result.PMTU)
	}

	// Validate suggestions are reasonable
	if result.Suggestions.TCPMSSv4 <= 0 || result.Suggestions.TCPMSSv4 >= result.PMTU {
		t.Errorf("invalid TCP MSS IPv4: %d (PMTU: %d)", result.Suggestions.TCPMSSv4, result.PMTU)
	}

	if result.Suggestions.TCPMSSv6 <= 0 || result.Suggestions.TCPMSSv6 >= result.PMTU {
		t.Errorf("invalid TCP MSS IPv6: %d (PMTU: %d)", result.Suggestions.TCPMSSv6, result.PMTU)
	}
}

// TestWatchCommand tests the watch subcommand (basic functionality)
func TestWatchCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectError    bool
		expectedSubstr string
	}{
		{
			name:           "no arguments",
			args:           []string{},
			expectError:    true,
			expectedSubstr: "",
		},
		{
			name:           "help flag",
			args:           []string{"--help"},
			expectError:    false,
			expectedSubstr: "continuously monitors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := captureCommandOutput(t, watchCmd, tt.args)
			assertTestResult(t, err, output, tt.expectError, tt.expectedSubstr)
		})
	}
}

// TestMTUResultStructure tests the MTUResult struct
func TestMTUResultStructure(t *testing.T) {
	result := &MTUResult{
		Target:    "example.com",
		Protocol:  "icmp",
		PMTU:      1500,
		MSS:       1460,
		Hops:      10,
		ElapsedMS: 250,
	}

	// Test JSON marshaling
	data, err := json.Marshal(result)
	if err != nil {
		t.Errorf("failed to marshal MTUResult: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled MTUResult
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Errorf("failed to unmarshal MTUResult: %v", err)
	}

	if unmarshaled.Target != result.Target {
		t.Errorf("target mismatch: got %q, want %q", unmarshaled.Target, result.Target)
	}

	if unmarshaled.PMTU != result.PMTU {
		t.Errorf("PMTU mismatch: got %d, want %d", unmarshaled.PMTU, result.PMTU)
	}
}

// TestOutputJSON tests JSON output formatting
func TestOutputJSON(t *testing.T) {
	result := &MTUResult{
		Target:    "test.example.com",
		Protocol:  "tcp",
		PMTU:      1472,
		MSS:       1432,
		Hops:      8,
		ElapsedMS: 180,
	}

	// Capture JSON output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputJSON(result)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	if err != nil {
		t.Errorf("outputJSON failed: %v", err)
	}

	// Validate JSON structure
	var parsed MTUResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("outputJSON produced invalid JSON: %v\nOutput: %s", err, output)
	}

	if parsed.Target != result.Target {
		t.Errorf("JSON target mismatch: got %q, want %q", parsed.Target, result.Target)
	}
}

// TestOutputTable tests table output formatting
func TestOutputTable(t *testing.T) {
	result := &MTUResult{
		Target:    "test.example.com",
		Protocol:  "udp",
		PMTU:      1500,
		MSS:       1460,
		Hops:      12,
		ElapsedMS: 220,
	}

	// Capture table output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := outputTable(result)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	output := strings.TrimSpace(buf.String())

	if err != nil {
		t.Errorf("outputTable failed: %v", err)
	}

	// Validate table contains expected fields
	expectedFields := []string{
		"Target:", result.Target,
		"Protocol:", result.Protocol,
		"Path MTU:", "1500",
		"TCP MSS:", "1460",
		"Hops:", "12",
		"Elapsed:", "220ms",
	}

	for _, field := range expectedFields {
		if !strings.Contains(output, field) {
			t.Errorf("table output missing field %q\nOutput: %s", field, output)
		}
	}
}

// TestCalculateSuggestions tests the suggestions calculation logic
func TestCalculateSuggestions(t *testing.T) {
	tests := []struct {
		name                string
		pmtu                int
		expectedTCPMSSv4    int
		expectedTCPMSSv6    int
		expectedWireGuard   int
		expectedIPSecESPUDP int
	}{
		{
			name:                "Standard Ethernet MTU",
			pmtu:                1500,
			expectedTCPMSSv4:    1460, // 1500 - 40
			expectedTCPMSSv6:    1440, // 1500 - 60
			expectedWireGuard:   1440, // 1500 - 60
			expectedIPSecESPUDP: 1416, // 1500 - 84
		},
		{
			name:                "Jumbo Frame MTU",
			pmtu:                9000,
			expectedTCPMSSv4:    8960, // 9000 - 40
			expectedTCPMSSv6:    8940, // 9000 - 60
			expectedWireGuard:   8940, // 9000 - 60
			expectedIPSecESPUDP: 8916, // 9000 - 84
		},
		{
			name:                "Low MTU",
			pmtu:                576,
			expectedTCPMSSv4:    536, // 576 - 40
			expectedTCPMSSv6:    516, // 576 - 60
			expectedWireGuard:   516, // 576 - 60
			expectedIPSecESPUDP: 492, // 576 - 84
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := calculateSuggestions(tt.pmtu)

			if suggestions.TCPMSSv4 != tt.expectedTCPMSSv4 {
				t.Errorf("TCP MSS IPv4: got %d, want %d", suggestions.TCPMSSv4, tt.expectedTCPMSSv4)
			}

			if suggestions.TCPMSSv6 != tt.expectedTCPMSSv6 {
				t.Errorf("TCP MSS IPv6: got %d, want %d", suggestions.TCPMSSv6, tt.expectedTCPMSSv6)
			}

			if suggestions.WireGuardPayload != tt.expectedWireGuard {
				t.Errorf("WireGuard payload: got %d, want %d", suggestions.WireGuardPayload, tt.expectedWireGuard)
			}

			if suggestions.IPSecESPUDP != tt.expectedIPSecESPUDP {
				t.Errorf("IPSec ESP+UDP: got %d, want %d", suggestions.IPSecESPUDP, tt.expectedIPSecESPUDP)
			}
		})
	}
}

// TestMockNetworkProber demonstrates proper mock usage
func TestMockNetworkProber(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*MockNetworkProber)
		probeSize   int
		expectError bool
	}{
		{
			name: "successful probe",
			setupMock: func(m *MockNetworkProber) {
				m.SetResponse(1500, true, nil)
			},
			probeSize:   1500,
			expectError: false,
		},
		{
			name: "fragmentation needed",
			setupMock: func(m *MockNetworkProber) {
				icmpErr := &ICMPError{
					Type:    3,
					Code:    4,
					Message: "Fragmentation Needed",
				}
				m.SetResponse(1600, false, icmpErr)
			},
			probeSize:   1600,
			expectError: false,
		},
		{
			name: "probe after closure",
			setupMock: func(m *MockNetworkProber) {
				_ = m.Close() // Ignore error in test setup
			},
			probeSize:   1500,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockNetworkProber()
			defer func() {
				if closeErr := mock.Close(); closeErr != nil {
					t.Logf("Warning: failed to close mock: %v", closeErr)
				}
			}()
			tt.setupMock(mock)

			ctx, cancel := context.WithTimeout(context.Background(), TestTimeouts.Short)
			defer cancel()

			result := mock.Probe(ctx, tt.probeSize)

			if tt.expectError {
				if result.Error == nil {
					t.Errorf("expected error, got none")
				}
			} else {
				if result.Error != nil {
					t.Errorf("unexpected error: %v", result.Error)
				}
				if result.Size != tt.probeSize {
					t.Errorf("size mismatch: got %d, want %d", result.Size, tt.probeSize)
				}
			}
		})
	}
}

// TestMockMTUDiscovery demonstrates proper MTU discovery mocking
func TestMockMTUDiscovery(t *testing.T) {
	// Test standard MTU scenarios using test constants
	for _, scenario := range TestMTUScenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			helper := NewTestHelper(t)
			testFunc := helper.CreateMockDiscoveryTest(scenario.Name, scenario.MTU, "tcp", true)
			testFunc(t)
		})
	}
}

// TestMTUDiscoveryFailureModes tests various failure scenarios
func TestMTUDiscoveryFailureModes(t *testing.T) {
	tests := []struct {
		name        string
		failureMode string
		expectedErr string
	}{
		{
			name:        "network unreachable",
			failureMode: "network_unreachable",
			expectedErr: "network unreachable",
		},
		{
			name:        "permission denied",
			failureMode: "permission_denied",
			expectedErr: "operation not permitted",
		},
		{
			name:        "timeout",
			failureMode: "timeout",
			expectedErr: "operation timed out",
		},
		{
			name:        "no working MTU",
			failureMode: "no_working_mtu",
			expectedErr: "no working MTU found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockMTUDiscoverer(TestTargets.Localhost, "tcp", 1500)
			mock.SetFailureMode(tt.failureMode)
			defer func() {
				if closeErr := mock.Close(); closeErr != nil {
					t.Logf("Warning: failed to close mock: %v", closeErr)
				}
			}()

			ctx, cancel := context.WithTimeout(context.Background(), TestTimeouts.Normal)
			defer cancel()

			_, err := mock.DiscoverPMTU(ctx, 576, 1500)
			if err == nil {
				t.Errorf("expected error, got nil")
				return
			}

			helper := NewTestHelper(t)
			helper.AssertErrorType(err, nil, tt.expectedErr)
		})
	}
}

// TestCalculateExpectedValues demonstrates using test constants
func TestCalculateExpectedValues(t *testing.T) {
	tests := []struct {
		name     string
		mtu      int
		protocol string
		ipv6     bool
		expected int
	}{
		{
			name:     "IPv4 TCP MSS",
			mtu:      1500,
			protocol: "tcp",
			ipv6:     false,
			expected: 1500 - IPv4HeaderSize - TCPHeaderSize,
		},
		{
			name:     "IPv6 TCP MSS",
			mtu:      1500,
			protocol: "tcp",
			ipv6:     true,
			expected: 1500 - IPv6HeaderSize - TCPHeaderSize,
		},
		{
			name:     "WireGuard payload",
			mtu:      1500,
			protocol: "wireguard",
			ipv6:     false,
			expected: 1500 - WireGuardOverhead,
		},
		{
			name:     "IPSec ESP+UDP",
			mtu:      1500,
			protocol: "ipsec",
			ipv6:     false,
			expected: 1500 - IPSecESPUDPOverhead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result int
			if tt.protocol == "tcp" {
				result = CalculateExpectedMSS(tt.mtu, tt.ipv6)
			} else {
				result = CalculateExpectedPayload(tt.mtu, tt.protocol, tt.ipv6)
			}

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// TestMTUFragmentationSimulation demonstrates realistic MTU discovery simulation
func TestMTUFragmentationSimulation(t *testing.T) {
	mock := NewMockNetworkProber()
	mock.SetMTUFragmentationPoint(1472) // Simulate path with 1472 MTU

	ctx := context.Background()

	// Test sizes around the fragmentation point
	testSizes := []struct {
		size            int
		expectedSuccess bool
	}{
		{1400, true},  // Should succeed
		{1472, true},  // Should succeed (at the limit)
		{1500, false}, // Should fail (too large)
		{1600, false}, // Should fail (too large)
	}

	for _, test := range testSizes {
		t.Run(fmt.Sprintf("size_%d", test.size), func(t *testing.T) {
			result := mock.Probe(ctx, test.size)

			if result.Success != test.expectedSuccess {
				t.Errorf("size %d: expected success=%v, got %v",
					test.size, test.expectedSuccess, result.Success)
			}

			if !test.expectedSuccess && result.ICMPErr == nil {
				t.Errorf("size %d: expected ICMP error for failed probe", test.size)
			}
		})
	}

	// Verify call count tracking
	expectedCalls := len(testSizes)
	if mock.GetCallCount() != expectedCalls {
		t.Errorf("expected %d calls, got %d", expectedCalls, mock.GetCallCount())
	}
}

// TestRateLimitingWithSkipping shows improved rate limiting tests
func TestRateLimitingWithSkipping(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing test in short mode")
	}

	limiter := NewRateLimiter(10) // 10 PPS

	start := time.Now()

	// Make 5 calls
	for i := 0; i < 5; i++ {
		limiter.Wait()
	}

	elapsed := time.Since(start)

	// Should take at least 400ms for 5 packets at 10 PPS
	expectedMin := 400 * time.Millisecond
	if elapsed < expectedMin {
		t.Errorf("rate limiting too fast: took %v, expected at least %v", elapsed, expectedMin)
	}
}

// Benchmark tests for performance validation
func BenchmarkCalculateSuggestions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateSuggestions(1500)
	}
}

func BenchmarkImprovedMTUCalculations(b *testing.B) {
	b.Run("CalculateExpectedMSS", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CalculateExpectedMSS(1500, false)
		}
	})

	b.Run("CalculateExpectedPayload", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			CalculateExpectedPayload(1500, "tcp", false)
		}
	})
}

func BenchmarkOutputJSON(b *testing.B) {
	result := &MTUResult{
		Target:    "benchmark.example.com",
		Protocol:  "icmp",
		PMTU:      1500,
		MSS:       1460,
		Hops:      10,
		ElapsedMS: 150,
	}

	// Redirect output to discard
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = outputJSON(result) // Ignore error in benchmark
	}
}
