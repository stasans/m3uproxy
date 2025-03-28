package users

import (
	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Run: func(cmd *cobra.Command, args []string) {

	},
}
