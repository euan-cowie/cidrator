//go:build darwin

package mtu

import (
	"errors"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestGetMTUOnDarwinWithInjectedSyscalls(t *testing.T) {
	originalOpen := openDarwinMTUSocket
	originalClose := closeDarwinFD
	originalIoctl := ioctlGetDarwinIfreqMTU
	t.Cleanup(func() {
		openDarwinMTUSocket = originalOpen
		closeDarwinFD = originalClose
		ioctlGetDarwinIfreqMTU = originalIoctl
	})

	t.Run("propagates socket failures", func(t *testing.T) {
		openDarwinMTUSocket = func(domain, typ, proto int) (int, error) {
			return 0, errors.New("socket failed")
		}
		mtu, err := getMTU("en0")
		if err == nil || !strings.Contains(err.Error(), "socket: socket failed") {
			t.Fatalf("expected socket error, got mtu=%d err=%v", mtu, err)
		}
	})

	t.Run("propagates ioctl failures", func(t *testing.T) {
		openDarwinMTUSocket = func(domain, typ, proto int) (int, error) {
			return 7, nil
		}
		ioctlGetDarwinIfreqMTU = func(fd int, interfaceName string) (*unix.IfreqMTU, error) {
			return nil, errors.New("ioctl failed")
		}
		mtu, err := getMTU("en0")
		if err == nil || !strings.Contains(err.Error(), "ioctl SIOCGIFMTU: ioctl failed") {
			t.Fatalf("expected ioctl error, got mtu=%d err=%v", mtu, err)
		}
	})

	t.Run("returns mtu from ioctl result", func(t *testing.T) {
		openDarwinMTUSocket = func(domain, typ, proto int) (int, error) {
			return 9, nil
		}
		ioctlGetDarwinIfreqMTU = func(fd int, interfaceName string) (*unix.IfreqMTU, error) {
			return &unix.IfreqMTU{MTU: 1500}, nil
		}
		if mtu, err := getMTU("en0"); err != nil || mtu != 1500 {
			t.Fatalf("expected mtu=1500, got mtu=%d err=%v", mtu, err)
		}
	})
}
