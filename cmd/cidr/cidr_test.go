package cidr

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Test helpers

// captureCommandOutput executes a command and captures its stdout output
func captureCommandOutput(t *testing.T, cmd *cobra.Command, args []string) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd.SetArgs(args)
	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return strings.TrimSpace(buf.String()), err
}

// createTestCommand creates a test command with the specified configuration
func createTestCommand(use string, args int, runE func(cmd *cobra.Command, args []string) error) *cobra.Command {
	return &cobra.Command{
		Use:  use,
		Args: cobra.ExactArgs(args),
		RunE: runE,
	}
}

// assertTestResult validates test results with common patterns
func assertTestResult(t *testing.T, err error, output string, expectErr bool, expectedOutput string) {
	t.Helper()

	if expectErr {
		if err == nil {
			t.Error("Expected error but got none")
		}
		return
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if expectedOutput != "" && output != expectedOutput {
		t.Errorf("Expected output %q, got %q", expectedOutput, output)
	}
}

func TestExplainCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(t *testing.T, output string)
	}{
		{
			name: "IPv4 /24 table format",
			args: []string{"explain", "192.168.1.0/24"},
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "192.168.1.0") {
					t.Error("Output should contain base address")
				}
				if !strings.Contains(output, "192.168.1.255") {
					t.Error("Output should contain broadcast address")
				}
				if !strings.Contains(output, "256") {
					t.Error("Output should contain total addresses")
				}
			},
		},
		{
			name: "IPv4 /24 JSON format",
			args: []string{"explain", "192.168.1.0/24", "--format", "json"},
			checkFunc: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid JSON output: %v", err)
				}
				if result["base_address"] != "192.168.1.0" {
					t.Error("JSON should contain correct base address")
				}
				if result["is_ipv6"] != false {
					t.Error("JSON should indicate IPv4")
				}
			},
		},
		{
			name: "IPv4 /24 YAML format",
			args: []string{"explain", "192.168.1.0/24", "--format", "yaml"},
			checkFunc: func(t *testing.T, output string) {
				var result map[string]interface{}
				if err := yaml.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("Invalid YAML output: %v", err)
				}
				if result["base_address"] != "192.168.1.0" {
					t.Error("YAML should contain correct base address")
				}
			},
		},
		{
			name: "IPv6 /64 table format",
			args: []string{"explain", "2001:db8::/64"},
			checkFunc: func(t *testing.T, output string) {
				if !strings.Contains(output, "2001:db8::") {
					t.Error("Output should contain IPv6 base address")
				}
				if !strings.Contains(output, "true") {
					t.Error("Output should indicate IPv6")
				}
			},
		},
		{
			name:      "Invalid CIDR",
			args:      []string{"explain", "invalid"},
			expectErr: true,
		},
		{
			name:      "Invalid format",
			args:      []string{"explain", "192.168.1.0/24", "--format", "xml"},
			expectErr: true,
		},
		{
			name:      "No arguments",
			args:      []string{"explain"},
			expectErr: true,
		},
		{
			name:      "Too many arguments",
			args:      []string{"explain", "192.168.1.0/24", "extra"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the output format flag for each test
			config.Explain.OutputFormat = "table"

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create a new command instance for each test
			cmd := &cobra.Command{
				Use:  "explain <CIDR>",
				Args: cobra.ExactArgs(1),
				RunE: explainCmd.RunE,
			}
			cmd.Flags().StringVarP(&config.Explain.OutputFormat, "format", "f", "table", "Output format")

			// Execute command
			cmd.SetArgs(tt.args[1:]) // Remove "explain" from args
			err := cmd.Execute()

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check results
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

func TestExpandCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
		checkFunc func(t *testing.T, output string)
	}{
		{
			name: "IPv4 /30 default output",
			args: []string{"expand", "192.168.1.0/30"},
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 4 {
					t.Errorf("Expected 4 IP addresses, got %d", len(lines))
				}
				if lines[0] != "192.168.1.0" {
					t.Errorf("Expected first IP 192.168.1.0, got %s", lines[0])
				}
				if lines[3] != "192.168.1.3" {
					t.Errorf("Expected last IP 192.168.1.3, got %s", lines[3])
				}
			},
		},
		{
			name: "IPv4 /30 one-line output",
			args: []string{"expand", "192.168.1.0/30", "--one-line"},
			checkFunc: func(t *testing.T, output string) {
				expected := "192.168.1.0, 192.168.1.1, 192.168.1.2, 192.168.1.3"
				if strings.TrimSpace(output) != expected {
					t.Errorf("Expected %s, got %s", expected, strings.TrimSpace(output))
				}
			},
		},
		{
			name: "IPv4 /32 single address",
			args: []string{"expand", "192.168.1.1/32"},
			checkFunc: func(t *testing.T, output string) {
				if strings.TrimSpace(output) != "192.168.1.1" {
					t.Errorf("Expected 192.168.1.1, got %s", strings.TrimSpace(output))
				}
			},
		},
		{
			name:      "IPv4 /29 with limit 5",
			args:      []string{"expand", "10.0.0.0/29", "--limit", "5"},
			expectErr: true, // Should exceed limit
		},
		{
			name:      "IPv4 /15 too large",
			args:      []string{"expand", "10.0.0.0/15"},
			expectErr: true,
		},
		{
			name:      "Invalid CIDR",
			args:      []string{"expand", "invalid"},
			expectErr: true,
		},
		{
			name:      "No arguments",
			args:      []string{"expand"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			config.Expand.Limit = 0
			config.Expand.OneLine = false

			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create command
			cmd := &cobra.Command{
				Use:  "expand <CIDR>",
				Args: cobra.ExactArgs(1),
				RunE: expandCmd.RunE,
			}
			cmd.Flags().IntVarP(&config.Expand.Limit, "limit", "l", 0, "Maximum number of IPs")
			cmd.Flags().BoolVarP(&config.Expand.OneLine, "one-line", "o", false, "One line output")

			// Execute
			cmd.SetArgs(tt.args[1:])
			err := cmd.Execute()

			// Capture output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check results
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

func TestContainsCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expected  string
		expectErr bool
	}{
		{
			name:     "IPv4 contained",
			args:     []string{"contains", "192.168.1.0/24", "192.168.1.100"},
			expected: "true",
		},
		{
			name:     "IPv4 not contained",
			args:     []string{"contains", "192.168.1.0/24", "192.168.2.100"},
			expected: "false",
		},
		{
			name:     "IPv6 contained",
			args:     []string{"contains", "2001:db8::/32", "2001:db8:1::1"},
			expected: "true",
		},
		{
			name:     "IPv6 not contained",
			args:     []string{"contains", "2001:db8::/32", "2001:db9::1"},
			expected: "false",
		},
		{
			name:      "Invalid CIDR",
			args:      []string{"contains", "invalid", "192.168.1.1"},
			expectErr: true,
		},
		{
			name:      "Invalid IP",
			args:      []string{"contains", "192.168.1.0/24", "invalid"},
			expectErr: true,
		},
		{
			name:      "Not enough arguments",
			args:      []string{"contains", "192.168.1.0/24"},
			expectErr: true,
		},
		{
			name:      "Too many arguments",
			args:      []string{"contains", "192.168.1.0/24", "192.168.1.1", "extra"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestCommand("contains <CIDR> <IP>", 2, containsCmd.RunE)
			output, err := captureCommandOutput(t, cmd, tt.args[1:])
			assertTestResult(t, err, output, tt.expectErr, tt.expected)
		})
	}
}

func TestCountCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expected  string
		expectErr bool
	}{
		{
			name:     "IPv4 /24",
			args:     []string{"count", "192.168.1.0/24"},
			expected: "256",
		},
		{
			name:     "IPv4 /16",
			args:     []string{"count", "10.0.0.0/16"},
			expected: "65536",
		},
		{
			name:     "IPv4 /32",
			args:     []string{"count", "192.168.1.1/32"},
			expected: "1",
		},
		{
			name:     "IPv6 /127",
			args:     []string{"count", "2001:db8::/127"},
			expected: "2",
		},
		{
			name:      "Invalid CIDR",
			args:      []string{"count", "invalid"},
			expectErr: true,
		},
		{
			name:      "No arguments",
			args:      []string{"count"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestCommand("count <CIDR>", 1, countCmd.RunE)
			output, err := captureCommandOutput(t, cmd, tt.args[1:])
			assertTestResult(t, err, output, tt.expectErr, tt.expected)
		})
	}
}

func TestOverlapsCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expected  string
		expectErr bool
	}{
		{
			name:     "IPv4 overlapping",
			args:     []string{"overlaps", "192.168.1.0/24", "192.168.1.128/25"},
			expected: "true",
		},
		{
			name:     "IPv4 non-overlapping",
			args:     []string{"overlaps", "192.168.1.0/24", "192.168.2.0/24"},
			expected: "false",
		},
		{
			name:     "IPv6 overlapping",
			args:     []string{"overlaps", "2001:db8::/32", "2001:db8:1::/48"},
			expected: "true",
		},
		{
			name:     "IPv6 non-overlapping",
			args:     []string{"overlaps", "2001:db8::/32", "2001:db9::/32"},
			expected: "false",
		},
		{
			name:      "Invalid first CIDR",
			args:      []string{"overlaps", "invalid", "192.168.1.0/24"},
			expectErr: true,
		},
		{
			name:      "Invalid second CIDR",
			args:      []string{"overlaps", "192.168.1.0/24", "invalid"},
			expectErr: true,
		},
		{
			name:      "Not enough arguments",
			args:      []string{"overlaps", "192.168.1.0/24"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestCommand("overlaps <CIDR1> <CIDR2>", 2, overlapsCmd.RunE)
			output, err := captureCommandOutput(t, cmd, tt.args[1:])
			assertTestResult(t, err, output, tt.expectErr, tt.expected)
		})
	}
}

