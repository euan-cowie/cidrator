package mtu

import (
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestCreateICMPPacket(t *testing.T) {
	t.Run("ipv4 packet size and type", func(t *testing.T) {
		discoverer := &MTUDiscoverer{
			ipv6:     false,
			security: NewSecurityConfig(0),
		}
		discoverer.security.Randomizer.useRandomData = false
		discoverer.security.Randomizer.useRandomID = false
		discoverer.security.Randomizer.useRandomSeq = false

		packet, err := discoverer.createICMPPacket(1500)
		if err != nil {
			t.Fatalf("createICMPPacket returned error: %v", err)
		}
		if len(packet) != 1480 {
			t.Fatalf("unexpected IPv4 ICMP packet length: got %d, want 1480", len(packet))
		}

		msg, err := icmp.ParseMessage(1, packet)
		if err != nil {
			t.Fatalf("failed to parse generated IPv4 ICMP packet: %v", err)
		}
		if msg.Type != ipv4.ICMPTypeEcho {
			t.Fatalf("unexpected IPv4 ICMP type: %v", msg.Type)
		}
	})

	t.Run("ipv6 packet size and type", func(t *testing.T) {
		discoverer := &MTUDiscoverer{
			ipv6:     true,
			security: NewSecurityConfig(0),
		}
		discoverer.security.Randomizer.useRandomData = false
		discoverer.security.Randomizer.useRandomID = false
		discoverer.security.Randomizer.useRandomSeq = false

		packet, err := discoverer.createICMPPacket(1500)
		if err != nil {
			t.Fatalf("createICMPPacket returned error: %v", err)
		}
		if len(packet) != 1460 {
			t.Fatalf("unexpected IPv6 ICMP packet length: got %d, want 1460", len(packet))
		}

		msg, err := icmp.ParseMessage(58, packet)
		if err != nil {
			t.Fatalf("failed to parse generated IPv6 ICMP packet: %v", err)
		}
		if msg.Type != ipv6.ICMPTypeEchoRequest {
			t.Fatalf("unexpected IPv6 ICMP type: %v", msg.Type)
		}
	})
}

func TestParseICMPResponses(t *testing.T) {
	t.Run("ipv4 echo reply succeeds", func(t *testing.T) {
		discoverer := &MTUDiscoverer{ipv6: false}
		data := mustMarshalICMP(t, &icmp.Message{
			Type: ipv4.ICMPTypeEchoReply,
			Code: 0,
			Body: &icmp.Echo{ID: 1, Seq: 1},
		})

		if err := discoverer.parseICMPResponse(data, &net.IPAddr{IP: net.ParseIP("127.0.0.1")}); err != nil {
			t.Fatalf("expected echo reply to parse as success, got %+v", err)
		}
	})

	t.Run("ipv4 fragmentation needed parses without mtu extraction helper", func(t *testing.T) {
		discoverer := &MTUDiscoverer{ipv6: false}
		data := mustMarshalICMP(t, &icmp.Message{
			Type: ipv4.ICMPTypeDestinationUnreachable,
			Code: 4,
			Body: &icmp.DstUnreach{Data: []byte{0x00, 0x00, 0x05, 0xdc}},
		})

		err := discoverer.parseICMPResponse(data, &net.IPAddr{IP: net.ParseIP("192.0.2.1")})
		if err == nil {
			t.Fatal("expected destination-unreachable error")
		}
		if err.Message != "Fragmentation Needed and Don't Fragment was Set" {
			t.Fatalf("unexpected IPv4 parse error: %+v", err)
		}
	})

	t.Run("ipv4 fragmentation needed exposes mtu", func(t *testing.T) {
		discoverer := &MTUDiscoverer{ipv6: false}
		data := mustMarshalICMP(t, &icmp.Message{
			Type: ipv4.ICMPTypeDestinationUnreachable,
			Code: 4,
			Body: &icmp.DstUnreach{
				Data: []byte{0x00, 0x00, 0x05, 0xdc},
			},
		})

		err := discoverer.parseICMPResponseWithMTU(data, &net.IPAddr{IP: net.ParseIP("192.0.2.1")})
		if err == nil {
			t.Fatal("expected fragmentation-needed error")
		}
		if err.Type != int(ipv4.ICMPTypeDestinationUnreachable) || err.Code != 4 || err.MTU != 1500 {
			t.Fatalf("unexpected IPv4 fragmentation error: %+v", err)
		}
	})

	t.Run("ipv6 packet too big exposes mtu", func(t *testing.T) {
		discoverer := &MTUDiscoverer{ipv6: true}
		data := mustMarshalICMP(t, &icmp.Message{
			Type: ipv6.ICMPTypePacketTooBig,
			Code: 0,
			Body: &icmp.PacketTooBig{MTU: 1280},
		})

		err := discoverer.parseICMPResponseWithMTU(data, &net.IPAddr{IP: net.ParseIP("2001:db8::1")})
		if err == nil {
			t.Fatal("expected packet-too-big error")
		}
		if err.Type != int(ipv6.ICMPTypePacketTooBig) || err.MTU != 1280 {
			t.Fatalf("unexpected IPv6 packet-too-big error: %+v", err)
		}
	})

	t.Run("ipv6 packet too big parses in basic path", func(t *testing.T) {
		discoverer := &MTUDiscoverer{ipv6: true}
		data := mustMarshalICMP(t, &icmp.Message{
			Type: ipv6.ICMPTypePacketTooBig,
			Code: 0,
			Body: &icmp.PacketTooBig{MTU: 1280},
		})

		err := discoverer.parseICMPResponse(data, &net.IPAddr{IP: net.ParseIP("2001:db8::1")})
		if err == nil {
			t.Fatal("expected packet-too-big error")
		}
		if err.Message != "Packet Too Big" {
			t.Fatalf("unexpected IPv6 parse error: %+v", err)
		}
	})

	t.Run("time exceeded parses correctly", func(t *testing.T) {
		discoverer := &MTUDiscoverer{ipv6: false}
		data := mustMarshalICMP(t, &icmp.Message{
			Type: ipv4.ICMPTypeTimeExceeded,
			Code: 0,
			Body: &icmp.TimeExceeded{},
		})

		err := discoverer.parseICMPResponseWithMTU(data, &net.IPAddr{IP: net.ParseIP("198.51.100.1")})
		if err == nil {
			t.Fatal("expected time-exceeded error")
		}
		if err.Type != int(ipv4.ICMPTypeTimeExceeded) || err.Message != "Time Exceeded" {
			t.Fatalf("unexpected time-exceeded error: %+v", err)
		}
	})

	t.Run("invalid data returns parse error", func(t *testing.T) {
		discoverer := &MTUDiscoverer{ipv6: false}
		err := discoverer.parseICMPResponseWithMTU([]byte{0x01, 0x02}, nil)
		if err == nil {
			t.Fatal("expected parse failure")
		}
		if err.Type != -1 || err.Code != -1 {
			t.Fatalf("unexpected parse failure payload: %+v", err)
		}
	})
}

func TestDestinationDetectionHelpers(t *testing.T) {
	discoverer := &MTUDiscoverer{
		targetAddr: &net.IPAddr{IP: net.ParseIP("192.0.2.10")},
	}

	if discoverer.isDestinationReached(&HopInfo{}) {
		t.Fatal("expected nil hop address to be treated as not reached")
	}
	if !discoverer.isDestinationReached(&HopInfo{Addr: net.ParseIP("192.0.2.10")}) {
		t.Fatal("expected matching hop address to be treated as destination reached")
	}
	if discoverer.isDestinationReached(&HopInfo{Addr: net.ParseIP("192.0.2.20")}) {
		t.Fatal("expected mismatched hop address to be treated as not reached")
	}
}

func mustMarshalICMP(t *testing.T, msg *icmp.Message) []byte {
	t.Helper()

	data, err := msg.Marshal(nil)
	if err != nil {
		t.Fatalf("failed to marshal ICMP message: %v", err)
	}
	return data
}
