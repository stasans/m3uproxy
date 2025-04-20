package users

import (
	"fmt"
	"os"

	restapi "github.com/a13labs/m3uproxy/cli/cmd/rest"
	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(passwordCmd)
}

var passwordCmd = &cobra.Command{
	Use:   "password",
	Short: "Change a user password",
	Run: func(cmd *cobra.Command, args []string) {
		// Add your code here to handle the "add" command
		if len(args) != 2 {
			cmd.PrintErrln("Usage: m3uproxy-cli users add <username> <password>")
			os.Exit(1)
		}
		err := restapi.Authenticate()
		if err != nil {
			cmd.PrintErrln("Error authenticating:", err)
			return
		}
		username := args[0]
		password := args[1]
		body := map[string]string{
			"username": username,
			"password": password,
			"role":     "",
		}
		resp, err := restapi.Call("PUT", fmt.Sprintf("/api/v1/user/%s", username), body)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println(resp)
		os.Exit(0)
	},
}
