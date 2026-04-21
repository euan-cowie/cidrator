package mtu

import (
	"context"
	"testing"
	"time"
)

func TestPLPMTUDRefinesToExactPMTU(t *testing.T) {
	prober := NewPLPMTUDProber("example.com", false, PLPMTUDOptions{
		PLPPort:     4821,
		MaxProbes:   3,
		StepSize:    64,
		BaseTimeout: 50 * time.Millisecond,
	})
	prober.probeUDP = func(_ context.Context, size int) bool {
		return size <= 1400
	}

	result, err := prober.DiscoverPMTUWithPLPMTUD(context.Background(), 576, 1500)
	if err != nil {
		t.Fatalf("DiscoverPMTUWithPLPMTUD returned error: %v", err)
	}

	if result.Protocol != "plpmtud" {
		t.Fatalf("Protocol = %q, want plpmtud", result.Protocol)
	}
	if result.PMTU != 1400 {
		t.Fatalf("PMTU = %d, want 1400", result.PMTU)
	}
	if result.MSS != 1360 {
		t.Fatalf("MSS = %d, want 1360", result.MSS)
	}
}

func TestPLPMTUDReturnsErrorWhenNoSizeWorks(t *testing.T) {
	prober := NewPLPMTUDProber("example.com", false, PLPMTUDOptions{
		PLPPort:     4821,
		MaxProbes:   3,
		StepSize:    64,
		BaseTimeout: 50 * time.Millisecond,
	})
	prober.probeUDP = func(_ context.Context, size int) bool {
		return false
	}

	_, err := prober.DiscoverPMTUWithPLPMTUD(context.Background(), 576, 1500)
	if err == nil {
		t.Fatal("DiscoverPMTUWithPLPMTUD returned nil error, want failure")
	}
}
