package mtu

import (
	"context"
	"fmt"
	"math/bits"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type discoveryOptions struct {
	Destination      string
	IPv6             bool
	Protocol         string
	MinMTU           int
	MaxMTU           int
	Step             int
	Timeout          time.Duration
	TTL              int
	Quiet            bool
	PacketsPerSecond int
	HopsMode         bool
	MaxHops          int
	Port             int
	PLPMTUD          bool
	PLPPort          int
}

func readDiscoveryOptions(cmd *cobra.Command, destination string) (discoveryOptions, error) {
	forceIPv4, _ := cmd.Flags().GetBool("4")
	forceIPv6, _ := cmd.Flags().GetBool("6")
	if forceIPv4 && forceIPv6 {
		return discoveryOptions{}, fmt.Errorf("--4 and --6 are mutually exclusive")
	}

	protocol, _ := cmd.Flags().GetString("proto")
	if !isSupportedProbeProtocol(protocol) {
		return discoveryOptions{}, fmt.Errorf("unsupported protocol: %s", protocol)
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	ipv6 := forceIPv6
	minMTU, _ := cmd.Flags().GetInt("min")
	if minMTU == 0 {
		minMTU = defaultMinMTU(ipv6)
	}

	maxMTU, _ := cmd.Flags().GetInt("max")
	step, _ := cmd.Flags().GetInt("step")
	ttl, _ := cmd.Flags().GetInt("ttl")
	quiet, _ := cmd.Flags().GetBool("quiet")
	pps, _ := cmd.Flags().GetInt("pps")
	hopsMode, _ := cmd.Flags().GetBool("hops")
	maxHops, _ := cmd.Flags().GetInt("max-hops")
	port, _ := cmd.Flags().GetInt("port")
	plpmtud, _ := cmd.Flags().GetBool("plpmtud")
	plpPort, _ := cmd.Flags().GetInt("plp-port")

	opts := discoveryOptions{
		Destination:      destination,
		IPv6:             ipv6,
		Protocol:         protocol,
		MinMTU:           minMTU,
		MaxMTU:           maxMTU,
		Step:             step,
		Timeout:          timeout,
		TTL:              ttl,
		Quiet:            quiet,
		PacketsPerSecond: pps,
		HopsMode:         hopsMode,
		MaxHops:          maxHops,
		Port:             port,
		PLPMTUD:          plpmtud,
		PLPPort:          plpPort,
	}

	if opts.MinMTU > opts.MaxMTU {
		return discoveryOptions{}, fmt.Errorf("minimum MTU %d exceeds maximum %d", opts.MinMTU, opts.MaxMTU)
	}
	if opts.Step < 0 {
		return discoveryOptions{}, fmt.Errorf("--step must be non-negative")
	}
	if opts.TTL <= 0 {
		return discoveryOptions{}, fmt.Errorf("--ttl must be positive")
	}
	if opts.PacketsPerSecond < 0 {
		return discoveryOptions{}, fmt.Errorf("--pps must be non-negative")
	}
	if opts.HopsMode && opts.MaxHops <= 0 {
		return discoveryOptions{}, fmt.Errorf("--max-hops must be positive")
	}
	if opts.Port < 0 {
		return discoveryOptions{}, fmt.Errorf("--port must be non-negative")
	}
	if opts.PLPPort < 0 {
		return discoveryOptions{}, fmt.Errorf("--plp-port must be non-negative")
	}

	return opts, nil
}

func defaultMinMTU(ipv6 bool) int {
	if ipv6 {
		return 1280
	}
	return 576
}

func isSupportedProbeProtocol(protocol string) bool {
	switch protocol {
	case "icmp", "tcp", "udp":
		return true
	default:
		return false
	}
}

func newMTUDiscoverer(opts discoveryOptions) (*MTUDiscoverer, error) {
	discoverer, err := NewMTUDiscoverer(
		opts.Destination,
		opts.IPv6,
		opts.Protocol,
		opts.Port,
		opts.Timeout,
		opts.TTL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer: %w", err)
	}

	discoverer.security.RateLimiter = NewRateLimiter(opts.PacketsPerSecond)
	return discoverer, nil
}

func performMTUDiscovery(ctx context.Context, opts discoveryOptions) (*MTUResult, error) {
	discoverer, err := newMTUDiscoverer(opts)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := discoverer.Close(); closeErr != nil && !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Warning: failed to close discoverer: %v\n", closeErr)
		}
	}()

	if opts.Protocol == "icmp" {
		icmpListener, icmpErr := NewICMPListener()
		if icmpErr == nil {
			discoverer.SetICMPListener(icmpListener)
			icmpListener.Start(ctx)
			defer func() {
				if closeErr := icmpListener.Close(); closeErr != nil && !opts.Quiet {
					fmt.Fprintf(os.Stderr, "Warning: failed to close ICMP listener: %v\n", closeErr)
				}
			}()
		}
	}

	switch {
	case opts.Step > 0:
		return discoverer.DiscoverPMTULinear(ctx, opts.MinMTU, opts.MaxMTU, opts.Step)
	case opts.PLPMTUD:
		return discoverer.WithPLPMTUDFallback(ctx, opts.MinMTU, opts.MaxMTU, opts.PLPPort)
	default:
		return discoverer.DiscoverPMTU(ctx, opts.MinMTU, opts.MaxMTU)
	}
}

func newDiscoveryContext(opts discoveryOptions) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), discoveryTimeoutBudget(opts))
}

func discoveryTimeoutBudget(opts discoveryOptions) time.Duration {
	estimated := time.Duration(estimatedDiscoveryProbes(opts))*opts.Timeout + 5*time.Second
	if estimated < 60*time.Second {
		return 60 * time.Second
	}
	return estimated
}

func estimatedDiscoveryProbes(opts discoveryOptions) int {
	if opts.Step > 0 {
		probes := ((opts.MaxMTU - opts.MinMTU) / opts.Step) + 1
		if probes < 1 {
			return 1
		}
		return probes
	}

	probes := bits.Len(uint(opts.MaxMTU - opts.MinMTU + 1))
	if probes < 1 {
		probes = 1
	}

	if opts.PLPMTUD {
		plpProbes := (((opts.MaxMTU - opts.MinMTU) / 64) + 1) * 3
		if plpProbes < 3 {
			plpProbes = 3
		}
		probes += plpProbes
	}

	return probes
}
