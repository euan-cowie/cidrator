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

// TestHelper provides common testing utilities
type TestHelper struct {
	t      *testing.T
	binary string
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()
	return &TestHelper{
		t:      t,
		binary: "../../bin/cidrator",
	}
}

// BuildBinaryOnce builds the test binary once and caches the result
var (
	binaryBuilt = false
	buildError  error
)

// EnsureBinaryBuilt ensures the test binary is built
func (h *TestHelper) EnsureBinaryBuilt() error {
	h.t.Helper()

	if binaryBuilt {
		return buildError
	}

	buildCmd := exec.Command("go", "build", "-o", h.binary, "../../main.go")
	buildError = buildCmd.Run()
	binaryBuilt = true

	if buildError != nil {
		h.t.Logf("Failed to build test binary: %v", buildError)
	}

	return buildError
}

// CaptureCommandOutput executes a command and captures its stdout output
func (h *TestHelper) CaptureCommandOutput(cmd *cobra.Command, args []string) (string, error) {
	h.t.Helper()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create fresh command instance to avoid state pollution
	cmdToRun := h.createFreshMTUCommand()

	// Determine subcommand and prepend to args
	switch cmd {
	case discoverCmd:
		args = append([]string{"discover"}, args...)
	case interfacesCmd:
		args = append([]string{"interfaces"}, args...)
	case suggestCmd:
		args = append([]string{"suggest"}, args...)
	case watchCmd:
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

// RunBinaryCommand executes the built binary with given arguments
func (h *TestHelper) RunBinaryCommand(args []string) (string, error) {
	h.t.Helper()

	if err := h.EnsureBinaryBuilt(); err != nil {
		return "", fmt.Errorf("binary not available: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, h.binary, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// createFreshMTUCommand creates a fresh MTU command instance for testing
func (h *TestHelper) createFreshMTUCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mtu",
		Short: "Path-MTU discovery & MTU toolbox",
		Long:  `MTU subcommand provides Path-MTU discovery and MTU analysis tools.`,
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

// AssertTestResult validates test results with enhanced error checking
func (h *TestHelper) AssertTestResult(err error, output string, expectErr bool, expectedSubstrings ...string) {
	h.t.Helper()

	if expectErr {
		if err == nil {
			h.t.Errorf("expected error, got nil")
		}
		return
	}

	if err != nil {
		h.t.Errorf("unexpected error: %v", err)
		return
	}

	for _, expected := range expectedSubstrings {
		if expected != "" && !strings.Contains(output, expected) {
			h.t.Errorf("expected output to contain %q, got %q", expected, output)
		}
	}
}

// AssertErrorType validates that an error is of a specific type or contains specific text
func (h *TestHelper) AssertErrorType(err error, expectedType error, expectedText string) {
	h.t.Helper()

	if err == nil {
		h.t.Errorf("expected error, got nil")
		return
	}

	if expectedType != nil && !errors.Is(err, expectedType) {
		h.t.Errorf("expected error type %T, got %T: %v", expectedType, err, err)
	}

	if expectedText != "" && !strings.Contains(err.Error(), expectedText) {
		h.t.Errorf("expected error to contain %q, got %q", expectedText, err.Error())
	}
}

// ValidateJSON validates that output is valid JSON and optionally checks structure
func (h *TestHelper) ValidateJSON(output string, validator func(map[string]interface{}) error) {
	h.t.Helper()

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		h.t.Errorf("invalid JSON output: %v\nOutput: %s", err, output)
		return
	}

	if validator != nil {
		if err := validator(data); err != nil {
			h.t.Errorf("JSON validation failed: %v\nData: %+v", err, data)
		}
	}
}

// ValidateInterfacesJSON validates the structure of interfaces JSON output
func (h *TestHelper) ValidateInterfacesJSON(output string) {
	h.t.Helper()

	var result struct {
		Interfaces []struct {
			Name string `json:"name"`
			MTU  int    `json:"mtu"`
			Type string `json:"type"`
		} `json:"interfaces"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		h.t.Errorf("failed to parse interfaces JSON: %v\nOutput: %s", err, output)
		return
	}

	if len(result.Interfaces) == 0 {
		h.t.Error("expected at least one interface in JSON output")
	}

	for i, iface := range result.Interfaces {
		if iface.Name == "" {
			h.t.Errorf("interface %d has empty name", i)
		}
		if iface.MTU <= 0 {
			h.t.Errorf("interface %d (%s) has invalid MTU: %d", i, iface.Name, iface.MTU)
		}
		if iface.Type == "" {
			h.t.Errorf("interface %d (%s) has empty type", i, iface.Name)
		}
	}
}

// ValidateSuggestionsJSON validates the structure of suggestions JSON output
func (h *TestHelper) ValidateSuggestionsJSON(output string, expectedTarget string) {
	h.t.Helper()

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
		h.t.Errorf("failed to parse suggestions JSON: %v\nOutput: %s", err, output)
		return
	}

	if result.Target != expectedTarget {
		h.t.Errorf("expected target %q, got %q", expectedTarget, result.Target)
	}

	if result.PMTU <= 0 {
		h.t.Errorf("expected positive PMTU, got %d", result.PMTU)
	}

	// Validate suggestions are reasonable
	if result.Suggestions.TCPMSSv4 <= 0 || result.Suggestions.TCPMSSv4 >= result.PMTU {
		h.t.Errorf("invalid TCP MSS IPv4: %d (PMTU: %d)", result.Suggestions.TCPMSSv4, result.PMTU)
	}

	if result.Suggestions.TCPMSSv6 <= 0 || result.Suggestions.TCPMSSv6 >= result.PMTU {
		h.t.Errorf("invalid TCP MSS IPv6: %d (PMTU: %d)", result.Suggestions.TCPMSSv6, result.PMTU)
	}
}

// ValidateDiscoveryJSON validates the structure of discovery JSON output
func (h *TestHelper) ValidateDiscoveryJSON(output string, expectedTarget string) {
	h.t.Helper()

	var result MTUResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		h.t.Errorf("failed to parse discovery JSON: %v\nOutput: %s", err, output)
		return
	}

	if result.Target != expectedTarget {
		h.t.Errorf("expected target %q, got %q", expectedTarget, result.Target)
	}

	if result.PMTU <= 0 {
		h.t.Errorf("expected positive PMTU, got %d", result.PMTU)
	}

	if result.MSS <= 0 {
		h.t.Errorf("expected positive MSS, got %d", result.MSS)
	}

	if result.ElapsedMS < 0 {
		h.t.Errorf("expected non-negative elapsed time, got %d", result.ElapsedMS)
	}
}

// CreateMockDiscoveryTest creates a standardized mock-based discovery test
func (h *TestHelper) CreateMockDiscoveryTest(name string, pmtu int, protocol string, expectSuccess bool) func(*testing.T) {
	return func(t *testing.T) {
		t.Helper()

		mockDiscoverer := NewMockMTUDiscoverer(TestTargets.Localhost, protocol, pmtu)
		if !expectSuccess {
			mockDiscoverer.SetFailureMode("network_unreachable")
		}
		defer func() {
			if closeErr := mockDiscoverer.Close(); closeErr != nil {
				t.Logf("Warning: failed to close mock discoverer: %v", closeErr)
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), TestTimeouts.Normal)
		defer cancel()

		result, err := mockDiscoverer.DiscoverPMTU(ctx, 576, 9000)

		if expectSuccess {
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("expected result, got nil")
				return
			}

			if result.PMTU != pmtu {
				t.Errorf("PMTU mismatch: got %d, want %d", result.PMTU, pmtu)
			}

			expectedMSS := CalculateExpectedMSS(pmtu, false)
			if result.MSS != expectedMSS {
				t.Errorf("MSS mismatch: got %d, want %d", result.MSS, expectedMSS)
			}
		} else {
			if err == nil {
				t.Errorf("expected error, got nil")
			}
		}
	}
}

// SkipIfNotRoot skips tests that require root privileges
func (h *TestHelper) SkipIfNotRoot() {
	h.t.Helper()

	if os.Geteuid() != 0 {
		h.t.Skip("test requires root privileges")
	}
}

// SkipIfShort skips tests in short mode
func (h *TestHelper) SkipIfShort() {
	h.t.Helper()

	if testing.Short() {
		h.t.Skip("skipping test in short mode")
	}
}
