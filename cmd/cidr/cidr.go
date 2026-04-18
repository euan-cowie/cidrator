package cidr

import (
	"github.com/spf13/cobra"
)

// CidrCmd represents the cidr command
var CidrCmd = &cobra.Command{
	Use:   "cidr",
	Short: "CIDR network analysis and manipulation",
	Long: `Inspect and manipulate IPv4 or IPv6 CIDR ranges.

The cidr command group covers explanation, expansion, containment checks,
counting, overlap detection, and subnet division.`,
}
