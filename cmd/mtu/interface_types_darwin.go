//go:build darwin

package mtu

import "golang.org/x/sys/unix"

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
