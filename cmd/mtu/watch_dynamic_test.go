package mtu

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type adjustableUDPEchoServer struct {
	conn       *net.UDPConn
	maxPayload atomic.Int64
	done       chan struct{}
	errCh      chan error
}

func startAdjustableUDPEchoServer(t *testing.T, maxPayload int) (*adjustableUDPEchoServer, func()) {
	t.Helper()

	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skip("sandbox does not allow local UDP listeners")
		}
		t.Fatalf("failed to start adjustable UDP echo server: %v", err)
	}

	server := &adjustableUDPEchoServer{
		conn:  conn,
		done:  make(chan struct{}),
		errCh: make(chan error, 1),
	}
	server.maxPayload.Store(int64(maxPayload))

	go func() {
		buf := make([]byte, 65535)
		for {
			if err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				server.errCh <- err
				return
			}

			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				var netErr net.Error
				if errors.As(err, &netErr) && netErr.Timeout() {
					select {
					case <-server.done:
						server.errCh <- nil
						return
					default:
						continue
					}
				}

				select {
				case <-server.done:
					server.errCh <- nil
				default:
					server.errCh <- err
				}
				return
			}

			if n > int(server.maxPayload.Load()) {
				continue
			}
			if _, err := conn.WriteToUDP(buf[:n], addr); err != nil {
				server.errCh <- err
				return
			}
		}
	}()

	shutdown := func() {
		close(server.done)
		_ = conn.Close()
		select {
		case err := <-server.errCh:
			if err != nil {
				t.Fatalf("adjustable UDP echo server returned error: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for adjustable UDP echo server shutdown")
		}
	}

	return server, shutdown
}

func TestRunWatchDetectsPMTUDropInJSONMode(t *testing.T) {
	initialPMTU := 1400
	droppedPMTU := 1360

	server, shutdown := startAdjustableUDPEchoServer(t, payloadSizeForPacket(initialPMTU, udpPacketOverhead(false)))
	defer shutdown()

	go func() {
		time.Sleep(400 * time.Millisecond)
		server.maxPayload.Store(int64(payloadSizeForPacket(droppedPMTU, udpPacketOverhead(false))))
	}()

	cmd := newDiscoveryOptionsCommand()
	cmd.Flags().Duration("interval", 50*time.Millisecond, "")
	cmd.Flags().Bool("mss-only", false, "")

	mustSetFlag(t, cmd, "proto", "udp")
	mustSetFlag(t, cmd, "port", strconv.Itoa(server.conn.LocalAddr().(*net.UDPAddr).Port))
	mustSetFlag(t, cmd, "min", "1300")
	mustSetFlag(t, cmd, "max", "1450")
	mustSetFlag(t, cmd, "timeout", "100ms")
	mustSetFlag(t, cmd, "json", "true")
	mustSetFlag(t, cmd, "interval", "50ms")

	output, err := captureStdout(t, func() error {
		return runWatch(cmd, []string{"127.0.0.1"})
	})
	if err == nil {
		t.Fatal("expected watch to exit on PMTU drop")
	}

	var previousPMTU, currentPMTU int
	if _, scanErr := fmt.Sscanf(err.Error(), "pmtu dropped from %d to %d", &previousPMTU, &currentPMTU); scanErr != nil {
		t.Fatalf("unexpected watch error format: %v", err)
	}
	if previousPMTU != initialPMTU {
		t.Fatalf("unexpected previous PMTU in watch error: %v", err)
	}
	if currentPMTU >= previousPMTU {
		t.Fatalf("expected watch error to report a drop, got %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least two JSON watch records, got %q", output)
	}

	var finalRecord struct {
		Target  string `json:"target"`
		PMTU    int    `json:"pmtu"`
		Changed bool   `json:"changed"`
	}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &finalRecord); err != nil {
		t.Fatalf("failed to parse final watch JSON record: %v", err)
	}
	if finalRecord.Target != "127.0.0.1" || finalRecord.PMTU != currentPMTU || !finalRecord.Changed {
		t.Fatalf("unexpected final watch record: %+v", finalRecord)
	}
}

func TestRunWatchRejectsHopMode(t *testing.T) {
	cmd := newDiscoveryOptionsCommand()
	cmd.Flags().Duration("interval", 10*time.Second, "")
	cmd.Flags().Bool("mss-only", false, "")
	mustSetFlag(t, cmd, "hops", "true")

	err := runWatch(cmd, []string{"127.0.0.1"})
	if err == nil {
		t.Fatal("expected hop-mode rejection")
	}
	if !strings.Contains(err.Error(), "--hops is only supported by mtu discover") {
		t.Fatalf("unexpected watch error: %v", err)
	}
}
