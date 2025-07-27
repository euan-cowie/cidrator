package mtu

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

// discoverCmd represents the discover command
var discoverCmd = &cobra.Command{
	Use:   "discover <destination>",
	Short: "Binary-search to the largest size that gets through",
	Long: `Discover performs Path-MTU discovery using binary search to find the largest
packet size that can reach the destination without fragmentation.

Examples:
  cidrator mtu discover 8.8.8.8
  cidrator mtu discover 2001:4860:4860::8888 --6
  cidrator mtu discover example.com --proto tcp --json`,
	Args: cobra.ExactArgs(1),
	RunE: runDiscover,
}

func runDiscover(cmd *cobra.Command, args []string) error {
	destination := args[0]

	// Get flags
	_, _ = cmd.Flags().GetBool("4") // ipv4 - TODO: implement IPv4/IPv6 logic
	ipv6, _ := cmd.Flags().GetBool("6")
	proto, _ := cmd.Flags().GetString("proto")
	minMTU, _ := cmd.Flags().GetInt("min")
	maxMTU, _ := cmd.Flags().GetInt("max")
	_, _ = cmd.Flags().GetInt("step") // step - TODO: implement linear sweep fallback
	timeout, _ := cmd.Flags().GetDuration("timeout")
	_, _ = cmd.Flags().GetInt("ttl") // ttl - TODO: implement hop limit
	jsonOutput, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")
	_, _ = cmd.Flags().GetInt("pps") // pps - TODO: implement rate limiting

	// Set default timeout if not specified
	if timeout == 0 {
		timeout = 2 * time.Second
	}

	// Set default min MTU based on IP version
	if minMTU == 0 {
		if ipv6 {
			minMTU = 1280 // IPv6 minimum
		} else {
			minMTU = 576 // IPv4 minimum
		}
	}

	if !quiet {
		fmt.Printf("Discovering MTU to %s...\n", destination)
		fmt.Printf("Protocol: %s, Range: %d-%d, Timeout: %v\n", proto, minMTU, maxMTU, timeout)
	}

	// Get TTL value
	ttl, _ := cmd.Flags().GetInt("ttl")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MTU discoverer
	discoverer, err := NewMTUDiscoverer(destination, ipv6, proto, timeout, ttl)
	if err != nil {
		return fmt.Errorf("failed to create discoverer: %w", err)
	}
	defer func() {
		if closeErr := discoverer.Close(); closeErr != nil {
			// Log the close error but don't override the main error
			fmt.Printf("Warning: failed to close discoverer: %v\n", closeErr)
		}
	}()

	// Perform MTU discovery
	result, err := discoverer.DiscoverPMTU(ctx, minMTU, maxMTU)
	if err != nil {
		return fmt.Errorf("MTU discovery failed: %w", err)
	}

	// Output result
	if jsonOutput {
		return outputJSON(result)
	}
	return outputTable(result)
}

// MTUResult represents the result of MTU discovery
type MTUResult struct {
	Target    string `json:"target"`
	Protocol  string `json:"protocol"`
	PMTU      int    `json:"pmtu"`
	MSS       int    `json:"mss"`
	Hops      int    `json:"hops"`
	ElapsedMS int    `json:"elapsed_ms"`
}

func outputJSON(result *MTUResult) error {
	fmt.Printf("{\n")
	fmt.Printf("  \"target\": \"%s\",\n", result.Target)
	fmt.Printf("  \"protocol\": \"%s\",\n", result.Protocol)
	fmt.Printf("  \"pmtu\": %d,\n", result.PMTU)
	fmt.Printf("  \"mss\": %d,\n", result.MSS)
	fmt.Printf("  \"hops\": %d,\n", result.Hops)
	fmt.Printf("  \"elapsed_ms\": %d\n", result.ElapsedMS)
	fmt.Printf("}\n")
	return nil
}

func outputTable(result *MTUResult) error {
	// TODO: Implement table output
	fmt.Printf("Target: %s\n", result.Target)
	fmt.Printf("Protocol: %s\n", result.Protocol)
	fmt.Printf("Path MTU: %d\n", result.PMTU)
	fmt.Printf("TCP MSS: %d\n", result.MSS)
	fmt.Printf("Hops: %d\n", result.Hops)
	fmt.Printf("Elapsed: %dms\n", result.ElapsedMS)
	return nil
}
