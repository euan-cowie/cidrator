//go:build linux

package mtu

// getInterfaceTypeFromOS gets interface type using Linux sysfs (Linux-specific)
func getInterfaceTypeFromOS(_ string) (string, bool) {
	return "", false
}
