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
