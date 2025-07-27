package mtu

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// NetworkInterface represents a network interface with MTU information
type NetworkInterface struct {
	Name string `json:"name"`
	MTU  int    `json:"mtu"`
	Type string `json:"type"`
}

// InterfaceResult represents the result of interface detection
type InterfaceResult struct {
	Interfaces []NetworkInterface `json:"interfaces"`
}

// getInterfaceTypeFromOS will be defined in platform-specific files

// GetNetworkInterfaces returns all network interfaces with their MTU values
func GetNetworkInterfaces() (*InterfaceResult, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("failed to get interfaces: %w", err)
	}

	var result []NetworkInterface

	for _, iface := range interfaces {
		// Skip interfaces that are down
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		interfaceType := determineInterfaceType(iface.Name, iface.Flags)

		// Get MTU - some platforms might need special handling
		mtu := iface.MTU
		if mtu <= 0 {
			// Fallback to platform-specific MTU detection
			if platformMTU, err := getPlatformSpecificMTU(iface.Name); err == nil {
				mtu = platformMTU
			}
		}

		result = append(result, NetworkInterface{
			Name: iface.Name,
			MTU:  mtu,
			Type: interfaceType,
		})
	}

	return &InterfaceResult{Interfaces: result}, nil
}

// determineInterfaceType determines the type of network interface
func determineInterfaceType(name string, flags net.Flags) string {
	name = strings.ToLower(name)

	// Try platform-specific interface type detection first
	if t, ok := getInterfaceTypeFromOS(name); ok {
		return t
	}

	// Check for loopback
	if flags&net.FlagLoopback != 0 {
		return "loopback"
	}

	// Common interface name patterns
	return "unknown"
}

// getPlatformSpecificMTU gets MTU using platform-specific methods
func getPlatformSpecificMTU(interfaceName string) (int, error) {
	switch runtime.GOOS {
	case "linux":
		return getLinuxMTU(interfaceName)
	case "darwin":
		return getDarwinMTU(interfaceName)
	default:
		return 0, fmt.Errorf("unsupported platform: %s (only Linux and Darwin are supported)", runtime.GOOS)
	}
}

// getLinuxMTU reads MTU from /sys/class/net/*/mtu
func getLinuxMTU(interfaceName string) (int, error) {
	// Sanitize path to prevent directory traversal
	path := filepath.Join("/sys/class/net/", interfaceName, "mtu")
	if !strings.HasPrefix(path, "/sys/class/net/") {
		return 0, fmt.Errorf("invalid interface name provided: %s", interfaceName)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	mtuStr := strings.TrimSpace(string(content))
	return strconv.Atoi(mtuStr)
}

// Platform-specific MTU functions are defined in mtu_*.go files

// GetMaxMTU returns the maximum MTU among all interfaces (useful for auto-setting --max)
func GetMaxMTU() (int, error) {
	result, err := GetNetworkInterfaces()
	if err != nil {
		return 0, err
	}

	maxMTU := 0
	for _, iface := range result.Interfaces {
		if iface.MTU > maxMTU {
			maxMTU = iface.MTU
		}
	}

	if maxMTU == 0 {
		return 1500, nil // Default fallback
	}

	return maxMTU, nil
}
