package mtu

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultPeerPort          = 4821
	defaultPeerListenAddress = "127.0.0.1"
	defaultPeerMaxPacketSize = 9216
	defaultPeerResponsePPS   = 100
)

type peerProtocolSet struct {
	udp bool
	tcp bool
}

type peerUDPConn interface {
	SetReadDeadline(time.Time) error
	ReadFromUDP([]byte) (int, *net.UDPAddr, error)
	WriteToUDP([]byte, *net.UDPAddr) (int, error)
	Close() error
	LocalAddr() net.Addr
}

type peerTCPListener interface {
	Accept() (net.Conn, error)
	Close() error
	Addr() net.Addr
}

type peerTCPConn interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	RemoteAddr() net.Addr
}

type peerRuntime struct {
	newContext func() (context.Context, context.CancelFunc)
	openUDP    func(listenAddr string, port int) (peerUDPConn, error)
	openTCP    func(listenAddr string, port int) (peerTCPListener, error)
	runUDP     func(ctx context.Context, conn peerUDPConn, verbose bool, maxPacketSize int, limiter *RateLimiter) error
	runTCP     func(ctx context.Context, listener peerTCPListener, verbose bool, maxPacketSize int, limiter *RateLimiter) error
	stdout     io.Writer
}

var defaultPeerRuntime = peerRuntime{
	newContext: func() (context.Context, context.CancelFunc) {
		return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	},
	openUDP: func(listenAddr string, port int) (peerUDPConn, error) {
		return openPeerUDPListener(listenAddr, port)
	},
	openTCP: func(listenAddr string, port int) (peerTCPListener, error) {
		return openPeerTCPListener(listenAddr, port)
	},
	runUDP: func(ctx context.Context, conn peerUDPConn, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
		return runUDPServer(ctx, conn, verbose, maxPacketSize, limiter)
	},
	runTCP: func(ctx context.Context, listener peerTCPListener, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
		return runTCPServer(ctx, listener, verbose, maxPacketSize, limiter)
	},
	stdout: os.Stdout,
}

var resolvePeerUDPAddr = func(network, address string) (*net.UDPAddr, error) {
	return net.ResolveUDPAddr(network, address)
}

var listenPeerUDP = func(network string, addr *net.UDPAddr) (*net.UDPConn, error) {
	return net.ListenUDP(network, addr)
}

var listenPeerTCP = func(network, address string) (net.Listener, error) {
	return net.Listen(network, address)
}

func (p peerProtocolSet) String() string {
	switch {
	case p.udp && p.tcp:
		return "udp,tcp"
	case p.udp:
		return "udp"
	case p.tcp:
		return "tcp"
	default:
		return ""
	}
}

// peerCmd represents the peer-assisted MTU endpoint command.
var peerCmd = &cobra.Command{
	Use:   "peer",
	Short: "Run an advanced peer-assisted endpoint for MTU verification",
	Long: `Peer runs a controlled TCP/UDP echo endpoint for peer-assisted MTU verification.

This is an advanced mode for cases where you control both ends of the path and
want application-to-application MTU validation. Use it with
'cidrator mtu discover --proto udp|tcp --port ...' against a host running this
endpoint.

Safety defaults:
- Binds to 127.0.0.1 by default
- Requires --allow-remote for non-loopback addresses
- Rate-limits responses and caps echoed packet size

Examples:
  cidrator mtu peer
  cidrator mtu peer --proto udp --listen 0.0.0.0 --allow-remote
  cidrator mtu discover branch-office.example.com --proto udp --port 4821`,
	RunE: runPeer,
}

func init() {
	peerCmd.Flags().Int("port", defaultPeerPort, "Port to listen on")
	peerCmd.Flags().String("proto", "udp,tcp", "Protocols to serve (udp, tcp, or udp,tcp)")
	peerCmd.Flags().String("listen", defaultPeerListenAddress, "Listen address (defaults to localhost only)")
	peerCmd.Flags().Bool("allow-remote", false, "Allow binding to non-loopback addresses for controlled remote testing")
	peerCmd.Flags().Int("max-packet-size", defaultPeerMaxPacketSize, "Maximum bytes echoed per packet or read")
	peerCmd.Flags().Int("response-pps", defaultPeerResponsePPS, "Maximum responses per second across all protocols (0 = unlimited)")
	peerCmd.Flags().Bool("verbose", false, "Log accepted packets and dropped oversized packets")
}

func runPeer(cmd *cobra.Command, args []string) error {
	return runPeerWithRuntime(cmd, defaultPeerRuntime)
}

