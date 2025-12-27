package cidr

import (
	"context"
	"fmt"

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

Use --limit to restrict output for large ranges.
Streaming output uses constant memory regardless of range size.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Expand.Validate(); err != nil {
			return err
		}

		cidrStr := args[0]
		opts := cidr.ExpansionOptions{
			Limit: config.Expand.Limit,
		}

		return streamExpandedIPs(cmd.Context(), cidrStr, opts, config.Expand)
	},
}

// streamExpandedIPs streams and outputs the expanded IP list
func streamExpandedIPs(ctx context.Context, cidrStr string, opts cidr.ExpansionOptions, cfg *ExpandConfig) error {
	results := cidr.Expand(ctx, cidrStr, opts)

	if cfg.OneLine {
		// Stream one-line output directly to stdout (constant memory)
		first := true
		for result := range results {
			if result.Err != nil {
				return fmt.Errorf("failed to expand CIDR: %v", result.Err)
			}
			if !first {
				fmt.Print(", ")
			}
			fmt.Print(result.IP)
			first = false
		}
		fmt.Println() // Final newline
		return nil
	}

	// Stream directly to stdout for constant memory
	for result := range results {
		if result.Err != nil {
			return fmt.Errorf("failed to expand CIDR: %v", result.Err)
		}
		fmt.Println(result.IP)
	}
	return nil
}

func init() {
	CidrCmd.AddCommand(expandCmd)

	// Add flags
	expandCmd.Flags().IntVarP(&config.Expand.Limit, "limit", "l", 0, "Maximum number of IPs to expand (0 = no limit)")
	expandCmd.Flags().BoolVarP(&config.Expand.OneLine, "one-line", "o", false, "Output all IPs on one line, comma-separated")
}
