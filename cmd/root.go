package cmd

import (
	"fmt"
	"os"

	"github.com/euan-cowie/cidrator/cmd/cidr"
	"github.com/euan-cowie/cidrator/cmd/dns"
	"github.com/euan-cowie/cidrator/cmd/fw"
	"github.com/euan-cowie/cidrator/cmd/scan"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cidrator",
	Short: "Comprehensive network analysis and manipulation toolkit",
	Long: `Cidrator is a comprehensive CLI toolkit for network analysis and manipulation.

Available command groups:
- cidr: IPv4/IPv6 CIDR network analysis (explain, expand, contains, count, overlaps, divide)
- dns: DNS analysis and lookup tools (coming soon)
- scan: Network scanning and discovery (coming soon)  
- fw: Firewall rule generation and analysis (coming soon)

Each command group provides specialized tools for different aspects of network operations.
Use 'cidrator <command> --help' for detailed information about each command group.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Add command groups
	rootCmd.AddCommand(cidr.CidrCmd)
	rootCmd.AddCommand(dns.DnsCmd)
	rootCmd.AddCommand(scan.ScanCmd)
	rootCmd.AddCommand(fw.FwCmd)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.cidrator.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cidrator" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".cidrator")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
