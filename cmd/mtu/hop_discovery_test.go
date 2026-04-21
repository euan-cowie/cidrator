package mtu

import (
	"bytes"
	"context"
	"net"
	"testing"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type fakeHopPacketConn struct {
	packetConn *fakePacketConn
	prepareErr map[int]error
	ttl        int
	readFor    func(ttl, size int) fakePacketResponse
}

func (f *fakeHopPacketConn) Prepare(ttl int) error {
	if err, ok := f.prepareErr[ttl]; ok {
		return err
	}
	f.ttl = ttl
	return nil
}

func (f *fakeHopPacketConn) ReadFrom(buf []byte) (int, net.Addr, error) {
	f.packetConn.mu.Lock()
	size := f.packetConn.lastProbeSize
	f.packetConn.mu.Unlock()

	response := f.readFor(f.ttl, size)
	if response.err != nil {
		return 0, response.addr, response.err
	}
	n := copy(buf, response.data)
	return n, response.addr, nil
}

func newHopDiscovererForTest(conn *fakePacketConn, hopConn *fakeHopPacketConn) *MTUDiscoverer {
	discoverer := newICMPDiscovererForTest(conn, false)
	discoverer.hopFactory = func(_ net.PacketConn, _ bool) (hopPacketConn, error) {
		return hopConn, nil
	}
	return discoverer
}

func TestHopDiscoveryHelpers(t *testing.T) {
	routerIP := net.ParseIP("10.10.0.1")
	targetIP := net.ParseIP("10.20.0.2")

	t.Run("probeHop handles ttl exceeded and fragmentation", func(t *testing.T) {
		conn := &fakePacketConn{}
		hopConn := &fakeHopPacketConn{
			packetConn: conn,
			readFor: func(ttl, size int) fakePacketResponse {
				if size <= 1400 {
					return fakePacketResponse{
						data: mustMarshalICMP(t, &icmp.Message{
							Type: ipv4.ICMPTypeTimeExceeded,
							Code: 0,
							Body: &icmp.TimeExceeded{},
						}),
						addr: &net.IPAddr{IP: routerIP},
					}
				}
				return fakePacketResponse{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeDestinationUnreachable,
						Code: 4,
						Body: &icmp.DstUnreach{Data: []byte{0x00, 0x00, 0x05, 0x78}},
					}),
					addr: &net.IPAddr{IP: routerIP},
				}
			},
		}
		discoverer := newHopDiscovererForTest(conn, hopConn)

		successHop := discoverer.probeHop(context.Background(), 1, 1400)
		if successHop.Timeout || successHop.Error != "" || !successHop.Addr.Equal(routerIP) {
			t.Fatalf("unexpected successful hop probe: %+v", successHop)
		}

		failedHop := discoverer.probeHop(context.Background(), 1, 1450)
		if failedHop.Error != "Fragmentation Needed and Don't Fragment was Set" || failedHop.MTU != 1400 {
			t.Fatalf("unexpected fragmented hop probe: %+v", failedHop)
		}
	})

	t.Run("discoverMTUToHop finds hop threshold", func(t *testing.T) {
		conn := &fakePacketConn{}
		hopConn := &fakeHopPacketConn{
			packetConn: conn,
			readFor: func(ttl, size int) fakePacketResponse {
				if size <= 1500 {
					return fakePacketResponse{
						data: mustMarshalICMP(t, &icmp.Message{
							Type: ipv4.ICMPTypeTimeExceeded,
							Code: 0,
							Body: &icmp.TimeExceeded{},
						}),
						addr: &net.IPAddr{IP: routerIP},
					}
				}
				return fakePacketResponse{err: timeoutNetError{message: "i/o timeout"}}
			},
		}
		discoverer := newHopDiscovererForTest(conn, hopConn)

		if mtu := discoverer.discoverMTUToHop(context.Background(), 1, 1400, 1600); mtu != 1500 {
			t.Fatalf("discoverMTUToHop returned %d, want 1500", mtu)
		}
	})

	t.Run("DiscoverHopByHopMTU reports per-hop mtu and final pmtu", func(t *testing.T) {
		conn := &fakePacketConn{
			responseForProbe: func(size int) fakePacketResponse {
				if size <= 1400 {
					return fakePacketResponse{
						data: mustMarshalICMP(t, &icmp.Message{
							Type: ipv4.ICMPTypeEchoReply,
							Code: 0,
							Body: &icmp.Echo{ID: 1, Seq: 1},
						}),
						addr: &net.IPAddr{IP: targetIP},
					}
				}
				return fakePacketResponse{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeDestinationUnreachable,
						Code: 4,
						Body: &icmp.DstUnreach{Data: []byte{0x00, 0x00, 0x05, 0x78}},
					}),
					addr: &net.IPAddr{IP: routerIP},
				}
			},
		}
		hopConn := &fakeHopPacketConn{
			packetConn: conn,
			readFor: func(ttl, size int) fakePacketResponse {
				switch ttl {
				case 1:
					if size <= 1500 {
						return fakePacketResponse{
							data: mustMarshalICMP(t, &icmp.Message{
								Type: ipv4.ICMPTypeTimeExceeded,
								Code: 0,
								Body: &icmp.TimeExceeded{},
							}),
							addr: &net.IPAddr{IP: routerIP},
						}
					}
					return fakePacketResponse{err: timeoutNetError{message: "i/o timeout"}}
				case 2:
					if size <= 1400 {
						return fakePacketResponse{
							data: mustMarshalICMP(t, &icmp.Message{
								Type: ipv4.ICMPTypeEchoReply,
								Code: 0,
								Body: &icmp.Echo{ID: 1, Seq: 1},
							}),
							addr: &net.IPAddr{IP: targetIP},
						}
					}
					return fakePacketResponse{
						data: mustMarshalICMP(t, &icmp.Message{
							Type: ipv4.ICMPTypeDestinationUnreachable,
							Code: 4,
							Body: &icmp.DstUnreach{Data: []byte{0x00, 0x00, 0x05, 0x78}},
						}),
						addr: &net.IPAddr{IP: routerIP},
					}
				default:
					return fakePacketResponse{err: timeoutNetError{message: "i/o timeout"}}
				}
			},
		}
		discoverer := newHopDiscovererForTest(conn, hopConn)
		discoverer.targetAddr = &net.IPAddr{IP: targetIP}

		var progress bytes.Buffer
		discoverer.SetProgressWriter(&progress)

		result, err := discoverer.DiscoverHopByHopMTU(context.Background(), 4, 1600)
		if err != nil {
			t.Fatalf("DiscoverHopByHopMTU returned error: %v", err)
		}
		if result.FinalPMTU != 1400 || len(result.Hops) != 2 {
			t.Fatalf("unexpected hop-by-hop result: %+v", result)
		}
		if !result.Hops[0].Addr.Equal(routerIP) || result.Hops[0].MTU != 1500 {
			t.Fatalf("unexpected first hop: %+v", result.Hops[0])
		}
		if !result.Hops[1].Addr.Equal(targetIP) || result.Hops[1].MTU != 1400 {
			t.Fatalf("unexpected destination hop: %+v", result.Hops[1])
		}
		if !bytes.Contains(progress.Bytes(), []byte("Hop 1:")) || !bytes.Contains(progress.Bytes(), []byte("Reached destination")) {
			t.Fatalf("expected hop progress output, got %q", progress.String())
		}
	})

	t.Run("DiscoverHopByHopMTU falls back after timeout burst", func(t *testing.T) {
		conn := &fakePacketConn{
			responseForProbe: func(size int) fakePacketResponse {
				if size <= 1380 {
					return fakePacketResponse{
						data: mustMarshalICMP(t, &icmp.Message{
							Type: ipv4.ICMPTypeEchoReply,
							Code: 0,
							Body: &icmp.Echo{ID: 1, Seq: 1},
						}),
						addr: &net.IPAddr{IP: targetIP},
					}
				}
				return fakePacketResponse{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeDestinationUnreachable,
						Code: 4,
						Body: &icmp.DstUnreach{Data: []byte{0x00, 0x00, 0x05, 0x64}},
					}),
					addr: &net.IPAddr{IP: routerIP},
				}
			},
		}
		hopConn := &fakeHopPacketConn{
			packetConn: conn,
			readFor: func(ttl, size int) fakePacketResponse {
				return fakePacketResponse{err: timeoutNetError{message: "i/o timeout"}}
			},
		}
		discoverer := newHopDiscovererForTest(conn, hopConn)
		discoverer.targetAddr = &net.IPAddr{IP: targetIP}

		result, err := discoverer.DiscoverHopByHopMTU(context.Background(), 5, 1500)
		if err != nil {
			t.Fatalf("DiscoverHopByHopMTU returned error: %v", err)
		}
		if result.FinalPMTU != 1380 || len(result.Hops) != 4 {
			t.Fatalf("unexpected timeout-burst hop result: %+v", result)
		}
		for i, hop := range result.Hops {
			if !hop.Timeout {
				t.Fatalf("expected timeout hop at index %d, got %+v", i, hop)
			}
		}
	})
}
