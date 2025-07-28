//go:build darwin

package mtu

import (
	"fmt"
	"net"
	"runtime"

	"golang.org/x/sys/unix"
)

// setIPv4DontFragment sets DF flag for IPv4 on Darwin
func setIPv4DontFragment(conn net.Conn) error {
	switch conn := conn.(type) {
	case *net.IPConn:
		rawConn, err := conn.SyscallConn()
		if err != nil {
			return fmt.Errorf("failed to get syscall conn: %w", err)
		}

		var sockErr error
		err = rawConn.Control(func(f uintptr) {
			fd := int(f)
			// Darwin uses IP_DONTFRAG
			sockErr = unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_DONTFRAG, 1)
			if sockErr == nil {
				fmt.Printf("✅ Successfully set DF flag on %s\n", runtime.GOOS)
			}
		})
		if err != nil {
			return fmt.Errorf("failed to control raw conn: %w", err)
		}
		return sockErr
	default:
		return fmt.Errorf("unsupported connection type: %T", conn)
	}
}

// setIPv6DontFragment sets DF flag for IPv6 on Darwin
func setIPv6DontFragment(conn net.Conn) error {
	switch conn := conn.(type) {
	case *net.IPConn:
		rawConn, err := conn.SyscallConn()
		if err != nil {
			return fmt.Errorf("failed to get syscall conn: %w", err)
		}

		var sockErr error
		err = rawConn.Control(func(f uintptr) {
			fd := int(f)
			sockErr = unix.SetsockoptInt(fd, unix.IPPROTO_IPV6, unix.IPV6_DONTFRAG, 1)
			if sockErr == nil {
				fmt.Printf("✅ Successfully set IPv6 DF flag\n")
			}
		})
		if err != nil {
			return fmt.Errorf("failed to control raw conn: %w", err)
		}
		return sockErr
	default:
		return fmt.Errorf("unsupported connection type: %T", conn)
	}
}
