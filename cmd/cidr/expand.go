package cidr

import (
	"fmt"
	"strings"

	"github.com/euan-cowie/cidrator/internal/cidr"
	"github.com/spf13/cobra"
)

// expandCmd represents the expand command
var expandCmd = &cobra.Command{
	Use:   "expand <CIDR>",
	Short: "List all IP addresses in a CIDR range",
	Long: `Expand lists all individual IP addresses contained within the specified CIDR range.

Examples:
  cidrator cidr expand 192.168.1.0/30
  cidrator cidr expand 10.0.0.0/29 --limit 10
  cidrator cidr expand 192.168.1.0/28 --one-line

Warning: Large CIDR ranges can produce many addresses. Use --limit to restrict output.
The command automatically prevents expansion of ranges larger than /16 (65,536 addresses).`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Expand.Validate(); err != nil {
			return err
		}

		cidrStr := args[0]
		opts := cidr.ExpansionOptions{
			Limit: config.Expand.Limit,
		}
		
		ips, err := cidr.Expand(cidrStr, opts)
		if err != nil {
			return fmt.Errorf("failed to expand CIDR: %v", err)
		}

		return outputExpandedIPs(ips, config.Expand)
	},
}

// outputExpandedIPs formats and outputs the expanded IP list
func outputExpandedIPs(ips []string, cfg *ExpandConfig) error {
	if cfg.OneLine {
		fmt.Println(strings.Join(ips, ", "))
		return nil
	}
	
	for _, ip := range ips {
		fmt.Println(ip)
	}
	return nil
}

func init() {
	CidrCmd.AddCommand(expandCmd)

	// Add flags
	expandCmd.Flags().IntVarP(&config.Expand.Limit, "limit", "l", 0, "Maximum number of IPs to expand (0 = no limit, subject to safety limits)")
	expandCmd.Flags().BoolVarP(&config.Expand.OneLine, "one-line", "o", false, "Output all IPs on one line, comma-separated")
}
