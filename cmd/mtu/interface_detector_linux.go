//go:build linux

package mtu

import (
	"os"
	"path/filepath"
	"strings"
)

// getInterfaceTypeFromOS gets interface type using Linux sysfs (Linux-specific)
func getInterfaceTypeFromOS(ifName string) (string, bool) {
	// Try to read interface type from sysfs
	typePath := filepath.Join("/sys/class/net", ifName, "type")
	if !strings.HasPrefix(typePath, "/sys/class/net/") {
		return "", false
	}

	typeData, err := os.ReadFile(typePath)
	if err != nil {
		return "", false
	}

	typeStr := strings.TrimSpace(string(typeData))

	// Linux uses ARPHRD_* constants, which map differently than IFT_* constants
	// Common mappings for Linux networking types
	switch typeStr {
	case "1": // ARPHRD_ETHER
		return "ethernet", true
	case "772": // ARPHRD_LOOPBACK
		return "loopback", true
	case "801": // ARPHRD_IEEE80211
		return "wifi", true
	case "768": // ARPHRD_TUNNEL
		return "tunnel", true
	case "512": // ARPHRD_PPP
		return "ppp", true
	case "774": // ARPHRD_IEEE80211_PRISM
		return "wifi", true
	case "776": // ARPHRD_IEEE80211_RADIOTAP
		return "wifi", true
	default:
		return "unknown", true
	}
}
