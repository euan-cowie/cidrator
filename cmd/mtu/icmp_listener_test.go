package mtu

import (
	"bytes"
	"context"
	"encoding/binary"
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

type fakeICMPReadResult struct {
	data []byte
	addr net.Addr
	err  error
}

type fakeICMPReadConn struct {
	mu               sync.Mutex
	setDeadlineErrs  []error
	reads            []fakeICMPReadResult
	setDeadlineCalls int
	readCalls        int
	closeErr         error
	closeCalls       int
	readHook         func()
}

func (c *fakeICMPReadConn) SetReadDeadline(time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	idx := c.setDeadlineCalls
	c.setDeadlineCalls++
	if idx < len(c.setDeadlineErrs) {
		return c.setDeadlineErrs[idx]
	}
	return nil
}

func (c *fakeICMPReadConn) ReadFrom(buf []byte) (int, net.Addr, error) {
	c.mu.Lock()
	hook := c.readHook
	c.readHook = nil
	idx := c.readCalls
	c.readCalls++
	var result fakeICMPReadResult
	if idx < len(c.reads) {
		result = c.reads[idx]
	} else {
		result = fakeICMPReadResult{err: timeoutNetError{message: "i/o timeout"}}
	}
	c.mu.Unlock()

	if hook != nil {
		hook()
	}
	if result.err != nil {
		return 0, result.addr, result.err
	}

	n := copy(buf, result.data)
	return n, result.addr, nil
}

func (c *fakeICMPReadConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeCalls++
	return c.closeErr
}

func TestICMPListenerHelpers(t *testing.T) {
	listener := &ICMPListener{
		errors: make(chan *FragmentationError, 4),
		done:   make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listener.Start(ctx)
	if !listener.running {
		t.Fatal("expected listener to be marked running after Start")
	}

	listener.Start(ctx)
	if listener.Errors() == nil {
		t.Fatal("expected Errors channel")
	}

	go func() {
		listener.errors <- &FragmentationError{OriginalDst: net.ParseIP("192.0.2.10"), NextHopMTU: 1400}
		listener.errors <- &FragmentationError{OriginalDst: net.ParseIP("192.0.2.20"), NextHopMTU: 1360}
	}()

	err := listener.WaitForError(context.Background(), net.ParseIP("192.0.2.20"), time.Second)
	if err == nil {
		t.Fatal("expected matching fragmentation error")
	}
	if err.NextHopMTU != 1360 {
		t.Fatalf("unexpected next-hop MTU: %d", err.NextHopMTU)
	}

	if closeErr := listener.Close(); closeErr != nil {
		t.Fatalf("listener Close returned error: %v", closeErr)
	}
	if listener.running {
		t.Fatal("expected listener to stop running after Close")
	}

	discoverer := &MTUDiscoverer{}
	discoverer.SetICMPListener(listener)
	if discoverer.icmpListener != listener {
		t.Fatal("expected SetICMPListener to attach the listener")
	}
}

func TestParseICMPv4Error(t *testing.T) {
	listener := &ICMPListener{}

	if err := listener.parseICMPv4Error([]byte{0x45}, nil); err != nil {
		t.Fatalf("expected short ICMP payload to be ignored, got %+v", err)
	}

	data := make([]byte, 28)
	data[0] = 0x45
	data[9] = 17
	copy(data[16:20], net.ParseIP("198.51.100.7").To4())
	binary.BigEndian.PutUint16(data[20:22], 53000)
	binary.BigEndian.PutUint16(data[22:24], 4821)

	err := listener.parseICMPv4Error(data, nil)
	if err == nil {
		t.Fatal("expected parsed ICMPv4 fragmentation error")
	}
	if !err.OriginalDst.Equal(net.ParseIP("198.51.100.7")) {
		t.Fatalf("unexpected original destination: %v", err.OriginalDst)
	}
	if err.OriginalSrcPort != 53000 || err.OriginalDstPort != 4821 {
		t.Fatalf("unexpected original ports: %+v", err)
	}
}

func TestNewICMPListener(t *testing.T) {
	originalOpen := openICMPListenPacket
	originalWarnings := icmpListenerWarningOutput
	t.Cleanup(func() {
		openICMPListenPacket = originalOpen
		icmpListenerWarningOutput = originalWarnings
	})

	t.Run("uses available sockets and warns for unavailable families", func(t *testing.T) {
		var warnings bytes.Buffer
		icmpListenerWarningOutput = &warnings

		ipv6Conn := &fakeICMPReadConn{}
		openICMPListenPacket = func(network, address string) (icmpReadConn, error) {
			if network == "ip4:icmp" {
				return nil, errors.New("permission denied")
			}
			if network == "ip6:ipv6-icmp" && address == "::" {
				return ipv6Conn, nil
			}
			t.Fatalf("unexpected listen request: %s %s", network, address)
			return nil, nil
		}

		listener, err := NewICMPListener()
		if err != nil {
			t.Fatalf("NewICMPListener returned error: %v", err)
		}
		if listener.conn4 != nil {
			t.Fatal("expected IPv4 socket to be unavailable")
		}
		if listener.conn6 != ipv6Conn {
			t.Fatal("expected IPv6 socket to be attached")
		}
		if !strings.Contains(warnings.String(), "Could not open IPv4 ICMP socket") {
			t.Fatalf("expected IPv4 warning, got %q", warnings.String())
		}
	})

	t.Run("fails when no ICMP sockets can be opened", func(t *testing.T) {
		openICMPListenPacket = func(network, address string) (icmpReadConn, error) {
			return nil, errors.New("denied")
		}

		listener, err := NewICMPListener()
		if err == nil || !strings.Contains(err.Error(), "failed to open any ICMP socket") {
			t.Fatalf("expected constructor failure, got listener=%+v err=%v", listener, err)
		}
	})
}

func TestICMPListenerListenIPv4(t *testing.T) {
	embedded := make([]byte, 28)
	embedded[0] = 0x45
	embedded[9] = 17
	copy(embedded[16:20], net.ParseIP("198.51.100.10").To4())
	binary.BigEndian.PutUint16(embedded[20:22], 53000)
	binary.BigEndian.PutUint16(embedded[22:24], 4821)

	listener := &ICMPListener{
		conn4: &fakeICMPReadConn{
			setDeadlineErrs: []error{errors.New("temporary deadline error")},
			reads: []fakeICMPReadResult{
				{data: []byte{0x01, 0x02, 0x03}, addr: &net.IPAddr{IP: net.ParseIP("10.10.0.1")}},
				{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeEchoReply,
						Code: 0,
						Body: &icmp.Echo{ID: 1, Seq: 1},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("10.10.0.1")},
				},
				{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv4.ICMPTypeDestinationUnreachable,
						Code: 4,
						Body: &icmp.DstUnreach{Data: embedded},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("10.10.0.1")},
				},
			},
		},
		errors: make(chan *FragmentationError, 1),
		done:   make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		listener.listenIPv4(ctx)
	}()

	var icmpErr *FragmentationError
	select {
	case icmpErr = <-listener.errors:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for IPv4 fragmentation error")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("listenIPv4 did not stop after cancellation")
	}

	if !icmpErr.OriginalDst.Equal(net.ParseIP("198.51.100.10")) {
		t.Fatalf("unexpected original destination: %v", icmpErr.OriginalDst)
	}
	if icmpErr.OriginalSrcPort != 53000 || icmpErr.OriginalDstPort != 4821 {
		t.Fatalf("unexpected original ports: %+v", icmpErr)
	}
}

