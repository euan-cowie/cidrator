package mtu

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
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

var ifTypeMap = map[int]string{
	unix.IFT_ETHER: "ethernet",
	// unix.IFT_IEEE80211: "wifi",
	unix.IFT_LOOP:   "loopback",
	unix.IFT_BRIDGE: "bridge",
	unix.IFT_PPP:    "ppp",
	unix.IFT_L2VLAN: "vlan",
	unix.IFT_GIF:    "tunnel",
	unix.IFT_STF:    "tunnel",
	// unix.IFT_UTUN:      "tunnel",
	unix.IFT_OTHER: "virtual",
	// Manually add the Apple Wi-Fi and utun constants:
	0x47: "wifi",   // IFT_IEEE80211
	0xf9: "tunnel", // IFT_UTUN
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

func getInterfaceTypeFromOS(ifName string) (string, bool) {
	rib, err := route.FetchRIB(0, route.RIBTypeInterface, 0)
	if err != nil {
		return "", false
	}
	msgs, err := route.ParseRIB(route.RIBTypeInterface, rib)
	if err != nil {
		return "", false
	}
	for _, m := range msgs {
		imsg, ok := m.(*route.InterfaceMessage)
		if !ok || imsg.Name != ifName {
			continue
		}
		for _, sys := range imsg.Sys() {
			if imx, ok := sys.(*route.InterfaceMetrics); ok {
				if s, exists := ifTypeMap[imx.Type]; exists {
					return s, true
				}
				return "unknown", true
			}
		}
	}
	return "", false
}

// determineInterfaceType determines the type of network interface
func determineInterfaceType(name string, flags net.Flags) string {
	name = strings.ToLower(name)

	if t, ok := getInterfaceTypeFromOS(name); ok { // <- build‑tagged helpers
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

// getDarwinMTU gets MTU on macOS
func getDarwinMTU(interfaceName string) (int, error) {
	// Open a dummy datagram socket; required for the ioctl.
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return 0, fmt.Errorf("socket: %w", err)
	}
	defer func() {
		_ = unix.Close(fd) // Explicitly ignore close error
	}()

	ifr, err := unix.IoctlGetIfreqMTU(fd, interfaceName) // <- libSystem wrapper
	if err != nil {
		return 0, fmt.Errorf("ioctl SIOCGIFMTU: %w", err)
	}
	return int(ifr.MTU), nil
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
