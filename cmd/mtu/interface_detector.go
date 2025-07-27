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

	// Check for loopback
	if flags&net.FlagLoopback != 0 {
		return "loopback"
	}

	// Common interface name patterns
	switch {
	case strings.HasPrefix(name, "lo"):
		return "loopback"
	case strings.HasPrefix(name, "eth"), strings.HasPrefix(name, "en"),
		strings.HasPrefix(name, "em"), strings.HasPrefix(name, "eno"):
		return "ethernet"
	case strings.HasPrefix(name, "wlan"), strings.HasPrefix(name, "wl"),
		strings.HasPrefix(name, "wifi"), strings.HasPrefix(name, "ath"):
		return "wireless"
	case strings.HasPrefix(name, "tun"), strings.HasPrefix(name, "tap"):
		return "tunnel"
	case strings.HasPrefix(name, "br"):
		return "bridge"
	case strings.HasPrefix(name, "docker"), strings.HasPrefix(name, "veth"):
		return "virtual"
	case strings.HasPrefix(name, "ppp"):
		return "ppp"
	case strings.HasPrefix(name, "bond"):
		return "bond"
	case strings.HasPrefix(name, "vlan"):
		return "vlan"
	default:
		return "unknown"
	}
}

// getPlatformSpecificMTU gets MTU using platform-specific methods
func getPlatformSpecificMTU(interfaceName string) (int, error) {
	switch runtime.GOOS {
	case "linux":
		return getLinuxMTU(interfaceName)
	case "darwin":
		return getDarwinMTU(interfaceName)
	case "windows":
		return getWindowsMTU(interfaceName)
	default:
		return 0, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
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

// getDarwinMTU uses ifconfig to get MTU on macOS
func getDarwinMTU(interfaceName string) (int, error) {
	// Try to parse from route table or use a system call
	// For now, return error to fall back to net.Interface.MTU
	return 0, fmt.Errorf("platform-specific MTU detection not implemented for macOS")
}

// getWindowsMTU gets MTU on Windows
func getWindowsMTU(interfaceName string) (int, error) {
	// Try to parse from netsh or WMI
	// For now, return error to fall back to net.Interface.MTU
	return 0, fmt.Errorf("platform-specific MTU detection not implemented for Windows")
}

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