func runPeerWithRuntime(cmd *cobra.Command, runtime peerRuntime) error {
	port, _ := cmd.Flags().GetInt("port")
	proto, _ := cmd.Flags().GetString("proto")
	listenAddr, _ := cmd.Flags().GetString("listen")
	allowRemote, _ := cmd.Flags().GetBool("allow-remote")
	maxPacketSize, _ := cmd.Flags().GetInt("max-packet-size")
	responsePPS, _ := cmd.Flags().GetInt("response-pps")
	verbose, _ := cmd.Flags().GetBool("verbose")

	if port < 1 || port > 65535 {
		return fmt.Errorf("--port must be between 1 and 65535")
	}
	if maxPacketSize <= 0 {
		return fmt.Errorf("--max-packet-size must be positive")
	}
	if responsePPS < 0 {
		return fmt.Errorf("--response-pps must be non-negative")
	}

	protocols, err := parsePeerProtocols(proto)
	if err != nil {
		return err
	}
	if err := validatePeerListenAddress(listenAddr, allowRemote); err != nil {
		return err
	}

	if runtime.newContext == nil {
		runtime.newContext = defaultPeerRuntime.newContext
	}
	if runtime.openUDP == nil {
		runtime.openUDP = defaultPeerRuntime.openUDP
	}
	if runtime.openTCP == nil {
		runtime.openTCP = defaultPeerRuntime.openTCP
	}
	if runtime.runUDP == nil {
		runtime.runUDP = defaultPeerRuntime.runUDP
	}
	if runtime.runTCP == nil {
		runtime.runTCP = defaultPeerRuntime.runTCP
	}
	if runtime.stdout == nil {
		runtime.stdout = io.Discard
	}

	ctx, stop := runtime.newContext()
	defer stop()

	limiter := NewRateLimiter(responsePPS)

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	var udpConn peerUDPConn
	if protocols.udp {
		udpConn, err = runtime.openUDP(listenAddr, port)
		if err != nil {
			return err
		}
		defer func() {
			_ = udpConn.Close()
		}()
	}

	var tcpListener peerTCPListener
	if protocols.tcp {
		tcpListener, err = runtime.openTCP(listenAddr, port)
		if err != nil {
			if udpConn != nil {
				_ = udpConn.Close()
			}
			return err
		}
		defer func() {
			_ = tcpListener.Close()
		}()
	}

	_, _ = fmt.Fprintf(runtime.stdout, "Advanced peer-assisted MTU endpoint listening on %s (%s)\n", net.JoinHostPort(listenAddr, strconv.Itoa(port)), protocols.String())
	_, _ = fmt.Fprintln(runtime.stdout, "Designed for controlled MTU verification between hosts you manage.")
	if allowRemote {
		_, _ = fmt.Fprintln(runtime.stdout, "Remote bind enabled. Do not expose this endpoint on the public internet.")
	} else {
		_, _ = fmt.Fprintln(runtime.stdout, "Bound to localhost only. Use --listen <addr> --allow-remote for controlled remote testing.")
	}
	_, _ = fmt.Fprintf(runtime.stdout, "Use 'cidrator mtu discover <host> --proto udp|tcp --port %d' from another host.\n", port)
	_, _ = fmt.Fprintln(runtime.stdout, "Press Ctrl+C to stop")

	if udpConn != nil {
		_, _ = fmt.Fprintf(runtime.stdout, "UDP peer endpoint listening on %s\n", udpConn.LocalAddr())
		wg.Add(1)
		go func() {
			defer wg.Done()
			if serveErr := runtime.runUDP(ctx, udpConn, verbose, maxPacketSize, limiter); serveErr != nil && ctx.Err() == nil {
				select {
				case errCh <- fmt.Errorf("udp peer error: %w", serveErr):
				default:
				}
			}
		}()
	}

	if tcpListener != nil {
		_, _ = fmt.Fprintf(runtime.stdout, "TCP peer endpoint listening on %s\n", tcpListener.Addr())
		wg.Add(1)
		go func() {
			defer wg.Done()
			if serveErr := runtime.runTCP(ctx, tcpListener, verbose, maxPacketSize, limiter); serveErr != nil && ctx.Err() == nil {
				select {
				case errCh <- fmt.Errorf("tcp peer error: %w", serveErr):
				default:
				}
			}
		}()
	}

	select {
	case err := <-errCh:
		stop()
		if udpConn != nil {
			_ = udpConn.Close()
		}
		if tcpListener != nil {
			_ = tcpListener.Close()
		}
		wg.Wait()
		return err
	case <-ctx.Done():
		_, _ = fmt.Fprintln(runtime.stdout, "\nShutting down peer endpoint...")
		if udpConn != nil {
			_ = udpConn.Close()
		}
		if tcpListener != nil {
			_ = tcpListener.Close()
		}
	}

	wg.Wait()
	return nil
}

