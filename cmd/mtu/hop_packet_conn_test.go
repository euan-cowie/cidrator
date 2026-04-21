package mtu

import (
	"errors"
	"net"
	"testing"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

func TestHopPacketConnWrappers(t *testing.T) {
	t.Run("ipv4 prepare and read use injected helpers", func(t *testing.T) {
		originalSetTTL := setIPv4HopTTL
		originalSetControl := setIPv4HopControlMessage
		originalRead := readIPv4HopPacket
		t.Cleanup(func() {
			setIPv4HopTTL = originalSetTTL
			setIPv4HopControlMessage = originalSetControl
			readIPv4HopPacket = originalRead
		})

		var gotTTL int
		setIPv4HopTTL = func(conn *ipv4.PacketConn, ttl int) error {
			gotTTL = ttl
			return nil
		}
		setIPv4HopControlMessage = func(conn *ipv4.PacketConn, on bool) error {
			if !on {
				t.Fatal("expected IPv4 control message to be enabled")
			}
			return nil
		}
		readIPv4HopPacket = func(conn *ipv4.PacketConn, buf []byte) (int, net.Addr, error) {
			copy(buf, []byte("ok"))
			return 2, &net.IPAddr{IP: net.ParseIP("192.0.2.1")}, nil
		}

		conn := &ipv4HopPacketConn{}
		if err := conn.Prepare(7); err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if gotTTL != 7 {
			t.Fatalf("Prepare set TTL %d, want 7", gotTTL)
		}

		buf := make([]byte, 8)
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			t.Fatalf("ReadFrom returned error: %v", err)
		}
		if n != 2 || addr.String() != "192.0.2.1" || string(buf[:n]) != "ok" {
			t.Fatalf("unexpected IPv4 read result: n=%d addr=%v data=%q", n, addr, string(buf[:n]))
		}
	})

	t.Run("ipv4 prepare wraps helper failures", func(t *testing.T) {
		originalSetTTL := setIPv4HopTTL
		originalSetControl := setIPv4HopControlMessage
		t.Cleanup(func() {
			setIPv4HopTTL = originalSetTTL
			setIPv4HopControlMessage = originalSetControl
		})

		setIPv4HopTTL = func(conn *ipv4.PacketConn, ttl int) error {
			return errors.New("ttl failed")
		}
		conn := &ipv4HopPacketConn{}
		if err := conn.Prepare(5); err == nil || err.Error() != "failed to set TTL: ttl failed" {
			t.Fatalf("expected wrapped TTL error, got %v", err)
		}

		setIPv4HopTTL = func(conn *ipv4.PacketConn, ttl int) error {
			return nil
		}
		setIPv4HopControlMessage = func(conn *ipv4.PacketConn, on bool) error {
			return errors.New("control failed")
		}
		if err := conn.Prepare(5); err == nil || err.Error() != "failed to set control message: control failed" {
			t.Fatalf("expected wrapped control error, got %v", err)
		}
	})

	t.Run("ipv6 prepare and read use injected helpers", func(t *testing.T) {
		originalSetHop := setIPv6HopLimit
		originalSetControl := setIPv6HopControlMessage
		originalRead := readIPv6HopPacket
		t.Cleanup(func() {
			setIPv6HopLimit = originalSetHop
			setIPv6HopControlMessage = originalSetControl
			readIPv6HopPacket = originalRead
		})

		var gotTTL int
		setIPv6HopLimit = func(conn *ipv6.PacketConn, ttl int) error {
			gotTTL = ttl
			return nil
		}
		setIPv6HopControlMessage = func(conn *ipv6.PacketConn, on bool) error {
			if !on {
				t.Fatal("expected IPv6 control message to be enabled")
			}
			return nil
		}
		readIPv6HopPacket = func(conn *ipv6.PacketConn, buf []byte) (int, net.Addr, error) {
			copy(buf, []byte("v6"))
			return 2, &net.IPAddr{IP: net.ParseIP("2001:db8::1")}, nil
		}

		conn := &ipv6HopPacketConn{}
		if err := conn.Prepare(9); err != nil {
			t.Fatalf("Prepare returned error: %v", err)
		}
		if gotTTL != 9 {
			t.Fatalf("Prepare set hop limit %d, want 9", gotTTL)
		}

		buf := make([]byte, 8)
		n, addr, err := conn.ReadFrom(buf)
		if err != nil {
			t.Fatalf("ReadFrom returned error: %v", err)
		}
		if n != 2 || addr.String() != "2001:db8::1" || string(buf[:n]) != "v6" {
			t.Fatalf("unexpected IPv6 read result: n=%d addr=%v data=%q", n, addr, string(buf[:n]))
		}
	})

	t.Run("ipv6 prepare wraps helper failures", func(t *testing.T) {
		originalSetHop := setIPv6HopLimit
		originalSetControl := setIPv6HopControlMessage
		t.Cleanup(func() {
			setIPv6HopLimit = originalSetHop
			setIPv6HopControlMessage = originalSetControl
		})

		setIPv6HopLimit = func(conn *ipv6.PacketConn, ttl int) error {
			return errors.New("hop failed")
		}
		conn := &ipv6HopPacketConn{}
		if err := conn.Prepare(11); err == nil || err.Error() != "failed to set hop limit: hop failed" {
			t.Fatalf("expected wrapped hop limit error, got %v", err)
		}

		setIPv6HopLimit = func(conn *ipv6.PacketConn, ttl int) error {
			return nil
		}
		setIPv6HopControlMessage = func(conn *ipv6.PacketConn, on bool) error {
			return errors.New("control failed")
		}
		if err := conn.Prepare(11); err == nil || err.Error() != "failed to set control message: control failed" {
			t.Fatalf("expected wrapped control error, got %v", err)
		}
	})
}
