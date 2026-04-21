package mtu

import (
	"fmt"
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
}

func runWatch(cmd *cobra.Command, args []string) error {
	opts, err := readDiscoveryOptions(cmd, args[0])
	if err != nil {
		return err
	}
	if opts.HopsMode {
		return fmt.Errorf("--hops is only supported by mtu discover")
	}

	interval, _ := cmd.Flags().GetDuration("interval")
	mssOnly, _ := cmd.Flags().GetBool("mss-only")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	if !jsonOutput {
		fmt.Printf("Watching MTU to %s every %v...\n", opts.Destination, interval)
		if mssOnly {
			fmt.Printf("Will only alert on MSS changes\n")
		}
		fmt.Printf("Press Ctrl+C to stop\n\n")
	}

	var lastResult *MTUResult

	for {
		// Perform MTU discovery
		ctx, cancel := newDiscoveryContext(opts)
		result, err := performMTUDiscovery(ctx, opts)
		cancel()

		timestamp := time.Now()

		if err != nil {
			if jsonOutput {
				if jsonErr := outputWatchErrorJSON(timestamp, opts.Destination, err); jsonErr != nil {
					return jsonErr
				}
			} else {
				fmt.Printf("[%s] Error: %v\n", timestamp.Format("15:04:05"), err)
			}
		} else {
			// Check for changes
			changed := lastResult == nil || result.PMTU != lastResult.PMTU
			mssChanged := lastResult == nil || result.MSS != lastResult.MSS

			// Output based on mode
			if jsonOutput {
				if jsonErr := outputWatchResultJSON(timestamp, result, changed, mssChanged); jsonErr != nil {
					return jsonErr
				}
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
				if mssOnly && !mssChanged {
					// Skip alert if only monitoring MSS changes
				} else {
					// Non-zero exit if PMTU drops as specified in requirements
					if result.PMTU < lastResult.PMTU {
						return newWatchDropError(cmd, lastResult.PMTU, result.PMTU, jsonOutput)
					}
				}
			}

			lastResult = result
		}

		time.Sleep(interval)
	}
}

func newWatchDropError(cmd *cobra.Command, previousPMTU, currentPMTU int, jsonOutput bool) error {
	cmd.SilenceUsage = true
	if jsonOutput {
		cmd.SilenceErrors = true
	}
	return fmt.Errorf("pmtu dropped from %d to %d", previousPMTU, currentPMTU)
}

func outputWatchErrorJSON(timestamp time.Time, destination string, err error) error {
	return writeJSONLine(struct {
		Timestamp string `json:"timestamp"`
		Target    string `json:"target"`
		Error     string `json:"error"`
	}{
		Timestamp: timestamp.Format(time.RFC3339),
		Target:    destination,
		Error:     err.Error(),
	})
}

func outputWatchResultJSON(timestamp time.Time, result *MTUResult, changed, mssChanged bool) error {
	return writeJSONLine(struct {
		Timestamp  string `json:"timestamp"`
		Target     string `json:"target"`
		PMTU       int    `json:"pmtu"`
		MSS        int    `json:"mss"`
		Changed    bool   `json:"changed"`
		MSSChanged bool   `json:"mss_changed"`
	}{
		Timestamp:  timestamp.Format(time.RFC3339),
		Target:     result.Target,
		PMTU:       result.PMTU,
		MSS:        result.MSS,
		Changed:    changed,
		MSSChanged: mssChanged,
	})
}
