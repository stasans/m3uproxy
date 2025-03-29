package server

import (
	"github.com/a13labs/m3uproxy/cli/cmd"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the M3U proxy server",
	Long:  `Start the M3U proxy server that proxies M3U playlists and EPG data.`,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	cmd.RootCmd.AddCommand(serverCmd)
}
