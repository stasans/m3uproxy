package users

import (
	"os"

	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/streamserver"
	rootCmd "github.com/a13labs/m3uproxy/server/cmd"

	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Run: func(cmd *cobra.Command, args []string) {

		err := streamserver.LoadServerConfig(rootCmd.ConfigFile)
		if err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}

		err = auth.InitializeAuth(streamserver.Config.Auth)
		if err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}

		users, err := auth.GetUsers()
		if err != nil {
			cmd.PrintErrln(err)
			os.Exit(1)
		}
		for _, user := range users {
			cmd.OutOrStdout().Write([]byte(user + "\n"))
		}
		os.Exit(0)
	},
}
