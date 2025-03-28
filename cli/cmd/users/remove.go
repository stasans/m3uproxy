package users

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(removeCmd)
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove a user",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			cmd.PrintErrln("Usage: m3uproxycli users remove <username>")
			os.Exit(1)
		}

		fmt.Println("User removed")
		os.Exit(0)
	},
}
