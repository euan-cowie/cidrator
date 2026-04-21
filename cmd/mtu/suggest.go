package mtu

import (
	"fmt"
	"net"

	"github.com/spf13/cobra"
)

var getSuggestionInterfaces = GetNetworkInterfaces
var suggestMTUDiscovery = performMTUDiscovery

// suggestCmd represents the suggest command
var suggestCmd = &cobra.Command{
	Use:   "suggest <destination>",
	Short: "Print TCP MSS / IPSec ESP / WireGuard frame sizes for the path",
	Long: `Suggest calculates optimal frame sizes for various protocols based on the
discovered Path-MTU to the destination.

Calculations:
• TCP MSS = PMTU - 40 (IPv4) or - 60 (IPv6)
• WireGuard payload = PMTU - 60
• IPSec ESP + UDP-encap = PMTU - (ESP + UDP + IP)

Examples:
  cidrator mtu suggest example.com --proto tcp
  cidrator mtu suggest 8.8.8.8 --proto tcp --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSuggest,
}

func runSuggest(cmd *cobra.Command, args []string) error {
	opts, err := readDiscoveryOptions(cmd, args[0])
	if err != nil {
		return err
	}
	if opts.HopsMode {
		return fmt.Errorf("--hops is only supported by mtu discover")
	}
	opts = applySuggestProbeDefaults(cmd, opts)

	jsonOutput, _ := cmd.Flags().GetBool("json")

	ctx, cancel := newDiscoveryContext(opts)
	defer cancel()

	result, err := suggestMTUDiscovery(ctx, opts)
	if err != nil {
		pmtu, fallbackErr := fallbackSuggestionPMTU(opts)
		if fallbackErr != nil {
			return fmt.Errorf("MTU discovery failed: %w", err)
		}
		result = &MTUResult{
			Target:   opts.Destination,
			Protocol: opts.Protocol,
			PMTU:     pmtu,
		}
	}

	suggestions := calculateSuggestions(result.PMTU)

	if jsonOutput {
		return outputSuggestionsJSON(result.Target, result.PMTU, suggestions)
	}
	return outputSuggestionsTable(result.Target, result.PMTU, suggestions)
}

func applySuggestProbeDefaults(cmd *cobra.Command, opts discoveryOptions) discoveryOptions {
	// Suggest should work for unprivileged users without requiring them to override
	// the shared raw-ICMP discovery default explicitly.
	if opts.Protocol == "icmp" && !cmd.Flags().Changed("proto") {
		opts.Protocol = "tcp"
	}
	return opts
}

type Suggestions struct {
	TCPMSSv4           int `json:"tcp_mss_ipv4"`
	TCPMSSv6           int `json:"tcp_mss_ipv6"`
	TCPMSSv4Timestamps int `json:"tcp_mss_ipv4_timestamps"` // With 12-byte TCP timestamps option
	TCPMSSv6Timestamps int `json:"tcp_mss_ipv6_timestamps"`
	WireGuardPayload   int `json:"wireguard_payload"`
	IPSecESPUDP        int `json:"ipsec_esp_udp"`
	GREPayload         int `json:"gre_payload"`   // 4-byte GRE header + IP
	VXLANPayload       int `json:"vxlan_payload"` // 50-byte VXLAN overhead
	MPLSPayload        int `json:"mpls_1label"`   // 4-byte per label
}

func calculateSuggestions(pmtu int) Suggestions {
	return Suggestions{
		TCPMSSv4:           pmtu - 40, // IPv4 header (20) + TCP header (20)
		TCPMSSv6:           pmtu - 60, // IPv6 header (40) + TCP header (20)
		TCPMSSv4Timestamps: pmtu - 52, // IPv4 (20) + TCP (20) + Timestamps (12)
		TCPMSSv6Timestamps: pmtu - 72, // IPv6 (40) + TCP (20) + Timestamps (12)
		WireGuardPayload:   pmtu - 60, // WireGuard overhead
		IPSecESPUDP:        pmtu - 84, // ESP + UDP + IP overhead
		GREPayload:         pmtu - 24, // IPv4 (20) + GRE (4)
		VXLANPayload:       pmtu - 50, // Per RFC 7348
		MPLSPayload:        pmtu - 4,  // Single label
	}
}

func outputSuggestionsJSON(destination string, pmtu int, suggestions Suggestions) error {
	return writePrettyJSON(struct {
		Target      string      `json:"target"`
		PMTU        int         `json:"pmtu"`
		Suggestions Suggestions `json:"suggestions"`
	}{
		Target:      destination,
		PMTU:        pmtu,
		Suggestions: suggestions,
	})
}

func outputSuggestionsTable(destination string, pmtu int, suggestions Suggestions) error {
	fmt.Printf("Suggestions for %s (PMTU: %d):\n\n", destination, pmtu)
	fmt.Printf("TCP MSS (IPv4):              %d\n", suggestions.TCPMSSv4)
	fmt.Printf("TCP MSS (IPv6):              %d\n", suggestions.TCPMSSv6)
	fmt.Printf("TCP MSS (IPv4+timestamps):   %d\n", suggestions.TCPMSSv4Timestamps)
	fmt.Printf("TCP MSS (IPv6+timestamps):   %d\n", suggestions.TCPMSSv6Timestamps)
	fmt.Printf("WireGuard payload:           %d\n", suggestions.WireGuardPayload)
	fmt.Printf("IPSec ESP+UDP:               %d\n", suggestions.IPSecESPUDP)
	fmt.Printf("GRE payload:                 %d\n", suggestions.GREPayload)
	fmt.Printf("VXLAN payload:               %d\n", suggestions.VXLANPayload)
	fmt.Printf("MPLS (1 label):              %d\n", suggestions.MPLSPayload)
	return nil
}

func fallbackSuggestionPMTU(opts discoveryOptions) (int, error) {
	ips, err := resolveTargetIPs(opts.Destination)
	if err != nil {
		return 0, err
	}

	isLoopback := false
	for _, ip := range ips {
		if !ip.IsLoopback() {
			continue
		}
		if opts.IPv6 && ip.To4() != nil {
			continue
		}
		if !opts.IPv6 && ip.To4() == nil {
			continue
		}
		isLoopback = true
		break
	}
	if !isLoopback {
		return 0, fmt.Errorf("%s does not resolve to a loopback address", opts.Destination)
	}

	result, err := getSuggestionInterfaces()
	if err != nil {
		return 0, err
	}

	loopbackMTU := 0
	for _, iface := range result.Interfaces {
		if iface.Type == "loopback" && iface.MTU > loopbackMTU {
			loopbackMTU = iface.MTU
		}
	}
	if loopbackMTU == 0 {
		return 0, fmt.Errorf("no loopback interface MTU available")
	}
	if loopbackMTU < opts.MinMTU {
		return 0, fmt.Errorf("loopback MTU %d is below the configured minimum %d", loopbackMTU, opts.MinMTU)
	}

	return loopbackMTU, nil
}

func resolveTargetIPs(target string) ([]net.IP, error) {
	if ip := net.ParseIP(target); ip != nil {
		return []net.IP{ip}, nil
	}
	return lookupIPAddrs(target)
}
