//go:build darwin

package mtu

import (
	"golang.org/x/net/route"
	"golang.org/x/sys/unix"
)

// Platform-specific interface type mappings for Darwin/macOS
var ifTypeMap = map[int]string{
	unix.IFT_ETHER:  "ethernet",
	unix.IFT_LOOP:   "loopback",
	unix.IFT_BRIDGE: "bridge",
	unix.IFT_PPP:    "ppp",
	unix.IFT_L2VLAN: "vlan",
	unix.IFT_GIF:    "tunnel",
	unix.IFT_STF:    "tunnel",
	unix.IFT_OTHER:  "virtual",
	// Darwin-specific constants
	0x47: "wifi",   // IFT_IEEE80211
	0xf9: "tunnel", // IFT_UTUN
}

// getInterfaceTypeFromOS gets interface type using BSD route information (Darwin/macOS specific)
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
