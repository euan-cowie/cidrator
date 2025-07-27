//go:build linux

package mtu

// Platform-specific interface type mappings for Linux
// Using standard values since unix.IFT_* constants aren't available on Linux
var ifTypeMap = map[int]string{
	0x6:  "ethernet", // IFT_ETHER
	0x18: "loopback", // IFT_LOOP
	0xd1: "bridge",   // IFT_BRIDGE
	0x17: "ppp",      // IFT_PPP
	0x87: "vlan",     // IFT_L2VLAN
	0x37: "tunnel",   // IFT_GIF
	0x39: "tunnel",   // IFT_STF
	0x1:  "virtual",  // IFT_OTHER
	0x47: "wifi",     // IFT_IEEE80211
	0xf9: "tunnel",   // IFT_UTUN
}
