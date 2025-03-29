package users

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(addCmd)
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new user",
	Run: func(cmd *cobra.Command, args []string) {
		// Add your code here to handle the "add" command
		if len(args) != 2 {
			cmd.PrintErrln("Usage: m3uproxycli users add <username> <password>")
			os.Exit(1)
		}

		cmd.Println("User Added")
		os.Exit(0)
	},
}
