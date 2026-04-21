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
	target   string
	ipv6     bool
	options  PLPMTUDOptions
	probeUDP func(ctx context.Context, size int) bool
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

	stepSize := p.options.StepSize
	if stepSize <= 0 {
		stepSize = 64
	}

	confirmedMTU := 0
	firstFailedMTU := maxMTU + 1

	// Coarse sweep to find the first failing region.
	for size := minMTU; size <= maxMTU; size += stepSize {
		success, err := p.confirmPacketSize(ctx, size)
		if err != nil {
			return nil, err
		}
		if success {
			confirmedMTU = size
		} else {
			firstFailedMTU = size
			break
		}
		if err := waitForNextPLPProbe(ctx); err != nil {
			return nil, err
		}
	}

	if confirmedMTU == 0 {
		return nil, fmt.Errorf("no working PLPMTUD size found in range %d-%d", minMTU, maxMTU)
	}

	refineUpperBound := maxMTU
	if firstFailedMTU <= maxMTU {
		refineUpperBound = firstFailedMTU - 1
	}

	// Refine inside the last successful coarse bucket so PLPMTUD returns an exact PMTU.
	low := confirmedMTU + 1
	high := refineUpperBound
	for low <= high {
		mid := low + (high-low)/2
		success, err := p.confirmPacketSize(ctx, mid)
		if err != nil {
			return nil, err
		}
		if success {
			confirmedMTU = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	elapsed := time.Since(start)

	return &MTUResult{
		Target:    p.target,
		Protocol:  "plpmtud",
		PMTU:      confirmedMTU,
		MSS:       tcpMSSForMTU(confirmedMTU, p.ipv6),
		Hops:      0, // Not applicable for PLPMTUD
		ElapsedMS: int(elapsed.Milliseconds()),
	}, nil
}

func (p *PLPMTUDProber) confirmPacketSize(ctx context.Context, size int) (bool, error) {
	maxProbes := p.options.MaxProbes
	if maxProbes <= 0 {
		maxProbes = 1
	}

	successCount := 0
	for attempt := 0; attempt < maxProbes; attempt++ {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
		}

		if p.testPacketSize(ctx, size) {
			successCount++
		}
	}

	return successCount > maxProbes/2, nil
}

func waitForNextPLPProbe(ctx context.Context) error {
	timer := time.NewTimer(100 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// testPacketSize tests if a packet of given size can be sent successfully
func (p *PLPMTUDProber) testPacketSize(ctx context.Context, size int) bool {
	if p.probeUDP != nil {
		return p.probeUDP(ctx, size)
	}

	prober, err := NewUDPProber(p.target, p.ipv6, p.options.PLPPort, p.options.BaseTimeout)
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
