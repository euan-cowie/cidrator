package mtu

import (
	"context"
	"fmt"
	"time"
)

// PLPMTUDOptions contains options for PLPMTUD fallback
type PLPMTUDOptions struct {
	PLPPort     int
	MaxProbes   int
	StepSize    int
	BaseTimeout time.Duration
}

// PLPMTUDProber implements RFC 4821 style PLPMTUD
type PLPMTUDProber struct {
	target  string
	ipv6    bool
	options PLPMTUDOptions
}

// NewPLPMTUDProber creates a new PLPMTUD prober
func NewPLPMTUDProber(target string, ipv6 bool, options PLPMTUDOptions) *PLPMTUDProber {
	return &PLPMTUDProber{
		target:  target,
		ipv6:    ipv6,
		options: options,
	}
}

// DiscoverPMTUWithPLPMTUD performs PLPMTUD-style MTU discovery
// This is used as a fallback when ICMP is filtered/blocked
func (p *PLPMTUDProber) DiscoverPMTUWithPLPMTUD(ctx context.Context, minMTU, maxMTU int) (*MTUResult, error) {
	start := time.Now()

	// Start with a conservative estimate
	confirmedMTU := minMTU

	// Gradually increase packet size in-band
	for size := minMTU; size <= maxMTU; size += p.options.StepSize {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Test this size multiple times for reliability
		successCount := 0
		for attempt := 0; attempt < p.options.MaxProbes; attempt++ {
			if p.testPacketSize(ctx, size) {
				successCount++
			}
		}

		// Require majority success for confirmation
		if successCount > p.options.MaxProbes/2 {
			confirmedMTU = size
		} else {
			// Failed at this size, stop probing
			break
		}

		// Add some delay between probes to be network-friendly
		time.Sleep(time.Millisecond * 100)
	}

	elapsed := time.Since(start)

	// Calculate MSS
	mss := confirmedMTU - 40 // Default to IPv4
	if p.ipv6 {
		mss = confirmedMTU - 60
	}

	return &MTUResult{
		Target:    p.target,
		Protocol:  "plpmtud",
		PMTU:      confirmedMTU,
		MSS:       mss,
		Hops:      0, // Not applicable for PLPMTUD
		ElapsedMS: int(elapsed.Milliseconds()),
	}, nil
}

// testPacketSize tests if a packet of given size can be sent successfully
func (p *PLPMTUDProber) testPacketSize(ctx context.Context, size int) bool {
	// In a real implementation, this would send application-layer data
	// to a willing echo server on the specified PLP port
	// For now, we'll simulate using UDP probes

	prober, err := NewUDPProber(p.target, p.ipv6, 0, p.options.BaseTimeout)
	if err != nil {
		return false
	}

	result := prober.ProbeUDP(ctx, size)
	return result.Success
}

// WithPLPMTUDFallback modifies MTU discovery to use PLPMTUD when ICMP fails
func (d *MTUDiscoverer) WithPLPMTUDFallback(ctx context.Context, minMTU, maxMTU int, plpPort int) (*MTUResult, error) {
	// First try normal ICMP discovery
	result, err := d.DiscoverPMTU(ctx, minMTU, maxMTU)
	if err == nil {
		return result, nil
	}

	// If ICMP failed, fall back to PLPMTUD
	options := PLPMTUDOptions{
		PLPPort:     plpPort,
		MaxProbes:   3,
		StepSize:    64, // Conservative step size
		BaseTimeout: d.timeout,
	}

	plpProber := NewPLPMTUDProber(d.target, d.ipv6, options)

	// Try PLPMTUD fallback
	plpResult, plpErr := plpProber.DiscoverPMTUWithPLPMTUD(ctx, minMTU, maxMTU)
	if plpErr == nil {
		return plpResult, nil
	}

	// Both methods failed
	return nil, fmt.Errorf("both ICMP and PLPMTUD discovery failed: icmp_error=%v, plpmtud_error=%v", err, plpErr)
}
