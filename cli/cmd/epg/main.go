package epg

import (
	"github.com/a13labs/m3uproxy/cli/cmd"

	"github.com/spf13/cobra"
)

var epgCmd = &cobra.Command{
	Use:   "epg",
	Short: "Generate M3U epg",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

	},
}

func init() {
	cmd.RootCmd.AddCommand(epgCmd)
}
