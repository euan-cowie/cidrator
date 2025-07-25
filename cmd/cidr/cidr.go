package cidr

import (
	"github.com/spf13/cobra"
)

// CidrCmd represents the cidr command
var CidrCmd = &cobra.Command{
	Use:   "cidr",
	Short: "CIDR network analysis and manipulation",
	Long: `CIDR subcommand provides comprehensive IPv4/IPv6 CIDR network analysis and manipulation tools.

Available operations:
- explain: Show detailed network information with multiple output formats
- expand: List all IP addresses in a CIDR range
- contains: Check if an IP address belongs to a CIDR range  
- count: Count total addresses in CIDR ranges
- overlaps: Check if two CIDR ranges overlap
- divide: Split CIDR ranges into smaller subnets

All commands support both IPv4 and IPv6 networks.`,
}
