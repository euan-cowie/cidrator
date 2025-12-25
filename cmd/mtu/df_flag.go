package mtu

import (
	"fmt"
	"net"
)

// setDontFragment sets the DF (Don't Fragment) flag on a connection.
// This is a platform-agnostic wrapper that calls the appropriate
// IPv4 or IPv6 DF flag setting function based on the ipv6 parameter.
func setDontFragment(conn net.Conn, ipv6 bool) error {
	if ipv6 {
		return setIPv6DontFragment(conn)
	}
	return setIPv4DontFragment(conn)
}

// setDontFragmentUDP sets the DF flag on a UDP connection.
// UDPConn needs special handling as it's not a generic net.Conn for some platforms.
func setDontFragmentUDP(conn *net.UDPConn, ipv6 bool) error {
	// net.UDPConn implements net.Conn, so we can use the generic function
	return setDontFragment(conn, ipv6)
}

// setDontFragmentTCP sets the DF flag on a TCP connection.
// TCPConn needs special handling as it's not a generic net.Conn for some platforms.
func setDontFragmentTCP(conn *net.TCPConn, ipv6 bool) error {
	// net.TCPConn implements net.Conn, so we can use the generic function
	return setDontFragment(conn, ipv6)
}

// validateDFSupport checks if the platform supports DF flag setting
func validateDFSupport() error {
	// This is a placeholder for future platform-specific validation
	// Currently all supported platforms (darwin, linux) support DF flags
	return nil
}

// DFError represents an error when setting the DF flag
type DFError struct {
	Protocol string
	IPv6     bool
	Err      error
}

func (e *DFError) Error() string {
	version := "IPv4"
	if e.IPv6 {
		version = "IPv6"
	}
	return fmt.Sprintf("failed to set DF flag for %s %s: %v", e.Protocol, version, e.Err)
}

func (e *DFError) Unwrap() error {
	return e.Err
}
