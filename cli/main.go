package main

import (
	"github.com/a13labs/m3uproxy/cli/cmd"

	_ "github.com/a13labs/m3uproxy/cli/cmd/config"
	_ "github.com/a13labs/m3uproxy/cli/cmd/diags"
	_ "github.com/a13labs/m3uproxy/cli/cmd/users"
)

func main() {
	cmd.Execute()
}
