package mtu

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start an echo server for RFC 1191 PMTUD testing",
	Long: `Starts a UDP and/or TCP echo server that reflects received packets back to the sender.

This enables RFC 1191 Path MTU Discovery testing by providing an endpoint
that will echo data back, allowing the client to determine if packets of
a given size can traverse the path.

Examples:
  cidrator mtu server --port 4821
  cidrator mtu server --port 4821 --proto udp
  cidrator mtu server --port 4821 --proto tcp`,
	RunE: runServer,
}

func init() {
	serverCmd.Flags().Int("port", 4821, "Port to listen on")
	serverCmd.Flags().String("proto", "udp,tcp", "Protocols to serve (udp, tcp, or udp,tcp)")
	serverCmd.Flags().Bool("verbose", false, "Log received packets")
}

func runServer(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	proto, _ := cmd.Flags().GetString("proto")
	verbose, _ := cmd.Flags().GetBool("verbose")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		cancel()
	}()

	var wg sync.WaitGroup

	// Start UDP server
	if proto == "udp" || proto == "udp,tcp" || proto == "tcp,udp" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runUDPServer(ctx, port, verbose); err != nil {
				fmt.Fprintf(os.Stderr, "UDP server error: %v\n", err)
			}
		}()
	}

	// Start TCP server
	if proto == "tcp" || proto == "udp,tcp" || proto == "tcp,udp" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := runTCPServer(ctx, port, verbose); err != nil {
				fmt.Fprintf(os.Stderr, "TCP server error: %v\n", err)
			}
		}()
	}

	fmt.Printf("PMTUD echo server listening on port %d (%s)\n", port, proto)
	fmt.Println("Press Ctrl+C to stop")

	wg.Wait()
	return nil
}

// runUDPServer starts a UDP echo server
func runUDPServer(ctx context.Context, port int, verbose bool) error {
	addr := &net.UDPAddr{Port: port}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start UDP server: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	fmt.Printf("UDP echo server listening on :%d\n", port)

	// Buffer for receiving packets (max jumbo frame size)
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

		// Echo the packet back
		_, err = conn.WriteToUDP(buf[:n], remoteAddr)
		if err != nil {
			if verbose {
				fmt.Printf("UDP: echo error to %s: %v\n", remoteAddr, err)
			}
		}
	}
}

// runTCPServer starts a TCP echo server
func runTCPServer(ctx context.Context, port int, verbose bool) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}
	defer func() {
		if closeErr := listener.Close(); closeErr != nil {
			_ = closeErr
		}
	}()

	fmt.Printf("TCP echo server listening on :%d\n", port)

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
			return fmt.Errorf("TCP accept error: %w", err)
		}

		go handleTCPConnection(ctx, conn, verbose)
	}
}

// handleTCPConnection handles a single TCP connection
func handleTCPConnection(ctx context.Context, conn net.Conn, verbose bool) {
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

		// Echo the data back
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
