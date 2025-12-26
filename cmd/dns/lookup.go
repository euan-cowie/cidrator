package dns

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/euan-cowie/cidrator/internal/dns"
	"github.com/spf13/cobra"
)

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
	result, err := dns.Lookup(domain, opts)
	if err != nil {
		return err
	}

	// Output result
	return outputLookupResult(result, format)
}

func outputLookupResult(result *dns.DNSResult, format string) error {
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
		outputLookupTable(result)
	default:
		return fmt.Errorf("unsupported output format: %s", format)
	}
	return nil
}

func outputLookupTable(result *dns.DNSResult) {
	fmt.Printf("Domain: %s\n", result.Domain)
	fmt.Printf("Query Type: %s\n", result.QueryType)
	if result.Server != "" {
		fmt.Printf("Server: %s\n", result.Server)
	}
	fmt.Printf("Query Time: %v\n\n", result.QueryTime.Round(time.Millisecond))

	if len(result.Records) == 0 {
		fmt.Println("No records found.")
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	defer func() { _ = w.Flush() }()

	// Check if any MX records exist (to show priority column)
	hasMX := false
	for _, r := range result.Records {
		if r.Type == "MX" {
			hasMX = true
			break
		}
	}

	if hasMX {
		_, _ = fmt.Fprintf(w, "TYPE\tPRIORITY\tVALUE\n")
		_, _ = fmt.Fprintf(w, "----\t--------\t-----\n")
		for _, r := range result.Records {
			if r.Type == "MX" {
				_, _ = fmt.Fprintf(w, "%s\t%d\t%s\n", r.Type, r.Priority, r.Value)
			} else {
				_, _ = fmt.Fprintf(w, "%s\t\t%s\n", r.Type, r.Value)
			}
		}
	} else {
		_, _ = fmt.Fprintf(w, "TYPE\tVALUE\n")
		_, _ = fmt.Fprintf(w, "----\t-----\n")
		for _, r := range result.Records {
			_, _ = fmt.Fprintf(w, "%s\t%s\n", r.Type, r.Value)
		}
	}
}
