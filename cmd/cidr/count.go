package cidr

import (
	"fmt"

	"github.com/euan-cowie/cidrator/internal/cidr"
	"github.com/spf13/cobra"
)

// countCmd represents the count command
var countCmd = &cobra.Command{
	Use:   "count <CIDR>",
	Short: "Count the total number of addresses in a CIDR range",
	Long: `Count returns the total number of IP addresses available in the specified CIDR range.

Examples:
  cidrator cidr count 10.0.0.0/16
  cidrator cidr count 2001:db8:1234:1a00::/106
  cidrator cidr count 172.16.18.0/31

This includes all addresses (network, broadcast, and host addresses for IPv4).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cidrStr := args[0]

		count, err := cidr.Count(cidrStr)
		if err != nil {
			return fmt.Errorf("failed to count addresses: %v", err)
		}

		fmt.Println(count.String())
		return nil
	},
}

func init() {
	CidrCmd.AddCommand(countCmd)
}
