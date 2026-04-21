//go:build darwin

package mtu

import (
	"net"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func dialLocalTCPConn(t *testing.T, network string) *net.TCPConn {
	t.Helper()

	listener, err := net.Listen(network, "127.0.0.1:0")
	if err != nil {
		if strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "bind: operation not permitted") {
			t.Skipf("sandbox does not allow local TCP listeners: %v", err)
		}
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

func TestDarwinSocketOptionsRejectUnsupportedConnTypes(t *testing.T) {
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

func TestDarwinSocketOptionsUseInjectedSyscalls(t *testing.T) {
	conn := dialLocalTCPConn(t, "tcp")

	originalSet := darwinSetsockoptInt
	originalGet := darwinGetsockoptInt
	originalTCPInfo := darwinGetsockoptTCPConnectionInfo
	t.Cleanup(func() {
		darwinSetsockoptInt = originalSet
		darwinGetsockoptInt = originalGet
		darwinGetsockoptTCPConnectionInfo = originalTCPInfo
	})

	var sawOption int
	darwinSetsockoptInt = func(fd, level, opt, value int) error {
		sawOption = opt
		return nil
	}

	if err := setIPv4DontFragment(conn); err != nil {
		t.Fatalf("setIPv4DontFragment returned error: %v", err)
	}
	if sawOption != unix.IP_DONTFRAG {
		t.Fatalf("setIPv4DontFragment used opt %d, want %d", sawOption, unix.IP_DONTFRAG)
	}

	if err := setIPv6DontFragment(conn); err != nil {
		t.Fatalf("setIPv6DontFragment returned error: %v", err)
	}
	if sawOption != unix.IPV6_DONTFRAG {
		t.Fatalf("setIPv6DontFragment used opt %d, want %d", sawOption, unix.IPV6_DONTFRAG)
	}

	if err := setTCPMSS(10, 1400); err != nil {
		t.Fatalf("setTCPMSS returned error: %v", err)
	}
	if sawOption != unix.TCP_MAXSEG {
		t.Fatalf("setTCPMSS used opt %d, want %d", sawOption, unix.TCP_MAXSEG)
	}

	darwinGetsockoptInt = func(fd, level, opt int) (int, error) {
		return 1412, nil
	}
	mss, err := getTCPMSS(conn)
	if err != nil {
		t.Fatalf("getTCPMSS returned error: %v", err)
	}
	if mss != 1412 {
		t.Fatalf("getTCPMSS returned %d, want 1412", mss)
	}

	darwinGetsockoptTCPConnectionInfo = func(fd, level, opt int) (*unix.TCPConnectionInfo, error) {
		return &unix.TCPConnectionInfo{Options: tcpciOptTimestamps}, nil
	}
	enabled, err := tcpTimestampsEnabled(conn)
	if err != nil {
		t.Fatalf("tcpTimestampsEnabled returned error: %v", err)
	}
	if !enabled {
		t.Fatal("expected injected TCP timestamp option to be reported as enabled")
	}
}
