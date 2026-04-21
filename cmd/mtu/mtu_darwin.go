//go:build darwin

package mtu

import (
	"fmt"

	"golang.org/x/sys/unix"
)

var openDarwinMTUSocket = func(domain, typ, proto int) (int, error) {
	return unix.Socket(domain, typ, proto)
}

var closeDarwinFD = func(fd int) error {
	return unix.Close(fd)
}

var ioctlGetDarwinIfreqMTU = func(fd int, interfaceName string) (*unix.IfreqMTU, error) {
	return unix.IoctlGetIfreqMTU(fd, interfaceName)
}

// getDarwinMTU gets MTU on macOS using ioctl
func getMTU(interfaceName string) (int, error) {
	// Open a dummy datagram socket; required for the ioctl.
	fd, err := openDarwinMTUSocket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return 0, fmt.Errorf("socket: %w", err)
	}
	defer func() {
		_ = closeDarwinFD(fd) // Explicitly ignore close error
	}()

	ifr, err := ioctlGetDarwinIfreqMTU(fd, interfaceName)
	if err != nil {
		return 0, fmt.Errorf("ioctl SIOCGIFMTU: %w", err)
	}
	return int(ifr.MTU), nil
}
