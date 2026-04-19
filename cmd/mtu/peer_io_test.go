package mtu

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

type fakeUDPReadResult struct {
	data  []byte
	addr  *net.UDPAddr
	err   error
	after func()
}

type fakePeerUDPConn struct {
	localAddr        net.Addr
	setDeadlineErrs  []error
	reads            []fakeUDPReadResult
	writeErrs        []error
	setDeadlineCalls int
	readCalls        int
	writeCalls       int
	writes           [][]byte
	writeAddrs       []*net.UDPAddr
	closeCalls       int
}

func (c *fakePeerUDPConn) SetReadDeadline(time.Time) error {
	if c.setDeadlineCalls < len(c.setDeadlineErrs) {
		err := c.setDeadlineErrs[c.setDeadlineCalls]
		c.setDeadlineCalls++
		return err
	}
	c.setDeadlineCalls++
	return nil
}

func (c *fakePeerUDPConn) ReadFromUDP(buf []byte) (int, *net.UDPAddr, error) {
	if c.readCalls >= len(c.reads) {
		c.readCalls++
		return 0, nil, timeoutNetError{message: "i/o timeout"}
	}
	result := c.reads[c.readCalls]
	c.readCalls++
	if result.after != nil {
		defer result.after()
	}
	if result.err != nil {
		return 0, result.addr, result.err
	}
	n := copy(buf, result.data)
	return n, result.addr, nil
}

func (c *fakePeerUDPConn) WriteToUDP(buf []byte, addr *net.UDPAddr) (int, error) {
	c.writeCalls++
	c.writes = append(c.writes, append([]byte(nil), buf...))
	c.writeAddrs = append(c.writeAddrs, addr)
	if c.writeCalls-1 < len(c.writeErrs) && c.writeErrs[c.writeCalls-1] != nil {
		return 0, c.writeErrs[c.writeCalls-1]
	}
	return len(buf), nil
}

func (c *fakePeerUDPConn) Close() error {
	c.closeCalls++
	return nil
}

func (c *fakePeerUDPConn) LocalAddr() net.Addr {
	if c.localAddr == nil {
		return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4821}
	}
	return c.localAddr
}

type fakeTCPReadResult struct {
	data []byte
	err  error
}

type fakePeerTCPConn struct {
	remote     net.Addr
	reads      []fakeTCPReadResult
	readCalls  int
	writeErr   error
	writes     [][]byte
	closeCalls int
}

func (c *fakePeerTCPConn) Read(buf []byte) (int, error) {
	if c.readCalls >= len(c.reads) {
		return 0, io.EOF
	}
	result := c.reads[c.readCalls]
	c.readCalls++
	if result.err != nil {
		return 0, result.err
	}
	n := copy(buf, result.data)
	return n, nil
}

func (c *fakePeerTCPConn) Write(buf []byte) (int, error) {
	c.writes = append(c.writes, append([]byte(nil), buf...))
	if c.writeErr != nil {
		return 0, c.writeErr
	}
	return len(buf), nil
}

func (c *fakePeerTCPConn) Close() error {
	c.closeCalls++
	return nil
}

func (c *fakePeerTCPConn) RemoteAddr() net.Addr {
	if c.remote == nil {
		return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4242}
	}
	return c.remote
}

type scriptedListener struct {
	addr       net.Addr
	acceptErrs []error
	acceptConn []net.Conn
	acceptCall int
	closeCalls int
}

func (l *scriptedListener) Accept() (net.Conn, error) {
	idx := l.acceptCall
	l.acceptCall++
	if idx < len(l.acceptErrs) && l.acceptErrs[idx] != nil {
		return nil, l.acceptErrs[idx]
	}
	if idx < len(l.acceptConn) && l.acceptConn[idx] != nil {
		return l.acceptConn[idx], nil
	}
	return nil, errors.New("unexpected accept")
}

func (l *scriptedListener) Close() error {
	l.closeCalls++
	return nil
}

func (l *scriptedListener) Addr() net.Addr {
	if l.addr == nil {
		return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4821}
	}
	return l.addr
}

func TestPeerListenerOpenersUseInjectedSockets(t *testing.T) {
	originalResolveUDP := resolvePeerUDPAddr
	originalListenUDP := listenPeerUDP
	originalListenTCP := listenPeerTCP
	t.Cleanup(func() {
		resolvePeerUDPAddr = originalResolveUDP
		listenPeerUDP = originalListenUDP
		listenPeerTCP = originalListenTCP
	})

	resolvePeerUDPAddr = func(network, address string) (*net.UDPAddr, error) {
		return nil, errors.New("resolve failed")
	}
	if _, err := openPeerUDPListener("127.0.0.1", 4821); err == nil || !strings.Contains(err.Error(), "failed to resolve UDP listen address: resolve failed") {
		t.Fatalf("expected UDP resolve error, got %v", err)
	}

	resolvePeerUDPAddr = func(network, address string) (*net.UDPAddr, error) {
		return &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4821}, nil
	}
	listenPeerUDP = func(network string, addr *net.UDPAddr) (*net.UDPConn, error) {
		return nil, errors.New("bind failed")
	}
	if _, err := openPeerUDPListener("127.0.0.1", 4821); err == nil || !strings.Contains(err.Error(), "failed to start UDP peer endpoint: bind failed") {
		t.Fatalf("expected UDP listen error, got %v", err)
	}

	listenPeerTCP = func(network, address string) (net.Listener, error) {
		return nil, errors.New("listen failed")
	}
	if _, err := openPeerTCPListener("127.0.0.1", 4821); err == nil || !strings.Contains(err.Error(), "failed to start TCP peer endpoint: listen failed") {
		t.Fatalf("expected TCP listen error, got %v", err)
	}
}

