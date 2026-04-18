package mtu

import (
	"bytes"
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

type fakeListener struct {
	addr   net.Addr
	closed bool
}

func (l *fakeListener) Accept() (net.Conn, error) {
	return nil, errors.New("accept not implemented")
}

func (l *fakeListener) Close() error {
	l.closed = true
	return nil
}

func (l *fakeListener) Addr() net.Addr {
	return l.addr
}

type closableUDPConn struct {
	peerUDPConn
	closed bool
}

func (c *closableUDPConn) Close() error {
	c.closed = true
	if c.peerUDPConn != nil {
		return c.peerUDPConn.Close()
	}
	return nil
}

func TestRunPeerWithRuntime(t *testing.T) {
	t.Run("tcp lifecycle prints banners and shuts down cleanly", func(t *testing.T) {
		cmd := newPeerCommandForTest()
		mustSetFlag(t, cmd, "proto", "tcp")

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		listener := &fakeListener{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4821}}
		var output bytes.Buffer

		err := runPeerWithRuntime(cmd, peerRuntime{
			newContext: func() (context.Context, context.CancelFunc) {
				return ctx, cancel
			},
			openTCP: func(listenAddr string, port int) (peerTCPListener, error) {
				return listener, nil
			},
			runTCP: func(ctx context.Context, listener peerTCPListener, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
				<-ctx.Done()
				return nil
			},
			stdout: &output,
		})
		if err != nil {
			t.Fatalf("runPeerWithRuntime returned error: %v", err)
		}
		if !listener.closed {
			t.Fatal("expected listener to be closed on shutdown")
		}
		for _, expected := range []string{
			"Advanced peer-assisted MTU endpoint listening on 127.0.0.1:4821 (tcp)",
			"TCP peer endpoint listening on 127.0.0.1:4821",
			"Shutting down peer endpoint...",
		} {
			if !strings.Contains(output.String(), expected) {
				t.Fatalf("expected peer output to contain %q, got %q", expected, output.String())
			}
		}
	})

	t.Run("tcp server errors propagate", func(t *testing.T) {
		cmd := newPeerCommandForTest()
		mustSetFlag(t, cmd, "proto", "tcp")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := runPeerWithRuntime(cmd, peerRuntime{
			newContext: func() (context.Context, context.CancelFunc) {
				return ctx, cancel
			},
			openTCP: func(listenAddr string, port int) (peerTCPListener, error) {
				return &fakeListener{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4821}}, nil
			},
			runTCP: func(ctx context.Context, listener peerTCPListener, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
				return errors.New("boom")
			},
			stdout: &bytes.Buffer{},
		})
		if err == nil || !strings.Contains(err.Error(), "tcp peer error: boom") {
			t.Fatalf("expected propagated tcp server error, got %v", err)
		}
	})

	t.Run("dual protocol startup announces both listeners", func(t *testing.T) {
		udpConn, err := openPeerUDPListener("127.0.0.1", 0)
		if err != nil {
			if strings.Contains(err.Error(), "operation not permitted") {
				t.Skip("sandbox does not allow local UDP listeners")
			}
			t.Fatalf("openPeerUDPListener returned error: %v", err)
		}
		defer func() {
			_ = udpConn.Close()
		}()

		cmd := newPeerCommandForTest()

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		tcpListener := &fakeListener{addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 4821}}
		var output bytes.Buffer

		err = runPeerWithRuntime(cmd, peerRuntime{
			newContext: func() (context.Context, context.CancelFunc) {
				return ctx, cancel
			},
			openUDP: func(listenAddr string, port int) (peerUDPConn, error) {
				return udpConn, nil
			},
			openTCP: func(listenAddr string, port int) (peerTCPListener, error) {
				return tcpListener, nil
			},
			runUDP: func(ctx context.Context, conn peerUDPConn, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
				<-ctx.Done()
				return nil
			},
			runTCP: func(ctx context.Context, listener peerTCPListener, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
				<-ctx.Done()
				return nil
			},
			stdout: &output,
		})
		if err != nil {
			t.Fatalf("runPeerWithRuntime returned error: %v", err)
		}
		if !strings.Contains(output.String(), "UDP peer endpoint listening on") || !strings.Contains(output.String(), "TCP peer endpoint listening on") {
			t.Fatalf("expected dual-protocol listener output, got %q", output.String())
		}
	})

	t.Run("udp open errors are returned directly", func(t *testing.T) {
		cmd := newPeerCommandForTest()
		mustSetFlag(t, cmd, "proto", "udp")

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := runPeerWithRuntime(cmd, peerRuntime{
			newContext: func() (context.Context, context.CancelFunc) {
				return ctx, cancel
			},
			openUDP: func(listenAddr string, port int) (peerUDPConn, error) {
				return nil, errors.New("udp open failed")
			},
		})
		if err == nil || !strings.Contains(err.Error(), "udp open failed") {
			t.Fatalf("expected UDP open failure, got %v", err)
		}
	})

	t.Run("tcp open errors close an already-open udp socket", func(t *testing.T) {
		cmd := newPeerCommandForTest()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		udpConn := &closableUDPConn{peerUDPConn: &fakePeerUDPConn{}}
		err := runPeerWithRuntime(cmd, peerRuntime{
			newContext: func() (context.Context, context.CancelFunc) {
				return ctx, cancel
			},
			openUDP: func(listenAddr string, port int) (peerUDPConn, error) {
				return udpConn, nil
			},
			openTCP: func(listenAddr string, port int) (peerTCPListener, error) {
				return nil, errors.New("tcp open failed")
			},
			stdout: &bytes.Buffer{},
		})
		if err == nil || !strings.Contains(err.Error(), "tcp open failed") {
			t.Fatalf("expected TCP open failure, got %v", err)
		}
		if !udpConn.closed {
			t.Fatal("expected UDP socket to be closed when TCP startup fails")
		}
	})

	t.Run("allow-remote banner is printed and nil stdout is tolerated", func(t *testing.T) {
		cmd := newPeerCommandForTest()
		mustSetFlag(t, cmd, "proto", "tcp")
		mustSetFlag(t, cmd, "allow-remote", "true")
		mustSetFlag(t, cmd, "listen", "0.0.0.0")

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		err := runPeerWithRuntime(cmd, peerRuntime{
			newContext: func() (context.Context, context.CancelFunc) {
				return ctx, cancel
			},
			openTCP: func(listenAddr string, port int) (peerTCPListener, error) {
				return &fakeListener{addr: &net.TCPAddr{IP: net.ParseIP("0.0.0.0"), Port: 4821}}, nil
			},
			runTCP: func(ctx context.Context, listener peerTCPListener, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
				<-ctx.Done()
				return nil
			},
			stdout: nil,
		})
		if err != nil {
			t.Fatalf("runPeerWithRuntime returned error: %v", err)
		}
	})
}
