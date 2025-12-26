package dns

import (
	"fmt"
	"time"

	"github.com/euan-cowie/cidrator/internal/dns"
	"github.com/spf13/cobra"
)

// reverseCmd represents the dns reverse command
var reverseCmd = &cobra.Command{
	Use:   "reverse <ip>",
	Short: "Perform reverse DNS lookups (PTR records)",
	Long: `Reverse performs reverse DNS lookups for IP addresses.

Returns the hostnames associated with the given IP address via PTR records.

Examples:
  cidrator dns reverse 8.8.8.8
  cidrator dns reverse 2001:4860:4860::8888
  cidrator dns reverse 8.8.8.8 --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runReverse,
}

func init() {
	DNSCmd.AddCommand(reverseCmd)

	// Add flags for reverse lookup
	reverseCmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")
	reverseCmd.Flags().DurationP("timeout", "", 5*time.Second, "Query timeout")
}

func runReverse(cmd *cobra.Command, args []string) error {
	ip := args[0]

	// Get flags
	format, _ := cmd.Flags().GetString("format")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// Perform reverse lookup
	result, err := dns.ReverseLookup(ip, timeout)
	if err != nil {
		return err
	}

	// Output result
	return outputReverseResult(result, format)
}

func outputReverseResult(result *dns.ReverseResult, format string) error {
	switch format {
	case "json":
		output, err := result.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON: %v", err)
		}
		fmt.Println(output)
	case "yaml":
		output, err := result.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to generate YAML: %v", err)
		}
		fmt.Print(output)
	case "table":
		outputReverseTable(result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
	return nil
}

func outputReverseTable(result *dns.ReverseResult) {
	fmt.Printf("IP: %s\n", result.IP)
	fmt.Printf("Query Time: %v\n\n", result.QueryTime.Round(time.Millisecond))

	if len(result.Hostnames) == 0 {
		fmt.Println("No PTR records found.")
		return
	}

	fmt.Println("Hostnames:")
	for _, hostname := range result.Hostnames {
		fmt.Printf("  - %s\n", hostname)
	}
}
