package mtu

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type timeoutNetError struct {
	message string
}

func (e timeoutNetError) Error() string   { return e.message }
func (e timeoutNetError) Timeout() bool   { return true }
func (e timeoutNetError) Temporary() bool { return true }

type fakePacketResponse struct {
	data []byte
	addr net.Addr
	err  error
}

type fakePacketConn struct {
	mu                 sync.Mutex
	ipv6               bool
	writeErr           error
	readErr            error
	setReadDeadlineErr error
	lastProbeSize      int
	writes             int
	responseForProbe   func(size int) fakePacketResponse
}

func (f *fakePacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	f.mu.Lock()
	size := f.lastProbeSize
	readErr := f.readErr
	responseFn := f.responseForProbe
	f.mu.Unlock()

	if readErr != nil {
		return 0, nil, readErr
	}
	if responseFn == nil {
		return 0, nil, errors.New("no fake response configured")
	}

	response := responseFn(size)
	if response.err != nil {
		return 0, response.addr, response.err
	}
	n := copy(p, response.data)
	return n, response.addr, nil
}

func (f *fakePacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.writeErr != nil {
		return 0, f.writeErr
	}

	f.writes++
	headerSize := 20
	if f.ipv6 {
		headerSize = 40
	}
	f.lastProbeSize = len(p) + headerSize
	return len(p), nil
}

func (f *fakePacketConn) Close() error { return nil }

func (f *fakePacketConn) LocalAddr() net.Addr {
	return &net.IPAddr{IP: net.ParseIP("127.0.0.1")}
}

func (f *fakePacketConn) SetDeadline(t time.Time) error      { return nil }
func (f *fakePacketConn) SetWriteDeadline(t time.Time) error { return nil }

func (f *fakePacketConn) SetReadDeadline(t time.Time) error {
	return f.setReadDeadlineErr
}

func newICMPDiscovererForTest(conn net.PacketConn, ipv6Mode bool) *MTUDiscoverer {
	d := &MTUDiscoverer{
		target:     "example.com",
		ipv6:       ipv6Mode,
		protocol:   "icmp",
		timeout:    50 * time.Millisecond,
		conn:       conn,
		targetAddr: &net.IPAddr{IP: net.ParseIP("198.51.100.10")},
		security:   NewSecurityConfig(0),
	}
	d.security.Randomizer.useRandomData = false
	d.security.Randomizer.useRandomID = false
	d.security.Randomizer.useRandomSeq = false
	return d
}

