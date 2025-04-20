package users

import (
	"fmt"
	"os"

	restapi "github.com/a13labs/m3uproxy/cli/cmd/rest"
	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List users",
	Run: func(cmd *cobra.Command, args []string) {
		err := restapi.Authenticate()
		if err != nil {
			cmd.PrintErrln("Error authenticating:", err)
			return
		}
		resp, err := restapi.Call("GET", "/api/v1/users", nil)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println(resp)
	},
}
