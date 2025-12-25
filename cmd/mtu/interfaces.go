package mtu

import (
	"fmt"

	"github.com/spf13/cobra"
)

// interfacesCmd represents the interfaces command
var interfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "List local interfaces + configured MTU",
	Long: `Interfaces lists all local network interfaces and their configured MTU values.
This helps establish baseline MTU values for discovery operations.

Examples:
  cidrator mtu interfaces
  cidrator mtu interfaces --json`,
	RunE: runInterfaces,
}

func runInterfaces(cmd *cobra.Command, args []string) error {
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Get real network interfaces
	result, err := GetNetworkInterfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %w", err)
	}

	if jsonOutput {
		return outputInterfacesJSON(result)
	}
	return outputInterfacesTable(result)
}

func outputInterfacesJSON(result *InterfaceResult) error {
	fmt.Printf("{\n  \"interfaces\": [\n")
	for i, iface := range result.Interfaces {
		comma := ""
		if i < len(result.Interfaces)-1 {
			comma = ","
		}
		fmt.Printf("    {\"name\": \"%s\", \"mtu\": %d, \"type\": \"%s\"}%s\n",
			iface.Name, iface.MTU, iface.Type, comma)
	}
	fmt.Printf("  ]\n}\n")
	return nil
}

func outputInterfacesTable(result *InterfaceResult) error {
	fmt.Printf("%-15s %-6s %s\n", "Interface", "MTU", "Type")
	fmt.Printf("%-15s %-6s %s\n", "---------------", "------", "--------")

	for _, iface := range result.Interfaces {
		fmt.Printf("%-15s %-6d %s\n", iface.Name, iface.MTU, iface.Type)
	}

	return nil
}
