package cidr

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/euan-cowie/cidrator/internal/cidr"
	"github.com/spf13/cobra"
)

var config = NewGlobalConfig()

// explainCmd represents the explain command
var explainCmd = &cobra.Command{
	Use:   "explain <CIDR>",
	Short: "Explain and show detailed information about a CIDR range",
	Long: `Explain shows comprehensive information about a CIDR range including:
- Base and broadcast addresses
- Usable address range  
- Number of total and usable addresses
- Network mask and host mask
- Prefix length and host bits

Works with both IPv4 and IPv6 CIDR ranges.

Output formats:
- table (default): Human-readable table format
- json: JSON format for programmatic use
- yaml: YAML format for configuration files`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Explain.Validate(); err != nil {
			return err
		}

		cidrStr := args[0]
		info, err := cidr.ParseCIDR(cidrStr)
		if err != nil {
			return fmt.Errorf("failed to parse CIDR: %v", err)
		}

		return generateOutput(info, config.Explain)
	},
}

// generateOutput produces output in the specified format
func generateOutput(info *cidr.NetworkInfo, cfg *ExplainConfig) error {
	switch cfg.OutputFormat {
	case "json":
		output, err := info.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON: %v", err)
		}
		fmt.Println(output)
	case "yaml":
		output, err := info.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to generate YAML: %v", err)
		}
		fmt.Print(output) // YAML includes trailing newline
	case "table":
		printTableFormat(info)
	default:
		return fmt.Errorf("unsupported output format: %s", cfg.OutputFormat)
	}
	return nil
}

func printTableFormat(info *cidr.NetworkInfo) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	defer w.Flush()

	fmt.Fprintf(w, "Property\tValue\n")
	fmt.Fprintf(w, "--------\t-----\n")
	fmt.Fprintf(w, "Base Address\t%s\n", info.BaseAddress)

	printUsableAddressRange(w, info)
	printBroadcastAddress(w, info)

	fmt.Fprintf(w, "Total Addresses\t%s\n", cidr.FormatBigInt(info.TotalAddresses))
	fmt.Fprintf(w, "Network Mask\t%s (/%d bits)\n", info.Netmask, info.PrefixLength)

	if !info.IsIPv6 {
		fmt.Fprintf(w, "Host Mask\t%s\n", info.HostMask)
	}

	fmt.Fprintf(w, "Prefix Length\t/%d\n", info.PrefixLength)
	fmt.Fprintf(w, "Host Bits\t%d\n", info.HostBits)
	fmt.Fprintf(w, "IPv6\t%t\n", info.IsIPv6)
}

// printUsableAddressRange prints the usable address range based on network type and host bits
func printUsableAddressRange(w *tabwriter.Writer, info *cidr.NetworkInfo) {
	if info.IsIPv6 {
		printIPv6UsableRange(w, info)
		return
	}
	printIPv4UsableRange(w, info)
}

// printIPv4UsableRange prints IPv4 usable address range
func printIPv4UsableRange(w *tabwriter.Writer, info *cidr.NetworkInfo) {
	if info.HostBits <= 1 {
		fmt.Fprintf(w, "Usable Address Range\t%s (%s)\n",
			info.FirstUsable, cidr.FormatBigInt(info.UsableAddresses))
		return
	}

	fmt.Fprintf(w, "Usable Address Range\t%s to %s (%s)\n",
		info.FirstUsable, info.LastUsable, cidr.FormatBigInt(info.UsableAddresses))
}

// printIPv6UsableRange prints IPv6 usable address range
func printIPv6UsableRange(w *tabwriter.Writer, info *cidr.NetworkInfo) {
	if info.HostBits == 0 {
		fmt.Fprintf(w, "Usable Address Range\t%s (%s)\n",
			info.FirstUsable, cidr.FormatBigInt(info.UsableAddresses))
		return
	}

	fmt.Fprintf(w, "Usable Address Range\t%s to %s (%s)\n",
		info.FirstUsable, info.LastUsable, cidr.FormatBigInt(info.UsableAddresses))
}

// printBroadcastAddress prints broadcast address for IPv4 networks if applicable
func printBroadcastAddress(w *tabwriter.Writer, info *cidr.NetworkInfo) {
	if info.IsIPv6 || info.HostBits <= 1 {
		return
	}
	fmt.Fprintf(w, "Broadcast Address\t%s\n", info.BroadcastAddr)
}

func init() {
	CidrCmd.AddCommand(explainCmd)

	// Add output format flag
	explainCmd.Flags().StringVarP(&config.Explain.OutputFormat, "format", "f", "table", "Output format (table, json, yaml)")
}