func parsePeerProtocols(raw string) (peerProtocolSet, error) {
	var protocols peerProtocolSet
	for _, entry := range strings.Split(raw, ",") {
		switch strings.ToLower(strings.TrimSpace(entry)) {
		case "udp":
			protocols.udp = true
		case "tcp":
			protocols.tcp = true
		case "":
		default:
			return peerProtocolSet{}, fmt.Errorf("unsupported peer protocol %q: use tcp, udp, or tcp,udp", entry)
		}
	}

	if !protocols.udp && !protocols.tcp {
		return peerProtocolSet{}, fmt.Errorf("at least one peer protocol must be selected")
	}

	return protocols, nil
}

func validatePeerListenAddress(listenAddr string, allowRemote bool) error {
	if listenAddr == "" {
		return fmt.Errorf("--listen must not be empty")
	}

	ips, err := resolvePeerListenIPs(listenAddr)
	if err != nil {
		return fmt.Errorf("failed to resolve listen address %q: %w", listenAddr, err)
	}

	if allowRemote {
		return nil
	}

	for _, ip := range ips {
		if !ip.IsLoopback() {
			return fmt.Errorf("refusing to bind peer endpoint to %q without --allow-remote; advanced mode defaults to localhost for safety", listenAddr)
		}
	}

	return nil
}

func resolvePeerListenIPs(listenAddr string) ([]net.IP, error) {
	if ip := net.ParseIP(listenAddr); ip != nil {
		return []net.IP{ip}, nil
	}
	return lookupIPAddrs(listenAddr)
}

func openPeerUDPListener(listenAddr string, port int) (*net.UDPConn, error) {
	addr, err := resolvePeerUDPAddr("udp", net.JoinHostPort(listenAddr, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve UDP listen address: %w", err)
	}

	conn, err := listenPeerUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to start UDP peer endpoint: %w", err)
	}
	return conn, nil
}

func openPeerTCPListener(listenAddr string, port int) (net.Listener, error) {
	listener, err := listenPeerTCP("tcp", net.JoinHostPort(listenAddr, strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("failed to start TCP peer endpoint: %w", err)
	}
	return listener, nil
}

// runUDPServer starts a UDP peer endpoint.
func runUDPServer(ctx context.Context, conn peerUDPConn, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
	buf := make([]byte, 65535)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		// Set read deadline to check for context cancellation
		if err := conn.SetReadDeadline(deadlineFromContext(ctx)); err != nil {
			return err
		}

		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue // Check context and retry
			}
			if ctx.Err() != nil {
				return nil // Context cancelled
			}
			return fmt.Errorf("UDP read error: %w", err)
		}

		if verbose {
			fmt.Printf("UDP: received %d bytes from %s\n", n, remoteAddr)
		}
		if n > maxPacketSize {
			if verbose {
				fmt.Printf("UDP: dropped %d-byte packet from %s (max %d)\n", n, remoteAddr, maxPacketSize)
			}
			continue
		}

		limiter.Wait()
		_, err = conn.WriteToUDP(buf[:n], remoteAddr)
		if err != nil {
			if verbose {
				fmt.Printf("UDP: echo error to %s: %v\n", remoteAddr, err)
			}
		}
	}
}

// runTCPServer starts a TCP peer endpoint.
func runTCPServer(ctx context.Context, listener peerTCPListener, verbose bool, maxPacketSize int, limiter *RateLimiter) error {
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil // Context cancelled
			}
			return fmt.Errorf("TCP peer accept error: %w", err)
		}

		go handleTCPConnection(ctx, conn, verbose, maxPacketSize, limiter)
	}
}

// handleTCPConnection handles a single TCP peer connection.
func handleTCPConnection(ctx context.Context, conn peerTCPConn, verbose bool, maxPacketSize int, limiter *RateLimiter) {
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	remoteAddr := conn.RemoteAddr().String()
	if verbose {
		fmt.Printf("TCP: connection from %s\n", remoteAddr)
	}

	buf := make([]byte, 65535)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				if verbose {
					fmt.Printf("TCP: connection closed by %s\n", remoteAddr)
				}
				return
			}
			if verbose {
				fmt.Printf("TCP: read error from %s: %v\n", remoteAddr, err)
			}
			return
		}

		if verbose {
			fmt.Printf("TCP: received %d bytes from %s\n", n, remoteAddr)
		}
		if n > maxPacketSize {
			if verbose {
				fmt.Printf("TCP: closing %s after %d-byte read exceeded max %d\n", remoteAddr, n, maxPacketSize)
			}
			return
		}

		limiter.Wait()
		_, err = conn.Write(buf[:n])
		if err != nil {
			if verbose {
				fmt.Printf("TCP: write error to %s: %v\n", remoteAddr, err)
			}
			return
		}
	}
}

// deadlineFromContext returns a deadline for network operations
func deadlineFromContext(ctx context.Context) time.Time {
	deadline, ok := ctx.Deadline()
	if !ok {
		// Use a short timeout to periodically check context
		return time.Now().Add(500 * time.Millisecond)
	}
	return deadline
}
