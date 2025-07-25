package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of cidrator",
	Long:  `Print the version number, commit hash, and build date of cidrator.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cidrator version %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Built: %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
