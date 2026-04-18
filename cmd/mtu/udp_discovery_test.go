package mtu

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"
)

func startLocalUDPPeer(t *testing.T, maxPacketSize int) (*net.UDPConn, func()) {
	t.Helper()

	conn, err := openPeerUDPListener("127.0.0.1", 0)
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("sandbox does not allow local UDP listeners")
		}
		t.Fatalf("openPeerUDPListener returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- runUDPServer(ctx, conn, false, maxPacketSize, NewRateLimiter(0))
	}()

	shutdown := func() {
		cancel()
		_ = conn.Close()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("runUDPServer returned error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for UDP peer shutdown")
		}
	}

	return conn, shutdown
}

func TestUDPProberAndDiscoveryAgainstPeer(t *testing.T) {
	maxPMTU := 1400
	conn, shutdown := startLocalUDPPeer(t, payloadSizeForPacket(maxPMTU, udpPacketOverhead(false)))
	defer shutdown()

	port := conn.LocalAddr().(*net.UDPAddr).Port
	prober, err := NewUDPProber("127.0.0.1", false, port, 150*time.Millisecond)
	if err != nil {
		t.Fatalf("NewUDPProber returned error: %v", err)
	}

	success := prober.ProbeUDP(context.Background(), maxPMTU)
	if !success.Success {
		t.Fatalf("expected UDP probe at PMTU %d to succeed, got %+v", maxPMTU, success)
	}

	failure := prober.ProbeUDP(context.Background(), maxPMTU+20)
	if failure.Success {
		t.Fatalf("expected oversized UDP probe to fail, got %+v", failure)
	}

	result, err := prober.DiscoverPMTUUDP(context.Background(), 1300, 1450)
	if err != nil {
		t.Fatalf("DiscoverPMTUUDP returned error: %v", err)
	}
	if result.Protocol != "udp" {
		t.Fatalf("unexpected protocol: %q", result.Protocol)
	}
	if result.PMTU != maxPMTU {
		t.Fatalf("unexpected PMTU: got %d, want %d", result.PMTU, maxPMTU)
	}
	if result.MSS != tcpMSSForMTU(maxPMTU, false) {
		t.Fatalf("unexpected MSS: got %d, want %d", result.MSS, tcpMSSForMTU(maxPMTU, false))
	}

	discoverer := &MTUDiscoverer{
		target:   "127.0.0.1",
		ipv6:     false,
		protocol: "udp",
		port:     port,
		timeout:  150 * time.Millisecond,
	}

	discovered, err := discoverer.DiscoverPMTU(context.Background(), 1300, 1450)
	if err != nil {
		t.Fatalf("DiscoverPMTU returned error: %v", err)
	}
	if discovered.Protocol != "udp" || discovered.PMTU != maxPMTU {
		t.Fatalf("unexpected discoverer result: %+v", discovered)
	}
}

