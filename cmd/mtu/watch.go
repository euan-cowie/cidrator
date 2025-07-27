package mtu

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// watchCmd represents the watch command
var watchCmd = &cobra.Command{
	Use:   "watch <destination>",
	Short: "Re-run discover every N seconds and notify on change",
	Long: `Watch continuously monitors the Path-MTU to a destination and alerts
when changes are detected. Useful for detecting MTU black holes or path changes.

Examples:
  cidrator mtu watch example.com -i 10s
  cidrator mtu watch 8.8.8.8 --interval 30s --mss-only`,
	Args: cobra.ExactArgs(1),
	RunE: runWatch,
}

func init() {
	watchCmd.Flags().Duration("interval", 10*time.Second, "Interval between checks")
	watchCmd.Flags().Bool("mss-only", false, "Only alert on MSS changes")
	watchCmd.Flags().Bool("syslog", false, "Send alerts to syslog")
}

func runWatch(cmd *cobra.Command, args []string) error {
	destination := args[0]
	interval, _ := cmd.Flags().GetDuration("interval")
	mssOnly, _ := cmd.Flags().GetBool("mss-only")
	useSyslog, _ := cmd.Flags().GetBool("syslog")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Get other flags for MTU discovery
	ipv6, _ := cmd.Flags().GetBool("6")
	proto, _ := cmd.Flags().GetString("proto")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	ttl, _ := cmd.Flags().GetInt("ttl")
	minMTU, _ := cmd.Flags().GetInt("min")
	maxMTU, _ := cmd.Flags().GetInt("max")

	// Set default min MTU based on IP version
	if minMTU == 0 {
		if ipv6 {
			minMTU = 1280
		} else {
			minMTU = 576
		}
	}

	if !jsonOutput {
		fmt.Printf("Watching MTU to %s every %v...\n", destination, interval)
		if mssOnly {
			fmt.Printf("Will only alert on MSS changes\n")
		}
		if useSyslog {
			fmt.Printf("Alerts will be sent to syslog\n")
		}
		fmt.Printf("Press Ctrl+C to stop\n\n")
	}

	var lastResult *MTUResult

	for {
		// Perform MTU discovery
		result, err := performMTUDiscovery(destination, ipv6, proto, timeout, ttl, minMTU, maxMTU)

		timestamp := time.Now()

		if err != nil {
			if jsonOutput {
				outputWatchErrorJSON(timestamp, destination, err)
			} else {
				fmt.Printf("[%s] Error: %v\n", timestamp.Format("15:04:05"), err)
			}
		} else {
			// Check for changes
			changed := lastResult == nil || result.PMTU != lastResult.PMTU
			mssChanged := lastResult == nil || result.MSS != lastResult.MSS

			// Output based on mode
			if jsonOutput {
				outputWatchResultJSON(timestamp, result, changed, mssChanged)
			} else {
				symbol := " "
				if changed {
					symbol = "!"
				}
				fmt.Printf("[%s]%s MTU: %d, MSS: %d",
					timestamp.Format("15:04:05"), symbol, result.PMTU, result.MSS)

				if changed {
					if lastResult != nil {
						fmt.Printf(" (was %d)", lastResult.PMTU)
					}
					fmt.Printf(" ← CHANGED")
				}
				fmt.Printf("\n")
			}

			// Handle alerts
			if changed && lastResult != nil {
				if useSyslog {
					// TODO: Send to syslog
				}
				if mssOnly && !mssChanged {
					// Skip alert if only monitoring MSS changes
				} else {
					// Non-zero exit if PMTU drops as specified in requirements
					if result.PMTU < lastResult.PMTU {
						if !jsonOutput {
							fmt.Printf("ERROR: PMTU dropped from %d to %d\n", lastResult.PMTU, result.PMTU)
						}
						os.Exit(1)
					}
				}
			}

			lastResult = result
		}

		time.Sleep(interval)
	}
}

func performMTUDiscovery(destination string, ipv6 bool, proto string, timeout time.Duration, ttl, minMTU, maxMTU int) (*MTUResult, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create MTU discoverer
	discoverer, err := NewMTUDiscoverer(destination, ipv6, proto, timeout, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to create discoverer: %w", err)
	}
	defer discoverer.Close()

	// Perform MTU discovery
	return discoverer.DiscoverPMTU(ctx, minMTU, maxMTU)
}

func outputWatchErrorJSON(timestamp time.Time, destination string, err error) {
	fmt.Printf("{\"timestamp\":\"%s\",\"target\":\"%s\",\"error\":\"%v\"}\n",
		timestamp.Format(time.RFC3339), destination, err)
}

func outputWatchResultJSON(timestamp time.Time, result *MTUResult, changed, mssChanged bool) {
	fmt.Printf("{\"timestamp\":\"%s\",\"target\":\"%s\",\"pmtu\":%d,\"mss\":%d,\"changed\":%t,\"mss_changed\":%t}\n",
		timestamp.Format(time.RFC3339), result.Target, result.PMTU, result.MSS, changed, mssChanged)
}
