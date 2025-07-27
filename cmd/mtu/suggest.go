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
	TCPMSSv4         int `json:"tcp_mss_ipv4"`
	TCPMSSv6         int `json:"tcp_mss_ipv6"`
	WireGuardPayload int `json:"wireguard_payload"`
	IPSecESPUDP      int `json:"ipsec_esp_udp"`
}

func calculateSuggestions(pmtu int) Suggestions {
	return Suggestions{
		TCPMSSv4:         pmtu - 40, // IPv4 header (20) + TCP header (20)
		TCPMSSv6:         pmtu - 60, // IPv6 header (40) + TCP header (20)
		WireGuardPayload: pmtu - 60, // WireGuard overhead
		IPSecESPUDP:      pmtu - 84, // ESP + UDP + IP overhead
	}
}

func outputSuggestionsJSON(destination string, pmtu int, suggestions Suggestions) error {
	fmt.Printf("{\n")
	fmt.Printf("  \"target\": \"%s\",\n", destination)
	fmt.Printf("  \"pmtu\": %d,\n", pmtu)
	fmt.Printf("  \"suggestions\": {\n")
	fmt.Printf("    \"tcp_mss_ipv4\": %d,\n", suggestions.TCPMSSv4)
	fmt.Printf("    \"tcp_mss_ipv6\": %d,\n", suggestions.TCPMSSv6)
	fmt.Printf("    \"wireguard_payload\": %d,\n", suggestions.WireGuardPayload)
	fmt.Printf("    \"ipsec_esp_udp\": %d\n", suggestions.IPSecESPUDP)
	fmt.Printf("  }\n")
	fmt.Printf("}\n")
	return nil
}

func outputSuggestionsTable(destination string, pmtu int, suggestions Suggestions) error {
	fmt.Printf("Suggestions for %s (PMTU: %d):\n\n", destination, pmtu)
	fmt.Printf("TCP MSS (IPv4):      %d\n", suggestions.TCPMSSv4)
	fmt.Printf("TCP MSS (IPv6):      %d\n", suggestions.TCPMSSv6)
	fmt.Printf("WireGuard payload:   %d\n", suggestions.WireGuardPayload)
	fmt.Printf("IPSec ESP+UDP:       %d\n", suggestions.IPSecESPUDP)
	return nil
}
