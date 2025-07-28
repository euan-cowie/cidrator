//go:build linux

package mtu

import (
	"fmt"
	"net"
	"runtime"
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
	switch conn := conn.(type) {
	case *net.IPConn:
		rawConn, err := conn.SyscallConn()
		if err != nil {
			return fmt.Errorf("failed to get syscall conn: %w", err)
		}

		var sockErr error
		err = rawConn.Control(func(f uintptr) {
			fd := int(f)
			// Linux uses IP_MTU_DISCOVER with IP_PMTUDISC_DO
			sockErr = syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, IP_MTU_DISCOVER, IP_PMTUDISC_DO)
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

// setIPv6DontFragment sets DF flag for IPv6 on Linux
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
			// Linux uses IPV6_MTU_DISCOVER
			sockErr = syscall.SetsockoptInt(fd, syscall.IPPROTO_IPV6, IPV6_MTU_DISCOVER, IPV6_PMTUDISC_DO)
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
