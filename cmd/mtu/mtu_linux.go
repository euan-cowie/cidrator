//go:build linux

package mtu

import "fmt"

// getDarwinMTU is not applicable on Linux - this is a no-op
func getDarwinMTU(interfaceName string) (int, error) {
	return 0, fmt.Errorf("getDarwinMTU not supported on Linux")
}
