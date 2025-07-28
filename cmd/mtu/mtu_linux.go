//go:build linux

package mtu

import "fmt"

// getLinuxMTU reads MTU from /sys/class/net/*/mtu
func getMTU(_ string) (int, error) {
	return 0, fmt.Errorf("getLinuxMTU not supported yet")
}