func TestDivideCommand(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		checkFunc func(t *testing.T, output string)
		expectErr bool
	}{
		{
			name: "IPv4 /24 into 4 parts",
			args: []string{"divide", "192.168.1.0/24", "4"},
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 4 {
					t.Errorf("Expected 4 subnets, got %d", len(lines))
				}
				if lines[0] != "192.168.1.0/26" {
					t.Errorf("Expected first subnet 192.168.1.0/26, got %s", lines[0])
				}
			},
		},
		{
			name: "IPv6 /64 into 4 parts",
			args: []string{"divide", "2001:db8::/64", "4"},
			checkFunc: func(t *testing.T, output string) {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				if len(lines) != 4 {
					t.Errorf("Expected 4 subnets, got %d", len(lines))
				}
				if lines[0] != "2001:db8::/66" {
					t.Errorf("Expected first subnet 2001:db8::/66, got %s", lines[0])
				}
			},
		},
		{
			name:      "Zero parts",
			args:      []string{"divide", "192.168.1.0/24", "0"},
			expectErr: true,
		},
		{
			name:      "Negative parts",
			args:      []string{"divide", "192.168.1.0/24", "-1"},
			expectErr: true,
		},
		{
			name:      "Invalid number",
			args:      []string{"divide", "192.168.1.0/24", "invalid"},
			expectErr: true,
		},
		{
			name:      "Invalid CIDR",
			args:      []string{"divide", "invalid", "4"},
			expectErr: true,
		},
		{
			name:      "Not enough arguments",
			args:      []string{"divide", "192.168.1.0/24"},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Create command
			cmd := &cobra.Command{
				Use:  "divide <CIDR> <N>",
				Args: cobra.ExactArgs(2),
				RunE: divideCmd.RunE,
			}

			// Execute
			cmd.SetArgs(tt.args[1:])
			err := cmd.Execute()

			// Capture output
			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Check results
			if tt.expectErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, output)
			}
		})
	}
}

func TestCidrCmdStructure(t *testing.T) {
	// Test that the CIDR command is properly configured
	if CidrCmd.Use != "cidr" {
		t.Errorf("Expected Use 'cidr', got %s", CidrCmd.Use)
	}

	if CidrCmd.Short == "" {
		t.Error("CIDR command should have a short description")
	}

	if CidrCmd.Long == "" {
		t.Error("CIDR command should have a long description")
	}

	// Test that all subcommands are registered
	subcommands := []string{"explain", "expand", "contains", "count", "overlaps", "divide"}
	for _, subcmd := range subcommands {
		found := false
		for _, cmd := range CidrCmd.Commands() {
			if cmd.Use == subcmd+" <CIDR>" ||
				cmd.Use == subcmd+" <CIDR> <IP>" ||
				cmd.Use == subcmd+" <CIDR1> <CIDR2>" ||
				cmd.Use == subcmd+" <CIDR> <N>" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Subcommand %s not found in CIDR command", subcmd)
		}
	}
}
