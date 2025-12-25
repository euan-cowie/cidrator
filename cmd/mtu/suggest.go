package mtu

import (
	"fmt"

	"github.com/spf13/cobra"
)

// suggestCmd represents the suggest command
var suggestCmd = &cobra.Command{
	Use:   "suggest <destination>",
	Short: "Print TCP MSS / IPSec ESP / WireGuard frame sizes for the path",
	Long: `Suggest calculates optimal frame sizes for various protocols based on the
discovered Path-MTU to the destination.

Calculations:
• TCP MSS = PMTU - 40 (IPv4) or - 48 (IPv6)
• WireGuard payload = PMTU - 60
• IPSec ESP + UDP-encap = PMTU - (ESP + UDP + IP)

Examples:
  cidrator mtu suggest example.com
  cidrator mtu suggest 8.8.8.8 --json`,
	Args: cobra.ExactArgs(1),
	RunE: runSuggest,
}

func runSuggest(cmd *cobra.Command, args []string) error {
	destination := args[0]
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// TODO: First discover MTU, then calculate suggestions
	// For now, use a reasonable default
	pmtu := 1472 // Placeholder - should be discovered

	suggestions := calculateSuggestions(pmtu)

	if jsonOutput {
		return outputSuggestionsJSON(destination, pmtu, suggestions)
	}
	return outputSuggestionsTable(destination, pmtu, suggestions)
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
	fmt.Printf("{\n")
	fmt.Printf("  \"target\": \"%s\",\n", destination)
	fmt.Printf("  \"pmtu\": %d,\n", pmtu)
	fmt.Printf("  \"suggestions\": {\n")
	fmt.Printf("    \"tcp_mss_ipv4\": %d,\n", suggestions.TCPMSSv4)
	fmt.Printf("    \"tcp_mss_ipv6\": %d,\n", suggestions.TCPMSSv6)
	fmt.Printf("    \"tcp_mss_ipv4_timestamps\": %d,\n", suggestions.TCPMSSv4Timestamps)
	fmt.Printf("    \"tcp_mss_ipv6_timestamps\": %d,\n", suggestions.TCPMSSv6Timestamps)
	fmt.Printf("    \"wireguard_payload\": %d,\n", suggestions.WireGuardPayload)
	fmt.Printf("    \"ipsec_esp_udp\": %d,\n", suggestions.IPSecESPUDP)
	fmt.Printf("    \"gre_payload\": %d,\n", suggestions.GREPayload)
	fmt.Printf("    \"vxlan_payload\": %d,\n", suggestions.VXLANPayload)
	fmt.Printf("    \"mpls_1label\": %d\n", suggestions.MPLSPayload)
	fmt.Printf("  }\n")
	fmt.Printf("}\n")
	return nil
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
