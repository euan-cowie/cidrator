package dns

import (
	"github.com/spf13/cobra"
)

// DNSCmd represents the dns command
var DNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "DNS analysis and lookup tools",
	Long: `DNS subcommand provides DNS analysis and lookup tools.

Available commands:
  lookup  - Perform DNS lookups (A, AAAA, MX, TXT, CNAME, NS)
  reverse - Reverse DNS lookups (PTR records) for IP addresses

Examples:
  cidrator dns lookup example.com
  cidrator dns lookup example.com --type MX --format json
  cidrator dns reverse 8.8.8.8`,
}