func TestICMPListenerListenIPv6(t *testing.T) {
	embedded := make([]byte, 40)
	copy(embedded[24:40], net.ParseIP("2001:db8::42").To16())

	listener := &ICMPListener{
		conn6: &fakeICMPReadConn{
			reads: []fakeICMPReadResult{
				{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv6.ICMPTypeEchoReply,
						Code: 0,
						Body: &icmp.Echo{ID: 1, Seq: 1},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("2001:db8::1")},
				},
				{
					data: mustMarshalICMP(t, &icmp.Message{
						Type: ipv6.ICMPTypePacketTooBig,
						Code: 0,
						Body: &icmp.PacketTooBig{MTU: 1400, Data: embedded},
					}),
					addr: &net.IPAddr{IP: net.ParseIP("2001:db8::1")},
				},
			},
		},
		errors: make(chan *FragmentationError, 1),
		done:   make(chan struct{}),
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		listener.listenIPv6(ctx)
	}()

	var icmpErr *FragmentationError
	select {
	case icmpErr = <-listener.errors:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for IPv6 packet-too-big error")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("listenIPv6 did not stop after cancellation")
	}

	if icmpErr.NextHopMTU != 1400 {
		t.Fatalf("unexpected IPv6 next-hop MTU: %d", icmpErr.NextHopMTU)
	}
	if !icmpErr.OriginalDst.Equal(net.ParseIP("2001:db8::42")) {
		t.Fatalf("unexpected IPv6 original destination: %v", icmpErr.OriginalDst)
	}
}

func TestICMPListenerReadErrorStopsWhenDoneClosed(t *testing.T) {
	listener := &ICMPListener{
		errors: make(chan *FragmentationError, 1),
		done:   make(chan struct{}),
	}
	conn := &fakeICMPReadConn{
		readHook: func() {
			close(listener.done)
		},
		reads: []fakeICMPReadResult{
			{err: errors.New("socket closed")},
		},
	}
	listener.conn4 = conn

	done := make(chan struct{})
	go func() {
		defer close(done)
		listener.listenIPv4(context.Background())
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("listenIPv4 did not stop after done was closed during read error")
	}
}

func TestICMPListenerCloseAggregatesErrors(t *testing.T) {
	listener := &ICMPListener{
		conn4:   &fakeICMPReadConn{closeErr: errors.New("ipv4 close failed")},
		conn6:   &fakeICMPReadConn{closeErr: errors.New("ipv6 close failed")},
		errors:  make(chan *FragmentationError, 1),
		done:    make(chan struct{}),
		running: true,
	}

	err := listener.Close()
	if err == nil || !strings.Contains(err.Error(), "close errors") {
		t.Fatalf("expected aggregated close error, got %v", err)
	}
}
