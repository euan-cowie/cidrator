package mtu

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Test helpers following existing patterns

// captureCommandOutput executes a command and captures its stdout output
func captureCommandOutput(t *testing.T, cmd *cobra.Command, args []string) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.SetArgs(args)
	err := cmd.Execute()

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
			expectedSubstr: "Binary-search to the largest size",
		},
		{
			name:           "invalid destination",
			args:           []string{"invalid..destination"},
			expectError:    true,
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
			expectedSubstr: "List local interfaces",
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
	output, err := captureCommandOutput(t, interfacesCmd, []string{"--json"})
	if err != nil {
		t.Fatalf("interfaces command failed: %v", err)
	}

	var result struct {
		Interfaces []struct {
			Name string `json:"name"`
			MTU  int    `json:"mtu"`
			Type string `json:"type"`
		} `json:"interfaces"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("failed to parse JSON output: %v\nOutput: %s", err, output)
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
			expectedSubstr: "Print TCP MSS",
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
			expectedSubstr: "suggestions",
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
	output, err := captureCommandOutput(t, suggestCmd, []string{"localhost", "--json"})
	if err != nil {
		t.Fatalf("suggest command failed: %v", err)
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

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Errorf("failed to parse JSON output: %v\nOutput: %s", err, output)
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
			expectedSubstr: "Re-run discover every N seconds",
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

// Benchmark tests for performance validation
func BenchmarkCalculateSuggestions(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateSuggestions(1500)
	}
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
		outputJSON(result)
	}
}
