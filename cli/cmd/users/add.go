package users

import (
	"fmt"
	"os"

	restapi "github.com/a13labs/m3uproxy/cli/cmd/rest"
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
			cmd.PrintErrln("Usage: m3uproxy-cli users add <username> <password> <role>")
			os.Exit(1)
		}
		err := restapi.Authenticate()
		if err != nil {
			cmd.PrintErrln("Error authenticating:", err)
			return
		}
		username := args[0]
		password := args[1]
		role := "view"
		if len(args) == 3 {
			role = args[2]
		}
		body := map[string]string{
			"username": username,
			"password": password,
			"role":     role,
		}
		resp, err := restapi.Call("POST", fmt.Sprintf("/api/v1/user/%s", username), body)
		if err != nil {
			fmt.Println("Error:", err)
			os.Exit(1)
		}
		fmt.Println(resp)
		os.Exit(0)
	},
}