func TestDiscoveryCommandFlowsAgainstUDPPeer(t *testing.T) {
	maxPMTU := 1400
	conn, shutdown := startLocalUDPPeer(t, payloadSizeForPacket(maxPMTU, udpPacketOverhead(false)))
	defer shutdown()

	port := conn.LocalAddr().(*net.UDPAddr).Port

	discoverCmd := newDiscoveryOptionsCommand()
	mustSetFlag(t, discoverCmd, "proto", "udp")
	mustSetFlag(t, discoverCmd, "port", strconv.Itoa(port))
	mustSetFlag(t, discoverCmd, "min", "1300")
	mustSetFlag(t, discoverCmd, "max", "1450")
	mustSetFlag(t, discoverCmd, "timeout", "150ms")
	mustSetFlag(t, discoverCmd, "json", "true")
	mustSetFlag(t, discoverCmd, "quiet", "true")

	discoverOutput, err := captureStdout(t, func() error {
		return runDiscover(discoverCmd, []string{"127.0.0.1"})
	})
	if err != nil {
		t.Fatalf("runDiscover returned error: %v", err)
	}

	var discovered struct {
		Target   string `json:"target"`
		Protocol string `json:"protocol"`
		PMTU     int    `json:"pmtu"`
		MSS      int    `json:"mss"`
	}
	if err := json.Unmarshal([]byte(discoverOutput), &discovered); err != nil {
		t.Fatalf("runDiscover produced invalid JSON: %v", err)
	}
	if discovered.Target != "127.0.0.1" || discovered.Protocol != "udp" || discovered.PMTU != maxPMTU {
		t.Fatalf("unexpected discover result: %+v", discovered)
	}

	suggestCmd := newDiscoveryOptionsCommand()
	mustSetFlag(t, suggestCmd, "proto", "udp")
	mustSetFlag(t, suggestCmd, "port", strconv.Itoa(port))
	mustSetFlag(t, suggestCmd, "min", "1300")
	mustSetFlag(t, suggestCmd, "max", "1450")
	mustSetFlag(t, suggestCmd, "timeout", "150ms")
	mustSetFlag(t, suggestCmd, "json", "true")

	suggestOutput, err := captureStdout(t, func() error {
		return runSuggest(suggestCmd, []string{"127.0.0.1"})
	})
	if err != nil {
		t.Fatalf("runSuggest returned error: %v", err)
	}

	var suggested struct {
		Target      string      `json:"target"`
		PMTU        int         `json:"pmtu"`
		Suggestions Suggestions `json:"suggestions"`
	}
	if err := json.Unmarshal([]byte(suggestOutput), &suggested); err != nil {
		t.Fatalf("runSuggest produced invalid JSON: %v", err)
	}
	if suggested.Target != "127.0.0.1" || suggested.PMTU != maxPMTU {
		t.Fatalf("unexpected suggest result: %+v", suggested)
	}
	if suggested.Suggestions.WireGuardPayload != maxPMTU-60 {
		t.Fatalf("unexpected WireGuard payload suggestion: %d", suggested.Suggestions.WireGuardPayload)
	}
}

func TestPerformMTUDiscoveryLinearAndFallback(t *testing.T) {
	maxPMTU := 1400
	conn, shutdown := startLocalUDPPeer(t, payloadSizeForPacket(maxPMTU, udpPacketOverhead(false)))
	defer shutdown()

	port := conn.LocalAddr().(*net.UDPAddr).Port

	linearResult, err := performMTUDiscovery(context.Background(), discoveryOptions{
		Destination: "127.0.0.1",
		Protocol:    "udp",
		MinMTU:      1300,
		MaxMTU:      1450,
		Step:        20,
		Timeout:     150 * time.Millisecond,
		TTL:         64,
		Port:        port,
	})
	if err != nil {
		t.Fatalf("performMTUDiscovery linear returned error: %v", err)
	}
	if linearResult.PMTU != maxPMTU {
		t.Fatalf("unexpected linear PMTU: got %d, want %d", linearResult.PMTU, maxPMTU)
	}

	fallbackDiscoverer := &MTUDiscoverer{
		target:   "127.0.0.1",
		ipv6:     false,
		protocol: "bogus",
		timeout:  150 * time.Millisecond,
	}

	fallbackResult, err := fallbackDiscoverer.WithPLPMTUDFallback(context.Background(), 1300, 1450, port)
	if err != nil {
		t.Fatalf("WithPLPMTUDFallback returned error: %v", err)
	}
	if fallbackResult.Protocol != "plpmtud" || fallbackResult.PMTU != maxPMTU {
		t.Fatalf("unexpected PLPMTUD fallback result: %+v", fallbackResult)
	}

	plpResult, err := performMTUDiscovery(context.Background(), discoveryOptions{
		Destination: "127.0.0.1",
		Protocol:    "bogus",
		MinMTU:      1300,
		MaxMTU:      1450,
		Timeout:     150 * time.Millisecond,
		TTL:         64,
		PLPMTUD:     true,
		PLPPort:     port,
	})
	if err != nil {
		t.Fatalf("performMTUDiscovery PLPMTUD returned error: %v", err)
	}
	if plpResult.Protocol != "plpmtud" || plpResult.PMTU != maxPMTU {
		t.Fatalf("unexpected PLPMTUD discovery result: %+v", plpResult)
	}
}
