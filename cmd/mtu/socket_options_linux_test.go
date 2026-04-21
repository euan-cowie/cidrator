//go:build linux

package mtu

import (
	"net"
	"strings"
	"syscall"
	"testing"

	"golang.org/x/sys/unix"
)

func dialLocalLinuxTCPConn(t *testing.T, network string) *net.TCPConn {
	t.Helper()

	listener, err := net.Listen(network, "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create local listener: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			accepted <- conn
		}
	}()

	connRaw, err := net.Dial(network, listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to dial local listener: %v", err)
	}
	conn := connRaw.(*net.TCPConn)
	t.Cleanup(func() {
		_ = conn.Close()
	})

	serverConn := <-accepted
	t.Cleanup(func() {
		_ = serverConn.Close()
	})

	return conn
}

func TestLinuxSocketOptionsRejectUnsupportedConnTypes(t *testing.T) {
	client, _ := net.Pipe()
	defer func() {
		_ = client.Close()
	}()

	if err := setIPv4DontFragment(client); err == nil || !strings.Contains(err.Error(), "unsupported connection type") {
		t.Fatalf("expected unsupported IPv4 DF error, got %v", err)
	}
	if err := setIPv6DontFragment(client); err == nil || !strings.Contains(err.Error(), "unsupported connection type") {
		t.Fatalf("expected unsupported IPv6 DF error, got %v", err)
	}
	if _, err := getTCPMSS(client); err == nil || !strings.Contains(err.Error(), "unsupported connection type") {
		t.Fatalf("expected unsupported TCP MSS error, got %v", err)
	}
	if _, err := tcpTimestampsEnabled(client); err == nil || !strings.Contains(err.Error(), "unsupported connection type") {
		t.Fatalf("expected unsupported TCP info error, got %v", err)
	}
}

func TestLinuxSocketOptionsUseInjectedSyscalls(t *testing.T) {
	conn := dialLocalLinuxTCPConn(t, "tcp")

	originalSet := linuxSetsockoptInt
	originalGet := linuxGetsockoptInt
	originalTCPInfo := linuxGetsockoptTCPInfo
	t.Cleanup(func() {
		linuxSetsockoptInt = originalSet
		linuxGetsockoptInt = originalGet
		linuxGetsockoptTCPInfo = originalTCPInfo
	})

	var sawOption int
	linuxSetsockoptInt = func(fd, level, opt, value int) error {
		sawOption = opt
		return nil
	}

	if err := setIPv4DontFragment(conn); err != nil {
		t.Fatalf("setIPv4DontFragment returned error: %v", err)
	}
	if sawOption != IP_MTU_DISCOVER {
		t.Fatalf("setIPv4DontFragment used opt %d, want %d", sawOption, IP_MTU_DISCOVER)
	}

	if err := setIPv6DontFragment(conn); err != nil {
		t.Fatalf("setIPv6DontFragment returned error: %v", err)
	}
	if sawOption != IPV6_MTU_DISCOVER {
		t.Fatalf("setIPv6DontFragment used opt %d, want %d", sawOption, IPV6_MTU_DISCOVER)
	}

	if err := setTCPMSS(10, 1400); err != nil {
		t.Fatalf("setTCPMSS returned error: %v", err)
	}
	if sawOption != syscall.TCP_MAXSEG {
		t.Fatalf("setTCPMSS used opt %d, want %d", sawOption, syscall.TCP_MAXSEG)
	}

	linuxGetsockoptInt = func(fd, level, opt int) (int, error) {
		return 1412, nil
	}
	mss, err := getTCPMSS(conn)
	if err != nil {
		t.Fatalf("getTCPMSS returned error: %v", err)
	}
	if mss != 1412 {
		t.Fatalf("getTCPMSS returned %d, want 1412", mss)
	}

	linuxGetsockoptTCPInfo = func(fd, level, opt int) (*unix.TCPInfo, error) {
		return &unix.TCPInfo{Options: tcpiOptTimestamps}, nil
	}
	enabled, err := tcpTimestampsEnabled(conn)
	if err != nil {
		t.Fatalf("tcpTimestampsEnabled returned error: %v", err)
	}
	if !enabled {
		t.Fatal("expected injected TCP timestamp option to be reported as enabled")
	}
}
