package scan

import (
	"github.com/spf13/cobra"
)

// ScanCmd represents the scan command
var ScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Network scanning and discovery tools",
	Long: `Scan subcommand provides network scanning and discovery tools.

Planned features:
- port: Port scanning for hosts and ranges
- ping: ICMP ping sweeps across networks  
- arp: ARP table scanning and discovery
- host: Host discovery and OS fingerprinting
- service: Service detection and enumeration

This is a scaffold for future scanning functionality.`,
}