func TestProbeAndDiscoverICMP(t *testing.T) {
	t.Run("probe succeeds on echo reply", func(t *testing.T) {
		conn := &fakePacketConn{
			responseForProbe: func(size int) fakePacketResponse {
				return fakePacketResponse{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeEchoReply,
						Code: 0,
						Body: &icmp.Echo{ID: 1, Seq: 1},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("198.51.100.10")},
				}
			},
		}
		discoverer := newICMPDiscovererForTest(conn, false)

		result := discoverer.probe(context.Background(), 1400)
		if !result.Success || result.ICMPErr != nil || result.Error != nil {
			t.Fatalf("expected successful ICMP probe, got %+v", result)
		}
		if conn.writes != 1 || conn.lastProbeSize != 1400 {
			t.Fatalf("unexpected probe write state: writes=%d size=%d", conn.writes, conn.lastProbeSize)
		}
	})

	t.Run("probe returns timeout error", func(t *testing.T) {
		conn := &fakePacketConn{readErr: timeoutNetError{message: "i/o timeout"}}
		discoverer := newICMPDiscovererForTest(conn, false)

		result := discoverer.probe(context.Background(), 1400)
		if result.Success {
			t.Fatalf("expected timeout probe to fail, got %+v", result)
		}
		var netErr net.Error
		if !errors.As(result.Error, &netErr) || !netErr.Timeout() {
			t.Fatalf("expected timeout net.Error, got %+v", result.Error)
		}
	})

	t.Run("probe uses fast path listener", func(t *testing.T) {
		conn := &fakePacketConn{
			responseForProbe: func(size int) fakePacketResponse {
				time.Sleep(100 * time.Millisecond)
				return fakePacketResponse{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeEchoReply,
						Code: 0,
						Body: &icmp.Echo{ID: 1, Seq: 1},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("198.51.100.10")},
				}
			},
		}
		discoverer := newICMPDiscovererForTest(conn, false)
		discoverer.icmpListener = &ICMPListener{errors: make(chan *FragmentationError, 1)}
		discoverer.icmpListener.errors <- &FragmentationError{NextHopMTU: 1360}

		result := discoverer.probe(context.Background(), 1400)
		if result.Success || result.ICMPErr == nil {
			t.Fatalf("expected fast-path ICMP failure, got %+v", result)
		}
		if result.ICMPErr.MTU != 1360 || result.ICMPErr.Message != "Fragmentation Needed (fast-path)" {
			t.Fatalf("unexpected fast-path ICMP error: %+v", result.ICMPErr)
		}
	})

	t.Run("discoverICMP binary-searches fragmentation point", func(t *testing.T) {
		mtuLimit := 1400
		conn := &fakePacketConn{
			responseForProbe: func(size int) fakePacketResponse {
				if size <= mtuLimit {
					return fakePacketResponse{
						data: mustMarshalICMP(t, &icmp.Message{
							Type: ipv4.ICMPTypeEchoReply,
							Code: 0,
							Body: &icmp.Echo{ID: 1, Seq: 1},
						}),
						addr: &net.IPAddr{IP: net.ParseIP("198.51.100.10")},
					}
				}

				return fakePacketResponse{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeDestinationUnreachable,
						Code: 4,
						Body: &icmp.DstUnreach{Data: []byte{0x00, 0x00, 0x05, 0x78}},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("192.0.2.1")},
				}
			},
		}
		discoverer := newICMPDiscovererForTest(conn, false)

		result, err := discoverer.discoverICMP(context.Background(), 1300, 1450)
		if err != nil {
			t.Fatalf("discoverICMP returned error: %v", err)
		}
		if result.PMTU != mtuLimit {
			t.Fatalf("unexpected PMTU: got %d, want %d", result.PMTU, mtuLimit)
		}
		if result.Protocol != "icmp" || result.Target != "example.com" {
			t.Fatalf("unexpected ICMP discovery result: %+v", result)
		}
	})

	t.Run("discoverICMP reports no working mtu", func(t *testing.T) {
		conn := &fakePacketConn{
			responseForProbe: func(size int) fakePacketResponse {
				return fakePacketResponse{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv6.ICMPTypePacketTooBig,
						Code: 0,
						Body: &icmp.PacketTooBig{MTU: 1280},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("2001:db8::1")},
				}
			},
			ipv6: true,
		}
		discoverer := newICMPDiscovererForTest(conn, true)

		_, err := discoverer.discoverICMP(context.Background(), 1280, 1400)
		if err == nil {
			t.Fatal("expected ICMP discovery failure when no size works")
		}
		if !strings.Contains(err.Error(), "no working MTU found in range 1280-1400") {
			t.Fatalf("unexpected discoverICMP error: %v", err)
		}
	})
}

func TestICMPDiscoverySocketHelpers(t *testing.T) {
	t.Run("set dont fragment socket rejects unsupported packet conn", func(t *testing.T) {
		discoverer := newICMPDiscovererForTest(&fakePacketConn{}, false)
		err := discoverer.setDontFragmentSocket()
		if err == nil {
			t.Fatal("expected unsupported connection type error")
		}
		if !strings.Contains(err.Error(), "unsupported connection type") {
			t.Fatalf("unexpected IPv4 DF socket error: %v", err)
		}
	})

	t.Run("set dont fragment socket rejects unsupported ipv6 packet conn", func(t *testing.T) {
		discoverer := newICMPDiscovererForTest(&fakePacketConn{ipv6: true}, true)
		err := discoverer.setDontFragmentSocket()
		if err == nil {
			t.Fatal("expected unsupported IPv6 connection type error")
		}
		if !strings.Contains(err.Error(), "unsupported connection type") {
			t.Fatalf("unexpected IPv6 DF socket error: %v", err)
		}
	})
}
