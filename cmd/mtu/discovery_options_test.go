package mtu

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func newDiscoveryOptionsCommand() *cobra.Command {
	cmd := &cobra.Command{Use: "discover"}
	flags := cmd.Flags()
	flags.Bool("4", false, "")
	flags.Bool("6", false, "")
	flags.Bool("json", false, "")
	flags.String("proto", "icmp", "")
	flags.Int("min", 0, "")
	flags.Int("max", 9216, "")
	flags.Int("step", 0, "")
	flags.Duration("timeout", 0, "")
	flags.Int("ttl", 64, "")
	flags.Bool("quiet", false, "")
	flags.Int("pps", 10, "")
	flags.Bool("hops", false, "")
	flags.Int("max-hops", 30, "")
	flags.Int("port", 0, "")
	flags.Bool("plpmtud", false, "")
	flags.Int("plp-port", 443, "")
	return cmd
}

func mustSetFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if err := cmd.Flags().Set(name, value); err != nil {
		t.Fatalf("failed to set %s=%s: %v", name, value, err)
	}
}

func TestReadDiscoveryOptions(t *testing.T) {
	t.Run("applies defaults", func(t *testing.T) {
		cmd := newDiscoveryOptionsCommand()

		opts, err := readDiscoveryOptions(cmd, "example.com")
		if err != nil {
			t.Fatalf("readDiscoveryOptions returned error: %v", err)
		}

		if opts.Destination != "example.com" {
			t.Fatalf("unexpected destination: %q", opts.Destination)
		}
		if opts.Protocol != "icmp" {
			t.Fatalf("unexpected protocol: %q", opts.Protocol)
		}
		if opts.Timeout != 2*time.Second {
			t.Fatalf("unexpected timeout: %v", opts.Timeout)
		}
		if opts.MinMTU != 576 {
			t.Fatalf("unexpected default IPv4 minimum MTU: %d", opts.MinMTU)
		}
		if opts.MaxHops != 30 || opts.PacketsPerSecond != 10 || opts.PLPPort != 443 {
			t.Fatalf("unexpected default option set: %+v", opts)
		}
	})

	t.Run("uses ipv6 minimum by default", func(t *testing.T) {
		cmd := newDiscoveryOptionsCommand()
		mustSetFlag(t, cmd, "6", "true")

		opts, err := readDiscoveryOptions(cmd, "2001:db8::1")
		if err != nil {
			t.Fatalf("readDiscoveryOptions returned error: %v", err)
		}
		if !opts.IPv6 {
			t.Fatal("expected IPv6 mode")
		}
		if opts.MinMTU != 1280 {
			t.Fatalf("unexpected default IPv6 minimum MTU: %d", opts.MinMTU)
		}
	})

	tests := []struct {
		name    string
		flags   map[string]string
		wantErr string
	}{
		{
			name:    "mutually exclusive address families",
			flags:   map[string]string{"4": "true", "6": "true"},
			wantErr: "--4 and --6 are mutually exclusive",
		},
		{
			name:    "unsupported protocol",
			flags:   map[string]string{"proto": "sctp"},
			wantErr: "unsupported protocol: sctp",
		},
		{
			name:    "minimum exceeds maximum",
			flags:   map[string]string{"min": "1600", "max": "1500"},
			wantErr: "minimum MTU 1600 exceeds maximum 1500",
		},
		{
			name:    "negative step",
			flags:   map[string]string{"step": "-1"},
			wantErr: "--step must be non-negative",
		},
		{
			name:    "non-positive ttl",
			flags:   map[string]string{"ttl": "0"},
			wantErr: "--ttl must be positive",
		},
		{
			name:    "negative pps",
			flags:   map[string]string{"pps": "-1"},
			wantErr: "--pps must be non-negative",
		},
		{
			name:    "invalid max hops",
			flags:   map[string]string{"hops": "true", "max-hops": "0"},
			wantErr: "--max-hops must be positive",
		},
		{
			name:    "negative port",
			flags:   map[string]string{"port": "-1"},
			wantErr: "--port must be non-negative",
		},
		{
			name:    "negative plp port",
			flags:   map[string]string{"plp-port": "-1"},
			wantErr: "--plp-port must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newDiscoveryOptionsCommand()
			for name, value := range tt.flags {
				mustSetFlag(t, cmd, name, value)
			}

			_, err := readDiscoveryOptions(cmd, "example.com")
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestDiscoveryOptionHelpers(t *testing.T) {
	if defaultMinMTU(false) != 576 {
		t.Fatalf("unexpected IPv4 minimum MTU: %d", defaultMinMTU(false))
	}
	if defaultMinMTU(true) != 1280 {
		t.Fatalf("unexpected IPv6 minimum MTU: %d", defaultMinMTU(true))
	}
	if !isSupportedProbeProtocol("icmp") || !isSupportedProbeProtocol("tcp") || !isSupportedProbeProtocol("udp") {
		t.Fatal("expected supported protocols to be accepted")
	}
	if isSupportedProbeProtocol("quic") {
		t.Fatal("expected unsupported protocol to be rejected")
	}
}

func TestNewDiscoveryContextUsesBudget(t *testing.T) {
	opts := discoveryOptions{
		MinMTU:  576,
		MaxMTU:  9216,
		Timeout: 2 * time.Second,
	}

	ctx, cancel := newDiscoveryContext(opts)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("expected discovery context to have a deadline")
	}

	remaining := time.Until(deadline)
	if remaining < 58*time.Second || remaining > 61*time.Second {
		t.Fatalf("expected discovery context to use the 60s floor, got %v remaining", remaining)
	}
}

func TestEstimatedPLPMTUDPauseBudget(t *testing.T) {
	if budget := estimatedPLPMTUDPauseBudget(discoveryOptions{MinMTU: 1400, MaxMTU: 1400}); budget != 0 {
		t.Fatalf("expected no pause budget for a single PLPMTUD probe, got %v", budget)
	}

	budget := estimatedPLPMTUDPauseBudget(discoveryOptions{MinMTU: 576, MaxMTU: 704})
	if budget != 200*time.Millisecond {
		t.Fatalf("unexpected PLPMTUD pause budget: %v", budget)
	}
}

func TestNewMTUDiscovererUsesRateLimit(t *testing.T) {
	discoverer, err := newMTUDiscoverer(discoveryOptions{
		Destination:      "127.0.0.1",
		Protocol:         "tcp",
		Timeout:          time.Second,
		TTL:              64,
		PacketsPerSecond: 7,
	})
	if err != nil {
		t.Fatalf("newMTUDiscoverer returned error: %v", err)
	}
	defer func() { _ = discoverer.Close() }()

	if discoverer.protocol != "tcp" {
		t.Fatalf("unexpected protocol: %q", discoverer.protocol)
	}
	if discoverer.security.RateLimiter.packetsPerSecond != 7 {
		t.Fatalf("unexpected rate limit: %d", discoverer.security.RateLimiter.packetsPerSecond)
	}
}

func TestCommandEntryPointsRejectInvalidHopsModes(t *testing.T) {
	discoverCmd := newDiscoveryOptionsCommand()
	mustSetFlag(t, discoverCmd, "hops", "true")
	mustSetFlag(t, discoverCmd, "proto", "tcp")

	err := runDiscover(discoverCmd, []string{"example.com"})
	if err == nil {
		t.Fatal("expected discover command to reject hop mode for non-ICMP")
	}
	if !strings.Contains(err.Error(), "hop-by-hop discovery only supports ICMP protocol") {
		t.Fatalf("unexpected discover error: %v", err)
	}

	suggestCmd := newDiscoveryOptionsCommand()
	mustSetFlag(t, suggestCmd, "hops", "true")

	err = runSuggest(suggestCmd, []string{"example.com"})
	if err == nil {
		t.Fatal("expected suggest command to reject hop mode")
	}
	if !strings.Contains(err.Error(), "--hops is only supported by mtu discover") {
		t.Fatalf("unexpected suggest error: %v", err)
	}

	watchCmd := newDiscoveryOptionsCommand()
	mustSetFlag(t, watchCmd, "hops", "true")

	err = runWatch(watchCmd, []string{"example.com"})
	if err == nil {
		t.Fatal("expected watch command to reject hop mode")
	}
	if !strings.Contains(err.Error(), "--hops is only supported by mtu discover") {
		t.Fatalf("unexpected watch error: %v", err)
	}
}

func TestRunSuggestProbeDefaults(t *testing.T) {
	original := suggestMTUDiscovery
	t.Cleanup(func() { suggestMTUDiscovery = original })

	t.Run("defaults to tcp when proto flag is unchanged", func(t *testing.T) {
		var gotOpts discoveryOptions
		suggestMTUDiscovery = func(ctx context.Context, opts discoveryOptions) (*MTUResult, error) {
			gotOpts = opts
			return &MTUResult{Target: opts.Destination, Protocol: opts.Protocol, PMTU: 1500}, nil
		}

		cmd := newDiscoveryOptionsCommand()
		_, err := captureStdout(t, func() error {
			return runSuggest(cmd, []string{"example.com"})
		})
		if err != nil {
			t.Fatalf("runSuggest returned error: %v", err)
		}
		if gotOpts.Protocol != "tcp" {
			t.Fatalf("expected suggest default protocol to switch to tcp, got %q", gotOpts.Protocol)
		}
	})

	t.Run("explicit proto is preserved", func(t *testing.T) {
		var gotOpts discoveryOptions
		suggestMTUDiscovery = func(ctx context.Context, opts discoveryOptions) (*MTUResult, error) {
			gotOpts = opts
			return &MTUResult{Target: opts.Destination, Protocol: opts.Protocol, PMTU: 1500}, nil
		}

		cmd := newDiscoveryOptionsCommand()
		mustSetFlag(t, cmd, "proto", "icmp")
		_, err := captureStdout(t, func() error {
			return runSuggest(cmd, []string{"example.com"})
		})
		if err != nil {
			t.Fatalf("runSuggest returned error: %v", err)
		}
		if gotOpts.Protocol != "icmp" {
			t.Fatalf("expected explicit protocol to be preserved, got %q", gotOpts.Protocol)
		}
	})
}

func TestPerformMTUDiscoveryRejectsUnsupportedProtocol(t *testing.T) {
	_, err := performMTUDiscovery(context.Background(), discoveryOptions{
		Destination: "example.com",
		Protocol:    "bogus",
		MinMTU:      576,
		MaxMTU:      1500,
		Timeout:     time.Second,
		TTL:         64,
	})
	if err == nil {
		t.Fatal("expected unsupported protocol error")
	}
	if !strings.Contains(err.Error(), "unsupported protocol: bogus") {
		t.Fatalf("unexpected error: %v", err)
	}
}