func TestRunUDPServerWithFakeConn(t *testing.T) {
	t.Run("returns deadline errors", func(t *testing.T) {
		err := runUDPServer(context.Background(), &fakePeerUDPConn{
			setDeadlineErrs: []error{errors.New("deadline failed")},
		}, false, 1200, NewRateLimiter(0))
		if err == nil || !strings.Contains(err.Error(), "deadline failed") {
			t.Fatalf("expected deadline error, got %v", err)
		}
	})

	t.Run("returns non-timeout read errors", func(t *testing.T) {
		err := runUDPServer(context.Background(), &fakePeerUDPConn{
			reads: []fakeUDPReadResult{{err: errors.New("read failed")}},
		}, false, 1200, NewRateLimiter(0))
		if err == nil || !strings.Contains(err.Error(), "UDP read error: read failed") {
			t.Fatalf("expected wrapped read error, got %v", err)
		}
	})

	t.Run("stops after timeout when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		conn := &fakePeerUDPConn{
			reads: []fakeUDPReadResult{{
				err: timeoutNetError{message: "i/o timeout"},
				after: func() {
					cancel()
				},
			}},
		}
		if err := runUDPServer(ctx, conn, false, 1200, NewRateLimiter(0)); err != nil {
			t.Fatalf("expected graceful shutdown, got %v", err)
		}
	})

	t.Run("oversized packets are dropped and write errors are ignored", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		maxPacketSize := udpPacketSizeFromPayload(2, false)
		conn := &fakePeerUDPConn{
			reads: []fakeUDPReadResult{
				{data: []byte("oversized"), addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9000}},
				{
					data: []byte("ok"),
					addr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 9001},
					after: func() {
						cancel()
					},
				},
			},
			writeErrs: []error{errors.New("write failed")},
		}

		if err := runUDPServer(ctx, conn, false, maxPacketSize, NewRateLimiter(0)); err != nil {
			t.Fatalf("expected graceful shutdown after ignored write error, got %v", err)
		}
		if len(conn.writes) != 1 || string(conn.writes[0]) != "ok" {
			t.Fatalf("expected only in-range packet to be written back, got %+v", conn.writes)
		}
	})

	t.Run("ipv6 packet cap uses ipv6 overhead", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		maxPacketSize := udpPacketSizeFromPayload(2, true)
		conn := &fakePeerUDPConn{
			reads: []fakeUDPReadResult{
				{data: []byte("abc"), addr: &net.UDPAddr{IP: net.ParseIP("2001:db8::1"), Port: 9000}},
				{
					data: []byte("ok"),
					addr: &net.UDPAddr{IP: net.ParseIP("2001:db8::2"), Port: 9001},
					after: func() {
						cancel()
					},
				},
			},
		}

		if err := runUDPServer(ctx, conn, false, maxPacketSize, NewRateLimiter(0)); err != nil {
			t.Fatalf("expected graceful shutdown, got %v", err)
		}
		if len(conn.writes) != 1 || string(conn.writes[0]) != "ok" {
			t.Fatalf("expected only in-range IPv6 packet to be written back, got %+v", conn.writes)
		}
	})
}

func TestRunTCPServerWithFakeListener(t *testing.T) {
	t.Run("returns wrapped accept errors", func(t *testing.T) {
		err := runTCPServer(context.Background(), &scriptedListener{
			acceptErrs: []error{errors.New("accept failed")},
		}, false, 1200, NewRateLimiter(0))
		if err == nil || !strings.Contains(err.Error(), "TCP peer accept error: accept failed") {
			t.Fatalf("expected wrapped accept error, got %v", err)
		}
	})

	t.Run("returns nil once context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		listener := &scriptedListener{
			acceptErrs: []error{errors.New("listener closed")},
		}
		cancel()
		if err := runTCPServer(ctx, listener, false, 1200, NewRateLimiter(0)); err != nil {
			t.Fatalf("expected nil on cancelled context, got %v", err)
		}
	})
}

func TestHandleTCPConnectionWithFakeConn(t *testing.T) {
	t.Run("returns on EOF", func(t *testing.T) {
		conn := &fakePeerTCPConn{
			reads: []fakeTCPReadResult{{err: io.EOF}},
		}
		handleTCPConnection(context.Background(), conn, false, 1200, NewRateLimiter(0))
		if conn.closeCalls == 0 {
			t.Fatal("expected connection to be closed on EOF")
		}
	})

	t.Run("returns on generic read errors", func(t *testing.T) {
		conn := &fakePeerTCPConn{
			reads: []fakeTCPReadResult{{err: errors.New("read failed")}},
		}
		handleTCPConnection(context.Background(), conn, false, 1200, NewRateLimiter(0))
		if conn.closeCalls == 0 {
			t.Fatal("expected connection to be closed on read error")
		}
	})

	t.Run("returns on write errors", func(t *testing.T) {
		conn := &fakePeerTCPConn{
			reads:    []fakeTCPReadResult{{data: []byte("hello")}},
			writeErr: errors.New("write failed"),
		}
		handleTCPConnection(context.Background(), conn, false, 1200, NewRateLimiter(0))
		if len(conn.writes) != 1 || string(conn.writes[0]) != "hello" {
			t.Fatalf("expected write attempt for echoed payload, got %+v", conn.writes)
		}
		if conn.closeCalls == 0 {
			t.Fatal("expected connection to be closed on write error")
		}
	})
}
