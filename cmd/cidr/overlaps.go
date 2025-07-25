package cidr

import (
	"fmt"

	"github.com/euan-cowie/cidrator/internal/cidr"
	"github.com/spf13/cobra"
)

// overlapsCmd represents the overlaps command
var overlapsCmd = &cobra.Command{
	Use:   "overlaps <CIDR1> <CIDR2>",
	Short: "Check if two CIDR ranges overlap",
	Long: `Overlaps checks whether two CIDR ranges have any IP addresses in common.

Examples:
  cidrator cidr overlaps 10.0.0.0/16 10.0.14.0/22
  cidrator cidr overlaps 2001:db8:1111:2222:1::/80 2001:db8:1111:2222:1:1::/96
  cidrator cidr overlaps 192.168.1.0/24 10.0.0.0/8

Returns 'true' if the ranges overlap, 'false' otherwise.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cidr1 := args[0]
		cidr2 := args[1]

		overlaps, err := cidr.Overlaps(cidr1, cidr2)
		if err != nil {
			return fmt.Errorf("failed to check overlap: %v", err)
		}

		fmt.Println(overlaps)
		return nil
	},
}

func init() {
	CidrCmd.AddCommand(overlapsCmd)
}
