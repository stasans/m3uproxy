package cmd

import (
	"os"

	restapi "github.com/a13labs/m3uproxy/cli/cmd/rest"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "m3uproxy-cli",
	Short: "Cli for m3uproxy",
	Long:  `m3uproxy is a simple HTTP server that proxies M3U streams.`,
}

func Execute() {

	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&restapi.Config.Host, "api-host", "http://localhost:8080", "API host")
	RootCmd.PersistentFlags().StringVar(&restapi.Config.Username, "api-user", "admin", "API username")
	RootCmd.PersistentFlags().StringVar(&restapi.Config.Password, "api-password", "admin", "API password")
}
