/*
Copyright © 2024 Alexandre Pires

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package users

import (
	"os"

	rootCmd "github.com/a13labs/m3uproxy/cmd"
	"github.com/a13labs/m3uproxy/pkg/auth"
	"github.com/a13labs/m3uproxy/pkg/streamserver"

	"github.com/spf13/cobra"
)

func init() {
	usersCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&usersFilePath, "users", "u", "users.json", "Path to the users JSON file")
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
