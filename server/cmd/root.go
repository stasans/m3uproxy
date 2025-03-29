package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var ConfigFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "m3uproxy",
	Short: "A simple HTTP server that proxies M3U streams",
	Long:  `m3uproxy is a simple HTTP server that proxies M3U streams.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {

	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&ConfigFile, "config", "c", "", "config file (default is m3uproxy.json)")
}
