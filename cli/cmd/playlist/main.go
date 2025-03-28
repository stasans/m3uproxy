package playlist

import (
	"github.com/a13labs/m3uproxy/server/cmd"

	"github.com/spf13/cobra"
)

var playlistCmd = &cobra.Command{
	Use:   "playlist",
	Short: "Generate M3U playlists",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	cmd.RootCmd.AddCommand(playlistCmd)
}
