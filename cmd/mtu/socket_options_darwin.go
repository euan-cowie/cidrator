//go:build darwin

package mtu

import (
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/unix"
)

// setIPv4DontFragment sets DF flag for IPv4 on Darwin
func setIPv4DontFragment(conn net.Conn) error {
	var rawConn syscall.RawConn
	var err error

	switch c := conn.(type) {
	case *net.IPConn:
		rawConn, err = c.SyscallConn()
	case *net.UDPConn:
		rawConn, err = c.SyscallConn()
	case *net.TCPConn:
		rawConn, err = c.SyscallConn()
	default:
		return fmt.Errorf("unsupported connection type for DF flag: %T", conn)
	}

	if err != nil {
		return fmt.Errorf("failed to get syscall conn: %w", err)
	}

	var sockErr error
	err = rawConn.Control(func(f uintptr) {
		fd := int(f)
		// Darwin uses IP_DONTFRAG
		sockErr = unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_DONTFRAG, 1)
	})
	if err != nil {
		return fmt.Errorf("failed to control raw conn: %w", err)
	}
	return sockErr
}

// setIPv6DontFragment sets DF flag for IPv6 on Darwin
func setIPv6DontFragment(conn net.Conn) error {
	var rawConn syscall.RawConn
	var err error

	switch c := conn.(type) {
	case *net.IPConn:
		rawConn, err = c.SyscallConn()
	case *net.UDPConn:
		rawConn, err = c.SyscallConn()
	case *net.TCPConn:
		rawConn, err = c.SyscallConn()
	default:
		return fmt.Errorf("unsupported connection type for DF flag: %T", conn)
	}

	if err != nil {
		return fmt.Errorf("failed to get syscall conn: %w", err)
	}

	var sockErr error
	err = rawConn.Control(func(f uintptr) {
		fd := int(f)
		sockErr = unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_DONTFRAG, 1)
	})
	if err != nil {
		return fmt.Errorf("failed to control raw conn: %w", err)
	}
	return sockErr
}

// setTCPMSS forces the kernel to cap the segment size for this socket.
// This helps bypass TSO/GSO by forcing the stack to packetize at this specific size.
func setTCPMSS(fd uintptr, mss int) error {
	return unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG, mss)
}

// getTCPMSS retrieves the current effective MSS for the connection.
// This allows us to detect if the kernel negotiated a smaller MSS than our probe size.
func getTCPMSS(conn net.Conn) (int, error) {
	var rawConn syscall.RawConn
	var err error

	switch c := conn.(type) {
	case *net.TCPConn:
		rawConn, err = c.SyscallConn()
	default:
		return 0, fmt.Errorf("unsupported connection type for TCP MSS: %T", conn)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to get syscall conn: %w", err)
	}

	var mss int
	var sockErr error
	err = rawConn.Control(func(f uintptr) {
		fd := int(f)
		mss, sockErr = unix.GetsockoptInt(fd, unix.IPPROTO_TCP, unix.TCP_MAXSEG)
	})

	if err != nil {
		return 0, fmt.Errorf("failed to control raw conn: %w", err)
	}
	if sockErr != nil {
		return 0, sockErr
	}
	return mss, nil
}
