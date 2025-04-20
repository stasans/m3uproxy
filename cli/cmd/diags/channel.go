package diags

import (
	"fmt"
	"os"

	restapi "github.com/a13labs/m3uproxy/cli/cmd/rest"
	"github.com/spf13/cobra"
)

func init() {
	diagsCmd.AddCommand(channelCmd)
}

var channelCmd = &cobra.Command{
	Use:   "channel",
	Short: "Run diagnostics on M3U proxy channel",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.PrintErrln("Please provide a channel ID")
			cmd.PrintErrln("Usage: m3uproxy-cli diags channel <channel_id>")
			os.Exit(1)
		}
		err := restapi.Authenticate()
		if err != nil {
			cmd.PrintErrln("Error authenticating:", err)
			return
		}
		channelID := args[0]
		resp, err := restapi.Call("GET", fmt.Sprintf("/api/v1/diags/channel/%s", channelID), nil)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println(resp)
	},
}
