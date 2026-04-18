package dns

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/euan-cowie/cidrator/internal/dns"
	"github.com/spf13/cobra"
)

var dnsLookup = dns.Lookup

// lookupCmd represents the dns lookup command
var lookupCmd = &cobra.Command{
	Use:   "lookup <domain>",
	Short: "Perform DNS lookups for a domain",
	Long: `Lookup performs DNS queries for the specified domain.

Supports multiple record types: A, AAAA, MX, TXT, CNAME, NS, and ALL.

Examples:
  cidrator dns lookup example.com
  cidrator dns lookup example.com --type MX
  cidrator dns lookup example.com --type AAAA --format json
  cidrator dns lookup example.com --type ALL
  cidrator dns lookup example.com --server 8.8.8.8`,
	Args: cobra.ExactArgs(1),
	RunE: runLookup,
}

func init() {
	DNSCmd.AddCommand(lookupCmd)

	// Add flags for DNS lookup
	lookupCmd.Flags().StringP("type", "t", "A", "DNS record type (A, AAAA, MX, TXT, CNAME, NS, ALL)")
	lookupCmd.Flags().StringP("format", "f", "table", "Output format (table, json, yaml)")
	lookupCmd.Flags().StringP("server", "s", "", "DNS server to query (e.g., 8.8.8.8)")
	lookupCmd.Flags().DurationP("timeout", "", 5*time.Second, "Query timeout")
}

func runLookup(cmd *cobra.Command, args []string) error {
	domain := args[0]

	// Get flags
	recordType, _ := cmd.Flags().GetString("type")
	format, _ := cmd.Flags().GetString("format")
	server, _ := cmd.Flags().GetString("server")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	// Create lookup options
	opts := dns.LookupOptions{
		RecordType: strings.ToUpper(recordType),
		Server:     server,
		Timeout:    timeout,
	}

	// Perform lookup
	result, err := dnsLookup(domain, opts)
	if err != nil {
		return err
	}

	// Output result
	return outputLookupResult(cmd.OutOrStdout(), result, format)
}

func outputLookupResult(w io.Writer, result *dns.DNSResult, format string) error {
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
		outputLookupTable(w, result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
	return nil
}

func outputLookupTable(w io.Writer, result *dns.DNSResult) {
	_, _ = fmt.Fprintf(w, "Domain: %s\n", result.Domain)
	_, _ = fmt.Fprintf(w, "Query Type: %s\n", result.QueryType)
	if result.Server != "" {
		_, _ = fmt.Fprintf(w, "Server: %s\n", result.Server)
	}
	_, _ = fmt.Fprintf(w, "Query Time: %v\n\n", result.QueryTime.Round(time.Millisecond))

	if len(result.Records) == 0 {
		_, _ = fmt.Fprintln(w, "No records found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer func() { _ = tw.Flush() }()

	// Check if any MX records exist (to show priority column)
	hasMX := false
	for _, r := range result.Records {
		if r.Type == "MX" {
			hasMX = true
			break
		}
	}

	if hasMX {
		_, _ = fmt.Fprintf(tw, "TYPE\tPRIORITY\tVALUE\n")
		_, _ = fmt.Fprintf(tw, "----\t--------\t-----\n")
		for _, r := range result.Records {
			if r.Type == "MX" {
				_, _ = fmt.Fprintf(tw, "%s\t%d\t%s\n", r.Type, r.Priority, r.Value)
			} else {
				_, _ = fmt.Fprintf(tw, "%s\t\t%s\n", r.Type, r.Value)
			}
		}
	} else {
		_, _ = fmt.Fprintf(tw, "TYPE\tVALUE\n")
		_, _ = fmt.Fprintf(tw, "----\t-----\n")
		for _, r := range result.Records {
			_, _ = fmt.Fprintf(tw, "%s\t%s\n", r.Type, r.Value)
		}
	}
}
