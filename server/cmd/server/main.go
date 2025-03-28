package server

import (
	"github.com/a13labs/m3uproxy/pkg/streamserver"
	"github.com/a13labs/m3uproxy/server/cmd"
	rootCmd "github.com/a13labs/m3uproxy/server/cmd"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the M3U proxy server",
	Long:  `Start the M3U proxy server that proxies M3U playlists and EPG data.`,
	Run: func(cmd *cobra.Command, args []string) {

		streamserver.Run(rootCmd.ConfigFile)
	},
}

func init() {
	cmd.RootCmd.AddCommand(serverCmd)
}
