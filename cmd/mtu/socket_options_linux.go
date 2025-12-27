//go:build linux

package mtu

import (
	"fmt"
	"net"
	"syscall"
)

// Linux constants for MTU discovery
const (
	IP_MTU_DISCOVER   = 10
	IP_PMTUDISC_DO    = 2
	IPV6_MTU_DISCOVER = 23
	IPV6_PMTUDISC_DO  = 2
)

// setIPv4DontFragment sets DF flag for IPv4 on Linux
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
		// Linux uses IP_MTU_DISCOVER with IP_PMTUDISC_DO
		sockErr = syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, IP_MTU_DISCOVER, IP_PMTUDISC_DO)
	})
	if err != nil {
		return fmt.Errorf("failed to control raw conn: %w", err)
	}
	return sockErr
}

// setIPv6DontFragment sets DF flag for IPv6 on Linux
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
		// Linux uses IPV6_MTU_DISCOVER
		sockErr = syscall.SetsockoptInt(fd, syscall.IPPROTO_IPV6, IPV6_MTU_DISCOVER, IPV6_PMTUDISC_DO)
	})
	if err != nil {
		return fmt.Errorf("failed to control raw conn: %w", err)
	}
	return sockErr
}

// setTCPMSS forces the kernel to cap the segment size for this socket.
// This helps bypass TSO/GSO by forcing the stack to packetize at this specific size.
func setTCPMSS(fd uintptr, mss int) error {
	// TCP_MAXSEG is option 2 on most *nix systems
	return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_MAXSEG, mss)
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
		// On Linux, reading TCP_MAXSEG returns the current effective MSS
		mss, sockErr = syscall.GetsockoptInt(fd, syscall.IPPROTO_TCP, syscall.TCP_MAXSEG)
	})

	if err != nil {
		return 0, fmt.Errorf("failed to control raw conn: %w", err)
	}
	if sockErr != nil {
		return 0, sockErr
	}
	return mss, nil
}
