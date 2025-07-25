package cidr

import (
	"fmt"
	"strconv"

	"github.com/euan-cowie/cidrator/internal/cidr"
	"github.com/spf13/cobra"
)

// divideCmd represents the divide command
var divideCmd = &cobra.Command{
	Use:   "divide <CIDR> <N>",
	Short: "Divide a CIDR range into N smaller subnets",
	Long: `Divide splits a CIDR range into the specified number of smaller, equally-sized subnets.

Examples:
  cidrator cidr divide 10.0.0.0/16 4
  cidrator cidr divide 2001:db8:1111:2222:1::/80 8
  cidrator cidr divide 192.168.0.0/24 2

The command calculates the appropriate subnet mask and returns the list of subnets.
Note: N must be a power of 2 or the subnets will not utilize the full address space.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cidrStr := args[0]
		nStr := args[1]

		n, err := strconv.Atoi(nStr)
		if err != nil {
			return fmt.Errorf("invalid number of parts: %v", err)
		}

		if n <= 0 {
			return fmt.Errorf("number of parts must be greater than 0")
		}

		opts := cidr.DivisionOptions{
			Parts: n,
		}
		
		subnets, err := cidr.Divide(cidrStr, opts)
		if err != nil {
			return fmt.Errorf("failed to divide CIDR: %v", err)
		}

		for _, subnet := range subnets {
			fmt.Println(subnet)
		}

		return nil
	},
}

func init() {
	CidrCmd.AddCommand(divideCmd)
}
