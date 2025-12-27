//go:build !linux && !darwin

package mtu

import (
	"fmt"
	"net"
)

// setIPv4DontFragment is a stub for unsupported platforms
func setIPv4DontFragment(conn net.Conn) error {
	return fmt.Errorf("platform not supported")
}

// setIPv6DontFragment is a stub for unsupported platforms
func setIPv6DontFragment(conn net.Conn) error {
	return fmt.Errorf("platform not supported")
}

// setTCPMSS is a stub for unsupported platforms
func setTCPMSS(fd uintptr, mss int) error {
	return nil // No-op on unsupported platforms
}

// getTCPMSS is a stub for unsupported platforms
func getTCPMSS(conn net.Conn) (int, error) {
	return 0, nil // Return 0 to skip validation on unsupported platforms
}
