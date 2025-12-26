package mtu

import (
	"github.com/spf13/cobra"
)

// MTUCmd represents the mtu command
var MTUCmd = &cobra.Command{
	Use:   "mtu",
	Short: "Path-MTU discovery & MTU toolbox",
	Long: `MTU subcommand provides Path-MTU discovery and MTU analysis tools.

The mtu sub-command is a smart wrapper around the techniques in RFC 1191 (IPv4),
RFC 8201 (IPv6) and RFC 4821 (PLPMTUD). It answers three everyday questions:
• What MTU can I safely send to that host?
• Did today's change introduce an MTU black-hole?
• What MSS or VPN segment size should I configure?

Available operations:
- discover: Binary-search to the largest size that gets through (default)
- watch: Re-run discover every N seconds and notify on change
- interfaces: List local interfaces + configured MTU
- suggest: Print TCP MSS / IPSec ESP / WireGuard frame sizes for the path

All commands support both IPv4 and IPv6 with multiple probe protocols.`,
}

func init() {
	// Add subcommands
	MTUCmd.AddCommand(discoverCmd)
	MTUCmd.AddCommand(watchCmd)
	MTUCmd.AddCommand(interfacesCmd)
	MTUCmd.AddCommand(suggestCmd)
	MTUCmd.AddCommand(serverCmd)

	// Global flags for MTU commands
	MTUCmd.PersistentFlags().Bool("4", false, "Force IPv4")
	MTUCmd.PersistentFlags().Bool("6", false, "Force IPv6")
	MTUCmd.PersistentFlags().String("proto", "icmp", "Probe method (icmp|udp|tcp)")
	MTUCmd.PersistentFlags().Int("min", 0, "Lower bound (IPv4 default: 576, IPv6: 1280)")
	MTUCmd.PersistentFlags().Int("max", 9216, "Upper bound")
	MTUCmd.PersistentFlags().Int("step", 0, "Granularity for linear sweep mode (0 = binary search)")
	MTUCmd.PersistentFlags().Duration("timeout", 0, "Wait per probe (default: 2s)")
	MTUCmd.PersistentFlags().Int("ttl", 64, "Initial hop limit")
	MTUCmd.PersistentFlags().Bool("json", false, "Structured output")
	MTUCmd.PersistentFlags().Bool("quiet", false, "Suppress progress bar")
	MTUCmd.PersistentFlags().Int("pps", 10, "Rate limit probes per second")
	MTUCmd.PersistentFlags().Bool("hops", false, "Enable hop-by-hop MTU discovery (similar to tracepath)")
	MTUCmd.PersistentFlags().Int("max-hops", 30, "Maximum hops for hop-by-hop discovery")
	MTUCmd.PersistentFlags().Int("port", 0, "Target port for TCP/UDP probes (0 = default)")
	MTUCmd.PersistentFlags().Bool("plpmtud", false, "Enable PLPMTUD fallback for black-hole detection (RFC 4821)")
	MTUCmd.PersistentFlags().Int("plp-port", 443, "Port for PLPMTUD probes")
}
