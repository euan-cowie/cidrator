package cidr

import (
	"fmt"

	"github.com/euan-cowie/cidrator/internal/cidr"
	"github.com/spf13/cobra"
)

// containsCmd represents the contains command
var containsCmd = &cobra.Command{
	Use:   "contains <CIDR> <IP>",
	Short: "Check if an IP address is contained within a CIDR range",
	Long: `Contains checks whether a given IP address falls within the specified CIDR range.

Examples:
  cidrator cidr contains 10.0.0.0/16 10.0.14.5
  cidrator cidr contains 2001:db8:1234:1a00::/106 2001:db8:1234:1a00::1

Returns 'true' if the IP is within the range, 'false' otherwise.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cidrStr := args[0]
		ipStr := args[1]

		contains, err := cidr.Contains(cidrStr, ipStr)
		if err != nil {
			return fmt.Errorf("failed to check containment: %v", err)
		}

		fmt.Println(contains)
		return nil
	},
}

func init() {
	CidrCmd.AddCommand(containsCmd)
}
