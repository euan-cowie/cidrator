package dns

import (
	"fmt"
	"io"
	"time"

	"github.com/euan-cowie/cidrator/internal/dns"
	"github.com/spf13/cobra"
)

var dnsReverseLookup = dns.ReverseLookup

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
	result, err := dnsReverseLookup(ip, timeout)
	if err != nil {
		return err
	}

	// Output result
	return outputReverseResult(cmd.OutOrStdout(), result, format)
}

func outputReverseResult(w io.Writer, result *dns.ReverseResult, format string) error {
	switch format {
	case "json":
		output, err := result.ToJSON()
		if err != nil {
			return fmt.Errorf("failed to generate JSON: %v", err)
		}
		_, _ = fmt.Fprintln(w, output)
	case "yaml":
		output, err := result.ToYAML()
		if err != nil {
			return fmt.Errorf("failed to generate YAML: %v", err)
		}
		_, _ = fmt.Fprint(w, output)
	case "table":
		outputReverseTable(w, result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
	return nil
}

func outputReverseTable(w io.Writer, result *dns.ReverseResult) {
	_, _ = fmt.Fprintf(w, "IP: %s\n", result.IP)
	_, _ = fmt.Fprintf(w, "Query Time: %v\n\n", result.QueryTime.Round(time.Millisecond))

	if len(result.Hostnames) == 0 {
		_, _ = fmt.Fprintln(w, "No PTR records found.")
		return
	}

	_, _ = fmt.Fprintln(w, "Hostnames:")
	for _, hostname := range result.Hostnames {
		_, _ = fmt.Fprintf(w, "  - %s\n", hostname)
	}
}
