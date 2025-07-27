//go:build darwin

package mtu

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// getDarwinMTU gets MTU on macOS using ioctl
func getDarwinMTU(interfaceName string) (int, error) {
	// Open a dummy datagram socket; required for the ioctl.
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return 0, fmt.Errorf("socket: %w", err)
	}
	defer func() {
		_ = unix.Close(fd) // Explicitly ignore close error
	}()

	ifr, err := unix.IoctlGetIfreqMTU(fd, interfaceName)
	if err != nil {
		return 0, fmt.Errorf("ioctl SIOCGIFMTU: %w", err)
	}
	return int(ifr.MTU), nil
}
