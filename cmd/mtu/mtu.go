package mtu

import (
	"github.com/spf13/cobra"
)

// MTUCmd represents the mtu command
var MTUCmd = &cobra.Command{
	Use:   "mtu",
	Short: "Path-MTU discovery & MTU toolbox",
	Long: `Path-MTU discovery and related sizing tools.

The mtu command group covers one-off discovery, continuous monitoring, local
interface inspection, payload sizing recommendations, and advanced peer-assisted
verification for controlled environments.`,
}

func init() {
	// Add subcommands
	MTUCmd.AddCommand(discoverCmd)
	MTUCmd.AddCommand(watchCmd)
	MTUCmd.AddCommand(interfacesCmd)
	MTUCmd.AddCommand(suggestCmd)
	MTUCmd.AddCommand(peerCmd)

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
	MTUCmd.PersistentFlags().Bool("quiet", false, "Suppress informational output")
	MTUCmd.PersistentFlags().Int("pps", 10, "Rate limit probes per second")
	MTUCmd.PersistentFlags().Bool("hops", false, "Enable hop-by-hop MTU discovery (similar to tracepath)")
	MTUCmd.PersistentFlags().Int("max-hops", 30, "Maximum hops for hop-by-hop discovery")
	MTUCmd.PersistentFlags().Int("port", 0, "Target port for TCP/UDP probes (0 = default)")
	MTUCmd.PersistentFlags().Bool("plpmtud", false, "Enable PLPMTUD fallback for black-hole detection (RFC 4821)")
	MTUCmd.PersistentFlags().Int("plp-port", 443, "Port for PLPMTUD probes")
}
