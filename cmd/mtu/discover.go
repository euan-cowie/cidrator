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
	step, _ := cmd.Flags().GetInt("step")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	_, _ = cmd.Flags().GetInt("ttl") // ttl - TODO: implement hop limit
	jsonOutput, _ := cmd.Flags().GetBool("json")
	quiet, _ := cmd.Flags().GetBool("quiet")
	_, _ = cmd.Flags().GetInt("pps") // pps - TODO: implement rate limiting
	hopsMode, _ := cmd.Flags().GetBool("hops")
	maxHops, _ := cmd.Flags().GetInt("max-hops")
	port, _ := cmd.Flags().GetInt("port")
	plpmtud, _ := cmd.Flags().GetBool("plpmtud")
	plpPort, _ := cmd.Flags().GetInt("plp-port")

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

	// Hop-by-hop discovery only supports ICMP
	if hopsMode && proto != "icmp" {
		return fmt.Errorf("hop-by-hop discovery only supports ICMP protocol")
	}

	if !quiet {
		if hopsMode {
			fmt.Printf("Hop-by-hop MTU discovery to %s...\n", destination)
			fmt.Printf("Protocol: %s, Max probe size: %d, Max hops: %d, Timeout: %v\n", proto, maxMTU, maxHops, timeout)
		} else if step > 0 {
			fmt.Printf("Linear sweep MTU discovery to %s...\n", destination)
			fmt.Printf("Protocol: %s, Range: %d-%d, Step: %d, Timeout: %v\n", proto, minMTU, maxMTU, step, timeout)
		} else {
			fmt.Printf("Discovering MTU to %s...\n", destination)
			fmt.Printf("Protocol: %s, Range: %d-%d, Timeout: %v\n", proto, minMTU, maxMTU, timeout)
		}
	}

	// Get TTL value
	ttl, _ := cmd.Flags().GetInt("ttl")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Longer timeout for hop-by-hop
	defer cancel()

	// Initialize ICMP listener for fail-fast fragmentation error detection
	// This runs in the background and detects ICMP "Fragmentation Needed" errors
	// without waiting for probe timeouts. Requires elevated privileges (root/sudo).
	var icmpListener *ICMPListener
	icmpListener, err := NewICMPListener()
	if err != nil {
		// Continue without ICMP listener (non-root mode or unsupported platform)
		if !quiet {
			fmt.Printf("Note: ICMP listener unavailable (%v), using probe timeouts only\n", err)
		}
	} else {
		icmpListener.Start(ctx)
		defer func() {
			if closeErr := icmpListener.Close(); closeErr != nil && !quiet {
				fmt.Printf("Warning: failed to close ICMP listener: %v\n", closeErr)
			}
		}()
	}

	// Create MTU discoverer
	discoverer, err := NewMTUDiscoverer(destination, ipv6, proto, port, timeout, ttl)
	if err != nil {
		return fmt.Errorf("failed to create discoverer: %w", err)
	}
	defer func() {
		if closeErr := discoverer.Close(); closeErr != nil {
			// Log the close error but don't override the main error
			fmt.Printf("Warning: failed to close discoverer: %v\n", closeErr)
		}
	}()

	// Wire up ICMP listener if available
	if icmpListener != nil {
		discoverer.SetICMPListener(icmpListener)
	}

	// Perform discovery based on mode
	if hopsMode {
		// Hop-by-hop discovery
		hopResult, err := discoverer.DiscoverHopByHopMTU(ctx, maxHops, maxMTU)
		if err != nil {
			return fmt.Errorf("hop-by-hop MTU discovery failed: %w", err)
		}

		// Output hop-by-hop result
		if jsonOutput {
			return outputHopJSON(hopResult)
		}
		return outputHopTable(hopResult)
	} else {
		// Regular PMTU discovery (binary search or linear sweep)
		var result *MTUResult
		var err error

		if step > 0 {
			// Linear sweep mode
			result, err = discoverer.DiscoverPMTULinear(ctx, minMTU, maxMTU, step)
		} else if plpmtud {
			// PLPMTUD fallback mode (for black-hole detection)
			result, err = discoverer.WithPLPMTUDFallback(ctx, minMTU, maxMTU, plpPort)
		} else {
			// Binary search mode (default)
			result, err = discoverer.DiscoverPMTU(ctx, minMTU, maxMTU)
		}

		if err != nil {
			return fmt.Errorf("MTU discovery failed: %w", err)
		}

		// Output result
		if jsonOutput {
			return outputJSON(result)
		}
		return outputTable(result)
	}
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

