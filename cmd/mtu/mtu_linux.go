//go:build linux

package mtu

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// getMTU reads MTU from sysfs for the specified interface.
// This is the Linux-specific implementation using /sys/class/net/{iface}/mtu.
func getMTU(iface string) (int, error) {
	path := fmt.Sprintf("/sys/class/net/%s/mtu", iface)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, fmt.Errorf("interface %s not found", iface)
		}
		return 0, fmt.Errorf("failed to read MTU for %s: %w", iface, err)
	}

	mtu, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid MTU value for %s: %w", iface, err)
	}

	return mtu, nil
}
