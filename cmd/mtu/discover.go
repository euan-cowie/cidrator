package mtu

import (
	"fmt"
	"os"
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
	opts, err := readDiscoveryOptions(cmd, args[0])
	if err != nil {
		return err
	}

	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Hop-by-hop discovery only supports ICMP
	if opts.HopsMode && opts.Protocol != "icmp" {
		return fmt.Errorf("hop-by-hop discovery only supports ICMP protocol")
	}

	if !opts.Quiet && !jsonOutput {
		if opts.HopsMode {
			fmt.Printf("Hop-by-hop MTU discovery to %s...\n", opts.Destination)
			fmt.Printf("Protocol: %s, Max probe size: %d, Max hops: %d, Timeout: %v\n", opts.Protocol, opts.MaxMTU, opts.MaxHops, opts.Timeout)
		} else if opts.Step > 0 {
			fmt.Printf("Linear sweep MTU discovery to %s...\n", opts.Destination)
			fmt.Printf("Protocol: %s, Range: %d-%d, Step: %d, Timeout: %v\n", opts.Protocol, opts.MinMTU, opts.MaxMTU, opts.Step, opts.Timeout)
		} else {
			fmt.Printf("Discovering MTU to %s...\n", opts.Destination)
			fmt.Printf("Protocol: %s, Range: %d-%d, Timeout: %v\n", opts.Protocol, opts.MinMTU, opts.MaxMTU, opts.Timeout)
		}
	}

	// Create context with a budget that scales with the discovery mode and per-probe timeout.
	ctx, cancel := newDiscoveryContext(opts)
	defer cancel()

	// Perform discovery based on mode
	if opts.HopsMode {
		discoverer, err := newMTUDiscoverer(opts)
		if err != nil {
			return err
		}
		if !jsonOutput && !opts.Quiet {
			discoverer.SetProgressWriter(os.Stdout)
		}
		defer func() {
			if closeErr := discoverer.Close(); closeErr != nil && !opts.Quiet {
				fmt.Fprintf(os.Stderr, "Warning: failed to close discoverer: %v\n", closeErr)
			}
		}()

		// Hop-by-hop discovery
		hopResult, err := discoverer.DiscoverHopByHopMTU(ctx, opts.MaxHops, opts.MaxMTU)
		if err != nil {
			return fmt.Errorf("hop-by-hop MTU discovery failed: %w", err)
		}

		// Output hop-by-hop result
		if jsonOutput {
			return outputHopJSON(hopResult)
		}
		return outputHopTable(hopResult)
	}

	result, err := performMTUDiscovery(ctx, opts)
	if err != nil {
		return fmt.Errorf("MTU discovery failed: %w", err)
	}

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
	return writePrettyJSON(result)
}

func outputTable(result *MTUResult) error {
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
	type hopJSON struct {
		Hop     int     `json:"hop"`
		Addr    string  `json:"addr,omitempty"`
		MTU     int     `json:"mtu,omitempty"`
		RTT     float64 `json:"rtt"`
		Timeout bool    `json:"timeout,omitempty"`
		Error   string  `json:"error,omitempty"`
	}

	hops := make([]hopJSON, 0, len(result.Hops))
	for _, hop := range result.Hops {
		entry := hopJSON{
			Hop:     hop.Hop,
			MTU:     hop.MTU,
			RTT:     float64(hop.RTT) / float64(time.Millisecond),
			Timeout: hop.Timeout,
			Error:   hop.Error,
		}
		if hop.Addr != nil {
			entry.Addr = hop.Addr.String()
		}
		hops = append(hops, entry)
	}

	return writePrettyJSON(struct {
		Target       string    `json:"target"`
		Protocol     string    `json:"protocol"`
		MaxProbeSize int       `json:"max_probe_size"`
		FinalPMTU    int       `json:"final_pmtu"`
		Hops         []hopJSON `json:"hops"`
		ElapsedMS    int       `json:"elapsed_ms"`
	}{
		Target:       result.Target,
		Protocol:     result.Protocol,
		MaxProbeSize: result.MaxProbeSize,
		FinalPMTU:    result.FinalPMTU,
		Hops:         hops,
		ElapsedMS:    result.ElapsedMS,
	})
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