// outputHopJSON outputs hop-by-hop discovery results in JSON format
func outputHopJSON(result *HopMTUResult) error {
	fmt.Printf("{\n")
	fmt.Printf("  \"target\": \"%s\",\n", result.Target)
	fmt.Printf("  \"protocol\": \"%s\",\n", result.Protocol)
	fmt.Printf("  \"max_probe_size\": %d,\n", result.MaxProbeSize)
	fmt.Printf("  \"final_pmtu\": %d,\n", result.FinalPMTU)
	fmt.Printf("  \"elapsed_ms\": %d,\n", result.ElapsedMS)
	fmt.Printf("  \"hops\": [\n")

	for i, hop := range result.Hops {
		fmt.Printf("    {\n")
		fmt.Printf("      \"hop\": %d,\n", hop.Hop)
		if hop.Addr != nil {
			fmt.Printf("      \"addr\": \"%s\",\n", hop.Addr.String())
		}
		if hop.MTU > 0 {
			fmt.Printf("      \"mtu\": %d,\n", hop.MTU)
		}
		fmt.Printf("      \"rtt\": %.2f,\n", float64(hop.RTT.Nanoseconds())/1000000.0)
		if hop.Timeout {
			fmt.Printf("      \"timeout\": true,\n")
		}
		if hop.Error != "" {
			fmt.Printf("      \"error\": \"%s\",\n", hop.Error)
		}
		// Remove trailing comma
		fmt.Printf("      \"hop_number\": %d\n", hop.Hop)
		if i < len(result.Hops)-1 {
			fmt.Printf("    },\n")
		} else {
			fmt.Printf("    }\n")
		}
	}

	fmt.Printf("  ]\n")
	fmt.Printf("}\n")
	return nil
}

// outputHopTable outputs hop-by-hop discovery results in table format
func outputHopTable(result *HopMTUResult) error {
	fmt.Printf("\nHop-by-hop MTU Discovery Results:\n")
	fmt.Printf("Target: %s\n", result.Target)
	fmt.Printf("Protocol: %s\n", result.Protocol)
	fmt.Printf("Max probe size: %d bytes\n", result.MaxProbeSize)
	if result.FinalPMTU > 0 {
		fmt.Printf("Final PMTU: %d bytes\n", result.FinalPMTU)
	}
	fmt.Printf("Total time: %dms\n\n", result.ElapsedMS)

	// Print table header
	fmt.Printf("%-4s %-15s %-6s %-10s %s\n", "Hop", "Address", "MTU", "RTT", "Status")
	fmt.Printf("%-4s %-15s %-6s %-10s %s\n", "---", "---------------", "-----", "----------", "------")

	// Print each hop
	for _, hop := range result.Hops {
		// Hop number
		fmt.Printf("%-4d ", hop.Hop)

		// Address
		addr := ""
		if hop.Addr != nil {
			addr = hop.Addr.String()
		}
		fmt.Printf("%-15s ", addr)

		// MTU
		mtu := ""
		if hop.MTU > 0 {
			mtu = fmt.Sprintf("%d", hop.MTU)
		}
		fmt.Printf("%-6s ", mtu)

		// RTT
		rtt := ""
		if !hop.Timeout && hop.Error == "" {
			rtt = fmt.Sprintf("%.2fms", float64(hop.RTT.Nanoseconds())/1000000.0)
		}
		fmt.Printf("%-10s ", rtt)

		// Status
		status := ""
		if hop.Timeout {
			status = "timeout"
		} else if hop.Error != "" {
			status = hop.Error
		} else if hop.Addr != nil {
			status = "ok"
		}
		fmt.Printf("%s\n", status)
	}

	return nil
}
