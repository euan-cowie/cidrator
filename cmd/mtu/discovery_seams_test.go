package mtu

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestResolveTargetWithInjectedLookup(t *testing.T) {
	originalLookup := lookupIPAddrs
	t.Cleanup(func() {
		lookupIPAddrs = originalLookup
	})

	lookupIPAddrs = func(host string) ([]net.IP, error) {
		switch host {
		case "mixed.example":
			return []net.IP{
				net.ParseIP("2001:db8::10"),
				net.ParseIP("192.0.2.10"),
			}, nil
		case "ipv6-only.example":
			return []net.IP{net.ParseIP("2001:db8::20")}, nil
		case "lookup-error.example":
			return nil, errors.New("dns failed")
		default:
			return nil, errors.New("unexpected lookup")
		}
	}

	t.Run("selects matching IPv4 address", func(t *testing.T) {
		discoverer := &MTUDiscoverer{
			target:   "mixed.example",
			ipv6:     false,
			protocol: "icmp",
			timeout:  time.Second,
			ttl:      64,
			security: NewSecurityConfig(10),
		}
		if err := discoverer.resolveTarget(); err != nil {
			t.Fatalf("resolveTarget returned error: %v", err)
		}
		if got := discoverer.targetAddr.(*net.IPAddr).IP.String(); got != "192.0.2.10" {
			t.Fatalf("resolveTarget selected %s, want 192.0.2.10", got)
		}
	})

	t.Run("returns missing family error", func(t *testing.T) {
		discoverer := &MTUDiscoverer{
			target:   "ipv6-only.example",
			ipv6:     false,
			protocol: "icmp",
			timeout:  time.Second,
			ttl:      64,
			security: NewSecurityConfig(10),
		}
		err := discoverer.resolveTarget()
		if err == nil || !strings.Contains(err.Error(), "no IPv4 address found") {
			t.Fatalf("expected missing IPv4 error, got %v", err)
		}
	})

	t.Run("propagates lookup failures", func(t *testing.T) {
		discoverer := &MTUDiscoverer{
			target:   "lookup-error.example",
			ipv6:     false,
			protocol: "icmp",
			timeout:  time.Second,
			ttl:      64,
			security: NewSecurityConfig(10),
		}
		err := discoverer.resolveTarget()
		if err == nil || !strings.Contains(err.Error(), "dns failed") {
			t.Fatalf("expected lookup error, got %v", err)
		}
	})
}

func TestResolveTargetHelpersUseInjectedLookup(t *testing.T) {
	originalLookup := lookupIPAddrs
	t.Cleanup(func() {
		lookupIPAddrs = originalLookup
	})

	lookupIPAddrs = func(host string) ([]net.IP, error) {
		if host == "peer.example" {
			return []net.IP{net.ParseIP("203.0.113.10")}, nil
		}
		return nil, errors.New("lookup failed")
	}

	ips, err := resolveTargetIPs("peer.example")
	if err != nil {
		t.Fatalf("resolveTargetIPs returned error: %v", err)
	}
	if len(ips) != 1 || !ips[0].Equal(net.ParseIP("203.0.113.10")) {
		t.Fatalf("resolveTargetIPs returned %+v", ips)
	}

	listenIPs, err := resolvePeerListenIPs("peer.example")
	if err != nil {
		t.Fatalf("resolvePeerListenIPs returned error: %v", err)
	}
	if len(listenIPs) != 1 || !listenIPs[0].Equal(net.ParseIP("203.0.113.10")) {
		t.Fatalf("resolvePeerListenIPs returned %+v", listenIPs)
	}
}

func TestNewMTUDiscovererUsesInjectedPacketListener(t *testing.T) {
	originalLookup := lookupIPAddrs
	originalListen := listenDiscoverPacket
	t.Cleanup(func() {
		lookupIPAddrs = originalLookup
		listenDiscoverPacket = originalListen
	})

	lookupIPAddrs = func(host string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1")}, nil
	}

	t.Run("icmp setup returns packet listener error", func(t *testing.T) {
		listenDiscoverPacket = func(network, address string) (net.PacketConn, error) {
			return nil, errors.New("listen failed")
		}

		discoverer, err := NewMTUDiscoverer("listener.example", false, "icmp", 0, time.Second, 64)
		if err == nil || !strings.Contains(err.Error(), "failed to setup connection: listen failed") {
			t.Fatalf("expected listen failure, got discoverer=%+v err=%v", discoverer, err)
		}
	})

	t.Run("tcp discoverer bypasses raw socket setup", func(t *testing.T) {
		listenDiscoverPacket = func(network, address string) (net.PacketConn, error) {
			t.Fatal("listenDiscoverPacket should not be called for TCP discoverers")
			return nil, nil
		}

		discoverer, err := NewMTUDiscoverer("listener.example", false, "tcp", 0, time.Second, 64)
		if err != nil {
			t.Fatalf("unexpected TCP discoverer error: %v", err)
		}
		if discoverer.conn != nil {
			t.Fatalf("expected TCP discoverer to defer raw socket setup, got %+v", discoverer.conn)
		}
	})
}
