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
