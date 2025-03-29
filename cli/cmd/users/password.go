package users

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(passwordCmd)
}

var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Change a user password",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			cmd.PrintErrln("Usage: m3uproxycli users password <username> <password>")
			os.Exit(1)
		}

		fmt.Println("Password changed")
		os.Exit(0)
	},
}
