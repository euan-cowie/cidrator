package cmd

import (
	"errors"
	"os"

	"github.com/euan-cowie/cidrator/cmd/cidr"
	"github.com/euan-cowie/cidrator/cmd/dns"
	"github.com/euan-cowie/cidrator/cmd/mtu"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cidrator",
	Short: "CIDR, DNS, and Path MTU diagnostics",
	Long: `Cidrator is a CLI for practical network diagnostics.

It provides focused tools for CIDR inspection, DNS queries, and Path MTU analysis.
Use 'cidrator <command> --help' for command-specific details.`,
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
	rootCmd.AddCommand(mtu.MTUCmd)
	rootCmd.AddCommand(dns.DNSCmd)

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

	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFound viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFound) {
			cobra.CheckErr(err)
		}
	}
}
