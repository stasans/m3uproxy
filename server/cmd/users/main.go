package users

import (
	"github.com/a13labs/m3uproxy/server/cmd"

	"github.com/spf13/cobra"
)

var usersCmd = &cobra.Command{
	Use:   "users",
	Short: "Control the M3U proxy users",
	Long:  ``,
}

func init() {
	cmd.RootCmd.AddCommand(usersCmd)
}
