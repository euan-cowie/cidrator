package mtu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	os.Stdout = w

	runErr := fn()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, readErr := buf.ReadFrom(r)
	if readErr != nil {
		t.Fatalf("failed to read captured stdout: %v", readErr)
	}

	return strings.TrimSpace(buf.String()), runErr
}

func TestWriteJSONLine(t *testing.T) {
	output, err := captureStdout(t, func() error {
		return writeJSONLine(map[string]any{
			"target": "example.com",
			"pmtu":   1500,
		})
	})
	if err != nil {
		t.Fatalf("writeJSONLine returned error: %v", err)
	}

	if strings.Contains(output, "\n") {
		t.Fatalf("expected single-line JSON output, got %q", output)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("writeJSONLine produced invalid JSON: %v", err)
	}

	if parsed["target"] != "example.com" {
		t.Fatalf("unexpected target: %#v", parsed["target"])
	}
}

func TestOutputSuggestionsHelpers(t *testing.T) {
	suggestions := calculateSuggestions(1500)

	jsonOutput, err := captureStdout(t, func() error {
		return outputSuggestionsJSON("example.com", 1500, suggestions)
	})
	if err != nil {
		t.Fatalf("outputSuggestionsJSON returned error: %v", err)
	}

	var jsonResult struct {
		Target      string      `json:"target"`
		PMTU        int         `json:"pmtu"`
		Suggestions Suggestions `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(jsonOutput), &jsonResult); err != nil {
		t.Fatalf("outputSuggestionsJSON produced invalid JSON: %v", err)
	}
	if jsonResult.Target != "example.com" || jsonResult.PMTU != 1500 {
		t.Fatalf("unexpected suggestion JSON payload: %+v", jsonResult)
	}
	if jsonResult.Suggestions.TCPMSSv4 != 1460 {
		t.Fatalf("unexpected IPv4 MSS suggestion: %d", jsonResult.Suggestions.TCPMSSv4)
	}

	tableOutput, err := captureStdout(t, func() error {
		return outputSuggestionsTable("example.com", 1500, suggestions)
	})
	if err != nil {
		t.Fatalf("outputSuggestionsTable returned error: %v", err)
	}

	expectedLines := []string{
		"Suggestions for example.com (PMTU: 1500):",
		"TCP MSS (IPv4):              1460",
		"WireGuard payload:           1440",
		"VXLAN payload:               1450",
	}
	for _, expected := range expectedLines {
		if !strings.Contains(tableOutput, expected) {
			t.Fatalf("expected suggestions table to contain %q, got %q", expected, tableOutput)
		}
	}
}

func TestOutputHopTable(t *testing.T) {
	result := &HopMTUResult{
		Target:       "example.com",
		Protocol:     "icmp",
		MaxProbeSize: 1500,
		FinalPMTU:    1480,
		ElapsedMS:    320,
		Hops: []*HopInfo{
			{
				Hop:  1,
				Addr: net.ParseIP("192.0.2.1"),
				MTU:  1500,
				RTT:  12 * time.Millisecond,
			},
			{
				Hop:     2,
				RTT:     2 * time.Second,
				Timeout: true,
			},
			{
				Hop:   3,
				Error: "administratively prohibited",
			},
		},
	}

	output, err := captureStdout(t, func() error {
		return outputHopTable(result)
	})
	if err != nil {
		t.Fatalf("outputHopTable returned error: %v", err)
	}

	expectedFragments := []string{
		"Hop-by-hop MTU Discovery Results:",
		"Final PMTU: 1480 bytes",
		"192.0.2.1",
		"12.00ms",
		"timeout",
		"administratively prohibited",
	}
	for _, expected := range expectedFragments {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected hop table to contain %q, got %q", expected, output)
		}
	}
}

func TestWatchJSONHelpers(t *testing.T) {
	timestamp := time.Date(2026, time.April, 18, 12, 30, 0, 0, time.UTC)

	errorOutput, err := captureStdout(t, func() error {
		return outputWatchErrorJSON(timestamp, "example.com", fmt.Errorf("timeout"))
	})
	if err != nil {
		t.Fatalf("outputWatchErrorJSON returned error: %v", err)
	}

	var errorResult struct {
		Timestamp string `json:"timestamp"`
		Target    string `json:"target"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal([]byte(errorOutput), &errorResult); err != nil {
		t.Fatalf("outputWatchErrorJSON produced invalid JSON: %v", err)
	}
	if errorResult.Target != "example.com" || errorResult.Error != "timeout" {
		t.Fatalf("unexpected watch error JSON payload: %+v", errorResult)
	}

	watchOutput, err := captureStdout(t, func() error {
		return outputWatchResultJSON(timestamp, &MTUResult{
			Target: "example.com",
			PMTU:   1480,
			MSS:    1440,
		}, true, false)
	})
	if err != nil {
		t.Fatalf("outputWatchResultJSON returned error: %v", err)
	}

	var watchResult struct {
		Target     string `json:"target"`
		PMTU       int    `json:"pmtu"`
		MSS        int    `json:"mss"`
		Changed    bool   `json:"changed"`
		MSSChanged bool   `json:"mss_changed"`
	}
	if err := json.Unmarshal([]byte(watchOutput), &watchResult); err != nil {
		t.Fatalf("outputWatchResultJSON produced invalid JSON: %v", err)
	}
	if watchResult.Target != "example.com" || !watchResult.Changed || watchResult.MSSChanged {
		t.Fatalf("unexpected watch result JSON payload: %+v", watchResult)
	}
}

func TestNewWatchDropErrorPreservesPlainTextErrors(t *testing.T) {
	cmd := &cobra.Command{Use: "watch"}

	err := newWatchDropError(cmd, 1500, 1400, false)
	if err == nil {
		t.Fatal("expected drop error")
	}
	if !cmd.SilenceUsage {
		t.Fatal("expected usage to be silenced")
	}
	if cmd.SilenceErrors {
		t.Fatal("expected non-JSON watch errors to remain visible")
	}
}
