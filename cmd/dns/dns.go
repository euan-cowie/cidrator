package dns

import (
	"github.com/spf13/cobra"
)

// DNSCmd represents the dns command
var DNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS analysis and lookup tools",
	Long: `Forward and reverse DNS lookup tools.

Use this command group to query common record types, direct queries to a
specific resolver, and inspect PTR records for IPv4 or IPv6 addresses.`,
}
