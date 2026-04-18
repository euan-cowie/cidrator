package mtu

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestPeerProtocolSetString(t *testing.T) {
	tests := []struct {
		name      string
		protocols peerProtocolSet
		want      string
	}{
		{name: "udp and tcp", protocols: peerProtocolSet{udp: true, tcp: true}, want: "udp,tcp"},
		{name: "udp only", protocols: peerProtocolSet{udp: true}, want: "udp"},
		{name: "tcp only", protocols: peerProtocolSet{tcp: true}, want: "tcp"},
		{name: "none", protocols: peerProtocolSet{}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.protocols.String(); got != tt.want {
				t.Fatalf("peerProtocolSet.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRunPeerRejectsInvalidFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "invalid port",
			args:    []string{"--port", "70000"},
			wantErr: "--port must be between 1 and 65535",
		},
		{
			name:    "invalid max packet size",
			args:    []string{"--max-packet-size", "0"},
			wantErr: "--max-packet-size must be positive",
		},
		{
			name:    "invalid response pps",
			args:    []string{"--response-pps", "-1"},
			wantErr: "--response-pps must be non-negative",
		},
		{
			name:    "invalid protocol",
			args:    []string{"--proto", "icmp"},
			wantErr: "unsupported peer protocol",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newPeerCommandForTest()
			for i := 0; i < len(tt.args); i += 2 {
				mustSetFlag(t, cmd, strings.TrimPrefix(tt.args[i], "--"), tt.args[i+1])
			}

			err := runPeer(cmd, nil)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestPeerListenersAndServers(t *testing.T) {
	t.Run("udp echoes and drops oversized packets", func(t *testing.T) {
		serverConn, err := openPeerUDPListener("127.0.0.1", 0)
		if err != nil {
			if strings.Contains(err.Error(), "operation not permitted") {
				t.Skip("sandbox does not allow local UDP listeners")
			}
			t.Fatalf("openPeerUDPListener returned error: %v", err)
		}
		defer func() { _ = serverConn.Close() }()

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			errCh <- runUDPServer(ctx, serverConn, false, 4, NewRateLimiter(0))
		}()

		clientConn, err := net.DialUDP("udp", nil, serverConn.LocalAddr().(*net.UDPAddr))
		if err != nil {
			cancel()
			t.Fatalf("failed to dial UDP peer: %v", err)
		}
		defer func() { _ = clientConn.Close() }()

		payload := []byte("ping")
		if _, err := clientConn.Write(payload); err != nil {
			cancel()
			t.Fatalf("failed to write UDP payload: %v", err)
		}

		reply := make([]byte, len(payload))
		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, _, err := clientConn.ReadFromUDP(reply)
		if err != nil {
			cancel()
			t.Fatalf("failed to read echoed UDP payload: %v", err)
		}
		if string(reply[:n]) != string(payload) {
			cancel()
			t.Fatalf("unexpected UDP echo: got %q, want %q", string(reply[:n]), string(payload))
		}

		if _, err := clientConn.Write([]byte("oversized")); err != nil {
			cancel()
			t.Fatalf("failed to write oversized UDP payload: %v", err)
		}

		_ = clientConn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, _, err = clientConn.ReadFromUDP(reply)
		var netErr net.Error
		if err == nil || !errors.As(err, &netErr) || !netErr.Timeout() {
			cancel()
			t.Fatalf("expected oversized UDP packet to be dropped, got %v", err)
		}

		cancel()
		_ = serverConn.Close()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("runUDPServer returned error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for UDP server shutdown")
		}
	})

	t.Run("tcp echoes and stops on cancel", func(t *testing.T) {
		listener, err := openPeerTCPListener("127.0.0.1", 0)
		if err != nil {
			if strings.Contains(err.Error(), "operation not permitted") {
				t.Skip("sandbox does not allow local TCP listeners")
			}
			t.Fatalf("openPeerTCPListener returned error: %v", err)
		}
		defer func() { _ = listener.Close() }()

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error, 1)
		go func() {
			errCh <- runTCPServer(ctx, listener, false, 8, NewRateLimiter(0))
		}()

		clientConn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			cancel()
			t.Fatalf("failed to dial TCP peer: %v", err)
		}
		defer func() { _ = clientConn.Close() }()

		payload := []byte("hello")
		if _, err := clientConn.Write(payload); err != nil {
			cancel()
			t.Fatalf("failed to write TCP payload: %v", err)
		}

		reply := make([]byte, len(payload))
		_ = clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, err := clientConn.Read(reply)
		if err != nil {
			cancel()
			t.Fatalf("failed to read echoed TCP payload: %v", err)
		}
		if string(reply[:n]) != string(payload) {
			cancel()
			t.Fatalf("unexpected TCP echo: got %q, want %q", string(reply[:n]), string(payload))
		}

		cancel()
		select {
		case err := <-errCh:
			if err != nil {
				t.Fatalf("runTCPServer returned error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for TCP server shutdown")
		}
	})
}

func newPeerCommandForTest() *cobra.Command {
	cmd := &cobra.Command{Use: "peer"}
	flags := cmd.Flags()
	flags.Int("port", defaultPeerPort, "")
	flags.String("proto", "udp,tcp", "")
	flags.String("listen", defaultPeerListenAddress, "")
	flags.Bool("allow-remote", false, "")
	flags.Int("max-packet-size", defaultPeerMaxPacketSize, "")
	flags.Int("response-pps", defaultPeerResponsePPS, "")
	flags.Bool("verbose", false, "")
	return cmd
}

func TestHandleTCPConnectionClosesOversizedReads(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go handleTCPConnection(ctx, serverConn, false, 3, NewRateLimiter(0))

	if _, err := clientConn.Write([]byte("hello")); err != nil {
		t.Fatalf("failed to write oversized TCP payload: %v", err)
	}

	buf := make([]byte, 8)
	_ = clientConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, err := clientConn.Read(buf)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected oversized TCP read to close the connection, got %v", err)
	}
}

func TestDeadlineFromContext(t *testing.T) {
	t.Run("uses context deadline when present", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		deadline := deadlineFromContext(ctx)
		if deadline.Before(time.Now().Add(1500*time.Millisecond)) || deadline.After(time.Now().Add(2500*time.Millisecond)) {
			t.Fatalf("expected deadline near context timeout, got %v", deadline)
		}
	})

	t.Run("uses short polling deadline without context deadline", func(t *testing.T) {
		deadline := deadlineFromContext(context.Background())
		if deadline.Before(time.Now().Add(400*time.Millisecond)) || deadline.After(time.Now().Add(700*time.Millisecond)) {
			t.Fatalf("expected short fallback deadline, got %v", deadline)
		}
	})
}
