//go:build darwin

package mtu

import (
	"golang.org/x/net/route"
)

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
